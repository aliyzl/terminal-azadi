package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/leejooy96/azad/internal/config"
	"github.com/leejooy96/azad/internal/engine"
	"github.com/leejooy96/azad/internal/killswitch"
	"github.com/leejooy96/azad/internal/lifecycle"
	"github.com/leejooy96/azad/internal/protocol"
	"github.com/leejooy96/azad/internal/serverstore"
	"github.com/leejooy96/azad/internal/sysproxy"
)

// connectServerCmd returns a tea.Cmd that runs the full connection flow
// for a specific server: start engine, set system proxy, write state,
// persist preferences.
func connectServerCmd(srv protocol.Server, eng *engine.Engine, cfg *config.Config, store *serverstore.Store) tea.Cmd {
	return func() tea.Msg {
		// Start the proxy engine.
		if err := eng.Start(context.Background(), srv, cfg.Proxy.SOCKSPort, cfg.Proxy.HTTPPort); err != nil {
			return connectResultMsg{Err: err}
		}

		// Detect network service for system proxy.
		svc, svcErr := sysproxy.DetectNetworkService()

		// Write proxy state for crash recovery (before setting proxy).
		if svcErr == nil {
			tuiWriteProxyState(svc, cfg.Proxy.SOCKSPort, cfg.Proxy.HTTPPort)
		}

		// Set system proxy (best-effort).
		if svcErr == nil {
			_ = sysproxy.SetSystemProxy(svc, cfg.Proxy.SOCKSPort, cfg.Proxy.HTTPPort)
		}

		// Persist preferences (best-effort).
		cfg.Server.LastUsed = srv.ID
		configPath, err := config.FilePath()
		if err == nil {
			_ = config.Save(cfg, configPath)
		}

		srv.LastConnected = time.Now()
		_ = store.UpdateServer(srv)

		// Verify IP (best-effort, ignore result).
		_, _ = engine.VerifyIP(cfg.Proxy.SOCKSPort)

		return connectResultMsg{Err: nil}
	}
}

// disconnectCmd returns a tea.Cmd that stops the engine, unsets the system
// proxy, and removes the proxy state file. If ksActive is true, it disables
// the kill switch before stopping.
func disconnectCmd(eng *engine.Engine, ksActive ...bool) tea.Cmd {
	disableKS := len(ksActive) > 0 && ksActive[0]
	return func() tea.Msg {
		// Disable kill switch before stopping engine.
		if disableKS {
			_ = killswitch.Disable()
		}

		// Detect network service.
		svc, svcErr := sysproxy.DetectNetworkService()

		// Stop engine.
		_ = eng.Stop()

		// Unset system proxy (best-effort).
		if svcErr == nil {
			_ = sysproxy.UnsetSystemProxy(svc)
		}

		// Remove state file.
		tuiRemoveStateFile()

		return disconnectMsg{}
	}
}

// autoConnectCmd returns a tea.Cmd that resolves the best server and
// connects on TUI startup. If the store is empty, it silently skips.
func autoConnectCmd(store *serverstore.Store, eng *engine.Engine, cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		servers := store.List()
		if len(servers) == 0 {
			return autoConnectMsg{} // skip silently
		}

		// Resolve best server: LastUsed > lowest positive LatencyMs > first.
		server := resolveBestServer(servers, cfg, store)

		// Start the proxy engine.
		if err := eng.Start(context.Background(), *server, cfg.Proxy.SOCKSPort, cfg.Proxy.HTTPPort); err != nil {
			return autoConnectMsg{Err: err}
		}

		// Detect network service for system proxy.
		svc, svcErr := sysproxy.DetectNetworkService()

		// Write proxy state for crash recovery.
		if svcErr == nil {
			tuiWriteProxyState(svc, cfg.Proxy.SOCKSPort, cfg.Proxy.HTTPPort)
		}

		// Set system proxy (best-effort).
		if svcErr == nil {
			_ = sysproxy.SetSystemProxy(svc, cfg.Proxy.SOCKSPort, cfg.Proxy.HTTPPort)
		}

		// Persist preferences (best-effort).
		cfg.Server.LastUsed = server.ID
		configPath, err := config.FilePath()
		if err == nil {
			_ = config.Save(cfg, configPath)
		}

		server.LastConnected = time.Now()
		_ = store.UpdateServer(*server)

		return autoConnectMsg{ServerID: server.ID}
	}
}

// resolveBestServer picks the best server to auto-connect to.
// Resolution order: LastUsed from config > lowest positive LatencyMs > first server.
// This mirrors the logic in cli/connect.go findServer (without the explicit arg tier).
func resolveBestServer(servers []protocol.Server, cfg *config.Config, store *serverstore.Store) *protocol.Server {
	// Try last-used from config.
	if cfg.Server.LastUsed != "" {
		if srv, ok := store.FindByID(cfg.Server.LastUsed); ok {
			return srv
		}
	}

	// Try lowest latency server (positive LatencyMs only).
	bestIdx := -1
	bestLatency := 0
	for i := range servers {
		if servers[i].LatencyMs > 0 {
			if bestIdx == -1 || servers[i].LatencyMs < bestLatency {
				bestIdx = i
				bestLatency = servers[i].LatencyMs
			}
		}
	}
	if bestIdx >= 0 {
		return &servers[bestIdx]
	}

	// Fall back to first server.
	return &servers[0]
}

// enableKillSwitchCmd returns a tea.Cmd that enables the kill switch.
// It resolves the connected server IP and calls killswitch.Enable.
func enableKillSwitchCmd(eng *engine.Engine, cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		_, srv, _ := eng.Status()
		if srv == nil {
			return killSwitchResultMsg{Err: fmt.Errorf("not connected")}
		}

		// Resolve server address to IP.
		resolvedIP := srv.Address
		if ips, err := net.LookupHost(srv.Address); err == nil && len(ips) > 0 {
			resolvedIP = ips[0]
		}

		if err := killswitch.Enable(resolvedIP, srv.Port); err != nil {
			return killSwitchResultMsg{Err: err}
		}

		// Update proxy state with kill switch fields.
		tuiWriteProxyStateWithKS(cfg.Proxy.SOCKSPort, cfg.Proxy.HTTPPort, true, resolvedIP, srv.Port)

		return killSwitchResultMsg{Enabled: true}
	}
}

// disableKillSwitchCmd returns a tea.Cmd that disables the kill switch.
func disableKillSwitchCmd() tea.Cmd {
	return func() tea.Msg {
		if err := killswitch.Disable(); err != nil {
			return killSwitchResultMsg{Err: err}
		}

		// Update proxy state to clear kill switch fields.
		tuiWriteProxyStateWithKS(0, 0, false, "", 0)

		return killSwitchResultMsg{Enabled: false}
	}
}

// tuiWriteProxyStateWithKS writes the proxy state with kill switch fields.
func tuiWriteProxyStateWithKS(socksPort, httpPort int, ksActive bool, serverAddr string, serverPort int) {
	statePath, err := config.StateFilePath()
	if err != nil {
		return
	}

	// Read existing state to preserve non-kill-switch fields.
	var state lifecycle.ProxyState
	if data, err := os.ReadFile(statePath); err == nil {
		_ = json.Unmarshal(data, &state)
	}

	// Update kill switch fields.
	state.KillSwitchActive = ksActive
	state.ServerAddress = serverAddr
	state.ServerPort = serverPort

	// Update proxy fields if provided (non-zero).
	if socksPort > 0 {
		state.SOCKSPort = socksPort
	}
	if httpPort > 0 {
		state.HTTPPort = httpPort
	}

	data, err := json.Marshal(state)
	if err != nil {
		return
	}

	_ = os.WriteFile(statePath, data, 0600)
}

// tuiWriteProxyState writes the proxy state file for crash recovery.
// Duplicated from cli/connect.go to avoid exporting internal helpers.
func tuiWriteProxyState(service string, socksPort, httpPort int) {
	statePath, err := config.StateFilePath()
	if err != nil {
		return
	}

	state := lifecycle.ProxyState{
		ProxySet:       true,
		SOCKSPort:      socksPort,
		HTTPPort:       httpPort,
		NetworkService: service,
		PID:            os.Getpid(),
	}

	data, err := json.Marshal(state)
	if err != nil {
		return
	}

	_ = os.WriteFile(statePath, data, 0600)
}

// tuiRemoveStateFile deletes the proxy state file after clean shutdown.
// Duplicated from cli/connect.go to avoid exporting internal helpers.
func tuiRemoveStateFile() {
	statePath, err := config.StateFilePath()
	if err != nil {
		return
	}
	_ = os.Remove(statePath)
}

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
	"github.com/leejooy96/azad/internal/splittunnel"
	"github.com/leejooy96/azad/internal/sysproxy"
)

// connectServerCmd returns a tea.Cmd that runs the full connection flow
// for a specific server: start engine, set system proxy, write state,
// persist preferences. splitCfg is passed to the engine for split tunnel routing.
func connectServerCmd(srv protocol.Server, eng *engine.Engine, cfg *config.Config, store *serverstore.Store, splitCfg *splittunnel.Config) tea.Cmd {
	return func() tea.Msg {
		// Start the proxy engine with optional split tunnel config.
		if err := eng.Start(context.Background(), srv, cfg.Proxy.SOCKSPort, cfg.Proxy.HTTPPort, splitCfg); err != nil {
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
// splitCfg is passed to the engine for split tunnel routing.
func autoConnectCmd(store *serverstore.Store, eng *engine.Engine, cfg *config.Config, splitCfg *splittunnel.Config) tea.Cmd {
	return func() tea.Msg {
		servers := store.List()
		if len(servers) == 0 {
			return autoConnectMsg{} // skip silently
		}

		// Resolve best server: LastUsed > lowest positive LatencyMs > first.
		server := resolveBestServer(servers, cfg, store)

		// Start the proxy engine with optional split tunnel config.
		if err := eng.Start(context.Background(), *server, cfg.Proxy.SOCKSPort, cfg.Proxy.HTTPPort, splitCfg); err != nil {
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
// bypassIPs are optional IPs/CIDRs to allow direct traffic (split tunnel coordination).
func enableKillSwitchCmd(eng *engine.Engine, cfg *config.Config, bypassIPs ...[]string) tea.Cmd {
	var bypass []string
	if len(bypassIPs) > 0 {
		bypass = bypassIPs[0]
	}
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

		if err := killswitch.Enable(resolvedIP, srv.Port, bypass); err != nil {
			return killSwitchResultMsg{Err: err}
		}

		// Update proxy state with kill switch fields.
		tuiWriteProxyStateWithKS(cfg.Proxy.SOCKSPort, cfg.Proxy.HTTPPort, true, resolvedIP, srv.Port)

		return killSwitchResultMsg{Enabled: true}
	}
}

// extractBypassIPs returns IP/CIDR rules from the split tunnel config
// when active in exclusive mode. Returns nil if inactive or inclusive mode.
// Domain rules are resolved best-effort.
func extractBypassIPs(cfg *config.Config) []string {
	if !cfg.SplitTunnel.Enabled || len(cfg.SplitTunnel.Rules) == 0 {
		return nil
	}
	if cfg.SplitTunnel.Mode == "inclusive" {
		return nil
	}
	// Exclusive mode: listed rules bypass VPN, so they need pf exceptions.
	var ips []string
	for _, r := range cfg.SplitTunnel.Rules {
		switch splittunnel.RuleType(r.Type) {
		case splittunnel.RuleTypeIP, splittunnel.RuleTypeCIDR:
			ips = append(ips, r.Value)
		case splittunnel.RuleTypeDomain:
			// Best-effort DNS resolution.
			if resolved, err := net.LookupHost(r.Value); err == nil {
				ips = append(ips, resolved...)
			}
		case splittunnel.RuleTypeWildcard:
			// Wildcards cannot be resolved meaningfully; skip.
		}
	}
	return ips
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

package tui

import (
	"context"
	"encoding/json"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/leejooy96/azad/internal/config"
	"github.com/leejooy96/azad/internal/engine"
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
// proxy, and removes the proxy state file.
func disconnectCmd(eng *engine.Engine) tea.Cmd {
	return func() tea.Msg {
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

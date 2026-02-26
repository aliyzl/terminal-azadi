package cli

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/leejooy96/azad/internal/config"
	"github.com/leejooy96/azad/internal/engine"
	"github.com/leejooy96/azad/internal/killswitch"
	"github.com/leejooy96/azad/internal/lifecycle"
	"github.com/leejooy96/azad/internal/protocol"
	"github.com/leejooy96/azad/internal/serverstore"
	"github.com/leejooy96/azad/internal/sysproxy"
	"github.com/spf13/cobra"
)

var killSwitchFlag bool

func newConnectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "connect [server-name]",
		Short: "Connect to a VPN server",
		Long:  "Connect to a VPN server using Xray-core proxy. Optionally specify a server name or ID.",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runConnect,
	}
	cmd.Flags().BoolVar(&killSwitchFlag, "kill-switch", false, "Enable kill switch to block all non-VPN traffic")
	return cmd
}

func runConnect(cmd *cobra.Command, args []string) error {
	// Load config.
	configPath, err := config.FilePath()
	if err != nil {
		return fmt.Errorf("resolving config path: %w", err)
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Load server store.
	dataDir, err := config.DataDir()
	if err != nil {
		return fmt.Errorf("resolving data dir: %w", err)
	}
	storePath := filepath.Join(dataDir, "servers.json")
	store := serverstore.New(storePath)
	if err := store.Load(); err != nil {
		return fmt.Errorf("loading servers: %w", err)
	}

	// Find server to connect to.
	server, err := findServer(store, cfg, args)
	if err != nil {
		return err
	}

	fmt.Printf("Connecting to %s (%s:%d)...\n", server.Name, server.Address, server.Port)

	// Start the proxy engine.
	eng := &engine.Engine{}
	ctx := cmd.Context()
	if err := eng.Start(ctx, *server, cfg.Proxy.SOCKSPort, cfg.Proxy.HTTPPort); err != nil {
		return fmt.Errorf("starting proxy: %w", err)
	}

	fmt.Printf("Proxy started on SOCKS5://127.0.0.1:%d and HTTP://127.0.0.1:%d\n",
		cfg.Proxy.SOCKSPort, cfg.Proxy.HTTPPort)

	// Detect network service for system proxy.
	svc, svcErr := sysproxy.DetectNetworkService()

	// Resolve server IP for kill switch (DNS resolution before enabling).
	resolvedIP := server.Address
	if killSwitchFlag {
		if ips, err := net.LookupHost(server.Address); err == nil && len(ips) > 0 {
			resolvedIP = ips[0]
		}
	}

	// Write ProxyState BEFORE setting proxy for crash safety.
	if svcErr == nil {
		writeProxyState(svc, cfg.Proxy.SOCKSPort, cfg.Proxy.HTTPPort, killSwitchFlag, resolvedIP, server.Port)
	}

	// Set system proxy.
	proxySetOK := false
	if svcErr != nil {
		fmt.Printf("Warning: could not detect network service: %v\n", svcErr)
	} else if err := sysproxy.SetSystemProxy(svc, cfg.Proxy.SOCKSPort, cfg.Proxy.HTTPPort); err != nil {
		fmt.Printf("Warning: could not set system proxy: %v\n", err)
	} else {
		proxySetOK = true
	}

	// Enable kill switch if requested.
	if killSwitchFlag {
		if err := killswitch.Enable(resolvedIP, server.Port); err != nil {
			fmt.Printf("Warning: could not enable kill switch: %v\n", err)
		} else {
			fmt.Println("Kill switch enabled -- all non-VPN traffic blocked.")
		}
	}

	// Fetch direct IP (best-effort).
	directIP, directErr := engine.GetDirectIP()

	// Verify connection through the proxy.
	proxyIP, verifyErr := engine.VerifyIP(cfg.Proxy.SOCKSPort)
	if verifyErr == nil {
		if directErr == nil && directIP != "" {
			if directIP != proxyIP {
				fmt.Printf("Direct IP: %s -> Proxy IP: %s (routing confirmed)\n", directIP, proxyIP)
			} else {
				fmt.Println("Warning: Proxy IP matches direct IP -- routing may not be working")
			}
		} else {
			fmt.Printf("Connected! Your IP: %s\n", proxyIP)
		}
	} else {
		fmt.Printf("Warning: Could not verify connection: %v\n", verifyErr)
	}

	// Persist connection preferences (non-fatal on failure).
	cfg.Server.LastUsed = server.ID
	if err := config.Save(cfg, configPath); err != nil {
		fmt.Printf("Warning: could not save config: %v\n", err)
	}

	server.LastConnected = time.Now()
	if err := store.UpdateServer(*server); err != nil {
		fmt.Printf("Warning: could not update server: %v\n", err)
	}

	fmt.Printf("Status: %s | Server: %s | Press Ctrl+C to disconnect\n",
		"connected", server.Name)

	// Wait for context cancellation (SIGINT/SIGTERM).
	<-ctx.Done()

	fmt.Println("Disconnecting...")

	// Disable kill switch before stopping engine/proxy.
	if killSwitchFlag {
		if err := killswitch.Disable(); err != nil {
			fmt.Printf("Warning: could not disable kill switch: %v\n", err)
		} else {
			fmt.Println("Kill switch disabled.")
		}
	}

	// Stop the engine.
	if err := eng.Stop(); err != nil {
		fmt.Printf("Warning: error stopping engine: %v\n", err)
	}

	// Unset system proxy.
	if proxySetOK {
		if err := sysproxy.UnsetSystemProxy(svc); err != nil {
			fmt.Printf("Warning: could not unset system proxy: %v\n", err)
		}
	}

	// Remove .state.json.
	removeStateFile()

	fmt.Println("Disconnected.")
	return nil
}

// findServer resolves which server to connect to based on args or config.
func findServer(store *serverstore.Store, cfg *config.Config, args []string) (*protocol.Server, error) {
	servers := store.List()
	if len(servers) == 0 {
		return nil, fmt.Errorf("no servers available. Add one with: azad servers add <uri>")
	}

	// If a name/ID argument was provided, search for it.
	if len(args) > 0 && args[0] != "" {
		query := args[0]

		// Try exact ID match first.
		if srv, ok := store.FindByID(query); ok {
			return srv, nil
		}

		// Try case-insensitive name contains match.
		queryLower := strings.ToLower(query)
		for i := range servers {
			if strings.Contains(strings.ToLower(servers[i].Name), queryLower) {
				return &servers[i], nil
			}
		}

		return nil, fmt.Errorf("server %q not found", query)
	}

	// No args: try last-used from config.
	if cfg.Server.LastUsed != "" {
		if srv, ok := store.FindByID(cfg.Server.LastUsed); ok {
			return srv, nil
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
		return &servers[bestIdx], nil
	}

	// Fall back to first server.
	return &servers[0], nil
}

// writeProxyState writes the proxy state file for crash recovery.
func writeProxyState(service string, socksPort, httpPort int, ksActive bool, serverAddr string, serverPort int) {
	statePath, err := config.StateFilePath()
	if err != nil {
		return
	}

	state := lifecycle.ProxyState{
		ProxySet:         true,
		SOCKSPort:        socksPort,
		HTTPPort:         httpPort,
		NetworkService:   service,
		PID:              os.Getpid(),
		KillSwitchActive: ksActive,
		ServerAddress:    serverAddr,
		ServerPort:       serverPort,
	}

	data, err := json.Marshal(state)
	if err != nil {
		return
	}

	_ = os.WriteFile(statePath, data, 0600)
}

// removeStateFile deletes the proxy state file after clean shutdown.
func removeStateFile() {
	statePath, err := config.StateFilePath()
	if err != nil {
		return
	}
	_ = os.Remove(statePath)
}

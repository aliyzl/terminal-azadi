package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/leejooy96/azad/internal/config"
	"github.com/leejooy96/azad/internal/engine"
	"github.com/leejooy96/azad/internal/lifecycle"
	"github.com/leejooy96/azad/internal/serverstore"
	"github.com/leejooy96/azad/internal/tui"
	"github.com/spf13/cobra"
)

var (
	cleanup       bool
	resetTerminal bool
)

// NewRootCmd creates and returns the root cobra command for azad.
func NewRootCmd(version string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "azad",
		Short: "Beautiful terminal VPN client",
		Long:  "Azad — One command to connect to the fastest VPN server through a stunning terminal interface",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Kill switch recovery detection: inform user if kill switch is active from a previous session.
			if !cleanup {
				if statePath, err := config.StateFilePath(); err == nil {
					if data, err := os.ReadFile(statePath); err == nil {
						var state lifecycle.ProxyState
						if err := json.Unmarshal(data, &state); err == nil && state.KillSwitchActive {
							fmt.Println("Kill switch is active from a previous session. Internet is blocked.")
							fmt.Println("Reconnecting to restore internet through VPN...")
						}
					}
				}
			}

			if cleanup {
				configDir, err := config.Dir()
				if err != nil {
					return err
				}
				if err := lifecycle.RunCleanup(configDir); err != nil {
					return err
				}
				os.Exit(0)
			}
			if resetTerminal {
				if err := lifecycle.RunResetTerminal(); err != nil {
					return err
				}
				os.Exit(0)
			}
			return nil
		},
		// RunE launches the TUI when no subcommand is given.
		// PersistentPreRunE intercepts --cleanup and --reset-terminal before this runs.
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load config
			configPath, err := config.FilePath()
			if err != nil {
				return fmt.Errorf("resolving config path: %w", err)
			}
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			// Load server store
			dataDir, err := config.DataDir()
			if err != nil {
				return fmt.Errorf("resolving data dir: %w", err)
			}
			storePath := filepath.Join(dataDir, "servers.json")
			store := serverstore.New(storePath)
			if err := store.Load(); err != nil {
				return fmt.Errorf("loading servers: %w", err)
			}

			// Create engine
			eng := &engine.Engine{}

			// Launch TUI
			m := tui.New(store, eng, cfg)
			p := tea.NewProgram(m)
			_, err = p.Run()
			return err
		},
	}

	rootCmd.Version = version

	rootCmd.PersistentFlags().BoolVar(&cleanup, "cleanup", false, "Remove dirty proxy state from a previous crash")
	rootCmd.PersistentFlags().BoolVar(&resetTerminal, "reset-terminal", false, "Restore terminal to usable state")

	rootCmd.AddCommand(newConnectCmd(), newServersCmd(), newConfigCmd(), newSplitTunnelCmd())

	return rootCmd
}

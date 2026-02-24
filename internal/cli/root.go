package cli

import (
	"fmt"
	"os"

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
			if cleanup {
				fmt.Println("cleanup: not yet implemented")
				os.Exit(0)
			}
			if resetTerminal {
				fmt.Println("reset-terminal: not yet implemented")
				os.Exit(0)
			}
			return nil
		},
		// RunE is needed so the root command is runnable (not help-only).
		// Without it, `azad --cleanup` would show help instead of running PersistentPreRunE.
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	rootCmd.Version = version

	rootCmd.PersistentFlags().BoolVar(&cleanup, "cleanup", false, "Remove dirty proxy state from a previous crash")
	rootCmd.PersistentFlags().BoolVar(&resetTerminal, "reset-terminal", false, "Restore terminal to usable state")

	rootCmd.AddCommand(newConnectCmd(), newServersCmd(), newConfigCmd())

	return rootCmd
}

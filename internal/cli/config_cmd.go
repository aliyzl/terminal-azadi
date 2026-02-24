package cli

import (
	"fmt"

	"github.com/leejooy96/azad/internal/config"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "View and modify configuration",
		Long:  "View current configuration values or modify settings.",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := config.FilePath()
			if err != nil {
				return fmt.Errorf("resolving config path: %w", err)
			}

			cfg, err := config.Load(path)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			fmt.Printf("Config file: %s\n\n", path)
			fmt.Println("Current configuration:")
			fmt.Printf("  proxy:\n")
			fmt.Printf("    socks_port: %d\n", cfg.Proxy.SOCKSPort)
			fmt.Printf("    http_port:  %d\n", cfg.Proxy.HTTPPort)
			fmt.Printf("  server:\n")
			fmt.Printf("    last_used:  %s\n", cfg.Server.LastUsed)

			return nil
		},
	}
}

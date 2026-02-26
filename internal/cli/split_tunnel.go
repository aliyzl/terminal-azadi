package cli

import (
	"fmt"

	"github.com/leejooy96/azad/internal/config"
	"github.com/leejooy96/azad/internal/splittunnel"
	"github.com/spf13/cobra"
)

func newSplitTunnelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "split-tunnel",
		Short:   "Manage split tunnel rules",
		Aliases: []string{"st"},
	}
	cmd.AddCommand(
		newSTAddCmd(),
		newSTRemoveCmd(),
		newSTListCmd(),
		newSTModeCmd(),
		newSTEnableCmd(),
		newSTDisableCmd(),
		newSTClearCmd(),
	)
	return cmd
}

func newSTAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <rule>",
		Short: "Add a split tunnel rule (IP, CIDR, domain, or *.domain)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, configPath, err := loadConfig()
			if err != nil {
				return err
			}

			// Parse and validate the rule.
			rule, err := splittunnel.ParseRule(args[0])
			if err != nil {
				return fmt.Errorf("invalid rule: %w", err)
			}

			// Check for duplicate.
			for _, r := range cfg.SplitTunnel.Rules {
				if r.Value == rule.Value {
					return fmt.Errorf("rule already exists: %s", rule.Value)
				}
			}

			cfg.SplitTunnel.Rules = append(cfg.SplitTunnel.Rules, config.SplitTunnelRule{
				Value: rule.Value,
				Type:  string(rule.Type),
			})

			if err := config.Save(cfg, configPath); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			fmt.Printf("Added %s rule: %s\n", rule.Type, rule.Value)
			return nil
		},
	}
}

func newSTRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <rule>",
		Short: "Remove a split tunnel rule by value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, configPath, err := loadConfig()
			if err != nil {
				return err
			}

			value := args[0]
			found := false
			var remaining []config.SplitTunnelRule
			for _, r := range cfg.SplitTunnel.Rules {
				if r.Value == value {
					found = true
					continue
				}
				remaining = append(remaining, r)
			}

			if !found {
				return fmt.Errorf("rule not found: %s", value)
			}

			cfg.SplitTunnel.Rules = remaining
			if err := config.Save(cfg, configPath); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			fmt.Printf("Removed rule: %s\n", value)
			return nil
		},
	}
}

func newSTListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List split tunnel rules",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := loadConfig()
			if err != nil {
				return err
			}

			// Status
			if cfg.SplitTunnel.Enabled {
				fmt.Println("Split tunneling: enabled")
			} else {
				fmt.Println("Split tunneling: disabled")
			}

			// Mode
			mode := cfg.SplitTunnel.Mode
			if mode == "" {
				mode = "exclusive"
			}
			fmt.Printf("Mode: %s\n", mode)

			// Rules
			if len(cfg.SplitTunnel.Rules) == 0 {
				fmt.Println("\nNo rules configured")
				return nil
			}

			fmt.Printf("\nRules (%d):\n", len(cfg.SplitTunnel.Rules))
			for i, r := range cfg.SplitTunnel.Rules {
				fmt.Printf("  %d. %-30s [%s]\n", i+1, r.Value, r.Type)
			}

			return nil
		},
	}
}

func newSTModeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mode <exclusive|inclusive>",
		Short: "Set split tunnel mode",
		Long: `Set split tunnel mode:
  exclusive: Listed rules bypass VPN (go direct)
  inclusive: Listed rules use VPN, everything else goes direct`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mode := args[0]
			if mode != "exclusive" && mode != "inclusive" {
				return fmt.Errorf("invalid mode %q: must be 'exclusive' or 'inclusive'", mode)
			}

			cfg, configPath, err := loadConfig()
			if err != nil {
				return err
			}

			cfg.SplitTunnel.Mode = mode
			if err := config.Save(cfg, configPath); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			fmt.Printf("Split tunnel mode set to %s\n", mode)
			return nil
		},
	}
}

func newSTEnableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "enable",
		Short: "Enable split tunneling",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, configPath, err := loadConfig()
			if err != nil {
				return err
			}

			cfg.SplitTunnel.Enabled = true
			if err := config.Save(cfg, configPath); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			fmt.Println("Split tunneling enabled")
			if len(cfg.SplitTunnel.Rules) == 0 {
				fmt.Println("Warning: No rules configured. Add rules with: azad split-tunnel add <rule>")
			}
			fmt.Println("Reconnect required for changes to take effect")
			return nil
		},
	}
}

func newSTDisableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disable",
		Short: "Disable split tunneling",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, configPath, err := loadConfig()
			if err != nil {
				return err
			}

			cfg.SplitTunnel.Enabled = false
			if err := config.Save(cfg, configPath); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			fmt.Println("Split tunneling disabled")
			return nil
		},
	}
}

func newSTClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Remove all split tunnel rules",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, configPath, err := loadConfig()
			if err != nil {
				return err
			}

			cfg.SplitTunnel.Rules = nil
			if err := config.Save(cfg, configPath); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			fmt.Println("All split tunnel rules cleared")
			return nil
		},
	}
}

// loadConfig loads the application config and returns it along with the config path.
func loadConfig() (*config.Config, string, error) {
	configPath, err := config.FilePath()
	if err != nil {
		return nil, "", fmt.Errorf("resolving config path: %w", err)
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, "", fmt.Errorf("loading config: %w", err)
	}
	return cfg, configPath, nil
}

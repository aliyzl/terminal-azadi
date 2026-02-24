package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "View and modify configuration",
		Long:  "View current configuration values or modify settings.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("config: not yet implemented")
			return nil
		},
	}
}

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newServersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "servers",
		Short: "Manage VPN servers",
		Long:  "List, add, remove, and ping VPN servers.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("servers: not yet implemented")
			return nil
		},
	}
}

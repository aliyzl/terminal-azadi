package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newConnectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "connect [server]",
		Short: "Connect to a VPN server",
		Long:  "Connect to a VPN server using Xray-core proxy. Optionally specify a server name or index.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("connect: not yet implemented")
			return nil
		},
	}
}

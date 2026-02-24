package main

import (
	"os"

	"github.com/leejooy96/azad/internal/cli"

	// Import xray-core to validate it compiles as a library dependency.
	_ "github.com/xtls/xray-core/core"

	// Register all xray-core features (protocol handlers, transports, config loaders).
	_ "github.com/xtls/xray-core/main/distro/all"
)

// version is set at build time via -ldflags.
var version = "dev"

func main() {
	cmd := cli.NewRootCmd(version)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/leejooy96/azad/internal/cli"
	"github.com/leejooy96/azad/internal/lifecycle"

	// Import xray-core to validate it compiles as a library dependency.
	_ "github.com/xtls/xray-core/core"

	// Register all xray-core features (protocol handlers, transports, config loaders).
	_ "github.com/xtls/xray-core/main/distro/all"
)

// version is set at build time via -ldflags.
var version = "dev"

func main() {
	// Set up signal-based context for graceful shutdown.
	ctx, cancel := lifecycle.WithShutdown(context.Background())
	defer cancel()

	// On context cancellation (SIGINT/SIGTERM), log the shutdown.
	go func() {
		<-ctx.Done()
		fmt.Fprintln(os.Stderr, "Shutting down gracefully...")
	}()

	cmd := cli.NewRootCmd(version)
	cmd.SetContext(ctx)
	if err := cmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}

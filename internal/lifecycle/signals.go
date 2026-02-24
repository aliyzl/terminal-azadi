package lifecycle

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// WithShutdown returns a context that cancels on SIGINT or SIGTERM.
// The returned cancel function should be deferred to release signal resources.
func WithShutdown(parent context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
}

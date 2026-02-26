package tui

import (
	"time"

	"github.com/leejooy96/azad/internal/protocol"
)

// pingStartMsg indicates a batch ping operation has started.
type pingStartMsg struct {
	Total int
}

// pingResultMsg carries the result of pinging a single server.
type pingResultMsg struct {
	ServerID  string
	LatencyMs int
	Err       error
}

// allPingsCompleteMsg indicates all server pings have finished.
type allPingsCompleteMsg struct{}

// serverAddedMsg indicates a server was successfully added.
type serverAddedMsg struct {
	Server protocol.Server
}

// serverRemovedMsg indicates a server was removed.
type serverRemovedMsg struct {
	ServerID string
}

// serversReplacedMsg indicates subscription servers were replaced.
type serversReplacedMsg struct {
	Count int
}

// subscriptionFetchedMsg carries the result of a subscription fetch.
type subscriptionFetchedMsg struct {
	Servers []protocol.Server
	Err     error
}

// connectResultMsg carries the result of a connection attempt.
type connectResultMsg struct {
	Err error
}

// disconnectMsg requests disconnection from the current server.
type disconnectMsg struct{}

// autoConnectMsg carries the result of the auto-connect attempt on TUI startup.
type autoConnectMsg struct {
	ServerID string // ID of the server auto-connected to (empty if skipped)
	Err      error  // nil on success, non-nil on failure, nil+empty ID means skipped
}

// tickMsg is sent on each uptime tick interval.
type tickMsg time.Time

// killSwitchResultMsg carries the result of a kill switch enable/disable operation.
type killSwitchResultMsg struct {
	Enabled bool
	Err     error
}

// splitTunnelSavedMsg indicates split tunnel config was saved to disk.
type splitTunnelSavedMsg struct{}

// errMsg carries a generic error.
type errMsg struct {
	Err error
}

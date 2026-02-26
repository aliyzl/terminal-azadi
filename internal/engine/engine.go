package engine

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/leejooy96/azad/internal/config"
	"github.com/leejooy96/azad/internal/geoasset"
	"github.com/leejooy96/azad/internal/protocol"
	"github.com/leejooy96/azad/internal/splittunnel"
	"github.com/xtls/xray-core/core"
)

// ConnectionStatus represents the current state of the proxy engine.
type ConnectionStatus int

const (
	// StatusDisconnected is the initial and final state.
	StatusDisconnected ConnectionStatus = iota
	// StatusConnecting is the transient state while starting the proxy.
	StatusConnecting
	// StatusConnected means the proxy is running and accepting connections.
	StatusConnected
	// StatusError means the proxy failed to start or encountered an error.
	StatusError
)

// String returns a human-readable representation of the connection status.
func (s ConnectionStatus) String() string {
	switch s {
	case StatusDisconnected:
		return "disconnected"
	case StatusConnecting:
		return "connecting"
	case StatusConnected:
		return "connected"
	case StatusError:
		return "error"
	default:
		return "unknown"
	}
}

// Engine manages the Xray-core proxy instance lifecycle.
type Engine struct {
	mu         sync.Mutex
	instance   *core.Instance
	logCapture *LogCapture
	status     ConnectionStatus
	server     *protocol.Server
	err        error
}

// Start creates and starts an Xray-core proxy instance for the given server.
// The engine transitions: Disconnected -> Connecting -> Connected (or Error).
// splitCfg is an optional variadic parameter for split tunnel configuration.
func (e *Engine) Start(ctx context.Context, srv protocol.Server, socksPort, httpPort int, splitCfg ...*splittunnel.Config) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.instance != nil {
		return fmt.Errorf("already connected")
	}

	e.status = StatusConnecting
	e.err = nil

	// Set XRAY_LOCATION_ASSET so Xray can find geoip.dat/geosite.dat.
	dataDir, err := config.DataDir()
	if err == nil {
		os.Setenv("XRAY_LOCATION_ASSET", dataDir)
	}

	// Ensure geo assets (geoip.dat, geosite.dat) exist before Xray init.
	// Xray panics if routing rules reference geoip:private with missing files.
	if err := geoasset.EnsureAssets(dataDir); err != nil {
		return fmt.Errorf("ensuring geo assets: %w", err)
	}

	// Extract split tunnel config if provided.
	var sc *splittunnel.Config
	if len(splitCfg) > 0 {
		sc = splitCfg[0]
	}

	// Create log capture pipe for access log.
	lc, accessLogPath, lcErr := NewLogCapture()
	if lcErr != nil {
		accessLogPath = "none"
	}

	closeLCOnError := func() {
		if lc != nil {
			lc.Close()
		}
	}

	// Build the Xray JSON config and load it into a core.Config.
	_, coreConfig, err := BuildConfig(srv, socksPort, httpPort, sc, accessLogPath)
	if err != nil {
		closeLCOnError()
		e.status = StatusError
		e.err = fmt.Errorf("building config: %w", err)
		return e.err
	}

	// Create the Xray instance from the protobuf config.
	instance, err := core.New(coreConfig)
	if err != nil {
		closeLCOnError()
		e.status = StatusError
		e.err = fmt.Errorf("creating xray instance: %w", err)
		return e.err
	}

	// Start the proxy.
	if err := instance.Start(); err != nil {
		_ = instance.Close()
		closeLCOnError()
		e.status = StatusError
		e.err = fmt.Errorf("starting xray instance: %w", err)
		return e.err
	}

	e.instance = instance
	e.logCapture = lc
	srvCopy := srv
	e.server = &srvCopy
	e.status = StatusConnected
	return nil
}

// Stop closes the Xray instance and transitions to Disconnected.
// A closed instance cannot be restarted; create a new Engine for each connection.
func (e *Engine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.instance == nil {
		e.status = StatusDisconnected
		return nil
	}

	err := e.instance.Close()
	if e.logCapture != nil {
		e.logCapture.Close()
		e.logCapture = nil
	}
	e.instance = nil
	e.server = nil
	e.status = StatusDisconnected
	e.err = nil
	return err
}

// Status returns the current connection status, the connected server (if any),
// and the last error (if in error state).
func (e *Engine) Status() (ConnectionStatus, *protocol.Server, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.status, e.server, e.err
}

// TrafficLog returns the last n access log entries.
// Returns nil if not connected or log capture is unavailable.
func (e *Engine) TrafficLog(n int) []LogEntry {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.logCapture == nil {
		return nil
	}
	return e.logCapture.Entries(n)
}

// ServerName returns the name of the currently connected server,
// or an empty string if not connected.
func (e *Engine) ServerName() string {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.server != nil {
		return e.server.Name
	}
	return ""
}

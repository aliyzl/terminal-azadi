package engine

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/leejooy96/azad/internal/config"
	"github.com/leejooy96/azad/internal/protocol"
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
	mu       sync.Mutex
	instance *core.Instance
	status   ConnectionStatus
	server   *protocol.Server
	err      error
}

// Start creates and starts an Xray-core proxy instance for the given server.
// The engine transitions: Disconnected -> Connecting -> Connected (or Error).
func (e *Engine) Start(ctx context.Context, srv protocol.Server, socksPort, httpPort int) error {
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

	// Build the Xray JSON config and load it into a core.Config.
	_, coreConfig, err := BuildConfig(srv, socksPort, httpPort)
	if err != nil {
		e.status = StatusError
		e.err = fmt.Errorf("building config: %w", err)
		return e.err
	}

	// Create the Xray instance from the protobuf config.
	instance, err := core.New(coreConfig)
	if err != nil {
		e.status = StatusError
		e.err = fmt.Errorf("creating xray instance: %w", err)
		return e.err
	}

	// Start the proxy.
	if err := instance.Start(); err != nil {
		_ = instance.Close()
		e.status = StatusError
		e.err = fmt.Errorf("starting xray instance: %w", err)
		return e.err
	}

	e.instance = instance
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

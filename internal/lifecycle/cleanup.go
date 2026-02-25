package lifecycle

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/leejooy96/azad/internal/sysproxy"
	"golang.org/x/term"
)

// ProxyState represents the proxy state written to .state.json when proxy is active.
// This allows --cleanup to know what to undo after a crash.
type ProxyState struct {
	ProxySet       bool   `json:"proxy_set"`
	SOCKSPort      int    `json:"socks_port"`
	HTTPPort       int    `json:"http_port"`
	NetworkService string `json:"network_service"`
	PID            int    `json:"pid"`
}

// RunCleanup checks for dirty proxy state from a previous crash and reverses it.
// It reads .state.json, calls networksetup to unset system proxy, then removes the state file.
func RunCleanup(configDir string) error {
	stateFile := filepath.Join(configDir, ".state.json")

	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No dirty proxy state found. System is clean.")
			return nil
		}
		return fmt.Errorf("reading state file: %w", err)
	}

	var state ProxyState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("parsing state file: %w", err)
	}

	if state.ProxySet {
		fmt.Println("Found dirty proxy state from previous session:")
		fmt.Printf("  Network service: %s\n", state.NetworkService)
		fmt.Printf("  SOCKS port:      %d\n", state.SOCKSPort)
		fmt.Printf("  HTTP port:       %d\n", state.HTTPPort)
		fmt.Printf("  PID:             %d\n", state.PID)
		fmt.Println()

		// Reverse the system proxy via networksetup
		if err := sysproxy.UnsetSystemProxy(state.NetworkService); err != nil {
			fmt.Printf("Warning: failed to unset system proxy: %v\n", err)
		} else {
			fmt.Printf("Reversed system proxy on: %s\n", state.NetworkService)
		}
		fmt.Println("Proxy state cleaned.")

		if err := os.Remove(stateFile); err != nil {
			return fmt.Errorf("removing state file: %w", err)
		}
	} else {
		fmt.Println("No dirty proxy state found. System is clean.")
	}

	return nil
}

// RunResetTerminal attempts to restore the terminal to a usable state.
// This is the crash-recovery fallback for when the TUI (bubbletea, Phase 4)
// exits without restoring terminal state. Uses stty sane as the primary
// recovery mechanism.
func RunResetTerminal() error {
	fd := int(os.Stdin.Fd())

	// If stdin is not a terminal (piped input, non-interactive shell),
	// stty sane will fail. Report and exit cleanly.
	if !term.IsTerminal(fd) {
		fmt.Println("Terminal is already in normal state (stdin is not a terminal).")
		return nil
	}

	// Run stty sane as the crash-recovery fallback.
	// This resets all terminal settings to sane defaults.
	cmd := exec.Command("stty", "sane")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("stty sane failed: %w", err)
	}

	fmt.Println("Terminal state restored.")
	return nil
}

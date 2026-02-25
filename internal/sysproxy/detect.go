package sysproxy

import (
	"fmt"
	"os/exec"
	"strings"
)

// execCommand is a package-level variable for testability.
// Tests can replace this with a mock to avoid running real commands.
var execCommand = exec.Command

// DetectNetworkService discovers the active macOS network service
// (e.g., "Wi-Fi", "Ethernet") by parsing networksetup output.
// It prefers "Wi-Fi" or any service containing "Ethernet", falling back
// to the first non-disabled service found.
func DetectNetworkService() (string, error) {
	cmd := execCommand("networksetup", "-listallnetworkservices")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("listing network services: %w", err)
	}

	lines := strings.Split(string(out), "\n")

	// First pass: prefer Wi-Fi or Ethernet
	for _, line := range lines {
		svc := strings.TrimSpace(line)
		if skipLine(svc) {
			continue
		}
		if svc == "Wi-Fi" || strings.Contains(svc, "Ethernet") {
			return svc, nil
		}
	}

	// Second pass: return first non-disabled, non-empty service
	for _, line := range lines {
		svc := strings.TrimSpace(line)
		if skipLine(svc) {
			continue
		}
		return svc, nil
	}

	return "", fmt.Errorf("no active network service found")
}

// skipLine returns true for lines that should be ignored:
// empty lines, the header line, and disabled services (prefixed with *).
func skipLine(svc string) bool {
	if svc == "" {
		return true
	}
	if strings.HasPrefix(svc, "An asterisk") {
		return true
	}
	if strings.HasPrefix(svc, "*") {
		return true
	}
	return false
}

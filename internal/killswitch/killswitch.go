package killswitch

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// anchorName is the pf anchor used by the kill switch.
const anchorName = "com.azad.killswitch"

// Enable loads kill switch pf rules into the kernel via a named anchor.
// It generates rules allowing only VPN server traffic, then loads them
// with pfctl and enables pf.
func Enable(serverIP string, serverPort int) error {
	rules := GenerateRules(serverIP, serverPort)
	encoded := base64.StdEncoding.EncodeToString([]byte(rules))

	command := fmt.Sprintf(
		"echo %s | base64 -d | /sbin/pfctl -a %s -f - && /sbin/pfctl -E",
		encoded, anchorName,
	)

	return runPrivilegedOrSudo(command)
}

// Disable flushes the kill switch anchor rules from pf.
// It does NOT call pfctl -d (which would disable ALL pf including Apple's rules).
func Disable() error {
	command := fmt.Sprintf("/sbin/pfctl -a %s -F all", anchorName)
	return runPrivilegedOrSudo(command)
}

// IsActive checks whether the kill switch pf anchor has active rules.
// This does not require sudo for read operations.
func IsActive() bool {
	cmd := execCommand("sh", "-c",
		fmt.Sprintf("/sbin/pfctl -a %s -sr 2>/dev/null", anchorName))
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) != ""
}

// Cleanup flushes the kill switch anchor rules with softer error handling.
// If privilege escalation fails, it prints a manual recovery command.
func Cleanup() error {
	command := fmt.Sprintf("/sbin/pfctl -a %s -F all", anchorName)
	err := runPrivilegedOrSudo(command)
	if err != nil {
		fmt.Printf("Warning: failed to flush kill switch rules: %v\n", err)
		fmt.Printf("Manual recovery: sudo pfctl -a %s -F all\n", anchorName)
		return err
	}
	return nil
}

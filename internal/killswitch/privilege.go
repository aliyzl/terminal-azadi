package killswitch

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// execCommand is a package-level variable for testability.
// Tests can replace this with a mock to avoid running real commands.
var execCommand = exec.Command

// runPrivileged runs a shell command via osascript with administrator privileges.
// This presents the native macOS password dialog (supports Touch ID).
func runPrivileged(command string) error {
	escaped := strings.ReplaceAll(command, `"`, `\"`)
	script := fmt.Sprintf(`do shell script "%s" with administrator privileges`, escaped)
	cmd := execCommand("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("privileged command failed: %w: %s", err, output)
	}
	return nil
}

// runPrivilegedOrSudo tries osascript first; if it fails (e.g. no GUI / SSH session),
// checks if the current process is already root and runs the command directly.
// This handles the headless/sudo case (research open question 3).
func runPrivilegedOrSudo(command string) error {
	err := runPrivileged(command)
	if err == nil {
		return nil
	}

	// osascript failed -- check if already running as root
	if os.Getuid() == 0 {
		cmd := execCommand("sh", "-c", command)
		output, execErr := cmd.CombinedOutput()
		if execErr != nil {
			return fmt.Errorf("root command failed: %w: %s", execErr, output)
		}
		return nil
	}

	return fmt.Errorf("privilege escalation failed (not root and osascript unavailable): %w", err)
}

package sysproxy

import (
	"fmt"
	"strconv"
)

// runCommand executes a command with arguments. This package-level variable
// defaults to running real exec.Command but can be replaced in tests.
var runCommand = defaultRunCommand

func defaultRunCommand(name string, args ...string) error {
	cmd := execCommand(name, args...)
	return cmd.Run()
}

// SetSystemProxy configures the macOS system proxy for the given network service.
// It sets SOCKS5, HTTP (web), and HTTPS (secure web) proxies to 127.0.0.1
// on the specified ports, then enables each proxy type.
func SetSystemProxy(service string, socksPort, httpPort int) error {
	cmds := [][]string{
		{"networksetup", "-setsocksfirewallproxy", service, "127.0.0.1", strconv.Itoa(socksPort)},
		{"networksetup", "-setsocksfirewallproxystate", service, "on"},
		{"networksetup", "-setwebproxy", service, "127.0.0.1", strconv.Itoa(httpPort)},
		{"networksetup", "-setwebproxystate", service, "on"},
		{"networksetup", "-setsecurewebproxy", service, "127.0.0.1", strconv.Itoa(httpPort)},
		{"networksetup", "-setsecurewebproxystate", service, "on"},
	}

	for _, args := range cmds {
		if err := runCommand(args[0], args[1:]...); err != nil {
			return fmt.Errorf("running %v: %w", args, err)
		}
	}

	return nil
}

// UnsetSystemProxy disables the SOCKS5, HTTP, and HTTPS system proxies
// for the given network service.
func UnsetSystemProxy(service string) error {
	cmds := [][]string{
		{"networksetup", "-setsocksfirewallproxystate", service, "off"},
		{"networksetup", "-setwebproxystate", service, "off"},
		{"networksetup", "-setsecurewebproxystate", service, "off"},
	}

	for _, args := range cmds {
		if err := runCommand(args[0], args[1:]...); err != nil {
			return fmt.Errorf("running %v: %w", args, err)
		}
	}

	return nil
}

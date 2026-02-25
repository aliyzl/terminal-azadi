package sysproxy

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

// commandCall records a single command invocation for test assertions.
type commandCall struct {
	name string
	args []string
}

// setupMockExec replaces package-level execCommand and runCommand with mocks.
// It returns the recorded calls and a cleanup function.
func setupMockExec(t *testing.T, outputMap map[string]string, errMap map[string]error) (*[]commandCall, func()) {
	t.Helper()
	calls := &[]commandCall{}

	origExecCommand := execCommand
	origRunCommand := runCommand

	// Mock execCommand for DetectNetworkService (needs Output())
	execCommand = func(name string, args ...string) *exec.Cmd {
		*calls = append(*calls, commandCall{name: name, args: args})
		key := name + " " + strings.Join(args, " ")
		output := ""
		if outputMap != nil {
			if v, ok := outputMap[key]; ok {
				output = v
			}
		}
		// Use echo to produce the expected output
		return exec.Command("echo", output)
	}

	// Mock runCommand for SetSystemProxy/UnsetSystemProxy
	runCommand = func(name string, args ...string) error {
		*calls = append(*calls, commandCall{name: name, args: args})
		key := name + " " + strings.Join(args, " ")
		if errMap != nil {
			if e, ok := errMap[key]; ok {
				return e
			}
		}
		return nil
	}

	cleanup := func() {
		execCommand = origExecCommand
		runCommand = origRunCommand
	}

	return calls, cleanup
}

func TestDetectNetworkService_WiFi(t *testing.T) {
	sampleOutput := "An asterisk (*) denotes that a network service is disabled.\nWi-Fi\nBluetooth PAN\nThunderbolt Bridge"
	calls, cleanup := setupMockExec(t,
		map[string]string{"networksetup -listallnetworkservices": sampleOutput},
		nil,
	)
	defer cleanup()

	svc, err := DetectNetworkService()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc != "Wi-Fi" {
		t.Errorf("expected Wi-Fi, got %q", svc)
	}
	if len(*calls) != 1 {
		t.Errorf("expected 1 call, got %d", len(*calls))
	}
}

func TestDetectNetworkService_Ethernet(t *testing.T) {
	sampleOutput := "An asterisk (*) denotes that a network service is disabled.\nThunderbolt Ethernet\nBluetooth PAN"
	calls, cleanup := setupMockExec(t,
		map[string]string{"networksetup -listallnetworkservices": sampleOutput},
		nil,
	)
	defer cleanup()

	svc, err := DetectNetworkService()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc != "Thunderbolt Ethernet" {
		t.Errorf("expected Thunderbolt Ethernet, got %q", svc)
	}
	_ = calls
}

func TestDetectNetworkService_SkipsDisabled(t *testing.T) {
	sampleOutput := "An asterisk (*) denotes that a network service is disabled.\n*Wi-Fi\nUSB Tethering"
	_, cleanup := setupMockExec(t,
		map[string]string{"networksetup -listallnetworkservices": sampleOutput},
		nil,
	)
	defer cleanup()

	svc, err := DetectNetworkService()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc != "USB Tethering" {
		t.Errorf("expected USB Tethering, got %q", svc)
	}
}

func TestDetectNetworkService_EmptyList(t *testing.T) {
	sampleOutput := "An asterisk (*) denotes that a network service is disabled."
	_, cleanup := setupMockExec(t,
		map[string]string{"networksetup -listallnetworkservices": sampleOutput},
		nil,
	)
	defer cleanup()

	_, err := DetectNetworkService()
	if err == nil {
		t.Fatal("expected error for empty service list")
	}
	if !strings.Contains(err.Error(), "no active network service found") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestSetSystemProxy_CallsSixCommands(t *testing.T) {
	calls, cleanup := setupMockExec(t, nil, nil)
	defer cleanup()

	err := SetSystemProxy("Wi-Fi", 10808, 10809)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Filter only runCommand calls (SetSystemProxy uses runCommand, not execCommand)
	runCalls := []commandCall{}
	for _, c := range *calls {
		if c.name == "networksetup" {
			runCalls = append(runCalls, c)
		}
	}

	if len(runCalls) != 6 {
		t.Fatalf("expected 6 networksetup commands, got %d", len(runCalls))
	}

	expected := [][]string{
		{"-setsocksfirewallproxy", "Wi-Fi", "127.0.0.1", "10808"},
		{"-setsocksfirewallproxystate", "Wi-Fi", "on"},
		{"-setwebproxy", "Wi-Fi", "127.0.0.1", "10809"},
		{"-setwebproxystate", "Wi-Fi", "on"},
		{"-setsecurewebproxy", "Wi-Fi", "127.0.0.1", "10809"},
		{"-setsecurewebproxystate", "Wi-Fi", "on"},
	}

	for i, exp := range expected {
		got := runCalls[i].args
		if strings.Join(got, " ") != strings.Join(exp, " ") {
			t.Errorf("command %d: expected %v, got %v", i, exp, got)
		}
	}
}

func TestSetSystemProxy_FailsOnError(t *testing.T) {
	_, cleanup := setupMockExec(t, nil,
		map[string]error{
			"networksetup -setwebproxy Wi-Fi 127.0.0.1 10809": fmt.Errorf("permission denied"),
		},
	)
	defer cleanup()

	err := SetSystemProxy("Wi-Fi", 10808, 10809)
	if err == nil {
		t.Fatal("expected error when networksetup fails")
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("error should contain cause: %v", err)
	}
}

func TestUnsetSystemProxy_CallsThreeCommands(t *testing.T) {
	calls, cleanup := setupMockExec(t, nil, nil)
	defer cleanup()

	err := UnsetSystemProxy("Ethernet")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	runCalls := []commandCall{}
	for _, c := range *calls {
		if c.name == "networksetup" {
			runCalls = append(runCalls, c)
		}
	}

	if len(runCalls) != 3 {
		t.Fatalf("expected 3 networksetup commands, got %d", len(runCalls))
	}

	expected := [][]string{
		{"-setsocksfirewallproxystate", "Ethernet", "off"},
		{"-setwebproxystate", "Ethernet", "off"},
		{"-setsecurewebproxystate", "Ethernet", "off"},
	}

	for i, exp := range expected {
		got := runCalls[i].args
		if strings.Join(got, " ") != strings.Join(exp, " ") {
			t.Errorf("command %d: expected %v, got %v", i, exp, got)
		}
	}
}

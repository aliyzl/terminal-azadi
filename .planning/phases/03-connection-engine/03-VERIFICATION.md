---
phase: 03-connection-engine
verified: 2026-02-25T11:22:00Z
status: passed
score: 13/13 must-haves verified
gaps: []
human_verification:
  - test: "Run azad connect with a real VLESS server"
    expected: "Direct IP: X.X.X.X -> Proxy IP: Y.Y.Y.Y (routing confirmed) printed after connection"
    why_human: "Requires live Xray-core instance connecting to a real VPN server; cannot verify actual traffic routing programmatically"
  - test: "Send SIGINT while azad connect is waiting"
    expected: "Disconnecting... printed, engine stops, system proxy unset, .state.json removed, Disconnected. printed"
    why_human: "Signal handling in a real process cannot be triggered programmatically in a test; context cancellation flow must be verified by running the binary"
  - test: "Run azad --cleanup after creating a dirty .state.json"
    expected: "networksetup commands run to unset proxy, state file removed, 'Reversed system proxy on: ...' printed"
    why_human: "Requires real networksetup binary with admin rights; test mocks skip actual OS system calls"
---

# Phase 3: Connection Engine Verification Report

**Phase Goal:** The app can start and stop an Xray-core proxy, route traffic through it, and manage system proxy state safely
**Verified:** 2026-02-25T11:22:00Z
**Status:** PASSED
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|---------|
| 1 | BuildConfig produces valid Xray JSON with SOCKS5 and HTTP inbounds on configurable ports | VERIFIED | `config.go:114-127` builds inbounds with configurable ports; test `Port_configuration` asserts SOCKS=2080/HTTP=9080 |
| 2 | VLESS outbound includes vnext with UUID, encryption, flow, and correct streamSettings | VERIFIED | `config.go:180-226` builds vnext with id/encryption/flow; test `VLESS_+_REALITY_+_tcp` verifies all fields |
| 3 | VMess outbound includes vnext with UUID, alterId, security, and correct streamSettings | VERIFIED | `config.go:229-275` builds vnext with id/alterId/security; test `VMess_+_TLS_+_ws` verifies all fields |
| 4 | Trojan outbound includes servers with password and correct streamSettings | VERIFIED | `config.go:278-308` builds servers with address/port/password; tests `Trojan_+_TLS_+_tcp` and `Trojan_+_TLS_+_grpc` verify |
| 5 | Shadowsocks outbound includes servers with method, password, and no streamSettings when plain | VERIFIED | `config.go:311-347` builds servers with method/password; `buildShadowsocksOutbound` skips streamSettings when network="" or "tcp"; test `Shadowsocks_plain` asserts nil streamSettings |
| 6 | StreamSettings correctly maps network types (tcp, ws, grpc, httpupgrade) with their sub-settings | VERIFIED | `config.go:367-385` switches on network; tests cover ws (path+Host header), grpc (serviceName), httpupgrade (path+host), tcp (no sub-settings) |
| 7 | TLS, REALITY, and no-TLS security modes produce correct security and settings blocks | VERIFIED | `config.go:387-413` switches on security; tests cover tls (serverName/fingerprint/ALPN/allowInsecure), reality (publicKey/shortId/fingerprint/spiderX), none (no TLS/REALITY blocks) |
| 8 | App detects active macOS network service dynamically (Wi-Fi, Ethernet, etc.) | VERIFIED | `detect.go:17-47` runs networksetup -listallnetworkservices, prefers Wi-Fi/Ethernet, falls back to first non-disabled; all 4 detection tests pass |
| 9 | App sets SOCKS5 and HTTP/HTTPS system proxy via networksetup on connect | VERIFIED | `sysproxy.go:20-37` executes 6 networksetup commands in order; `TestSetSystemProxy_CallsSixCommands` verifies exact command sequence |
| 10 | App unsets all three proxy types via networksetup on disconnect | VERIFIED | `sysproxy.go:39-55` executes 3 networksetup off commands; `TestUnsetSystemProxy_CallsThreeCommands` verifies exact sequence |
| 11 | On startup with dirty .state.json, app reverses system proxy and removes state file | VERIFIED | `cleanup.go:43-65` calls `sysproxy.UnsetSystemProxy(state.NetworkService)` before removing the file; warning printed but cleanup continues if unset fails |
| 12 | Engine.Start creates Xray instance from protocol.Server and starts proxy on configured ports | VERIFIED | `engine.go:55-101` locks, checks not already connected, sets XRAY_LOCATION_ASSET, calls BuildConfig -> core.New -> instance.Start; sets StatusConnected on success |
| 13 | Engine.Stop calls instance.Close, nils the reference, and transitions to Disconnected | VERIFIED | `engine.go:105-120` calls instance.Close(), sets instance=nil, server=nil, status=StatusDisconnected |

**Score:** 13/13 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/engine/config.go` | Xray JSON config builder from protocol.Server | VERIFIED | 417 lines; exports BuildConfig; imports protocol.Server and serial.LoadJSONConfig |
| `internal/engine/config_test.go` | Table-driven tests for all 4 protocols | VERIFIED | 660 lines (min 100); 12 table cases + 1 defaults test; all pass |
| `internal/sysproxy/detect.go` | Active network service detection | VERIFIED | Exports DetectNetworkService; uses execCommand var for testability |
| `internal/sysproxy/sysproxy.go` | System proxy set/unset via networksetup | VERIFIED | Exports SetSystemProxy and UnsetSystemProxy; uses runCommand var for testability |
| `internal/sysproxy/sysproxy_test.go` | Unit tests with command injection | VERIFIED | 218 lines (min 30); 7 tests; all pass |
| `internal/lifecycle/cleanup.go` | Upgraded cleanup with actual proxy reversal | VERIFIED | Imports sysproxy; calls UnsetSystemProxy on dirty state; removes state file |
| `internal/engine/engine.go` | Engine struct with Start/Stop and ConnectionStatus state machine | VERIFIED | Exports Engine, ConnectionStatus, StatusDisconnected, StatusConnecting, StatusConnected, StatusError |
| `internal/engine/verify.go` | IP verification through SOCKS5 proxy | VERIFIED | Exports VerifyIP (SOCKS5 dialer) and GetDirectIP (direct HTTP client) |
| `internal/cli/connect.go` | Wired connect command using engine, sysproxy, and verify | VERIFIED | 201 lines (min 40); imports all required packages; full flow implemented |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `internal/engine/config.go` | `internal/protocol/server.go` | import protocol.Server, protocol.Protocol* constants | WIRED | `config.go:9` imports protocol; `protocol.Server` used on lines 105, 160, 180, 229, 278, 311, 350 |
| `internal/engine/config.go` | `github.com/xtls/xray-core/infra/conf/serial` | serial.LoadJSONConfig for JSON->protobuf conversion | WIRED | `config.go:12` imports serial; `serial.LoadJSONConfig` called at line 151 |
| `internal/engine/engine.go` | `internal/engine/config.go` | Calls BuildConfig to get *core.Config | WIRED | `engine.go:73` calls `BuildConfig(srv, socksPort, httpPort)` |
| `internal/engine/engine.go` | `github.com/xtls/xray-core/core` | core.New + instance.Start / instance.Close | WIRED | `engine.go:11` imports core; `core.New` at line 81, `instance.Start()` at line 89, `instance.Close()` at lines 90 and 114 |
| `internal/engine/verify.go` | `golang.org/x/net/proxy` | SOCKS5 proxy dialer for HTTP client | WIRED | `verify.go:11` imports proxy; `proxy.SOCKS5` called at line 20 |
| `internal/sysproxy/sysproxy.go` | `os/exec` | exec.Command for networksetup calls | WIRED | `detect.go:11` imports os/exec; `execCommand` (= exec.Command) used in detect.go line 18 and shared via package var |
| `internal/lifecycle/cleanup.go` | `internal/sysproxy/sysproxy.go` | Calls UnsetSystemProxy with saved network service | WIRED | `cleanup.go:11` imports sysproxy; `sysproxy.UnsetSystemProxy(state.NetworkService)` called at line 52 |
| `internal/cli/connect.go` | `internal/engine/engine.go` | Creates Engine, calls Start/Stop | WIRED | `connect.go:11` imports engine; `engine.Engine{}` at line 60, `eng.Start` at line 62, `eng.Stop` at line 115 |
| `internal/cli/connect.go` | `internal/sysproxy` | Calls DetectNetworkService, SetSystemProxy, UnsetSystemProxy | WIRED | `connect.go:15` imports sysproxy; calls at lines 70, 81, 121 |
| `internal/cli/connect.go` | `internal/engine/verify.go` | Calls GetDirectIP before proxy, then VerifyIP after connecting | WIRED | `connect.go:88` calls `engine.GetDirectIP()`, line 91 calls `engine.VerifyIP(cfg.Proxy.SOCKSPort)` |
| `internal/cli/connect.go` | `internal/lifecycle/cleanup.go` | Writes ProxyState to .state.json before setting system proxy | WIRED | `connect.go:13` imports lifecycle; `lifecycle.ProxyState{}` constructed at line 178; `config.StateFilePath()` used at line 173 |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|---------|
| CONN-01 | 03-01-PLAN, 03-03-PLAN | App starts Xray-core proxy via Go library API (core.New/instance.Start) on configurable SOCKS5 and HTTP ports | SATISFIED | `engine.go:81,89` calls core.New + instance.Start; ports flow from config through BuildConfig to inbound listener config |
| CONN-02 | 03-03-PLAN | App stops Xray-core proxy cleanly via instance.Close() | SATISFIED | `engine.go:114` calls instance.Close(); instance nilled at line 115 |
| CONN-03 | 03-03-PLAN | App displays connection status (disconnected/connecting/connected/error) with current server name | SATISFIED | ConnectionStatus enum with String() in engine.go:29-42; connect.go:106-107 prints "connected" status and server name; "Connecting...", "Disconnecting...", "Disconnected." also printed |
| CONN-04 | 03-03-PLAN | App verifies connection works by checking external IP through the proxy | SATISFIED | verify.go:19-46 implements VerifyIP; connect.go:88-103 calls GetDirectIP then VerifyIP, compares both IPs and prints routing confirmation |
| CONN-05 | 03-02-PLAN | App sets/unsets macOS system proxy (SOCKS + HTTP) via networksetup or sysproxy | SATISFIED | sysproxy.go SetSystemProxy (6 commands) and UnsetSystemProxy (3 commands); detect.go finds active network service |
| CONN-06 | 03-02-PLAN | App detects and cleans up dirty proxy state on startup (previous crash left system proxy set) | SATISFIED | cleanup.go:43-65 reads .state.json, calls sysproxy.UnsetSystemProxy, removes file |

**Requirement coverage: 6/6 (CONN-01 through CONN-06) — all satisfied**

No orphaned requirements found. All 6 CONN-* requirements declared in plan frontmatter match the REQUIREMENTS.md Phase 3 entries and have implementation evidence.

### Anti-Patterns Found

| File | Pattern | Severity | Assessment |
|------|---------|----------|------------|
| `internal/cli/connect.go:107` | Status string hardcoded as `"connected"` instead of `eng.Status().String()` | Info | Functional at this point in code (engine has successfully started); does not block goal. The ConnectionStatus enum's String() method exists and is correct; this is cosmetic redundancy. |

No TODO/FIXME/placeholder comments found in any Phase 3 files.
No empty return stubs found.
No console.log-equivalent only implementations found.

### Human Verification Required

### 1. Live connection routing confirmation

**Test:** Run `azad connect <server-name>` with a configured VLESS/VMess/Trojan/Shadowsocks server.
**Expected:** After startup, prints "Direct IP: X.X.X.X -> Proxy IP: Y.Y.Y.Y (routing confirmed)" where the two IPs differ. The SOCKS5 proxy on the configured port accepts connections and routes them through the remote server.
**Why human:** Requires live Xray-core instance connecting to an actual VPN server. The actual traffic routing path (Xray geoip.dat routing, kernel network stack, remote server) cannot be mocked or tested programmatically.

### 2. SIGINT graceful shutdown

**Test:** Start `azad connect <server>`, wait for "Press Ctrl+C to disconnect", send SIGINT.
**Expected:** Prints "Shutting down gracefully..." (from main.go goroutine), then "Disconnecting...", stops engine, unsets system proxy (networksetup commands run), removes .state.json, prints "Disconnected." and exits cleanly.
**Why human:** Signal delivery to a running process and the exact sequence of cleanup operations must be observed in a terminal. Context cancellation flow involves OS signal handling that cannot be triggered in unit tests.

### 3. Crash recovery with dirty state

**Test:** Manually create `~/.config/azad/.state.json` with `{"proxy_set":true,"socks_port":1080,"http_port":8080,"network_service":"Wi-Fi","pid":99999}`, then run `azad --cleanup`.
**Expected:** Prints network service/port/PID from state file, runs networksetup to disable all proxies, prints "Reversed system proxy on: Wi-Fi", removes .state.json.
**Why human:** Requires real networksetup binary with admin rights; the mock-based unit tests verify command construction but not actual OS system proxy reversal.

### Overall Assessment

All 13 must-have truths are verified. All 9 required artifacts exist, are substantive, and are properly wired. All 11 key links are confirmed in the actual code. All 6 CONN-* requirements have implementation evidence. `go build ./...` succeeds with no errors. `go vet ./...` passes cleanly. All 12 engine config builder tests pass. All 7 sysproxy tests pass.

The one cosmetic finding (hardcoded `"connected"` string vs. `eng.Status().String()`) does not affect goal achievement — the status IS displayed correctly to the user at the moment it matters, and the enum machinery works correctly as evidenced by the String() method and its use in other parts of the code.

---

_Verified: 2026-02-25T11:22:00Z_
_Verifier: Claude (gsd-verifier)_

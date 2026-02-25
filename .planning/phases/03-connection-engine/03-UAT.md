---
status: complete
phase: 03-connection-engine
source: [03-01-SUMMARY.md, 03-02-SUMMARY.md, 03-03-SUMMARY.md]
started: 2026-02-25T08:00:00Z
updated: 2026-02-25T08:20:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Unit tests pass
expected: Run `go test ./internal/engine/ ./internal/sysproxy/ -v` — all 19 tests pass (12 config builder + 7 sysproxy). No failures, no skips.
result: pass

### 2. Build compiles cleanly
expected: Run `go build ./...` — exits 0 with no errors. All packages including engine, sysproxy, cli, and lifecycle compile.
result: pass

### 3. Connect command registered
expected: Run `go run ./cmd/azad connect --help` — shows usage "connect [server-name]" with description about connecting to a VPN server. No errors.
result: pass

### 4. Connect to a server
expected: With a server configured in servers.json, run `azad connect <server-name>`. Output shows "Connecting to <name>...", then "Proxy started on SOCKS5://127.0.0.1:<port> and HTTP://127.0.0.1:<port>". Xray proxy is running.
result: pass

### 5. IP verification and routing
expected: After connecting, the app fetches your direct IP and proxy IP and prints "Direct IP: X.X.X.X -> Proxy IP: Y.Y.Y.Y (routing confirmed)" showing they differ. This confirms traffic is routed through the remote server.
result: pass

### 6. System proxy set on macOS
expected: While connected, open macOS System Settings > Network > Wi-Fi > Proxies (or run `networksetup -getsocksfirewallproxy Wi-Fi`). SOCKS5 and HTTP/HTTPS proxies are set to 127.0.0.1 on the configured ports.
result: pass

### 7. Graceful disconnect
expected: Press Ctrl+C while connected. Output shows "Disconnecting..." then "Disconnected." System proxy is unset (verify with `networksetup -getsocksfirewallproxy Wi-Fi` showing "No"). State file (.state.json) is removed.
result: pass

### 8. Crash cleanup
expected: If the app crashes or is force-killed while connected (leaving dirty .state.json), running `azad --cleanup` detects dirty state, calls networksetup to unset proxy, removes .state.json, and prints confirmation.
result: pass

## Summary

total: 8
passed: 8
issues: 0
pending: 0
skipped: 0

## Gaps

[none yet]

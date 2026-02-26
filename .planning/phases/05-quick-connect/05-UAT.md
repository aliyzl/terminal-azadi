---
status: complete
phase: 05-quick-connect
source: [05-01-SUMMARY.md, 05-02-SUMMARY.md]
started: 2026-02-26T08:30:00Z
updated: 2026-02-26T09:10:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Headless connect — first server fallback
expected: Run `azad connect` with no prior connection history. It should connect to the first server in the list and print connection status to stdout.
result: pass

### 2. Headless connect — remembers last used
expected: After a successful `azad connect`, run `azad connect` again with no args. It should reconnect to the same server you used last time (LastUsed persisted to config).
result: pass

### 3. TUI auto-connect on launch
expected: Run `/tmp/azad-test` (no args) in a real terminal. The TUI should launch and automatically connect to the best server (last-used). You should see the connection status update in the TUI.
result: pass

### 4. TUI auto-connect — empty store
expected: Run `/tmp/azad-test` with an empty server store (no servers loaded). The TUI should launch normally without crashing or showing an error flash. Auto-connect silently skips.
result: skipped
reason: Only 1 server in store, would require manual file manipulation to test

### 5. TUI manual connect via Enter/c
expected: In the TUI, select a server from the list and press Enter (or c). The TUI should initiate a connection to that server, showing connection progress and status.
result: pass

### 6. TUI server switch while connected
expected: While connected to a server in the TUI, select a different server and press Enter/c. It should disconnect from the current server first, then connect to the newly selected one.
result: skipped
reason: Only 1 server available

### 7. TUI quit cleanup
expected: While connected in the TUI, press q or Ctrl+C. The engine should stop, system proxy should be unset, and the .state.json file should be removed. The TUI exits cleanly.
result: pass

### 8. TUI persistence across sessions
expected: After connecting to a server via TUI, quit and re-launch `/tmp/azad-test`. On re-launch, the TUI should auto-connect to the same server you used in the previous session.
result: pass

## Summary

total: 8
passed: 6
issues: 0
pending: 0
skipped: 2

## Gaps

[none yet]

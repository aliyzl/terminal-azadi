---
status: passed
phase: 07-kill-switch
source: [07-01-SUMMARY.md, 07-02-SUMMARY.md]
started: 2026-02-26T10:00:00Z
updated: 2026-02-26T12:00:00Z
---

## Tests

### 1. Headless connect with --kill-switch flag
expected: Running `go run ./cmd/azad connect --kill-switch` connects and enables kill switch with pf rules loaded.
result: pass

### 2. Kill switch blocks non-VPN traffic
expected: Non-VPN traffic blocked. Pf rules include block-all with pass for loopback, VPN server, DHCP, DNS.
result: pass

### 3. TUI menu (m key) shows settings overlay
expected: Pressing m opens Settings menu with kill switch toggle (enter to toggle, esc to close).
result: pass

### 4. TUI kill switch enable via menu
expected: m → enter → confirmation → y enables kill switch. Red banner appears at top, status bar shows indicator.
result: pass

### 5. TUI kill switch disable via menu
expected: m → enter (while active) disables directly. Banner disappears, status bar indicator gone.
result: pass

### 6. Quit with kill switch active disables it
expected: Pressing q auto-disables kill switch before exit. No leftover pf rules.
result: pass

### 7. Crash recovery via --cleanup
expected: After force-kill with kill switch active, `go run ./cmd/azad --cleanup` restores internet.
result: pass

### 8. Startup recovery detects active kill switch
expected: After crash, re-launching TUI detects active kill switch — banner and menu show active state.
result: pass

## Summary

total: 8
passed: 8
issues: 0
pending: 0
skipped: 0

## Notes

- Kill switch keybinding changed from K (broken in bubbletea v2) to m (settings menu)
- Menu approach: m → Settings overlay → enter toggles kill switch
- Enable requires confirmation dialog (y/n), disable is immediate
- Red full-width banner at top when active: "KILL SWITCH ACTIVE — All non-VPN traffic blocked"
- Xray routing fix applied during UAT: sniffing moved to inbounds, domainStrategy changed to AsIs

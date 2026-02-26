---
status: complete
phase: 04-tui-and-server-interaction
source: [04-01-SUMMARY.md, 04-02-SUMMARY.md, 04-03-SUMMARY.md]
started: 2026-02-25T11:00:00Z
updated: 2026-02-25T11:00:00Z
---

## Current Test
<!-- OVERWRITE each test - shows where we are -->

[testing complete]

## Tests

### 1. TUI Launch and Layout
expected: Run `go run ./cmd/azad/` with no arguments. The TUI launches in alt-screen mode showing a split-pane layout: server list panel on the left (~1/3 width), detail panel on the right (~2/3 width), and a status bar at the bottom showing connection state, server name, proxy port, and uptime.
result: pass

### 2. Keyboard Navigation
expected: Press j/k to navigate the server list up and down. The detail panel on the right updates to show the selected server's info (name, protocol, address, transport, TLS). Press enter on a server to select it.
result: pass

### 3. Help Overlay
expected: Press ? to open the help overlay. A centered bordered box appears showing all keybindings organized by group. Press ? or esc to dismiss it and return to the normal view.
result: pass

### 4. Fuzzy Filter
expected: Press / to activate filter mode. A text input appears. Type a search term (server name, address, or protocol). The server list narrows in real-time to matching entries. Press esc to clear filter and return to full list.
result: pass

### 5. Add Server via URI
expected: Press a to open the "Add Server" input modal. Paste a valid server URI (Cmd+V or Ctrl+V) and it appears in the text field. Press Enter — the server appears in the list.
result: pass

### 6. Add Subscription
expected: Press s to open the "Add Subscription" input modal. Paste a subscription URL (Cmd+V or Ctrl+V) and it appears in the text field. Press Enter — servers from the subscription appear in the list.
result: pass

### 7. Delete and Clear Servers
expected: Select a server and press d to delete it — the server is removed from the list. Press D (shift+d) to clear all servers — a confirmation dialog appears asking y/n. Press y to confirm deletion, or n/esc to cancel.
result: pass

### 8. Ping All Servers
expected: Press p to ping all servers. The list title updates with a progress counter (e.g., "Servers (pinging 2/5...)"). When complete, the server list re-sorts by latency (fastest first, errors last). Title returns to "Servers".
result: pass

### 9. Terminal Resize and Minimum Size
expected: Resize the terminal window — the layout adapts smoothly, panels resize proportionally. Shrink the terminal below 60x20 — a "Terminal too small" message appears. Resize back above 60x20 — the normal layout restores.
result: pass

### 10. Clean Exit
expected: Press q to exit the TUI. The terminal returns cleanly to the shell prompt with no visual artifacts or hanging processes.
result: pass

## Summary

total: 10
passed: 10
issues: 0
pending: 0
skipped: 0

## Gaps

- truth: "User can paste a server URI into the add-server input modal"
  status: resolved
  reason: "User reported: modal opens but can't paste into the text input"
  severity: major
  test: 5
  root_cause: "tea.PasteMsg not routed to input model — falls through to serverList. Also clipboard command result from Ctrl+V never reaches textinput on second hop."
  fix: "04-04-PLAN.md — Added tea.PasteMsg case and view-aware default fallthrough"
  debug_session: ".planning/debug/paste-not-working.md"

- truth: "User can paste a subscription URL into the add-subscription modal"
  status: resolved
  reason: "User reported: can't paste into the text input, but the subscription modal page opens"
  severity: major
  test: 6
  root_cause: "Same root cause as test 5 — tea.PasteMsg and clipboard result messages not routed to input model"
  fix: "04-04-PLAN.md — Same fix as test 5"
  debug_session: ".planning/debug/paste-not-working.md"

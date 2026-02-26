---
phase: 08-split-tunneling
plan: 03
subsystem: tui
tags: [split-tunneling, bubbletea, tui, lipgloss, config-persistence]

# Dependency graph
requires:
  - phase: 08-split-tunneling
    provides: "splittunnel package with ParseRule, ToXrayRules, Config types; BuildConfig split tunnel integration; SplitTunnelConfig persistence"
  - phase: 07-kill-switch
    provides: "killswitch.Enable with variadic bypassIPs, GenerateRules with bypass support"
provides:
  - "Split tunnel TUI management view with rule CRUD, mode toggle, enable/disable"
  - "Input modal for adding rules with validation via splittunnel.ParseRule"
  - "Status bar SPLIT indicator when split tunneling is active"
  - "TUI connect commands pass split tunnel config to Engine.Start"
  - "Kill switch enable passes bypass IPs for split tunnel pf coordination"
affects: [tui, split-tunneling]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Split tunnel overlay view follows existing menu/kill-switch modal pattern"
    - "saveSplitTunnelCmd pattern: config.Save in tea.Cmd goroutine, return splitTunnelSavedMsg"
    - "buildSplitTunnelConfig converts persistence types to runtime types at call site"
    - "extractBypassIPs resolves domain rules best-effort for pf firewall exceptions"

key-files:
  created:
    - internal/tui/split_tunnel.go
  modified:
    - internal/tui/app.go
    - internal/tui/connect_cmd.go
    - internal/tui/input.go
    - internal/tui/messages.go
    - internal/tui/statusbar.go
    - internal/tui/keys.go

key-decisions:
  - "Status bar and field additions pulled into Task 1 for compilation (status bar needed by splitTunnelSavedMsg handler)"
  - "extractBypassIPs skips wildcard rules since they cannot be meaningfully resolved to IPs"
  - "keyMap KillSwitch binding renamed to Menu (pre-existing uncommitted fix committed with Task 2)"

patterns-established:
  - "Pattern: Overlay views use renderXxxView(m) function in separate file, called from View() switch"
  - "Pattern: Config mutation in key handler -> saveSplitTunnelCmd -> splitTunnelSavedMsg -> statusBar update"

requirements-completed: [SPLT-04]

# Metrics
duration: 7min
completed: 2026-02-26
---

# Phase 8 Plan 3: Split Tunnel TUI Integration Summary

**Split tunnel management overlay in TUI settings menu with rule add/remove, mode toggle, enable/disable, status bar SPLIT indicator, and connect command wiring to pass split config to engine**

## Performance

- **Duration:** 7 min
- **Started:** 2026-02-26T16:06:52Z
- **Completed:** 2026-02-26T16:14:18Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- Full split tunnel management UI accessible from settings menu (m > t) with rule list, add/delete, mode toggle, enable/disable
- Input modal for adding rules with IP/CIDR/domain/wildcard validation via splittunnel.ParseRule
- TUI connect commands (manual and auto-connect) pass split tunnel config to Engine.Start for Xray routing
- Kill switch enable passes bypass IPs from exclusive-mode split tunnel rules for pf coordination
- Status bar shows SPLIT indicator when split tunneling is active with rules

## Task Commits

Each task was committed atomically:

1. **Task 1: TUI split tunnel view, input modal, and menu integration** - `202c4b1` (feat)
2. **Task 2: TUI connect wiring and status bar split tunnel indicator** - `c196908` (feat)

## Files Created/Modified
- `internal/tui/split_tunnel.go` - renderSplitTunnelView overlay, buildSplitTunnelConfig converter, saveSplitTunnelCmd
- `internal/tui/app.go` - viewSplitTunnel/viewAddSplitRule states, key handling, connect wiring, Init with split config
- `internal/tui/connect_cmd.go` - connectServerCmd/autoConnectCmd with splitCfg param, enableKillSwitchCmd with bypassIPs, extractBypassIPs helper
- `internal/tui/input.go` - inputAddSplitRule mode with placeholder text
- `internal/tui/messages.go` - splitTunnelSavedMsg type
- `internal/tui/statusbar.go` - splitTunnel field, SetSplitTunnel method, SPLIT indicator in View
- `internal/tui/keys.go` - KillSwitch binding renamed to Menu (pre-existing fix)

## Decisions Made
- Pulled status bar splitTunnel field and SetSplitTunnel method into Task 1 (originally planned for Task 2) because Task 1's splitTunnelSavedMsg handler needs it for compilation
- extractBypassIPs only resolves IP/CIDR and domain rules; wildcards are skipped since they cannot be meaningfully resolved to specific IPs
- Committed pre-existing keys.go rename (KillSwitch -> Menu) with Task 2 since it was required for correct TUI operation

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Status bar field and method pulled into Task 1**
- **Found during:** Task 1 (compilation)
- **Issue:** Task 1 handler for splitTunnelSavedMsg calls m.statusBar.SetSplitTunnel which was planned for Task 2
- **Fix:** Added splitTunnel field, SetSplitTunnel method, and SPLIT indicator to statusbar.go in Task 1
- **Files modified:** internal/tui/statusbar.go
- **Verification:** `go build ./...` succeeds
- **Committed in:** 202c4b1 (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Minor reordering of status bar additions from Task 2 to Task 1. No scope creep. Task 2 focused on connect wiring and bypass IPs.

## Issues Encountered
- Pre-existing go vet warning in internal/tui/ping.go (IPv6 format string) -- out of scope, not caused by this plan's changes

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Split tunnel fully integrated in both CLI (08-02) and TUI (08-03)
- Users can manage split tunnel rules interactively via TUI settings menu
- Connect commands pass split tunnel config through to Xray engine
- Kill switch coordinates with split tunnel bypass IPs

---
*Phase: 08-split-tunneling*
*Completed: 2026-02-26*

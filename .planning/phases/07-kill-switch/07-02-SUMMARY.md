---
phase: 07-kill-switch
plan: 02
subsystem: infra
tags: [killswitch, pf, cli, tui, keybinding, statusbar, crash-recovery, dns-resolution]

# Dependency graph
requires:
  - phase: 07-kill-switch/01
    provides: "killswitch.Enable/Disable/IsActive/Cleanup API and ProxyState kill switch fields"
  - phase: 03-connection-engine/02
    provides: "sysproxy pattern and writeProxyState for crash recovery"
  - phase: 05-quick-connect/02
    provides: "TUI connect_cmd.go with disconnectCmd/connectServerCmd/autoConnectCmd"
provides:
  - "azad connect --kill-switch flag for headless kill switch enable with DNS resolution"
  - "TUI K keybinding toggles kill switch with confirmation overlay"
  - "Status bar KILL SW: ON indicator when kill switch active"
  - "Quit/disconnect sequence disables kill switch automatically"
  - "Startup recovery detects active kill switch and informs user"
  - "TUI init checks killswitch.IsActive() for crash recovery state"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns: [variadic bool parameter for optional disconnect kill switch disable, read-modify-write for ProxyState updates in TUI]

key-files:
  created: []
  modified:
    - internal/cli/connect.go
    - internal/cli/root.go
    - internal/tui/app.go
    - internal/tui/connect_cmd.go
    - internal/tui/keys.go
    - internal/tui/messages.go
    - internal/tui/statusbar.go

key-decisions:
  - "Variadic bool parameter on disconnectCmd for optional kill switch disable (avoids breaking existing callers)"
  - "Read-modify-write pattern for tuiWriteProxyStateWithKS to preserve existing proxy state fields"
  - "K (uppercase) keybinding to avoid conflict with k (lowercase) navigation"
  - "No confirmation needed when disabling kill switch (only when enabling)"
  - "Kill switch recovery on TUI init via killswitch.IsActive() check (no state file dependency)"

patterns-established:
  - "Variadic optional parameters on tea.Cmd functions for backwards compatibility"
  - "Confirmation overlay pattern: viewConfirmX state + y/n/esc key handling"
  - "Status bar indicator: bool field + SetX method + conditional render in View()"

requirements-completed: [KILL-01, KILL-02, KILL-03, KILL-04, KILL-05]

# Metrics
duration: 5min
completed: 2026-02-26
---

# Phase 7 Plan 02: Kill Switch CLI/TUI Integration Summary

**Kill switch wired into CLI --kill-switch flag with DNS resolution, TUI K keybinding with confirmation overlay, status bar indicator, and startup crash recovery detection**

## Performance

- **Duration:** 5 min
- **Started:** 2026-02-26T09:41:19Z
- **Completed:** 2026-02-26T09:47:13Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- Added --kill-switch flag to headless `azad connect` with server hostname DNS resolution before enabling pf rules
- Wired K keybinding in TUI with confirmation overlay for enabling, direct disable without confirmation
- Added KILL SW: ON status bar indicator, quit/disconnect auto-disables kill switch, startup recovery detection in both CLI and TUI

## Task Commits

Each task was committed atomically:

1. **Task 1: Wire kill switch into headless connect and startup recovery** - `1837d94` (feat)
2. **Task 2: Wire kill switch toggle into TUI with status display** - `133439d` (feat)

## Files Created/Modified
- `internal/cli/connect.go` - Added --kill-switch flag, DNS resolution, enable/disable in connect flow, extended writeProxyState
- `internal/cli/root.go` - Added startup kill switch recovery detection in PersistentPreRunE
- `internal/tui/app.go` - Added viewConfirmKillSwitch, K key handler, killSwitchResultMsg handler, overlay rendering, quit integration
- `internal/tui/connect_cmd.go` - Added enableKillSwitchCmd/disableKillSwitchCmd, tuiWriteProxyStateWithKS, updated disconnectCmd with kill switch support
- `internal/tui/keys.go` - Added KillSwitch keybinding (K) to keyMap and FullHelp
- `internal/tui/messages.go` - Added killSwitchResultMsg struct
- `internal/tui/statusbar.go` - Added killSwitch field, SetKillSwitch method, KILL SW: ON indicator in View

## Decisions Made
- Used variadic `ksActive ...bool` parameter on `disconnectCmd` to optionally disable kill switch without breaking existing callers (sequence-based disconnect in TUI)
- Added `tuiWriteProxyStateWithKS` with read-modify-write pattern to preserve existing ProxyState fields when updating kill switch state
- Uppercase K keybinding avoids conflict with lowercase k for list navigation
- Disable does not require confirmation (only enabling does) -- disabling restores normal network, no destructive action
- TUI init checks `killswitch.IsActive()` directly (reads pf anchor) rather than relying on state file alone -- more reliable for crash recovery

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Kill switch feature is complete across all integration points (CLI headless, TUI interactive, crash recovery)
- Phase 7 Kill Switch is fully implemented: infrastructure (Plan 01) + integration (Plan 02)
- All KILL-01 through KILL-05 requirements addressed

## Self-Check: PASSED

- All 7 modified files verified on disk
- Commit 1837d94 (Task 1) verified in git log
- Commit 133439d (Task 2) verified in git log
- go build ./... passes

---
*Phase: 07-kill-switch*
*Completed: 2026-02-26*

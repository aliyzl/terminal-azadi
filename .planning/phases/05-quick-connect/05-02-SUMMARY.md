---
phase: 05-quick-connect
plan: 02
subsystem: tui
tags: [bubbletea, connection-lifecycle, auto-connect, proxy, tea-cmd]

# Dependency graph
requires:
  - phase: 05-quick-connect
    provides: UpdateServer, findServer resolution logic, LastUsed/LastConnected persistence
  - phase: 03-connection-engine
    provides: Engine.Start/Stop, VerifyIP, SetSystemProxy/UnsetSystemProxy
  - phase: 04-tui-server-interaction
    provides: TUI model with server list, detail panel, status bar, keybindings
provides:
  - connectServerCmd tea.Cmd for manual connect with proxy setup and persistence
  - disconnectCmd tea.Cmd for engine stop, proxy unset, and state file cleanup
  - autoConnectCmd tea.Cmd for startup auto-connect using best server resolution
  - TUI Init auto-connect on launch, Enter/c connect keybinding, quit cleanup
affects: [06-polish, tui-enhancements]

# Tech tracking
tech-stack:
  added: []
  patterns: [async tea.Cmd for connection lifecycle, tea.Sequence for disconnect-then-reconnect, tea.Batch for Init multi-command]

key-files:
  created:
    - internal/tui/connect_cmd.go
  modified:
    - internal/tui/app.go
    - internal/tui/messages.go

key-decisions:
  - "Duplicated writeProxyState/removeStateFile helpers from cli/connect.go to avoid exporting internals or circular dependency"
  - "tea.Sequence for disconnect-then-reconnect ensures serial execution when switching servers"
  - "Auto-connect skips silently on empty store (no error flash) per requirement"

patterns-established:
  - "Connection lifecycle as tea.Cmd functions (connectServerCmd, disconnectCmd, autoConnectCmd)"
  - "selectServerByID helper for programmatic list selection after connect"
  - "tea.Sequence for ordered disconnect-reconnect flow"

requirements-completed: [QCON-01, QCON-03]

# Metrics
duration: 3min
completed: 2026-02-26
---

# Phase 5 Plan 2: TUI Connection Lifecycle Summary

**Full TUI connection lifecycle with auto-connect on startup, Enter/c manual connect, and quit cleanup via async tea.Cmd functions**

## Performance

- **Duration:** 3 min
- **Started:** 2026-02-26T08:06:36Z
- **Completed:** 2026-02-26T08:09:36Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- Created `connect_cmd.go` with three tea.Cmd functions: `connectServerCmd` (manual connect), `disconnectCmd` (cleanup), `autoConnectCmd` (startup)
- Wired TUI Init to auto-connect on launch using best server resolution (LastUsed > latency > first)
- Enter/c keybinding connects to selected server with disconnect-first logic when already connected
- Quit (q/ctrl+c) properly cleans up engine, system proxy, and state file before exit

## Task Commits

Each task was committed atomically:

1. **Task 1: Create TUI connection command functions** - `6abe02e` (feat)
2. **Task 2: Wire auto-connect, Enter/c connect, and quit cleanup into TUI model** - `84711c5` (feat)

## Files Created/Modified
- `internal/tui/connect_cmd.go` - New file with connectServerCmd, disconnectCmd, autoConnectCmd, resolveBestServer, tuiWriteProxyState, tuiRemoveStateFile
- `internal/tui/app.go` - Init returns tea.Batch with autoConnectCmd; Enter/c dispatches connect; quit cleans up; autoConnectMsg/connectResultMsg handlers select server
- `internal/tui/messages.go` - Added autoConnectMsg struct for auto-connect result routing

## Decisions Made
- Duplicated `writeProxyState` and `removeStateFile` helpers (10 lines each) from `cli/connect.go` as `tuiWriteProxyState`/`tuiRemoveStateFile` to avoid exporting cli internals or creating a shared package for trivial functions
- Used `tea.Sequence(disconnectCmd, connectServerCmd)` for serial disconnect-then-reconnect when switching servers while connected
- Auto-connect silently skips when server store is empty (returns `autoConnectMsg{}` with empty ServerID, no error) per QCON-01 requirement
- `resolveBestServer` inlines the 3-tier resolution (LastUsed > lowest positive LatencyMs > first) to avoid circular dependency with cli package

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- Pre-existing `go vet` warning in `internal/tui/ping.go` (IPv6 address format) continues to show when vetting `./...` -- not related to our changes, out of scope

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- TUI now has complete connection lifecycle: auto-connect on launch, manual connect via Enter/c, clean disconnect on quit
- Both headless (`azad connect`) and TUI paths now persist LastUsed/LastConnected
- Phase 5 (Quick Connect) is complete -- ready for Phase 6 (Polish)
- All connection flows handle system proxy setup and state file management for crash recovery

## Self-Check: PASSED

- FOUND: internal/tui/connect_cmd.go
- FOUND: internal/tui/app.go
- FOUND: internal/tui/messages.go
- FOUND: 05-02-SUMMARY.md
- FOUND: 6abe02e (Task 1 commit)
- FOUND: 84711c5 (Task 2 commit)

---
*Phase: 05-quick-connect*
*Completed: 2026-02-26*

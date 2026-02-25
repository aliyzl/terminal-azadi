---
phase: 04-tui-and-server-interaction
plan: 02
subsystem: ui
tags: [bubbletea-v2, lipgloss-v2, bubbles-v2, tui, split-pane, help-overlay, cobra]

# Dependency graph
requires:
  - phase: 04-tui-and-server-interaction
    provides: Theme, keyMap, messages, serverList, detail, statusBar components from 04-01
  - phase: 03-connection-engine
    provides: engine.Engine with Status/ServerName for status bar integration
  - phase: 01-foundation
    provides: config.Load, config.DataDir, serverstore.Store for CLI wiring
provides:
  - Root TUI model composing all child components into split-pane layout
  - Help overlay with centered bordered keybinding display
  - CLI root command launching TUI via tea.NewProgram when no subcommand given
  - Keyboard navigation routing (j/k/enter/esc/q/?//) between view states
affects: [04-03-server-interaction]

# Tech tracking
tech-stack:
  added: []
  patterns: [viewState enum for modal routing, syncDetail after list navigation, tickCmd for uptime refresh]

key-files:
  created:
    - internal/tui/app.go
    - internal/tui/help.go
  modified:
    - internal/cli/root.go

key-decisions:
  - "KeyPressMsg.Keystroke() string matching instead of field inspection -- more readable key routing"
  - "Help overlay replaces content entirely via lipgloss.Place centering instead of transparent overlay"
  - "List filtering state checked before key routing -- Filtering mode delegates all keys to list"
  - "Root RunE loads config/store/engine inline instead of package-level state -- clean initialization"

patterns-established:
  - "viewState enum pattern: viewNormal/viewHelp/viewAddServer/viewAddSubscription/viewConfirmDelete for modal routing"
  - "syncDetail pattern: update detail panel after any list navigation to keep selection in sync"
  - "tickCmd loop: Init returns tickCmd, Update on tickMsg returns next tickCmd for continuous uptime"

requirements-completed: [TUI-01, TUI-02, TUI-03, TUI-05, TUI-06]

# Metrics
duration: 3min
completed: 2026-02-25
---

# Phase 4 Plan 2: Root Model and TUI Launch Summary

**Split-pane TUI root model composing list/detail/statusbar with vim-style navigation, help overlay, and CLI launch via cobra root command**

## Performance

- **Duration:** 3 min
- **Started:** 2026-02-25T10:02:57Z
- **Completed:** 2026-02-25T10:06:09Z
- **Tasks:** 2
- **Files modified:** 3 (2 created, 1 modified)

## Accomplishments
- Root model composing server list (1/3 width), detail panel (2/3 width), and status bar into split-pane layout with border separator
- Keyboard navigation routing: j/k navigate, enter select, q quit, ? help overlay, / fuzzy filter, esc back/clear filter
- Help overlay rendering centered bordered box with all 14 keybindings from FullHelp groups
- CLI root command RunE launches TUI via tea.NewProgram, loading config, server store, and engine

## Task Commits

Each task was committed atomically:

1. **Task 1: Create root model with split-pane layout and navigation** - `3bacdaf` (feat)
2. **Task 2: Create help overlay and wire TUI launch from CLI** - `bfa0423` (feat)

## Files Created/Modified
- `internal/tui/app.go` - Root model with New constructor, Init/Update/View, viewState routing, split-pane layout
- `internal/tui/help.go` - Help overlay with centered bordered box via lipgloss.Place
- `internal/cli/root.go` - Root command RunE now launches TUI instead of showing help

## Decisions Made
- Used KeyPressMsg.Keystroke() string matching for key routing -- more readable than inspecting Key struct fields
- Help overlay replaces content entirely (lipgloss.Place centers the help box) rather than transparency/dimming
- List filtering state gates key routing: when Filtering, all keys delegated to list model
- Root RunE creates config/store/engine inline rather than package-level initialization

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- TUI application fully launchable via `azad` command with no arguments
- All visual components composed and responsive to terminal resizing
- Ready for Plan 03 to add server interaction (add server, subscriptions, delete, connect flow)
- viewAddServer, viewAddSubscription, viewConfirmDelete states stubbed and ready for implementation

## Self-Check: PASSED

- internal/tui/app.go: EXISTS
- internal/tui/help.go: EXISTS
- internal/cli/root.go: MODIFIED
- Task 1 commit 3bacdaf: VERIFIED
- Task 2 commit bfa0423: VERIFIED
- go build ./...: OK
- go vet ./internal/tui/...: OK
- go test ./...: OK

---
*Phase: 04-tui-and-server-interaction*
*Completed: 2026-02-25*

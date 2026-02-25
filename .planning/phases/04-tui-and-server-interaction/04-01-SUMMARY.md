---
phase: 04-tui-and-server-interaction
plan: 01
subsystem: ui
tags: [bubbletea-v2, lipgloss-v2, bubbles-v2, tui, charmbracelet, adaptive-colors]

# Dependency graph
requires:
  - phase: 02-protocol-parsing
    provides: protocol.Server struct used by serverItem and detailModel
  - phase: 03-connection-engine
    provides: engine.ConnectionStatus enum used by statusBarModel
provides:
  - Theme system with LightDark-based adaptive colors (ColorPair, Styles, NewStyles)
  - Centralized keyMap with help.KeyMap interface for help overlay
  - Custom Bubble Tea message types for async TUI operations
  - Server list panel wrapping bubbles/list with fuzzy filtering
  - Detail panel rendering full server info
  - Status bar with connection state, server name, port, uptime
affects: [04-02-root-model, 04-03-server-interaction]

# Tech tracking
tech-stack:
  added: [charm.land/bubbletea/v2, charm.land/lipgloss/v2, charm.land/bubbles/v2]
  patterns: [LightDark adaptive colors, help.KeyMap interface, list.DefaultItem, ColorPair struct for theme]

key-files:
  created:
    - internal/tui/theme.go
    - internal/tui/keys.go
    - internal/tui/messages.go
    - internal/tui/serverlist.go
    - internal/tui/detail.go
    - internal/tui/statusbar.go
  modified:
    - go.mod
    - go.sum

key-decisions:
  - "LightDark function replaces AdaptiveColor in lipgloss v2 -- Theme uses ColorPair with Resolve(isDark) method"
  - "list.New in bubbles v2 still takes 4 args (items, delegate, width, height) despite research suggesting otherwise"
  - "Styles struct holds resolved styles; NewStyles(theme, isDark) creates them at render time from BackgroundColorMsg"

patterns-established:
  - "ColorPair pattern: store light/dark color.Color pairs, resolve via lipgloss.LightDark at render time"
  - "Styles struct: all themed styles resolved together, passed to child components via SetStyles()"
  - "statusBarHeight constant for layout calculations"

requirements-completed: [TUI-04, TUI-07, SRVR-01]

# Metrics
duration: 4min
completed: 2026-02-25
---

# Phase 4 Plan 1: TUI Visual Foundation Summary

**Charmbracelet v2 TUI building blocks: adaptive theme with LightDark colors, vim-style keyMap, 11 async message types, server list with fuzzy filtering, detail panel, and connection status bar**

## Performance

- **Duration:** 4 min
- **Started:** 2026-02-25T09:55:31Z
- **Completed:** 2026-02-25T09:59:45Z
- **Tasks:** 3
- **Files modified:** 8 (6 created, 2 modified)

## Accomplishments
- Theme system using lipgloss v2 LightDark pattern with ColorPair and resolved Styles struct
- Centralized keyMap with 14 bindings satisfying help.KeyMap interface for automatic help generation
- Server list panel wrapping bubbles/list with DefaultItem interface and fuzzy search by name, address, protocol
- Status bar rendering connection state (color-coded), server name, SOCKS5 port, and formatted uptime

## Task Commits

Each task was committed atomically:

1. **Task 1: Create TUI foundation types (theme, keys, messages)** - `c796558` (feat)
2. **Task 2: Create server list panel with fuzzy filtering** - `f25c9ad` (feat)
3. **Task 3: Create detail panel and status bar components** - `6373846` (feat)

## Files Created/Modified
- `internal/tui/theme.go` - ColorPair type, Theme struct, DefaultTheme, Styles struct, NewStyles resolver
- `internal/tui/keys.go` - keyMap struct with 14 bindings, ShortHelp/FullHelp for help.KeyMap
- `internal/tui/messages.go` - 11 custom message types for async TUI operations
- `internal/tui/serverlist.go` - serverItem (list.DefaultItem), serversToItems, newServerList
- `internal/tui/detail.go` - detailModel with server info rendering and nil placeholder
- `internal/tui/statusbar.go` - statusBarModel with 4-section status display
- `go.mod` - Added charm.land/bubbletea/v2, lipgloss/v2, bubbles/v2 v2.0.0
- `go.sum` - Updated with new dependency checksums

## Decisions Made
- lipgloss v2 replaced AdaptiveColor with LightDark function -- adapted theme to use ColorPair struct with Resolve(isDark bool) method that uses lipgloss.LightDark internally
- list.New in bubbles v2 actually takes 4 arguments (items, delegate, width, height), not just items as research suggested -- used correct 4-arg constructor
- Styles resolved at runtime via NewStyles(theme, isDark) pattern, allowing re-resolution when BackgroundColorMsg arrives

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Adapted theme for lipgloss v2 API changes**
- **Found during:** Task 1 (theme.go creation)
- **Issue:** lipgloss v2 removed AdaptiveColor type, replacing it with LightDark function
- **Fix:** Created ColorPair struct with Resolve(isDark) method; Styles struct holds resolved styles; NewStyles() factory creates them from theme and isDark flag
- **Files modified:** internal/tui/theme.go
- **Verification:** go build ./internal/tui/... passes
- **Committed in:** c796558

**2. [Rule 3 - Blocking] Fixed missing transitive dependencies**
- **Found during:** Task 2 (serverlist.go compilation)
- **Issue:** go.sum missing entries for github.com/sahilm/fuzzy and github.com/atotto/clipboard
- **Fix:** Ran go get for bubbles/v2/list and bubbles/v2/textinput to pull transitive deps
- **Files modified:** go.mod, go.sum
- **Verification:** go build ./internal/tui/... passes
- **Committed in:** f25c9ad

---

**Total deviations:** 2 auto-fixed (2 blocking issues)
**Impact on plan:** Both fixes necessary for compilation. No scope creep -- v2 API differences from research required adaptation.

## Issues Encountered
None beyond the deviations documented above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All six TUI building blocks ready for composition in Plan 02 (root model)
- Theme, keys, messages, server list, detail panel, and status bar are typed and compile
- Root model will compose these components into a split-pane layout with message routing

## Self-Check: PASSED

- All 6 files exist in internal/tui/
- All 3 task commits verified (c796558, f25c9ad, 6373846)
- Package build: OK
- Full project build: OK

---
*Phase: 04-tui-and-server-interaction*
*Completed: 2026-02-25*

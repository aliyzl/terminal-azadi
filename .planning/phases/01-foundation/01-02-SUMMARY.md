---
phase: 01-foundation
plan: 02
subsystem: config, lifecycle
tags: [go, koanf, yaml, xdg, signal-handling, cleanup, terminal-reset]

# Dependency graph
requires:
  - phase: 01-foundation/01
    provides: "Go module with cobra CLI skeleton and project layout"
provides:
  - "koanf-based YAML config system with XDG-compliant paths and defaults"
  - "Config Load/Save with struct binding and round-trip persistence"
  - "Signal handling via context cancellation (SIGINT/SIGTERM)"
  - "azad --cleanup reads proxy state file and reports/cleans dirty state"
  - "azad --reset-terminal restores terminal via stty sane fallback"
  - "azad config displays current configuration with file path"
affects: [02-protocol-parsing, 03-connection-engine, 04-tui, 05-quick-connect]

# Tech tracking
tech-stack:
  added: [koanf/v2 v2.3.2, koanf/parsers/yaml, koanf/providers/file, koanf/providers/confmap, koanf/providers/structs, golang.org/x/term]
  patterns: [fresh koanf instance for writes (no global mutable state), confmap defaults with file overlay, XDG path via os.UserConfigDir, signal.NotifyContext for context-based shutdown]

key-files:
  created:
    - internal/config/paths.go
    - internal/config/config.go
    - internal/config/config_test.go
    - internal/lifecycle/signals.go
    - internal/lifecycle/cleanup.go
  modified:
    - cmd/azad/main.go
    - internal/cli/root.go
    - internal/cli/config_cmd.go
    - go.mod
    - go.sum

key-decisions:
  - "Fresh koanf instance for Save() to avoid race conditions on package-level state"
  - "Non-terminal stdin handled gracefully in reset-terminal (no error when piped)"
  - "ProxyState struct with JSON serialization for .state.json crash recovery file"
  - "stty sane as crash-recovery fallback (bubbletea handles normal terminal restore in Phase 4)"

patterns-established:
  - "Config Load pattern: confmap defaults -> file overlay -> unmarshal to struct"
  - "Config Save pattern: fresh koanf instance -> structs.Provider -> yaml.Marshal -> WriteFile 0600"
  - "Signal context: lifecycle.WithShutdown wraps root context in main.go"
  - "Cleanup flag pattern: PersistentPreRunE checks flags, calls lifecycle functions, os.Exit(0)"

requirements-completed: [FNDN-02, FNDN-04]

# Metrics
duration: 4min
completed: 2026-02-25
---

# Phase 1 Plan 02: Config System and Lifecycle Management Summary

**koanf-based YAML config with XDG paths, signal handling via context cancellation, and crash-recovery commands (--cleanup, --reset-terminal)**

## Performance

- **Duration:** 4 min
- **Started:** 2026-02-24T22:50:12Z
- **Completed:** 2026-02-24T22:54:40Z
- **Tasks:** 2
- **Files modified:** 10

## Accomplishments
- koanf v2 config system reads/writes YAML at XDG-compliant path with correct defaults (socks_port=1080, http_port=8080)
- Config round-trips through Save/Load with struct binding, verified by unit tests
- Signal handling wraps root context -- SIGINT/SIGTERM trigger clean cancellation chain
- --cleanup reads .state.json, reports dirty proxy state, and removes state file
- --reset-terminal restores terminal via stty sane with graceful handling of non-terminal stdin

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement koanf config system with XDG paths** - `1c0db9c` (feat)
2. **Task 2: Implement signal handling and cleanup commands** - `9172ad6` (feat)

## Files Created/Modified
- `internal/config/paths.go` - XDG path resolution (Dir, FilePath, EnsureDir, DataDir, StateFilePath)
- `internal/config/config.go` - Config struct with koanf Load/Save, defaults
- `internal/config/config_test.go` - Unit tests for save/load round-trip and defaults
- `internal/lifecycle/signals.go` - WithShutdown wrapping signal.NotifyContext
- `internal/lifecycle/cleanup.go` - RunCleanup (proxy state) and RunResetTerminal (stty sane)
- `cmd/azad/main.go` - Signal context wrapping cobra execution
- `internal/cli/root.go` - Real lifecycle calls in PersistentPreRunE
- `internal/cli/config_cmd.go` - Config display with path and values
- `go.mod` - Added koanf and term dependencies
- `go.sum` - Updated checksums

## Decisions Made
- **Fresh koanf instance for writes:** Save() creates a new koanf.New(".") instance each time to avoid race conditions on shared state, following RESEARCH anti-pattern guidance.
- **Non-terminal stdin handling:** RunResetTerminal checks term.IsTerminal() before running stty sane, printing a clean message instead of failing when stdin is piped or non-interactive.
- **ProxyState as JSON struct:** The .state.json file uses a typed Go struct for type-safe serialization, with fields for proxy_set, ports, network_service, and PID. Phase 3 will write this file; Phase 1 only reads it for cleanup.
- **stty sane for crash recovery:** Chosen over term.Restore because crash recovery has no saved state to restore from. stty sane resets all terminal attributes to sane defaults, which is the correct approach for recovering from an unclean exit.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed reset-terminal failing on non-terminal stdin**
- **Found during:** Task 2 (verification step)
- **Issue:** stty sane fails with "stdin isn't a terminal" when running in non-interactive shell or piped context
- **Fix:** Added term.IsTerminal() check before running stty sane; returns clean message for non-terminal stdin
- **Files modified:** internal/lifecycle/cleanup.go
- **Verification:** --reset-terminal succeeds both in interactive terminal and piped contexts
- **Committed in:** 9172ad6

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Auto-fix necessary for correctness in non-interactive environments. No scope creep.

## Issues Encountered
None -- all dependencies resolved cleanly and existing CLI skeleton integrated without friction.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Foundation phase complete: Go module with xray-core, CLI routing, config persistence, and lifecycle management
- Phase 2 (Protocol Parsing) can use config.Load/Save for storing parsed server data
- Phase 3 (Connection Engine) will write .state.json on proxy set and read it on cleanup
- Phase 4 (TUI) will rely on signal context for graceful shutdown of bubbletea
- Config directory (~/.config/azad/) is ready for servers.json and other data files

## Self-Check: PASSED

- All 9 created/modified files verified on disk
- Commit 1c0db9c (Task 1) verified in git log
- Commit 9172ad6 (Task 2) verified in git log

---
*Phase: 01-foundation*
*Completed: 2026-02-25*

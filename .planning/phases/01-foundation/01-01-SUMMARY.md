---
phase: 01-foundation
plan: 01
subsystem: cli
tags: [go, xray-core, cobra, cli-skeleton]

# Dependency graph
requires:
  - phase: none
    provides: "First plan in project"
provides:
  - "Go module with xray-core as library dependency"
  - "Cobra CLI skeleton with connect, servers, config subcommands"
  - "Root-level --cleanup and --reset-terminal flags (stubs)"
  - "Project layout: cmd/azad/ + internal/{cli,config,lifecycle}/"
affects: [01-02, 02-protocol-parsing, 03-connection-engine]

# Tech tracking
tech-stack:
  added: [xray-core v1.260206.0, cobra v1.9.1]
  patterns: [cmd/internal Go layout, explicit AddCommand registration, PersistentPreRunE for root flags]

key-files:
  created:
    - go.mod
    - go.sum
    - .gitignore
    - cmd/azad/main.go
    - internal/cli/root.go
    - internal/cli/connect.go
    - internal/cli/servers.go
    - internal/cli/config_cmd.go
  modified: []

key-decisions:
  - "Used go 1.25.7 toolchain directive (Go 1.26 runtime resolves xray-core requirement automatically)"
  - "Added root RunE returning Help() so standalone --cleanup/--reset-terminal flags trigger PersistentPreRunE"
  - "koanf and golang.org/x/term not added yet (go mod tidy removes unused deps; will be added in plan 01-02)"

patterns-established:
  - "Explicit AddCommand() in NewRootCmd builder function, no init() registration"
  - "Subcommand per file under internal/cli/ (connect.go, servers.go, config_cmd.go)"
  - "Blank imports for xray-core side-effect registration in main.go"
  - "Build-time version injection via var version = dev"

requirements-completed: [FNDN-01, FNDN-03]

# Metrics
duration: 35min
completed: 2026-02-25
---

# Phase 1 Plan 01: Go Module and CLI Skeleton Summary

**Go module with xray-core v1.260206.0 as library dependency and cobra CLI skeleton with 3 subcommands and 2 root flags**

## Performance

- **Duration:** 35 min
- **Started:** 2026-02-24T22:11:00Z
- **Completed:** 2026-02-24T22:46:50Z
- **Tasks:** 2
- **Files modified:** 8

## Accomplishments
- Go module initialized with xray-core compiling as library dependency (44MB binary validates full protocol support)
- Cobra CLI skeleton with connect, servers, config subcommands producing stub output
- Root-level --cleanup and --reset-terminal persistent flags working standalone (not just with subcommands)
- Project follows standard Go cmd/ + internal/ layout with no init() functions

## Task Commits

Each task was committed atomically:

1. **Task 1: Initialize Go module with dependencies and project structure** - `77bd245` (feat)
2. **Task 2: Create cobra CLI skeleton with subcommands and root flags** - `592b73b` (feat)

## Files Created/Modified
- `go.mod` - Go module definition with xray-core, cobra, and all transitive dependencies
- `go.sum` - Dependency checksums for reproducible builds
- `.gitignore` - Ignores compiled binary, geo data files, OS artifacts, logs
- `cmd/azad/main.go` - Application entry point with xray-core blank imports and version injection
- `internal/cli/root.go` - Root cobra command with --cleanup, --reset-terminal flags and subcommand registration
- `internal/cli/connect.go` - Connect subcommand stub (Phase 3)
- `internal/cli/servers.go` - Servers subcommand stub (Phase 4)
- `internal/cli/config_cmd.go` - Config subcommand stub (Plan 02)

## Decisions Made
- **go 1.25.7 in go.mod:** Go 1.26 runtime handles the xray-core go directive requirement via toolchain auto-download. The build succeeds without manually bumping go.mod to 1.26.
- **Root RunE returning Help():** Without a RunE on the root command, cobra treats it as help-only, so `azad --cleanup` would show help instead of running PersistentPreRunE. Adding `RunE: func(...) { return cmd.Help() }` fixes this.
- **Deferred koanf/term deps:** go mod tidy correctly removes unused dependencies. koanf, golang.org/x/term, and koanf providers will be added in plan 01-02 when actually imported.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Resolved incomplete go.sum for xray-core transitive dependencies**
- **Found during:** Task 1
- **Issue:** go.sum only had direct dependency checksums; xray-core's 100+ transitive dependencies (protobuf, grpc, quic-go, wireguard, etc.) were missing
- **Fix:** Ran `go mod tidy` which downloaded and checksummed all transitive dependencies
- **Files modified:** go.mod, go.sum
- **Verification:** `go build ./cmd/azad` compiles successfully
- **Committed in:** 77bd245

**2. [Rule 1 - Bug] Fixed --cleanup and --reset-terminal not working without subcommand**
- **Found during:** Task 2
- **Issue:** Running `azad --cleanup` showed help output instead of stub message because root command had no RunE, making it help-only
- **Fix:** Added `RunE` to root command that returns `cmd.Help()`, making root command runnable so PersistentPreRunE triggers
- **Files modified:** internal/cli/root.go
- **Verification:** `azad --cleanup` prints "cleanup: not yet implemented" and exits 0
- **Committed in:** 592b73b

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 bug)
**Impact on plan:** Both auto-fixes necessary for correctness. No scope creep.

## Issues Encountered
- xray-core transitive dependency download took several minutes due to the massive dependency tree (grpc, protobuf, wireguard, quic-go, gvisor, etc.)
- Binary size is 44MB as expected (documented concern in STATE.md); `-ldflags="-s -w"` can reduce this 20-30% for distribution

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Go module compiles with xray-core as library dependency
- CLI routing complete: 3 subcommands + 2 root flags all functional
- Plan 01-02 can wire koanf config system, signal handling, and real cleanup/reset-terminal logic
- internal/config/ and internal/lifecycle/ directories exist (empty, ready for Plan 02)

## Self-Check: PASSED

- All 8 created files verified on disk
- Commit 77bd245 (Task 1) verified in git log
- Commit 592b73b (Task 2) verified in git log

---
*Phase: 01-foundation*
*Completed: 2026-02-25*

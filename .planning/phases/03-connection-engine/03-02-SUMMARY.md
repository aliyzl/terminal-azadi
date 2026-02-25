---
phase: 03-connection-engine
plan: 02
subsystem: infra
tags: [networksetup, sysproxy, macos, exec, cleanup]

# Dependency graph
requires:
  - phase: 01-foundation
    provides: "lifecycle.ProxyState struct and RunCleanup stub"
provides:
  - "DetectNetworkService finds active macOS network service dynamically"
  - "SetSystemProxy sets SOCKS5/HTTP/HTTPS proxy via networksetup"
  - "UnsetSystemProxy disables all proxy types"
  - "RunCleanup actually reverses dirty proxy state (not just reporting)"
affects: [03-connection-engine, 04-tui]

# Tech tracking
tech-stack:
  added: []
  patterns: [package-level var for exec.Command testability, mock exec in unit tests]

key-files:
  created:
    - internal/sysproxy/detect.go
    - internal/sysproxy/sysproxy.go
    - internal/sysproxy/sysproxy_test.go
  modified:
    - internal/lifecycle/cleanup.go

key-decisions:
  - "Package-level var runCommand for testability instead of interface/dependency injection"
  - "Direct networksetup calls via os/exec instead of third-party sysproxy library"
  - "Cleanup warns but continues if UnsetSystemProxy fails (state file removal is critical)"

patterns-established:
  - "Mock exec pattern: package-level var replaced in tests with cleanup via defer"
  - "Proxy error tolerance: warn on networksetup failure, don't block cleanup"

requirements-completed: [CONN-05, CONN-06]

# Metrics
duration: 3min
completed: 2026-02-25
---

# Phase 3 Plan 2: System Proxy Management Summary

**macOS system proxy set/unset via networksetup with dynamic network service detection and crash-recovery cleanup reversal**

## Performance

- **Duration:** 3 min
- **Started:** 2026-02-25T07:36:18Z
- **Completed:** 2026-02-25T07:39:00Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Created internal/sysproxy package with DetectNetworkService, SetSystemProxy, and UnsetSystemProxy
- 7 unit tests verify all proxy operations without running real networksetup commands
- Upgraded RunCleanup to actually reverse dirty proxy state via networksetup (replaces Phase 1 stub)

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement system proxy detection and set/unset** - `0e3bfb2` (feat)
2. **Task 2: Upgrade cleanup with actual proxy reversal** - `dc4b17d` (feat)

## Files Created/Modified
- `internal/sysproxy/detect.go` - DetectNetworkService: parses networksetup output, prefers Wi-Fi/Ethernet
- `internal/sysproxy/sysproxy.go` - SetSystemProxy (6 commands) and UnsetSystemProxy (3 commands)
- `internal/sysproxy/sysproxy_test.go` - 7 tests with mock exec pattern for command injection
- `internal/lifecycle/cleanup.go` - Upgraded RunCleanup to call sysproxy.UnsetSystemProxy

## Decisions Made
- Used package-level `var runCommand` and `var execCommand` for testability instead of an interface -- simpler for this use case where there's only one implementation
- Direct `networksetup` calls via `os/exec` instead of `getlantern/sysproxy` library -- sysproxy is unmaintained (2016) and adds unnecessary complexity
- Cleanup warns but continues on UnsetSystemProxy failure -- removing the state file is more important than guaranteed proxy reversal (user can re-run --cleanup)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- sysproxy package ready for use by the connection engine (Plan 01/03)
- RunCleanup is fully operational for crash recovery
- SetSystemProxy ready to be called after Xray proxy starts (Plan 03)

## Self-Check: PASSED

- All 4 files exist on disk
- Both task commits (0e3bfb2, dc4b17d) verified in git log
- go build ./... passes
- go test ./internal/sysproxy/ passes (7/7)
- go vet ./internal/sysproxy/ ./internal/lifecycle/ passes

---
*Phase: 03-connection-engine*
*Completed: 2026-02-25*

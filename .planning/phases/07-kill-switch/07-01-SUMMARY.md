---
phase: 07-kill-switch
plan: 01
subsystem: infra
tags: [pfctl, pf, firewall, killswitch, osascript, privilege-escalation, macos]

# Dependency graph
requires:
  - phase: 01-foundation/02
    provides: "lifecycle.ProxyState struct and RunCleanup for crash recovery"
  - phase: 03-connection-engine/02
    provides: "sysproxy pattern with package-level var execCommand for testability"
provides:
  - "killswitch.Enable loads pf anchor rules blocking all non-VPN traffic"
  - "killswitch.Disable flushes anchor without touching Apple's pf state"
  - "killswitch.IsActive checks anchor for loaded rules"
  - "killswitch.Cleanup flushes anchor with soft error handling and manual recovery message"
  - "ProxyState extended with KillSwitchActive/ServerAddress/ServerPort (backwards-compatible)"
  - "RunCleanup handles kill switch state recovery alongside proxy state"
affects: [07-kill-switch/02]

# Tech tracking
tech-stack:
  added: []
  patterns: [pf anchor-based kill switch via pfctl, osascript privilege escalation with root fallback, base64 encoding for safe shell piping of pf rules]

key-files:
  created:
    - internal/killswitch/rules.go
    - internal/killswitch/privilege.go
    - internal/killswitch/killswitch.go
  modified:
    - internal/lifecycle/cleanup.go

key-decisions:
  - "Package-level var execCommand for testability (follows sysproxy pattern)"
  - "runPrivilegedOrSudo tries osascript then falls back to direct exec if root (handles headless/SSH)"
  - "Base64-encode rules for safe shell piping through osascript"
  - "Disable only flushes anchor (pfctl -a ... -F all), never calls pfctl -d which would break Apple's pf"
  - "Cleanup prints manual recovery command on privilege failure (prevents user lockout)"
  - "ProxyState kill switch fields use omitempty for backwards compatibility with existing state files"

patterns-established:
  - "pf anchor management: load via stdin pipe, flush via -F all, read via -sr"
  - "Privilege escalation: osascript with admin privileges, root fallback for headless"
  - "Kill switch state in ProxyState with omitempty for safe migration"

requirements-completed: [KILL-01, KILL-02, KILL-04]

# Metrics
duration: 2min
completed: 2026-02-26
---

# Phase 7 Plan 01: Kill Switch Infrastructure Summary

**pf anchor-based kill switch with IPv4+IPv6 blocking, osascript/root privilege escalation, and backwards-compatible ProxyState crash recovery**

## Performance

- **Duration:** 2 min
- **Started:** 2026-02-26T09:34:36Z
- **Completed:** 2026-02-26T09:37:05Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Created internal/killswitch package with GenerateRules producing pf anchor rules (block-policy drop, loopback pass, server pass, DHCP pass, DNS pass, IPv4+IPv6 block)
- Enable/Disable/IsActive/Cleanup public API handles both GUI (osascript) and headless (root) privilege escalation
- Extended ProxyState with kill switch fields (backwards-compatible via omitempty) and upgraded RunCleanup to flush pf anchor on crash recovery

## Task Commits

Each task was committed atomically:

1. **Task 1: Create killswitch package with pf rules, privilege escalation, and public API** - `efd812b` (feat)
2. **Task 2: Extend ProxyState for kill switch and upgrade RunCleanup** - `58e41fa` (feat)

## Files Created/Modified
- `internal/killswitch/rules.go` - GenerateRules produces pf anchor rules with IPv4+IPv6 blocking
- `internal/killswitch/privilege.go` - runPrivileged (osascript) and runPrivilegedOrSudo (with root fallback)
- `internal/killswitch/killswitch.go` - Enable/Disable/IsActive/Cleanup public API using pf anchor
- `internal/lifecycle/cleanup.go` - Extended ProxyState with KillSwitchActive/ServerAddress/ServerPort; upgraded RunCleanup

## Decisions Made
- Used package-level `var execCommand` for testability, consistent with the sysproxy package pattern
- `runPrivilegedOrSudo` tries osascript first, then checks for root -- handles the headless/SSH use case from research open question 3
- Base64-encode rules before piping through shell to avoid escaping issues with osascript
- Disable only flushes our anchor (`pfctl -a com.azad.killswitch -F all`), never calls `pfctl -d` which would disable all pf including Apple's rules (anti-pattern from research)
- Cleanup prints manual recovery command on failure to prevent user lockout (research pitfall 1)
- Kill switch ProxyState fields use `omitempty` so existing state files without these fields remain compatible (research pitfall 6)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- killswitch package ready for Plan 02 to wire into CLI (--kill-switch flag) and TUI (K keybinding toggle)
- ProxyState kill switch fields ready for the engine to write on connect with kill switch enabled
- RunCleanup handles both proxy and kill switch state, ready for startup recovery flow

## Self-Check: PASSED

- All 4 created/modified files verified on disk
- Commit efd812b (Task 1) verified in git log
- Commit 58e41fa (Task 2) verified in git log
- go build ./... passes
- go vet ./internal/killswitch/ ./internal/lifecycle/ passes

---
*Phase: 07-kill-switch*
*Completed: 2026-02-26*

---
phase: 08-split-tunneling
plan: 02
subsystem: networking
tags: [split-tunneling, cli, killswitch, pf-rules, cobra]

# Dependency graph
requires:
  - phase: 08-split-tunneling
    provides: "splittunnel package with Rule, Config, ParseRule, ToXrayRules"
  - phase: 07-kill-switch
    provides: "killswitch Enable/Disable, GenerateRules, pf anchor management"
  - phase: 03-connection-engine
    provides: "Engine.Start, BuildConfig, proxy lifecycle"
provides:
  - "GenerateRules with bypass IPs for split tunnel pf coordination"
  - "Engine.Start with optional split tunnel config (variadic)"
  - "CLI split-tunnel subcommand with add/remove/list/mode/enable/disable/clear"
  - "Connect flow loads split tunnel config and passes to engine and kill switch"
affects: [08-split-tunneling, tui, cli]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Variadic parameters for backwards-compatible API extensions (Enable, Engine.Start)"
    - "strings.Builder for efficient pf rule string construction"
    - "loadConfig helper factored out for reuse across CLI subcommands"
    - "Bypass IP extraction with domain DNS resolution for pf coordination"

key-files:
  created:
    - internal/cli/split_tunnel.go
  modified:
    - internal/killswitch/rules.go
    - internal/killswitch/killswitch.go
    - internal/engine/engine.go
    - internal/cli/connect.go
    - internal/cli/root.go

key-decisions:
  - "Variadic parameters on Enable and Engine.Start for zero-breaking-change API evolution"
  - "strings.Builder replaces fmt.Sprintf for rules.go to support dynamic bypass IP injection"
  - "loadConfig helper extracted in split_tunnel.go for DRY config load/save pattern"

patterns-established:
  - "Pattern: Variadic optional params for backwards-compatible function extensions"
  - "Pattern: CLI subcommand with shared loadConfig helper for config CRUD"
  - "Pattern: Domain-to-IP resolution for pf bypass with best-effort error handling"

requirements-completed: [SPLT-05, SPLT-06]

# Metrics
duration: 6min
completed: 2026-02-26
---

# Phase 8 Plan 2: Split Tunnel Wiring Summary

**Kill switch pf bypass rules for split tunnel IPs, Engine.Start with optional split tunnel config, and CLI subcommand with full CRUD for rule management**

## Performance

- **Duration:** 6 min
- **Started:** 2026-02-26T16:06:09Z
- **Completed:** 2026-02-26T16:12:49Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- GenerateRules accepts bypassIPs and injects pf pass rules for split tunnel direct traffic
- Engine.Start accepts optional variadic split tunnel config, backwards-compatible with all existing callers
- CLI split-tunnel subcommand registered with add/remove/list/mode/enable/disable/clear operations
- Connect flow loads split tunnel config from persisted config, passes to engine, and coordinates bypass IPs with kill switch

## Task Commits

Each task was committed atomically:

1. **Task 1: Kill switch coordination and Engine.Start split tunnel wiring** - `a591c49` (feat)
2. **Task 2: CLI split-tunnel subcommand and connect wiring** - `b230594` (feat)

## Files Created/Modified
- `internal/killswitch/rules.go` - GenerateRules with bypassIPs parameter, strings.Builder for dynamic rule construction
- `internal/killswitch/killswitch.go` - Enable with variadic bypassIPs for backwards compatibility
- `internal/engine/engine.go` - Engine.Start with variadic splittunnel.Config parameter
- `internal/cli/split_tunnel.go` - Full split-tunnel subcommand with add/remove/list/mode/enable/disable/clear
- `internal/cli/connect.go` - Split tunnel config loading, engine pass-through, bypass IP extraction for kill switch
- `internal/cli/root.go` - Registered newSplitTunnelCmd in root AddCommand

## Decisions Made
- Used variadic parameters on both Enable and Engine.Start so existing callers (CLI connect, TUI connect, TUI auto-connect, TUI kill switch toggle) compile without any changes
- Replaced fmt.Sprintf in GenerateRules with strings.Builder for flexible bypass IP injection between VPN server pass and block rules
- Extracted loadConfig helper function in split_tunnel.go to reduce duplication across 7 subcommands
- Domain/wildcard rules resolved to IPs via net.LookupHost for pf bypass (best-effort, non-fatal on DNS failure)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

**Pre-existing TUI build breakage (out of scope):**
Untracked/uncommitted TUI files from plan 08-03 break `go build ./...` (split_tunnel.go references undefined model fields, app.go references undefined methods). Package-specific builds for all plan-relevant packages (`killswitch`, `engine`, `cli`, `splittunnel`, `config`) succeed. Binary `go build ./cmd/azad` compiles successfully. Documented in `deferred-items.md`. Will be resolved when 08-03 executes.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Engine and kill switch wired for split tunnel, ready for TUI integration (08-03)
- CLI subcommand fully functional for rule management
- Connect flow passes split tunnel config end-to-end
- TUI integration (08-03) needs to add splitTunnelIdx field and SetSplitTunnel method to complete the wiring

---
*Phase: 08-split-tunneling*
*Completed: 2026-02-26*

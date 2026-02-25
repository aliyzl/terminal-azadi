---
phase: 03-connection-engine
plan: 03
subsystem: engine
tags: [xray-core, proxy-lifecycle, socks5, ip-verification, connect-command, cobra]

# Dependency graph
requires:
  - phase: 03-connection-engine
    provides: "BuildConfig converting protocol.Server to *core.Config (03-01)"
  - phase: 03-connection-engine
    provides: "DetectNetworkService, SetSystemProxy, UnsetSystemProxy (03-02)"
  - phase: 02-protocol-parsing
    provides: "protocol.Server struct and serverstore.Store"
  - phase: 01-foundation
    provides: "config.Load, lifecycle.ProxyState, WithShutdown context"
provides:
  - "Engine struct with Start/Stop lifecycle and ConnectionStatus state machine"
  - "VerifyIP fetches external IP through SOCKS5 proxy for routing confirmation"
  - "GetDirectIP fetches direct IP for comparison with proxy IP"
  - "azad connect command orchestrating full connection flow end-to-end"
affects: [04-tui, connect-command, disconnect-flow]

# Tech tracking
tech-stack:
  added: [golang.org/x/net/proxy]
  patterns: [state-machine-lifecycle, proxy-ip-verification, crash-safe-state-file]

key-files:
  created:
    - internal/engine/engine.go
    - internal/engine/verify.go
  modified:
    - internal/cli/connect.go

key-decisions:
  - "Engine stores server copy (not pointer) to avoid external mutation of connected server"
  - "VerifyIP uses Dial (not DialContext) on SOCKS5 dialer -- proxy.Dialer interface returns Dial"
  - "Connection errors are fatal (return error); system proxy and IP verify errors are warnings (print but continue)"
  - "Direct IP fetched best-effort before proxy verification; failure does not block connection"

patterns-established:
  - "Engine lifecycle: Start creates instance, Stop closes+nils, never reuse closed instance"
  - "Connection flow: get direct IP -> start engine -> write state -> set proxy -> verify -> compare IPs -> wait -> cleanup"
  - "Crash safety: write ProxyState before setting proxy so cleanup can reverse on crash"

requirements-completed: [CONN-01, CONN-02, CONN-03, CONN-04]

# Metrics
duration: 2min
completed: 2026-02-25
---

# Phase 3 Plan 03: Engine Lifecycle and Connect Command Summary

**Xray proxy engine with Start/Stop state machine, SOCKS5 IP verification, and wired connect command orchestrating server lookup through graceful shutdown**

## Performance

- **Duration:** 2 min
- **Started:** 2026-02-25T07:44:59Z
- **Completed:** 2026-02-25T07:47:21Z
- **Tasks:** 3
- **Files modified:** 3

## Accomplishments
- Engine struct wraps xray-core Instance with mutex-protected state machine (Disconnected -> Connecting -> Connected/Error)
- VerifyIP confirms proxy routing by fetching external IP through SOCKS5 dialer, GetDirectIP fetches baseline for comparison
- azad connect orchestrates the full flow: find server -> start engine -> write crash-recovery state -> set system proxy -> verify IP routing -> wait for SIGINT -> cleanup
- All Phase 3 components (config builder, system proxy, engine, connect command) wired together and compiling

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement engine lifecycle with state machine** - `fe26ccb` (feat)
2. **Task 2: Implement IP verification through proxy** - `d657448` (feat)
3. **Task 3: Wire connect command with full connection flow** - `7c0d731` (feat)

## Files Created/Modified
- `internal/engine/engine.go` - Engine struct with Start/Stop lifecycle, ConnectionStatus state machine, thread-safe accessors
- `internal/engine/verify.go` - VerifyIP (SOCKS5 proxy) and GetDirectIP (direct) for IP comparison
- `internal/cli/connect.go` - Full connect command: server lookup, engine start, proxy set, IP verify, graceful shutdown

## Decisions Made
- Engine stores a copy of the server (not a pointer to external data) to prevent mutation of connected server state
- VerifyIP uses `Dial` (not `DialContext`) since the `proxy.Dialer` interface only exposes `Dial` -- the HTTP client timeout handles cancellation
- Connection errors (engine start failure) are fatal and returned as errors; system proxy and IP verification failures are non-fatal warnings that print but don't block the proxy
- Direct IP is fetched best-effort before proxy verification; if it fails, connect prints only the proxy IP without comparison
- ProxyState is written to .state.json BEFORE calling SetSystemProxy for crash safety (cleanup can reverse even if crash occurs during proxy setup)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Phase 3 complete: all connection engine components are wired and compiling
- `azad connect` is the functional entry point for VPN connections
- Engine lifecycle, system proxy, crash recovery, and IP verification all integrated
- Ready for Phase 4 (TUI) to wrap the connect flow in a terminal interface

## Self-Check: PASSED

- All 3 files exist on disk (engine.go, verify.go, connect.go)
- All 3 task commits verified in git log (fe26ccb, d657448, 7c0d731)
- go build ./... passes
- go vet ./... passes
- go test ./internal/engine/ passes (12/12 config builder tests)

---
*Phase: 03-connection-engine*
*Completed: 2026-02-25*

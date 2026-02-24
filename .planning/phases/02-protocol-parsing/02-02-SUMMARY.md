---
phase: 02-protocol-parsing
plan: 02
subsystem: protocol
tags: [go, server-store, subscription, persistence, atomic-write, base64, tdd, concurrent]

# Dependency graph
requires:
  - phase: 02-protocol-parsing
    plan: 01
    provides: "Server struct, Protocol enum, ParseURI dispatcher, 4 protocol parsers"
provides:
  - "Server store with CRUD, atomic writes, and RWMutex concurrency safety"
  - "Subscription fetcher with base64 decode, BOM/CRLF handling, multi-protocol parsing"
  - "DecodeSubscription utility for base64 variant fallback and line normalization"
  - "ReplaceBySource for subscription refresh without losing manual servers"
affects: [03-connection-engine, 04-tui]

# Tech tracking
tech-stack:
  added: []
  patterns: [atomic file write (CreateTemp + Rename), RWMutex for concurrent store access, base64 4-variant fallback in subscription decode, httptest mock server for HTTP tests]

key-files:
  created:
    - internal/serverstore/store.go
    - internal/serverstore/store_test.go
    - internal/subscription/fetch.go
    - internal/subscription/decode.go
    - internal/subscription/fetch_test.go
  modified: []

key-decisions:
  - "Local decodeBase64 in subscription package to avoid coupling to protocol internals"
  - "Atomic write uses os.CreateTemp in same directory then os.Rename for crash safety"
  - "Load from non-existent file returns empty store (not error) for first-run experience"
  - "ReplaceBySource filters by SubscriptionSource field for subscription refresh"
  - "Internal save() helper called by locked methods avoids double-locking"

patterns-established:
  - "Atomic file persistence: CreateTemp + Write + Close + Rename with defer Remove cleanup"
  - "Internal unlocked helper (save) called by public locked methods (Add, Remove, Clear, ReplaceBySource)"
  - "httptest.NewServer for subscription fetch testing without real HTTP"
  - "Table-driven base64 variant tests with encoder functions"

requirements-completed: [PROT-05, PROT-06]

# Metrics
duration: 4min
completed: 2026-02-25
---

# Phase 2 Plan 02: Server Store & Subscription Fetcher Summary

**JSON server store with atomic writes, RWMutex concurrency, and subscription fetcher handling 4 base64 variants, BOM stripping, and CRLF normalization**

## Performance

- **Duration:** 4 min
- **Started:** 2026-02-24T23:32:55Z
- **Completed:** 2026-02-24T23:36:41Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- Server store with full CRUD (Add, Remove, List, Clear, FindByID, Count, ReplaceBySource) and atomic file persistence
- Subscription fetcher downloading URLs, decoding base64, parsing all 4 protocol types, setting SubscriptionSource
- DecodeSubscription utility handling UTF-8 BOM, 4 base64 encoding variants, and \r\n/\r line ending normalization
- 18 tests total (8 store + 10 subscription) all passing with -race flag, zero data races

## Task Commits

Each task was committed atomically:

1. **Task 1: Create store and subscription stubs with failing tests** - `6b7df1f` (test)
2. **Task 2: Implement store and subscription to pass all tests** - `bc2f83f` (feat)

_TDD plan: Task 1 = RED phase (failing tests + stubs), Task 2 = GREEN phase (implementations pass all tests)_

## Files Created/Modified
- `internal/serverstore/store.go` - Store struct with CRUD operations, atomic writes (CreateTemp+Rename), RWMutex
- `internal/serverstore/store_test.go` - 8 tests: round-trip, CRUD, FindByID, LoadEmpty, ReplaceBySource, concurrent access
- `internal/subscription/decode.go` - DecodeSubscription with BOM stripping, base64 4-variant fallback, CRLF normalization
- `internal/subscription/fetch.go` - Fetch with HTTP GET, User-Agent header, DecodeSubscription, ParseURI per line
- `internal/subscription/fetch_test.go` - 10 tests with httptest: valid, mixed, URL-safe, Windows CRLF, BOM, invalid, errors, base64 variants

## Decisions Made
- **Local decodeBase64:** Created a local copy in subscription package instead of exporting from protocol package, avoiding coupling between subscription and protocol internals. Same 4-variant fallback logic.
- **Internal save() helper:** Unlocked save() called by methods that already hold the mutex, avoiding double-locking issues. Public Save() acquires RLock itself.
- **Empty file = empty store:** Load from non-existent path returns nil error with empty server list, providing clean first-run experience.
- **ReplaceBySource for subscription refresh:** Filters out all servers matching a SubscriptionSource then appends new ones, preserving manually-added servers (per RESEARCH Open Question 3).

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None - all stdlib functions worked as expected. No external dependencies needed.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Complete data pipeline ready: URI string -> parsed Server -> persistent store
- Subscription fetcher provides bulk import from provider URLs
- ReplaceBySource supports subscription refresh without losing manual servers
- Phase 2 complete -- ready for Phase 3 (Connection Engine) which generates Xray JSON configs from Server structs
- No new dependencies added -- everything uses Go stdlib

## Self-Check: PASSED

- All 5 created source/test files verified on disk
- Commit 6b7df1f (Task 1 RED) verified in git log
- Commit bc2f83f (Task 2 GREEN) verified in git log

---
*Phase: 02-protocol-parsing*
*Completed: 2026-02-25*

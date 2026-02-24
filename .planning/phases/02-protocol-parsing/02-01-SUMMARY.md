---
phase: 02-protocol-parsing
plan: 01
subsystem: protocol
tags: [go, uri-parsing, vless, vmess, trojan, shadowsocks, tdd, base64]

# Dependency graph
requires:
  - phase: 01-foundation
    provides: "Go module with xray-core and project layout"
provides:
  - "Server struct with Protocol enum and all proxy fields"
  - "ParseVLESS, ParseVMess, ParseTrojan, ParseShadowsocks parsers"
  - "ParseURI scheme dispatcher routing by URI prefix"
  - "Shared helpers: defaultString, decodeBase64, jsonFlexInt"
  - "41 table-driven tests covering all protocols and edge cases"
affects: [02-02, 03-connection-engine, 04-tui]

# Tech tracking
tech-stack:
  added: []
  patterns: [net/url.Parse for RFC 3986 protocols, base64 fallback chain for VMess, jsonFlexInt for VMess string-typed numbers, SIP002 dual userinfo detection by colon presence]

key-files:
  created:
    - internal/protocol/server.go
    - internal/protocol/helpers.go
    - internal/protocol/parse.go
    - internal/protocol/vless.go
    - internal/protocol/vmess.go
    - internal/protocol/trojan.go
    - internal/protocol/shadowsocks.go
    - internal/protocol/parse_test.go
    - internal/protocol/vless_test.go
    - internal/protocol/vmess_test.go
    - internal/protocol/trojan_test.go
    - internal/protocol/shadowsocks_test.go
  modified: []

key-decisions:
  - "Flat Server struct with omitempty JSON tags for optional fields (no protocol-specific sub-structs)"
  - "decodeBase64 tries 4 encoding variants in order: StdEncoding, RawStdEncoding, URLEncoding, RawURLEncoding"
  - "Trojan defaults port to 443 and TLS to tls (unlike VLESS which defaults to none)"
  - "SS userinfo detection: colon present means plaintext method:password, absent means base64-encoded"

patterns-established:
  - "Table-driven tests with check functions for protocol parsers"
  - "url.Hostname()/url.Port() for IPv6-safe host:port extraction (never split url.Host on colon)"
  - "Fragment fallback to address:port when no server name in URI"
  - "NewID() generates UUID v4 via crypto/rand for server identity"

requirements-completed: [PROT-01, PROT-02, PROT-03, PROT-04]

# Metrics
duration: 5min
completed: 2026-02-25
---

# Phase 2 Plan 01: Protocol URI Parsers Summary

**Four protocol URI parsers (VLESS, VMess, Trojan, Shadowsocks) with TDD, unified Server struct, and 41 table-driven tests covering base64 variants, IPv6, jsonFlexInt, and SIP002 dual format**

## Performance

- **Duration:** 5 min
- **Started:** 2026-02-24T23:23:50Z
- **Completed:** 2026-02-24T23:29:15Z
- **Tasks:** 2
- **Files modified:** 12

## Accomplishments
- Server struct with Protocol enum, 30+ fields covering all 4 protocols, JSON tags with omitempty
- All 4 protocol parsers: VLESS (net/url), VMess (base64+JSON with jsonFlexInt), Trojan (net/url, port default 443), Shadowsocks (SIP002 dual base64/plaintext userinfo)
- ParseURI dispatcher routing by scheme prefix with whitespace trimming
- Shared helpers: decodeBase64 with 4-encoding fallback chain, jsonFlexInt for string-or-number JSON fields, defaultString
- 41 table-driven tests: 10 VLESS (including REALITY, gRPC, IPv6, error cases), 10 VMess (padded/unpadded/URL-safe base64, string port/aid, error cases), 8 Trojan (WS, gRPC, REALITY, default port, error cases), 6 SS (base64 SIP002, AEAD-2022 plaintext, URL-encoded password, error cases), 7 ParseURI (dispatch + error + whitespace)

## Task Commits

Each task was committed atomically:

1. **Task 1: Create Server struct, helpers, and failing tests for all parsers** - `48299c3` (test)
2. **Task 2: Implement all parsers to pass tests** - `8f9f43a` (feat)

_TDD plan: Task 1 = RED phase (failing tests + stubs), Task 2 = GREEN phase (implementations pass all tests)_

## Files Created/Modified
- `internal/protocol/server.go` - Server struct, Protocol enum, NewID() UUID generator
- `internal/protocol/helpers.go` - defaultString, decodeBase64 (4-variant fallback), jsonFlexInt
- `internal/protocol/parse.go` - ParseURI scheme dispatcher
- `internal/protocol/vless.go` - ParseVLESS with net/url, query param extraction, REALITY support
- `internal/protocol/vmess.go` - ParseVMess with base64 decode, JSON unmarshal, jsonFlexInt
- `internal/protocol/trojan.go` - ParseTrojan with password auth, default port 443, TLS default
- `internal/protocol/shadowsocks.go` - ParseShadowsocks with SIP002 dual userinfo detection
- `internal/protocol/parse_test.go` - 7 table-driven tests for ParseURI dispatcher
- `internal/protocol/vless_test.go` - 10 tests including REALITY, gRPC, IPv6, error cases
- `internal/protocol/vmess_test.go` - 10 tests with test helper for base64 payload generation
- `internal/protocol/trojan_test.go` - 8 tests including WS, gRPC, REALITY, default port
- `internal/protocol/shadowsocks_test.go` - 6 tests for base64, AEAD-2022, URL-encoded, error cases

## Decisions Made
- **Flat Server struct:** All protocols share one struct with optional fields rather than protocol-specific sub-types. This simplifies serialization and the server store (plan 02-02).
- **decodeBase64 ordering:** StdEncoding first (most common), then RawStdEncoding, URLEncoding, RawURLEncoding. The ordering handles the most common VMess encoding variant first.
- **Trojan default TLS:** Defaults to "tls" (not "none") since Trojan is inherently TLS-based, unlike VLESS which defaults to "none".
- **SS colon detection:** Checking `u.User.String()` for ":" before deciding base64 vs plaintext, following SIP002 convention where AEAD-2022 uses plaintext and legacy uses base64.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None - all stdlib parsing functions worked as expected. No external dependencies needed.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Server struct and all 4 parsers ready for server store persistence (02-02)
- ParseURI dispatcher provides single entry point for subscription parsing (02-02)
- RawURI field preserved for re-parsing or display purposes
- AddedAt timestamp and NewID() provide metadata for server management
- No new dependencies added - everything uses Go stdlib

## Self-Check: PASSED

- All 12 created source/test files verified on disk
- Commit 48299c3 (Task 1 RED) verified in git log
- Commit 8f9f43a (Task 2 GREEN) verified in git log

---
*Phase: 02-protocol-parsing*
*Completed: 2026-02-25*

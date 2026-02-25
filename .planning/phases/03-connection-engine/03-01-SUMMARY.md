---
phase: 03-connection-engine
plan: 01
subsystem: engine
tags: [xray-core, json-config, vless, vmess, trojan, shadowsocks, serial]

# Dependency graph
requires:
  - phase: 02-protocol-parsing
    provides: "protocol.Server struct with all protocol/transport/security fields"
provides:
  - "BuildConfig function converting protocol.Server to valid *core.Config"
  - "XrayConfig JSON struct types for Xray config generation"
  - "Protocol-specific outbound builders (VLESS, VMess, Trojan, Shadowsocks)"
  - "Stream settings builder for transport (tcp, ws, grpc, httpupgrade) and security (tls, reality, none)"
affects: [03-connection-engine, engine-lifecycle, connect-command]

# Tech tracking
tech-stack:
  added: [xray-core/infra/conf/serial]
  patterns: [json-config-builder, protocol-switch-dispatch, local-type-definitions]

key-files:
  created:
    - internal/engine/config.go
    - internal/engine/config_test.go
  modified: []

key-decisions:
  - "Return both XrayConfig and *core.Config from BuildConfig for testability and inspection"
  - "Use local type definitions inside each builder function to avoid package-level type pollution"
  - "Shadowsocks plain gets no streamSettings; non-tcp Shadowsocks gets stream settings"
  - "REALITY fingerprint defaults to chrome when empty"
  - "VMess security defaults to auto when empty; VLESS encryption defaults to none when empty"

patterns-established:
  - "JSON config builder pattern: struct -> json.Marshal -> serial.LoadJSONConfig -> *core.Config"
  - "Protocol outbound switch dispatch with per-protocol builder helpers"
  - "TestMain sets XRAY_LOCATION_ASSET for geoip.dat access in tests"

requirements-completed: [CONN-01]

# Metrics
duration: 5min
completed: 2026-02-25
---

# Phase 3 Plan 01: Xray JSON Config Builder Summary

**BuildConfig converts protocol.Server to valid Xray *core.Config for VLESS/VMess/Trojan/Shadowsocks with transport and security variants via serial.LoadJSONConfig**

## Performance

- **Duration:** 5 min
- **Started:** 2026-02-25T07:36:23Z
- **Completed:** 2026-02-25T07:41:54Z
- **Tasks:** 2 (TDD RED + GREEN; no refactor needed)
- **Files modified:** 2

## Accomplishments
- BuildConfig produces valid Xray *core.Config for all 4 protocols verified by serial.LoadJSONConfig
- Transport settings correctly map tcp, ws (path + Host header), grpc (serviceName), and httpupgrade (path + host)
- Security settings correctly map tls (serverName, fingerprint, ALPN, allowInsecure), reality (publicKey, shortId, fingerprint, spiderX), and none
- 12 table-driven tests covering all protocol/transport/security combinations plus defaults and error cases

## Task Commits

Each task was committed atomically:

1. **Task 1: TDD RED - Failing tests** - `bb31fb6` (test)
2. **Task 2: TDD GREEN - Implementation** - `5d700bd` (feat)

_No refactor commit needed -- code was clean after GREEN phase._

## Files Created/Modified
- `internal/engine/config.go` - Xray JSON config builder with BuildConfig function and all struct types
- `internal/engine/config_test.go` - 12 table-driven tests for all protocols, transports, security modes, and edge cases

## Decisions Made
- Return both XrayConfig (for test inspection) and *core.Config (for engine use) from BuildConfig -- enables thorough JSON structure verification in tests while providing the protobuf config for core.New()
- Use local type definitions inside each builder function rather than package-level types -- keeps the namespace clean since these types are only used for JSON marshaling
- Shadowsocks without explicit non-tcp network gets no streamSettings (matches Xray convention for plain SS)
- Test data uses valid REALITY public keys (base64-raw-url 32-byte) and hex shortIds to pass xray-core's strict validation

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed REALITY test data to use valid key formats**
- **Found during:** Task 2 (TDD GREEN implementation)
- **Issue:** Test used placeholder strings "test-public-key" and "sid123" for REALITY publicKey and shortId. xray-core validates publicKey as base64-raw-url 32-byte and shortId as hex up to 16 chars.
- **Fix:** Generated valid 32-byte base64-raw-url public key and used hex shortId "abcd1234" / "0a1b2c3d"
- **Files modified:** internal/engine/config_test.go
- **Verification:** serial.LoadJSONConfig accepts REALITY config without error
- **Committed in:** 5d700bd (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Necessary fix for test correctness. No scope creep.

## Issues Encountered
None beyond the REALITY key format issue documented above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- BuildConfig is ready for use by the engine lifecycle (03-03-PLAN.md)
- XrayConfig struct types are exported for any future inspection needs
- All protocol/transport/security combinations tested and validated by xray-core itself

---
*Phase: 03-connection-engine*
*Completed: 2026-02-25*

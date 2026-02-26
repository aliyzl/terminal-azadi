# Testing

## Current State

**No automated test framework is present.** The project has zero test files, no test runner configuration, and no CI/CD pipeline.

## Built-in Diagnostic Features

The codebase includes manual testing capabilities within `menu.sh`:

| Feature | Menu Option | What It Tests |
|---------|-------------|---------------|
| Ping servers | 4 (`do_ping`) | TCP connectivity to each server via `nc -z` with timing |
| Test connection | 11 (`test_connection`) | Verifies VPN works by curling `ifconfig.me` through SOCKS proxy |
| Full tunnel test | 17 (`test_full_tunnel`) | Compares direct IP vs VPN IP to confirm tunneling |
| Status check | 12 (`show_status`) | Shows server count, selected server, proxy PID state |

## Test Gaps

### Critical (no coverage)
- **VLESS URI parsing** — `link-to-full-config.sh` and `vless-link-to-config.sh` parse complex URIs with regex/string manipulation; no tests for edge cases (malformed links, special characters, missing params)
- **Config JSON generation** — Generated JSON is never validated; malformed output silently breaks xray
- **Subscription fetching** — Base64 decoding, line parsing, and server extraction untested
- **Process lifecycle** — PID file management (stale PIDs, race conditions) untested

### Important (no coverage)
- **System proxy toggling** — `networksetup` calls could fail silently
- **Shell helper installation** — `.zshrc` modification could corrupt existing config
- **Server list CRUD** — Add/remove/clear operations on `data/servers.txt`

## Recommendations

If testing is added, consider:
- **bats-core** (Bash Automated Testing System) — natural fit for bash scripts
- Focus on pure functions first: `get_host_port`, URI parsing in `link-to-full-config.sh`
- Test config generation by comparing output JSON against known-good fixtures
- Mock `nc`, `curl`, `networksetup` for integration tests

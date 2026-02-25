# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-24)

**Core value:** One command to connect to the fastest VPN server through a stunning terminal interface
**Current focus:** Phase 4: TUI and Server Interaction

## Current Position

Phase: 4 of 6 (TUI and Server Interaction)
Plan: 2 of 3 in current phase
Status: Executing Phase 4
Last activity: 2026-02-25 -- Completed 04-02-PLAN.md

Progress: [███████░░░] 69%

## Performance Metrics

**Velocity:**
- Total plans completed: 9
- Average duration: 7min
- Total execution time: 1.09 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 1. Foundation | 2 | 39min | 20min |
| 2. Protocol Parsing | 2 | 9min | 5min |
| 3. Connection Engine | 3 | 10min | 3min |
| 4. TUI & Server Interaction | 2 | 7min | 4min |

**Recent Trend:**
- Last 5 plans: 5min, 3min, 2min, 4min, 3min
- Trend: Stable

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- Xray-core as Go library (not external binary) -- single binary, no IPC
- Charmbracelet v2 stack (bubbletea/lipgloss/bubbles) -- charm.land module paths
- koanf over Viper for config -- avoids key-lowercasing bug and dep bloat
- cobra for CLI routing -- standard Go CLI framework
- Root RunE returning Help() so standalone --cleanup/--reset-terminal work without subcommand
- Explicit AddCommand() in builder function, no init() registration (01-01)
- Fresh koanf instance for Save() to avoid race conditions on shared state (01-02)
- stty sane for crash-recovery terminal reset, bubbletea handles normal restore (01-02)
- ProxyState JSON struct in .state.json for crash recovery (01-02)
- Flat Server struct with omitempty JSON tags for optional fields (02-01)
- decodeBase64 tries 4 encoding variants: StdEncoding, RawStdEncoding, URLEncoding, RawURLEncoding (02-01)
- Trojan defaults port 443 and TLS "tls"; VLESS defaults TLS "none" (02-01)
- SS userinfo colon detection for base64 vs plaintext format (02-01)
- Local decodeBase64 in subscription package to avoid coupling to protocol internals (02-02)
- Atomic write uses CreateTemp then Rename for crash-safe persistence (02-02)
- Load from non-existent file returns empty store (not error) for first-run (02-02)
- ReplaceBySource for subscription refresh: filter by SubscriptionSource, preserve manual servers (02-02)
- Direct networksetup calls via os/exec instead of getlantern/sysproxy library (03-02)
- Package-level var runCommand for exec testability instead of interface/DI (03-02)
- Cleanup warns but continues if UnsetSystemProxy fails -- state file removal is critical (03-02)
- Return both XrayConfig and *core.Config from BuildConfig for testability and inspection (03-01)
- Local type definitions inside builder functions to avoid package-level type pollution (03-01)
- REALITY fingerprint defaults to chrome; VMess security defaults to auto; VLESS encryption defaults to none (03-01)
- Engine stores server copy (not pointer) to avoid external mutation of connected server (03-03)
- Connection errors fatal, proxy/verify errors are warnings (continue with SOCKS5 proxy) (03-03)
- ProxyState written BEFORE SetSystemProxy for crash safety (03-03)
- VerifyIP uses Dial not DialContext -- proxy.Dialer interface constraint (03-03)
- LightDark replaces AdaptiveColor in lipgloss v2 -- ColorPair struct with Resolve(isDark) method (04-01)
- list.New in bubbles v2 takes 4 args (items, delegate, w, h) not just items (04-01)
- Styles resolved at runtime via NewStyles(theme, isDark) from BackgroundColorMsg (04-01)
- KeyPressMsg.Keystroke() string matching for key routing -- more readable than Key struct inspection (04-02)
- Help overlay replaces content via lipgloss.Place centering, not transparent overlay (04-02)
- List filtering state gates key routing -- Filtering mode delegates all keys to list (04-02)
- Root RunE loads config/store/engine inline and launches TUI via tea.NewProgram (04-02)

### Pending Todos

None yet.

### Blockers/Concerns

- (RESOLVED) sysproxy library rejected -- using direct networksetup calls via os/exec (03-02)
- Xray-core binary size confirmed at 44MB -- use -ldflags="-s -w" for distribution builds
- (RESOLVED) lipgloss v2 now stable v2.0.0 -- successfully integrated with LightDark pattern (04-01)

## Session Continuity

Last session: 2026-02-25
Stopped at: Completed 04-02-PLAN.md (Root model and TUI launch). Phase 4 in progress (2/3 plans).
Resume file: .planning/phases/04-tui-and-server-interaction/04-02-SUMMARY.md

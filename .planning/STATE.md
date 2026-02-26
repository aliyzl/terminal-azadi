# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-24)

**Core value:** One command to connect to the fastest VPN server through a stunning terminal interface
**Current focus:** Phase 6: Distribution

## Current Position

Phase: 6 of 8 (Distribution)
Plan: 2 of 4 in current phase
Status: In progress
Last activity: 2026-02-26 -- Completed 06-02 Geo asset auto-download and platform-gated cleanup

Progress: [█████████░] 90% (7/8 phases complete)

## Performance Metrics

**Velocity:**
- Total plans completed: 20
- Average duration: 6min
- Total execution time: 1.8 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 1. Foundation | 2 | 39min | 20min |
| 2. Protocol Parsing | 2 | 9min | 5min |
| 3. Connection Engine | 3 | 10min | 3min |
| 4. TUI & Server Interaction | 4 | 13min | 3min |
| 5. Quick Connect | 2 | 5min | 3min |
| 7. Kill Switch | 2 | 7min | 4min |
| 8. Split Tunneling | 3 | 20min | 7min |
| 6. Distribution | 2 | 5min | 3min |

**Recent Trend:**
- Last 5 plans: 7min, 6min, 7min, 2min, 3min
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
- Track ping latencies in model map rather than modifying serverstore -- avoids store API changes (04-03)
- Input modal command functions take store as parameter (not closing over model state) for goroutine safety (04-03)
- Overlay modals use lipgloss.Place centering over base content -- consistent with help overlay pattern (04-03)
- Explicit tea.PasteMsg case + view-aware default fallthrough for paste routing -- targeted fix over broader refactor (04-04)
- Latency fallback uses positive LatencyMs only (> 0) to skip servers with no ping data (05-01)
- Persistence errors are warnings-only since the primary proxy connection is already established (05-01)
- Duplicated writeProxyState/removeStateFile in TUI to avoid exporting cli internals or circular dependency (05-02)
- tea.Sequence for disconnect-then-reconnect ensures serial execution when switching servers (05-02)
- Auto-connect skips silently on empty store (no error flash) per QCON-01 requirement (05-02)
- Package-level var execCommand for killswitch testability, follows sysproxy pattern (07-01)
- runPrivilegedOrSudo tries osascript then falls back to direct exec if root for headless/SSH (07-01)
- Base64-encode pf rules for safe shell piping through osascript (07-01)
- Disable only flushes anchor, never calls pfctl -d which would break Apple's pf (07-01)
- Cleanup prints manual recovery command on privilege failure to prevent user lockout (07-01)
- ProxyState kill switch fields use omitempty for backwards compatibility (07-01)
- Variadic bool on disconnectCmd for optional kill switch disable (07-02)
- Read-modify-write tuiWriteProxyStateWithKS preserves existing proxy fields (07-02)
- Uppercase K keybinding avoids conflict with k navigation (07-02)
- No confirmation for disabling kill switch, only for enabling (07-02)
- TUI init checks killswitch.IsActive() directly from pf anchor for crash recovery (07-02)
- XrayRoutingRule in splittunnel package to break circular dependency with engine (08-01)
- Fix pre-existing test expecting IPIfNonMatch for nil split config -- was always AsIs (08-01)
- Variadic params on Enable and Engine.Start for zero-breaking-change API evolution (08-02)
- strings.Builder in GenerateRules for dynamic bypass IP injection (08-02)
- loadConfig helper extracted in split_tunnel.go for DRY CLI config pattern (08-02)
- Status bar additions pulled into Task 1 for compilation when splitTunnelSavedMsg handler needs SetSplitTunnel (08-03)
- extractBypassIPs skips wildcard rules -- cannot be resolved to specific IPs (08-03)
- keyMap KillSwitch renamed to Menu -- pre-existing fix committed with Task 2 (08-03)
- Single ldflags line with -s -w and version injection for GoReleaser simplicity (06-01)
- GH_PAT secret instead of GITHUB_TOKEN to prepare for Homebrew tap cross-repo push (06-01)
- tar.gz archive format for both macOS and Linux (06-01)
- Package-level var httpClient and Assets slice for test overridability without interfaces (06-02)
- runtime.GOOS conditional in cleanup.go preserves macOS behavior, adds Linux messages (06-02)

### Pending Todos

None yet.

### Blockers/Concerns

- (RESOLVED) sysproxy library rejected -- using direct networksetup calls via os/exec (03-02)
- Xray-core binary size confirmed at 44MB -- use -ldflags="-s -w" for distribution builds
- (RESOLVED) lipgloss v2 now stable v2.0.0 -- successfully integrated with LightDark pattern (04-01)

## Session Continuity

Last session: 2026-02-26
Stopped at: Completed 06-02-PLAN.md (Geo asset auto-download and platform-gated cleanup)
Resume file: .planning/phases/06-distribution/06-02-SUMMARY.md

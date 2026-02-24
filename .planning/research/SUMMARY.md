# Project Research Summary

**Project:** Azad -- Terminal VPN Client
**Domain:** Go TUI application wrapping a multi-protocol proxy engine (Xray-core)
**Researched:** 2026-02-24
**Confidence:** HIGH

## Executive Summary

Azad is a full rewrite of an existing bash-based VLESS terminal tool into a professional Go TUI application. The research confirms this is a well-charted path: Go 1.24 with the Charmbracelet v2 stack (bubbletea, lipgloss, bubbles) provides a battle-tested TUI framework, and Xray-core can be imported directly as a Go library to produce a true single-binary distribution with zero runtime dependencies. No custom proxy protocol work is needed -- Xray-core handles VLESS, VMess, Trojan, and Shadowsocks out of the box. The competitive landscape reveals a clear gap: no existing tool combines multi-protocol Xray support with a lazygit-quality terminal interface.

The recommended approach is a layered architecture with strict separation between TUI, application core, proxy engine, and platform abstraction layers. The most consequential technical decision -- importing Xray-core as a Go library rather than managing it as an external binary -- eliminates an entire class of problems (IPC, process management, binary distribution per platform) at the cost of a larger binary (~40-60MB), which is acceptable for a desktop VPN client. The Elm Architecture enforced by bubbletea v2 naturally prevents many concurrency bugs, but only if the team commits to the Cmd/Msg pattern from day one and never performs I/O inside Update() or View().

The primary risks are: (1) orphaned proxy state -- if the app crashes, system proxy settings and Xray processes can be left running, breaking the user's internet; (2) subscription/URI parsing fragility -- the VLESS/VMess/Trojan URI schemes lack formal standards, and real-world providers emit inconsistent formats; (3) terminal restoration failure after panics in Cmd goroutines, which bubbletea does not catch by default. All three are solvable with disciplined engineering: proxy state files with startup cleanup, defensive URI parsing with extensive test fixtures, and a safeCmd wrapper for all tea.Cmd functions.

## Key Findings

### Recommended Stack

The stack is Go-centric with minimal external dependencies. All core libraries are stable, well-maintained, and verified on pkg.go.dev with recent releases (Feb 2026). The Charmbracelet v2 ecosystem just shipped stable releases, making this the ideal time to build -- mature enough to trust, new enough that the project starts on the latest version without migration debt.

**Core technologies:**
- **Go 1.24** (go.mod target, build with 1.26): Single binary, cross-compilation, native concurrency, Charmbracelet ecosystem is Go-only
- **bubbletea v2 + lipgloss v2 + bubbles v2** (charm.land paths): Gold standard Go TUI framework; Elm Architecture enforces clean state management; new Cursed Renderer for better performance
- **Xray-core v1.260206.0** (as Go library): Multi-protocol proxy engine imported as a module -- single binary, no IPC, no process management; call `core.New(config)` / `instance.Start()` / `instance.Close()` directly
- **cobra v1.10.2**: CLI command routing for subcommands (connect, servers, config) and headless/scripted usage
- **koanf v2.3.2**: Lightweight config management (YAML); replaces Viper without the key-lowercasing bug and 50+ transitive dependencies
- **slog (stdlib)**: Zero-dependency structured logging to file (stdout is the TUI)
- **GoReleaser v2.14.0**: Automated cross-compilation, Homebrew/Scoop/AUR publishing, SBOM generation

**Critical version note:** All Charm v2 libraries moved from `github.com/charmbracelet/*` to `charm.land/*`. Use the new module paths exclusively.

### Expected Features

**Must have (table stakes):**
- Multi-protocol support (VLESS, VMess, Trojan, Shadowsocks) via Xray-core
- Subscription URL management (fetch, decode, parse, refresh)
- Server list with concurrent latency display
- One-command quick connect (auto-fastest or last-used)
- System proxy integration (macOS, Linux, Windows)
- Keyboard-driven navigation (vim-style + arrows) with help system
- Connection status display (server, IP, uptime, state)
- Graceful lifecycle (clean start/stop, no orphans)
- Configuration persistence (YAML, XDG paths)
- Import from clipboard/URI (vless://, vmess://, etc.)
- Cross-platform support
- Colored, readable output with adaptive terminal support
- Actionable error messages

**Should have (differentiators):**
- Live connection statistics (speed, data transferred) -- the "btop for your VPN" moment
- Concurrent visual ping sweep with progressive results
- Smart server selection (score based on latency, history, preference)
- Split-pane TUI layout (lazygit-style panels)
- Fuzzy search/filter for large server lists
- Auto-reconnect on connection drop with server failover
- Server grouping by country/provider
- Theme support (dark/light/custom)

**Defer (v2+):**
- Live connection statistics (High complexity, requires Xray stats API research)
- Auto-reconnect (needs robust health detection first)
- Theme support beyond default dark theme
- Server grouping (flat list + search is sufficient for launch)
- Profile/config switching
- Export/QR code sharing
- Toast notifications / event log

### Architecture Approach

A five-layer architecture: CLI entry point (cobra) at the top, TUI layer (bubbletea Elm Architecture) for all user interaction, Application Core (pure Go business logic with no UI awareness), Proxy Engine (xray-core library wrapper), and Platform Layer (OS-specific abstractions behind interfaces). The root bubbletea model owns shared state and routes messages to child view models (dashboard, server list, ping, settings). All I/O flows through tea.Cmd functions that return tea.Msg values -- the TUI never blocks.

**Major components:**
1. **CLI Entry Point** (`cmd/`) -- cobra routing: `azad` launches TUI, `azad connect` for headless, `azad --cleanup` for recovery
2. **TUI Layer** (`internal/tui/`) -- Root model with view switching; child models for dashboard, server list, ping, settings; persistent status bar
3. **Application Core** (`internal/app/`) -- Config, server store, subscription parser, protocol URI parsers, ping service, connection stats; zero TUI imports
4. **Proxy Engine** (`internal/proxy/`) -- Xray-core lifecycle wrapper with Start/Stop/Status/Stats; context-based cancellation; mutex-protected state
5. **Platform Layer** (`internal/platform/`) -- SystemProxy interface with darwin/linux/windows implementations via build tags

### Critical Pitfalls

1. **Blocking the bubbletea event loop** -- All I/O must go through tea.Cmd. A single blocking call in Update() or View() freezes the entire UI. Establish the Cmd/Msg pattern from the first line of TUI code; retrofitting later means rewriting every feature.

2. **Orphaned Xray processes and dirty system proxy** -- If the app crashes or is force-killed, Xray keeps running and system proxy stays set, breaking the user's internet. Implement proxy state file with startup cleanup, `azad --cleanup` command, and signal handlers. Consider defaulting to local SOCKS/HTTP proxy without auto-setting system proxy.

3. **Xray-core library import side-effects** -- Xray-core requires explicit side-effect imports for JSON config loading (`_ "github.com/xtls/xray-core/main/json"`) and for each protocol (`_ "github.com/xtls/xray-core/proxy/vless/inbound"`, etc.). Missing these causes silent failures. Validate in a spike before building features on top.

4. **Subscription/URI parsing fragility** -- No formal spec exists for VLESS/VMess/Trojan URI schemes. Real providers emit inconsistent base64 encoding, missing padding, Unicode fragments. Build defensive parsers with test fixtures from 5+ providers.

5. **Terminal restoration after panics** -- Bubbletea only catches panics in the main goroutine, not in Cmd goroutines. Write a `safeCmd` wrapper that recovers from panics and returns error messages instead of crashing. Ship `azad --reset-terminal` for manual recovery.

## Implications for Roadmap

### Phase 1: Foundation and Core Architecture
**Rationale:** Every other phase depends on config, data models, the xray-core library spike, and the bubbletea root model. This phase validates the most consequential technical decision (xray-core as library) and establishes patterns that prevent the top pitfalls.
**Delivers:** Working Go module with config system, protocol URI parsers, server data store, xray-core library spike (Start/Stop lifecycle), cobra CLI skeleton, bubbletea root model with view switching and status bar, `safeCmd` wrapper, signal handling, `--cleanup` and `--reset-terminal` flags.
**Features addressed:** Configuration persistence, graceful lifecycle, CLI framework
**Pitfalls addressed:** Blocking event loop (pattern established), orphaned processes (signal handlers + PID/state files), xray-core integration choice (spike validates library approach), race conditions (CI with `-race`), terminal restoration (`safeCmd` wrapper)

### Phase 2: Protocol Support and Server Management
**Rationale:** With the foundation in place, build the data pipeline: parse URIs, manage subscriptions, store servers. This is the core data flow that feeds all UI features. URI parsing is a known fragile area requiring thorough testing.
**Delivers:** Multi-protocol URI parsers (VLESS, VMess, Trojan, SS), subscription fetcher/decoder, server CRUD with JSON persistence, import from clipboard, geoip/geosite asset management.
**Features addressed:** Multi-protocol support, subscription management, import from clipboard/URI
**Pitfalls addressed:** URI parsing fragility (defensive parsers + test fixtures), geoip/geosite management (auto-download with integrity verification)

### Phase 3: TUI Server List and Connection
**Rationale:** With servers parsed and stored, build the primary user-facing interaction: see servers, ping them, connect to one. This is where the app becomes functional. System proxy integration ships here but defaults to opt-in.
**Delivers:** Server list view with table display, concurrent TCP ping with progressive results, connect/disconnect flow, system proxy toggle (macOS first), connection status in status bar, IP verification through proxy.
**Features addressed:** Server list with latency, one-command connect, system proxy integration, connection status display, concurrent ping sweep
**Pitfalls addressed:** System proxy dirty state (state file + startup cleanup + opt-in default), sequential ping replaced by concurrent batch

### Phase 4: Intelligence and UX Polish
**Rationale:** With core functionality working, add the features that make Azad delightful -- smart selection, search, keyboard help, split-pane layout. These build on top of everything in phases 1-3.
**Delivers:** Smart server selection (fastest/last-used/scored), fuzzy search/filter, full keyboard navigation with help overlay, split-pane layout (server list + detail panel), quick connect CLI (`azad connect --fastest`).
**Features addressed:** Smart server selection, fuzzy search, split-pane layout, keyboard help, one-key quick actions
**Pitfalls addressed:** UX pitfalls (feedback during operations, cancellable operations, minimum terminal size handling)

### Phase 5: Cross-Platform and Distribution
**Rationale:** With the app working well on the primary platform (macOS), extend to Linux and Windows. GoReleaser handles cross-compilation; the platform abstraction layer from Phase 1 contains OS-specific work.
**Delivers:** Linux system proxy support, Windows system proxy support, GoReleaser pipeline, Homebrew tap, GitHub releases with checksums/SBOM, CI/CD with GitHub Actions, cross-platform testing.
**Features addressed:** Cross-platform support, colored output (adaptive terminal detection)
**Pitfalls addressed:** Cross-platform proxy differences, Windows terminal compatibility, antivirus false positives (binary signing)

### Phase 6: Advanced Features (Post-MVP)
**Rationale:** Features that require deeper research or build on mature infrastructure. Ship MVP without these; add based on user feedback.
**Delivers:** Live connection statistics, auto-reconnect with failover, theme support, server grouping, profile switching, export/QR sharing.
**Features addressed:** All remaining differentiators from FEATURES.md
**Pitfalls addressed:** Stats API integration complexity (needs Xray stats research), reconnection health detection

### Phase Ordering Rationale

- **Dependency chain drives order:** Config -> URI parsers -> subscriptions -> server store -> server list UI -> ping -> connect -> system proxy. Each phase produces artifacts the next phase consumes.
- **Risk-first:** The highest-risk decision (xray-core as library) is validated in Phase 1 via spike. URI parsing fragility is tackled in Phase 2 with thorough testing before any UI depends on it.
- **Architecture patterns before features:** Phase 1 establishes the Cmd/Msg pattern, safeCmd wrapper, signal handling, and state management. Every subsequent phase inherits these patterns rather than inventing ad-hoc solutions.
- **Value delivery cadence:** Phase 1 is infrastructure with minimal user value. Phase 2 adds data. Phase 3 is the first usable product. Phase 4 makes it delightful. Phase 5 expands the audience. Phase 6 is ongoing improvement.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 1:** Xray-core library import -- needs a hands-on spike to validate side-effect imports, config generation (protobuf vs JSON), and cross-compilation behavior. The GoXRay project provides reference but the exact import graph must be verified.
- **Phase 2:** VMess URI parsing -- the base64-encoded JSON blob format has no formal spec. Collect real-world test fixtures from multiple providers before implementing.
- **Phase 3:** System proxy on Linux -- varies by desktop environment (GNOME gsettings, KDE, env vars). sysproxy library coverage may be incomplete.
- **Phase 6:** Xray stats API -- need to research how to read connection statistics from an embedded xray-core instance (gRPC stats service vs in-process counters).

Phases with standard patterns (skip deep research):
- **Phase 4:** TUI patterns (fuzzy search, split pane, help overlay) are well-documented in bubbletea examples and community projects (lazygit, k9s patterns).
- **Phase 5:** GoReleaser cross-compilation and GitHub Actions CI/CD are thoroughly documented with official templates.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All core libraries verified on pkg.go.dev with Feb 2026 releases. Charm v2 is stable. Xray-core library API confirmed. Go version strategy sound. |
| Features | MEDIUM-HIGH | Competitive analysis covers 6+ tools. Feature priorities validated against real user expectations. Gap analysis is solid. Uncertainty: exact complexity of live stats and auto-reconnect. |
| Architecture | HIGH | Layered pattern with Elm Architecture is well-documented. Multiple community sources confirm parent-model-with-child-views approach. Xray-core-as-library validated by GoXRay project. |
| Pitfalls | HIGH | Pitfalls verified against official docs, GitHub issues, and community guides. Process lifecycle and terminal restoration issues confirmed with specific bubbletea issue numbers. |

**Overall confidence:** HIGH

### Gaps to Address

- **sysproxy library maintenance status:** The getlantern/sysproxy library's maintenance is unclear. Fallback plan (direct platform commands) is documented but the library should be evaluated hands-on in Phase 1. If abandoned, implement platform-specific proxy management behind the same interface.
- **lipgloss v2 beta label:** Despite being "production-ready" per maintainers, lipgloss v2 is technically labeled beta. Monitor for breaking changes. The risk is low since the maintainers use it in production, but pin the version.
- **huh v2 pre-release:** The forms library is pre-release. Use sparingly; most UI should be custom bubbletea models. If huh v2 stabilizes during development, expand usage.
- **Xray-core binary size:** Estimated 40-60MB with xray-core embedded. Needs validation. If unacceptable, `-ldflags="-s -w"` and UPX compression can reduce it, but actual numbers depend on the import graph.
- **geoip/geosite asset strategy:** Auto-download vs embed decision impacts first-run experience and binary size. Prototype both in Phase 2 and decide based on measured binary size.
- **Windows terminal compatibility:** bubbletea v2 claims full Windows Terminal support but legacy cmd.exe has limitations. Needs hands-on testing in Phase 5.

## Sources

### Primary (HIGH confidence)
- [pkg.go.dev/charm.land/bubbletea/v2](https://pkg.go.dev/charm.land/bubbletea/v2) -- v2.0.0 stable, Feb 24 2026
- [pkg.go.dev/charm.land/lipgloss/v2](https://pkg.go.dev/charm.land/lipgloss/v2) -- v2.0.0, Feb 24 2026
- [pkg.go.dev/charm.land/bubbles/v2](https://pkg.go.dev/charm.land/bubbles/v2) -- v2.0.0, Feb 24 2026
- [pkg.go.dev/github.com/xtls/xray-core/core](https://pkg.go.dev/github.com/xtls/xray-core/core) -- v1.260206.0, Feb 6 2026
- [github.com/XTLS/Xray-core/releases](https://github.com/XTLS/Xray-core/releases) -- Active development verified
- [github.com/spf13/cobra/releases](https://github.com/spf13/cobra/releases) -- v1.10.2, Dec 2024
- [go.dev/doc/devel/release](https://go.dev/doc/devel/release) -- Go 1.26, Feb 10 2026
- [Xray-core official docs](https://xtls.github.io/en/config/) -- Routing, DNS, protocol configuration
- [Bubbletea pitfalls guide](https://leg100.github.io/en/posts/building-bubbletea-programs/) -- Cmd/Msg patterns, common mistakes
- [Bubbletea GitHub issue #1459](https://github.com/charmbracelet/bubbletea/issues/1459) -- Terminal restoration after panics

### Secondary (MEDIUM confidence)
- [GoXRay project](https://github.com/goxray) -- Validates xray-core-as-library approach
- [xray-knife](https://github.com/lilendian0x00/xray-knife) -- URI parsing reference implementation
- [getlantern/sysproxy](https://github.com/getlantern/sysproxy) -- Cross-platform system proxy (maintenance uncertain)
- [knadh/koanf](https://github.com/knadh/koanf) -- v2.3.2, Jan 2026
- [DeepWiki analyses](https://deepwiki.com/) -- v2rayN architecture, Xray-core internals, bubbletea components
- [Competitive tools](https://github.com/) -- proton-tui, vortix, Clash Verge Rev, IRBox

### Tertiary (needs validation)
- Xray-core binary size estimate (40-60MB) -- needs build spike to confirm
- sysproxy library stability -- needs hands-on evaluation
- huh v2 readiness -- pre-release, monitor stability

---
*Research completed: 2026-02-24*
*Ready for roadmap: yes*

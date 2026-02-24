# Technology Stack

**Project:** Azad -- Terminal VPN Client
**Researched:** 2026-02-24

## Recommended Stack

### Language & Runtime

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| Go | 1.24 (go.mod directive) | Primary language | Single binary distribution, excellent cross-compilation, native concurrency, the Charmbracelet TUI ecosystem is Go-only. Go 1.24 is the sweet spot: still supported (until May 2026), stable, and compatible with all dependencies. Go 1.26 is out but too bleeding-edge for go.mod minimum -- our dependencies may not require it yet. | HIGH |

**Note on Go version strategy:** Set `go 1.24` in go.mod for broad compatibility (Go 1.24 supported until May 2026, Go 1.25 until ~Nov 2026). Build and test with Go 1.26 toolchain locally. GoReleaser will use the latest toolchain regardless.

### TUI Framework (Charmbracelet Stack)

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| charm.land/bubbletea/v2 | v2.0.0 | TUI framework (Elm Architecture) | The gold standard for Go terminal UIs. v2 shipped stable Feb 24, 2025. New "Cursed Renderer" based on ncurses for better performance. Declarative View struct replaces imperative commands. Native clipboard (OSC52), progressive keyboard enhancements (key release, shift+enter), synchronized output (mode 2026). Used by 18,000+ projects including Microsoft Azure, CockroachDB, NVIDIA. Module path changed from github.com/charmbracelet/bubbletea to charm.land/bubbletea/v2. | HIGH |
| charm.land/lipgloss/v2 | v2.0.0 | Terminal styling & layout | Deterministic styles, precise I/O control, works in lockstep with bubbletea v2. Tables overhauled with smart column width algorithms. New module path charm.land/lipgloss/v2. Technically still labeled beta but the maintainers say "ready for production -- we depend on it heavily and all new work is against v2." Published Feb 24, 2026 on pkg.go.dev. | MEDIUM (beta label, but production-used) |
| charm.land/bubbles/v2 | v2.0.0 | Pre-built TUI components | Spinner, TextInput, TextArea, Table, Progress, Viewport, List (with fuzzy filtering), FilePicker, Timer, Stopwatch, Help, Key bindings. All components needed for a VPN client UI. Published Feb 24, 2026. | HIGH |
| charm.land/huh/v2 | v2.0.0 (pre-release) | Terminal forms & prompts | For subscription URL input, server configuration dialogs, confirmation prompts. Built-in themes (Charm, Dracula, Catppuccin). Accessibility mode for screen readers. Currently pre-release (Jan 2026), but functional. Use sparingly -- most UI should be custom bubbletea models. | MEDIUM (pre-release) |

**IMPORTANT -- Module Path Migration:** All Charm v2 libraries moved from `github.com/charmbracelet/*` to `charm.land/*`. Use the new paths exclusively. The old paths point to v1.x only.

### Proxy Engine

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| github.com/xtls/xray-core | v1.260206.0 | Multi-protocol proxy engine | Proven, actively maintained (3 releases in Feb 2026 alone). Supports VLESS, VMess, Trojan, Shadowsocks, REALITY, XHTTP, and more. Clean Go library API: `core.New(config)`, `instance.Start()`, `instance.Close()`. Can be imported as a Go library -- no need to shell out to a separate binary. This is the key architectural win: embed Xray-core as a Go dependency, not as an external binary, achieving true single-binary distribution. 585 known importers on pkg.go.dev. Licensed MPL-2.0. | HIGH |

**Critical decision -- Library vs Binary:**

Use Xray-core as a **Go library dependency**, not an embedded/sidecar binary. This means:
- `go get github.com/xtls/xray-core@latest` in go.mod
- Call `core.New(config)` / `instance.Start()` directly in Go code
- Build config programmatically using Xray's protobuf types, or load JSON config via `core.LoadConfig("json", reader)`
- True single binary -- no extracting embedded binaries at runtime, no temp directories, no permission issues
- Xray-core versions tracked via standard Go module tooling

The GoXRay project (github.com/goxray) validates this approach -- it's a fully functional Xray VPN client written in Go that embeds xray-core as a library.

### CLI Framework

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| github.com/spf13/cobra | v1.10.2 | CLI command routing | Industry standard (Kubernetes, Docker, Hugo, GitHub CLI, 173K+ projects). Provides subcommands (`azad connect`, `azad servers`, `azad config`), auto-generated help, shell completions (bash, zsh, fish, powershell). The TUI is the primary interface but CLI flags are needed for headless/scripted usage (e.g., `azad connect --fastest --background`). | HIGH |

### Configuration

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| github.com/knadh/koanf/v2 | v2.3.2 | Configuration management | Lightweight, modular, no forced dependencies. Supports YAML, TOML, JSON, env vars, flags. Unlike Viper: does not lowercase keys (spec-compliant), 313% smaller binary impact, decoupled parsers (only pull what you use). For a VPN client, config is simple -- koanf is right-sized. Viper is overkill with its remote config, etcd support, etc. | HIGH |

**Config format: YAML.** Human-readable, well-supported, familiar to the target audience (technical users comfortable with terminals). Store at `~/.config/azad/config.yaml` (XDG Base Directory spec) with fallback to `~/.azad/config.yaml`.

### Logging

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| log/slog (stdlib) | Go 1.24+ | Structured logging | Zero dependencies. Standard library since Go 1.21. Structured key-value pairs. Good enough for a client application. Avoid adding logging libraries to a client tool -- slog handles debug/info/warn/error levels, JSON output when needed, and custom handlers. The Charm log library (v0.4.2) is pretty but adds a dependency for marginal value in a VPN client where logs go to a file, not the terminal. | HIGH |

**Logging strategy:** Log to file (`~/.config/azad/azad.log`), not stdout (stdout is the TUI). Use `tea.LogToFile()` in bubbletea for debug logging during development.

### System Proxy Management

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| github.com/getlantern/sysproxy | latest | Cross-platform system proxy toggle | Simple API: `sysproxy.On(addr)` / `sysproxy.Off()`. Supports macOS, Windows, Linux (GNOME via GSettings). From the Lantern project (battle-tested VPN client). Uses platform-specific helper tools extracted at runtime. | MEDIUM (maintenance status unclear) |

**Fallback plan:** If sysproxy proves unmaintained or problematic, implement platform-specific proxy setting directly:
- macOS: `networksetup` commands (already proven in current bash version)
- Linux: Set env vars + optional GNOME/KDE proxy settings
- Windows: Registry manipulation or `netsh` commands

This is a contained concern -- abstract behind an interface early so the implementation can be swapped.

### Networking & Testing

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| net (stdlib) | Go 1.24+ | TCP ping, connectivity checks | Go stdlib net package for TCP dial with timeout -- replaces current netcat dependency. `net.DialTimeout("tcp", addr, timeout)` is all we need for latency measurement. | HIGH |
| net/http (stdlib) | Go 1.24+ | Subscription fetching, IP checks | HTTP client for fetching subscription URLs, IP verification endpoints. Go's stdlib HTTP client is excellent. No need for resty/req/etc. | HIGH |
| encoding/base64 (stdlib) | Go 1.24+ | Subscription decoding | Base64 decode subscription content. Stdlib. | HIGH |
| net/url (stdlib) | Go 1.24+ | Protocol URL parsing | Parse vless://, vmess://, trojan://, ss:// URLs. Custom parsers on top of stdlib url.Parse. | HIGH |

### Build & Distribution

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| GoReleaser | v2.14.0 | Build, package, release | Automated cross-compilation (darwin/amd64, darwin/arm64, linux/amd64, linux/arm64, windows/amd64, windows/arm64). Auto-publishes to Homebrew, Scoop, Winget, AUR. SBOM generation. GitHub Actions integration. The standard for Go binary releases. | HIGH |
| GitHub Actions | N/A | CI/CD | Build, test, lint, release pipeline. GoReleaser has first-class GitHub Actions support. | HIGH |
| golangci-lint | latest | Linting | Standard Go linter aggregator. Run in CI and pre-commit. | HIGH |

### Data Files

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| geoip.dat / geosite.dat | embedded or auto-download | Xray routing rules | Required by Xray-core for geographic/domain-based routing. Options: (1) embed via go:embed (~30MB binary size increase), (2) auto-download on first run, (3) let users provide their own. Recommend option 2 (auto-download) with option 3 as override -- keeps binary small. Cache in `~/.config/azad/data/`. | MEDIUM (design decision, not tech risk) |

### Testing

| Technology | Version | Purpose | Why | Confidence |
|------------|---------|---------|-----|------------|
| testing (stdlib) | Go 1.24+ | Unit & integration tests | Go stdlib testing. No need for testify/gomega in a client app. Table-driven tests are idiomatic Go. | HIGH |
| github.com/charmbracelet/x/exp/teatest | v2 | TUI testing | Official bubbletea test harness. Send messages, assert view output, test TUI interactions programmatically. | MEDIUM (verify v2 compatibility) |

## Complete Dependency List

```bash
# Core dependencies (go get)
go get charm.land/bubbletea/v2@latest
go get charm.land/lipgloss/v2@latest
go get charm.land/bubbles/v2@latest
go get charm.land/huh/v2@latest
go get github.com/xtls/xray-core@latest
go get github.com/spf13/cobra@latest
go get github.com/knadh/koanf/v2@latest
go get github.com/getlantern/sysproxy@latest

# Koanf providers (install only what you need)
go get github.com/knadh/koanf/parsers/yaml@latest
go get github.com/knadh/koanf/providers/file@latest
go get github.com/knadh/koanf/providers/env@latest
go get github.com/knadh/koanf/providers/structs@latest

# Dev tools (go install)
go install github.com/goreleaser/goreleaser/v2@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

## Alternatives Considered

| Category | Recommended | Alternative | Why Not |
|----------|-------------|-------------|---------|
| TUI Framework | bubbletea v2 | tview | tview uses a widget-based model (like traditional GUI) that fights against terminal constraints. Bubbletea's Elm Architecture produces cleaner, more testable code. tview also has no v2 evolution and less active development. |
| TUI Framework | bubbletea v2 | tcell (raw) | Too low-level. Bubbletea handles terminal state, rendering, input, and resize. Building from tcell means reimplementing what bubbletea already provides. |
| Config | koanf | Viper | Viper lowercases all keys (breaks YAML/TOML spec), pulls 50+ transitive dependencies, and is overkill for a client config file. koanf is modular -- only import what you use. |
| Config | koanf | raw YAML (gopkg.in/yaml.v3) | No env var overlay, no default merging, no multi-source config. koanf adds this with minimal overhead. |
| Config format | YAML | TOML | YAML is more widely known among the target audience (VPN/proxy users often interact with YAML in other tools). TOML is fine but less familiar. |
| Logging | slog (stdlib) | charmbracelet/log | charmbracelet/log is beautiful for terminal output, but our logs go to a file (stdout is the TUI). Adding a dependency for file logging provides no visual benefit. slog is zero-dependency and standard. |
| Logging | slog (stdlib) | zerolog | zerolog is faster than slog, but logging performance is irrelevant in a VPN client. We log connection events, not millions of requests/sec. Zero dependencies wins. |
| Proxy engine | xray-core (library) | xray-core (binary sidecar) | Embedding as a library gives true single-binary distribution. Sidecar means extracting binaries to temp dirs, managing permissions, handling cleanup, and platform-specific binary selection at runtime. The library approach is cleaner in every way. |
| Proxy engine | xray-core | sing-box | sing-box is an alternative proxy platform, but xray-core has stronger VLESS/REALITY support (it created these protocols), more active development (3 releases in 2 weeks), and the existing user base uses xray-compatible subscription URLs. |
| CLI | cobra | urfave/cli | cobra is the industry standard with 173K+ dependents. Better documentation, more contributors, richer ecosystem (shell completions, man pages). |
| System proxy | sysproxy | Manual per-platform | sysproxy handles the cross-platform abstraction. Manual implementation is the fallback, not the first choice. |
| Release | GoReleaser | Manual goreleaser scripts | GoReleaser automates Homebrew taps, Scoop manifests, checksums, SBOMs, changelogs. Manual scripts would need to replicate all of this. |

## Architecture-Impacting Stack Decisions

### 1. Xray-core as Library (not binary)

This is the most consequential stack decision. By importing xray-core as a Go module:

- **Config generation** is programmatic (build protobuf structs in Go), not file-based JSON templating
- **Lifecycle management** is function calls (`Start()`, `Close()`), not process management (`exec.Command`, PID files, signal handling)
- **Error handling** is Go errors, not stderr parsing
- **Binary size** will be larger (~40-60MB estimated due to xray-core dependencies, crypto libs, protocol implementations) but this is acceptable for a VPN client
- **Build time** will be longer due to xray-core compilation, but this only affects CI/release builds

### 2. Bubbletea v2 Elm Architecture

The entire UI follows Model-Update-View:
- **Model** holds all state (connection status, server list, ping results, config)
- **Update** handles all messages (key presses, connection events, tick timers)
- **View** renders the current state (pure function, no side effects)

This is not optional -- it is how bubbletea works. All state mutations go through Update. All I/O goes through Commands (Cmd). The View is a pure function of state.

### 3. Single Binary via Go Modules

The stack produces one binary containing:
- The TUI application (bubbletea)
- The proxy engine (xray-core)
- All protocol support (VLESS, VMess, Trojan, Shadowsocks)
- CLI framework (cobra)
- Config management (koanf)

No runtime dependencies. No Python. No netcat. No curl. No external binaries.

## Platform-Specific Considerations

| Platform | Consideration | Approach |
|----------|--------------|----------|
| macOS | System proxy via networksetup | sysproxy library (wraps networksetup) |
| macOS | Code signing / notarization | GoReleaser supports Apple notarization |
| macOS | Gatekeeper warnings | Sign with Apple Developer ID or document `xattr -d` workaround |
| Linux | System proxy varies by DE | sysproxy for GNOME; env vars for others; document limitations |
| Linux | TUN device for system-wide VPN | Future feature (requires root/capabilities); start with SOCKS5/HTTP proxy |
| Windows | System proxy via registry | sysproxy library (wraps registry calls) |
| Windows | Terminal compatibility | Windows Terminal and modern terminals support all bubbletea v2 features. Legacy cmd.exe has limited support. |
| All | Terminal color detection | bubbletea v2 handles this automatically via ColorProfileMsg |

## Version Pinning Strategy

Pin major+minor in go.mod, allow patch updates:

```go
// go.mod
module github.com/user/azad

go 1.24

require (
    charm.land/bubbletea/v2 v2.0.0
    charm.land/lipgloss/v2  v2.0.0
    charm.land/bubbles/v2   v2.0.0
    github.com/xtls/xray-core v1.260206.0
    github.com/spf13/cobra  v1.10.2
    github.com/knadh/koanf/v2 v2.3.2
)
```

Run `go get -u ./...` periodically to pick up patch releases. Test before committing updated go.sum.

## Sources

### Verified (HIGH confidence)
- [pkg.go.dev/charm.land/bubbletea/v2](https://pkg.go.dev/charm.land/bubbletea/v2) -- v2.0.0, published Feb 24, 2026
- [pkg.go.dev/charm.land/lipgloss/v2](https://pkg.go.dev/charm.land/lipgloss/v2) -- v2.0.0, published Feb 24, 2026
- [pkg.go.dev/charm.land/bubbles/v2](https://pkg.go.dev/charm.land/bubbles/v2) -- v2.0.0, published Feb 24, 2026
- [pkg.go.dev/github.com/xtls/xray-core/core](https://pkg.go.dev/github.com/xtls/xray-core/core) -- v1.260206.0, published Feb 6, 2026
- [github.com/charmbracelet/bubbletea/releases](https://github.com/charmbracelet/bubbletea/releases) -- release history verified
- [github.com/XTLS/Xray-core/releases](https://github.com/XTLS/Xray-core/releases) -- v26.2.6, Feb 6, 2026
- [github.com/spf13/cobra/releases](https://github.com/spf13/cobra/releases) -- v1.10.2, Dec 4, 2024
- [github.com/goreleaser/goreleaser/releases](https://github.com/goreleaser/goreleaser/releases) -- v2.14.0, Feb 21, 2026
- [go.dev/doc/devel/release](https://go.dev/doc/devel/release) -- Go 1.26 released Feb 10, 2026

### Verified via multiple sources (MEDIUM confidence)
- [github.com/knadh/koanf](https://github.com/knadh/koanf) -- v2.3.2, Jan 25, 2026 (pkg.go.dev)
- [github.com/charmbracelet/lipgloss/discussions/506](https://github.com/charmbracelet/lipgloss/discussions/506) -- Lipgloss v2 discussion, maintainer confirms production-ready
- [github.com/charmbracelet/bubbletea/discussions/1374](https://github.com/charmbracelet/bubbletea/discussions/1374) -- Bubbletea v2 migration guide
- [github.com/getlantern/sysproxy](https://github.com/getlantern/sysproxy) -- cross-platform proxy management
- [github.com/goxray](https://github.com/goxray) -- GoXRay validates xray-core-as-library approach

### Community sources (verified patterns)
- [threedots.tech/post/list-of-recommended-libraries](https://threedots.tech/post/list-of-recommended-libraries/) -- koanf recommendation
- [betterstack.com/community/guides/logging/logging-in-go](https://betterstack.com/community/guides/logging/logging-in-go/) -- slog best practices
- [hatchet.run/blog/tuis-are-easy-now](https://hatchet.run/blog/tuis-are-easy-now) -- bubbletea ecosystem overview

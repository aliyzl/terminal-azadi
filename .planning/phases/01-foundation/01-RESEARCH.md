# Phase 1: Foundation - Research

**Researched:** 2026-02-25
**Domain:** Go CLI application scaffolding, configuration management, signal handling, xray-core embedding
**Confidence:** HIGH

## Summary

Phase 1 establishes the Go application skeleton for Azad: a Go module embedding xray-core as a library dependency, a cobra-based CLI with subcommands, a koanf-driven YAML configuration system using XDG-compliant paths, and graceful signal handling with crash recovery commands. This phase has no external blockers -- all technologies are mature, well-documented, and the patterns are well-established in the Go ecosystem.

The primary technical risk is xray-core binary size (~40-60MB) due to its extensive protocol support. This is a known tradeoff documented in STATE.md and should be validated with a build spike early in the phase. All other components (cobra, koanf, signal handling) are straightforward and battle-tested.

**Primary recommendation:** Use `cmd/azad/main.go` entry point with `internal/` packages for config, cli, and lifecycle. Initialize Go module with xray-core v1.260206.0 as dependency. Wire cobra root command with `connect`, `servers`, and `config` subcommands plus `--cleanup` and `--reset-terminal` root flags. Use koanf v2 with YAML parser for config at `os.UserConfigDir()/azad/config.yaml`. Use `signal.NotifyContext` for graceful shutdown.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| FNDN-01 | App initializes Go module with xray-core as library dependency (not external binary) | Standard `go mod init` + `go get github.com/xtls/xray-core@v1.260206.0`. Import `core` package for `core.New()`, `instance.Start()`, `instance.Close()`. Requires Go 1.26. Build command: `CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" ./cmd/azad` |
| FNDN-02 | App reads/writes YAML config from XDG-compliant path (~/.config/azad/config.yaml) | Use `os.UserConfigDir()` (returns `$XDG_CONFIG_HOME` or `~/.config` on Unix, `%APPDATA%` on Windows). koanf v2 with `file.Provider` + `yaml.Parser()` for reading. `k.Marshal(yamlParser)` + `os.WriteFile()` for writing. Create dir with `os.MkdirAll(path, 0700)`. |
| FNDN-03 | App provides cobra CLI with subcommands (connect, servers, config, --cleanup, --reset-terminal) | cobra v1.9.x provides subcommand routing, auto-generated help, persistent flags. `--cleanup` and `--reset-terminal` as root-level flags with `PersistentPreRun` hooks. |
| FNDN-04 | App handles SIGTERM/SIGINT gracefully, cleaning up proxy and terminal state | `signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)` for context-based cancellation. Defer cleanup functions for proxy state and terminal restoration. |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/xtls/xray-core | v1.260206.0 | Proxy engine (VLESS, VMess, Trojan, SS) | The proxy engine; project decision to embed as library, not shell out to binary |
| github.com/spf13/cobra | v1.9.1 | CLI framework with subcommands | De facto Go CLI standard; auto help, subcommand routing, persistent flags |
| github.com/knadh/koanf/v2 | v2.3.0 | Configuration management | Project decision over Viper; no key-lowercasing bug, lighter deps, clean API |
| gopkg.in/yaml.v3 | v3.0.1 | YAML parsing (via koanf parser) | Standard Go YAML library; koanf's yaml parser wraps this |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| github.com/knadh/koanf/parsers/yaml | v2 | YAML parser for koanf | Config file read/write |
| github.com/knadh/koanf/providers/file | v2 | File provider for koanf | Loading config from disk |
| github.com/knadh/koanf/providers/confmap | v2 | Map provider for koanf | Setting default config values |
| golang.org/x/term | latest | Terminal state save/restore | `--reset-terminal` command; save/restore raw mode state |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| koanf | Viper | Viper lowercases all keys (known bug), heavier dependency tree; koanf is project decision |
| cobra | urfave/cli | cobra has richer subcommand model and wider ecosystem adoption; project decision |
| os.UserConfigDir | adrg/xdg | stdlib is sufficient for config dir; adrg/xdg adds full XDG spec but unnecessary for this use case |
| YAML config | TOML config | YAML is more familiar to VPN/proxy users; matches existing ecosystem (Xray uses JSON/YAML) |

**Installation:**
```bash
go mod init github.com/user/azad
go get github.com/xtls/xray-core@v1.260206.0
go get github.com/spf13/cobra@v1.9.1
go get github.com/knadh/koanf/v2@latest
go get github.com/knadh/koanf/parsers/yaml@latest
go get github.com/knadh/koanf/providers/file@latest
go get github.com/knadh/koanf/providers/confmap@latest
go get golang.org/x/term@latest
```

## Architecture Patterns

### Recommended Project Structure
```
azad/
├── cmd/
│   └── azad/
│       └── main.go           # Entry point: wire cobra root, run
├── internal/
│   ├── cli/
│   │   ├── root.go           # Root command, persistent flags, global setup
│   │   ├── connect.go        # `azad connect` subcommand (stub in Phase 1)
│   │   ├── servers.go        # `azad servers` subcommand (stub in Phase 1)
│   │   └── config.go         # `azad config` subcommand
│   ├── config/
│   │   ├── config.go         # Config struct, defaults, load/save logic
│   │   └── paths.go          # XDG path resolution, dir creation
│   └── lifecycle/
│       ├── signals.go         # Signal handling, context cancellation
│       └── cleanup.go         # --cleanup and --reset-terminal logic
├── go.mod
├── go.sum
└── .gitignore
```

### Pattern 1: Cobra Command Organization
**What:** Each subcommand in its own file under `internal/cli/`, registered in `root.go`
**When to use:** Always -- standard cobra project layout
**Example:**
```go
// Source: Context7 /spf13/cobra - user guide
// internal/cli/root.go
package cli

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"
)

var (
    cleanup       bool
    resetTerminal bool
)

func NewRootCmd(version string) *cobra.Command {
    rootCmd := &cobra.Command{
        Use:   "azad",
        Short: "Beautiful terminal VPN client",
        Long:  "Azad - One command to connect to the fastest VPN server through a stunning terminal interface",
        PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
            if cleanup {
                return runCleanup()
            }
            if resetTerminal {
                return runResetTerminal()
            }
            return nil
        },
    }

    rootCmd.PersistentFlags().BoolVar(&cleanup, "cleanup", false, "Remove dirty proxy state from a previous crash")
    rootCmd.PersistentFlags().BoolVar(&resetTerminal, "reset-terminal", false, "Restore terminal to usable state")

    rootCmd.AddCommand(newConnectCmd())
    rootCmd.AddCommand(newServersCmd())
    rootCmd.AddCommand(newConfigCmd())

    return rootCmd
}
```

### Pattern 2: koanf Config Load/Save with Struct Binding
**What:** Define config struct with `koanf` tags, load from YAML file, marshal back for writes
**When to use:** All config read/write operations
**Example:**
```go
// Source: Context7 /knadh/koanf - README examples
// internal/config/config.go
package config

import (
    "os"
    "path/filepath"

    "github.com/knadh/koanf/v2"
    "github.com/knadh/koanf/parsers/yaml"
    "github.com/knadh/koanf/providers/confmap"
    "github.com/knadh/koanf/providers/file"
)

type Config struct {
    Proxy    ProxyConfig    `koanf:"proxy"`
    Server   ServerConfig   `koanf:"server"`
}

type ProxyConfig struct {
    SOCKSPort int `koanf:"socks_port"`
    HTTPPort  int `koanf:"http_port"`
}

type ServerConfig struct {
    LastUsed string `koanf:"last_used"`
}

var k = koanf.New(".")
var yamlParser = yaml.Parser()

func Load(path string) (*Config, error) {
    // Load defaults
    k.Load(confmap.Provider(map[string]interface{}{
        "proxy.socks_port": 1080,
        "proxy.http_port":  8080,
    }, "."), nil)

    // Load from file (ok if doesn't exist yet)
    if _, err := os.Stat(path); err == nil {
        if err := k.Load(file.Provider(path), yamlParser); err != nil {
            return nil, fmt.Errorf("loading config: %w", err)
        }
    }

    var cfg Config
    if err := k.Unmarshal("", &cfg); err != nil {
        return nil, fmt.Errorf("parsing config: %w", err)
    }
    return &cfg, nil
}

func Save(cfg *Config, path string) error {
    // Ensure directory exists
    if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
        return fmt.Errorf("creating config dir: %w", err)
    }

    // Load struct into koanf and marshal to YAML
    k := koanf.New(".")
    k.Load(structs.Provider(cfg, "koanf"), nil)
    b, err := k.Marshal(yamlParser)
    if err != nil {
        return fmt.Errorf("marshaling config: %w", err)
    }

    return os.WriteFile(path, b, 0600)
}
```

### Pattern 3: Signal Handling with Context
**What:** Use `signal.NotifyContext` for clean shutdown via context cancellation
**When to use:** Main application lifecycle
**Example:**
```go
// Source: Go standard library + community best practices
// internal/lifecycle/signals.go
package lifecycle

import (
    "context"
    "os"
    "os/signal"
    "syscall"
)

// WithShutdown returns a context that cancels on SIGINT or SIGTERM.
func WithShutdown(parent context.Context) (context.Context, context.CancelFunc) {
    ctx, cancel := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
    return ctx, cancel
}
```

### Pattern 4: XDG Config Path Resolution
**What:** Resolve config file path using stdlib `os.UserConfigDir`
**When to use:** On startup, before loading config
**Example:**
```go
// Source: Go stdlib os.UserConfigDir documentation
// internal/config/paths.go
package config

import (
    "os"
    "path/filepath"
)

const appName = "azad"
const configFileName = "config.yaml"

// Dir returns the config directory path (~/.config/azad or XDG equivalent).
func Dir() (string, error) {
    base, err := os.UserConfigDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(base, appName), nil
}

// FilePath returns the full config file path.
func FilePath() (string, error) {
    dir, err := Dir()
    if err != nil {
        return "", err
    }
    return filepath.Join(dir, configFileName), nil
}

// EnsureDir creates the config directory if it doesn't exist.
func EnsureDir() error {
    dir, err := Dir()
    if err != nil {
        return err
    }
    return os.MkdirAll(dir, 0700)
}
```

### Anti-Patterns to Avoid
- **Hardcoded paths:** Never use `~/.config/azad` directly; always go through `os.UserConfigDir()` which respects `$XDG_CONFIG_HOME`
- **Global mutable state:** Don't use package-level koanf instance for writes; create new instances for save operations to avoid race conditions
- **Blocking signal handler:** Don't use `signal.Notify` with raw channel reads when context-based cancellation suffices; `signal.NotifyContext` is cleaner
- **Ignoring config write errors:** Always handle and propagate errors from `os.WriteFile` and `os.MkdirAll`; silent failures corrupt user config
- **init() for command registration:** Prefer explicit `AddCommand()` calls in a builder function over `init()` functions scattered across files; easier to test and reason about

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| CLI routing and help | Custom arg parser | cobra | Handles help generation, flag parsing, subcommand routing, shell completion, error messages |
| Config file read/write | Custom YAML marshal/unmarshal | koanf + yaml parser | Handles defaults, merging, struct binding, multiple formats |
| XDG path resolution | String concatenation with HOME | `os.UserConfigDir()` | Handles macOS (`~/Library/Application Support`), Linux (`~/.config`), Windows (`%APPDATA%`) correctly |
| Signal handling | Raw `os.Signal` channels | `signal.NotifyContext` | Integrates with context cancellation pattern; cleaner than manual channel management |
| Terminal state restore | Manual termios manipulation | `golang.org/x/term` + `stty sane` | `term.GetState`/`term.Restore` for programmatic restore; `stty sane` for crash recovery |
| Proxy engine | Custom VLESS/VMess/Trojan impl | xray-core `core.New()` | Thousands of edge cases in proxy protocols; xray-core is battle-tested |

**Key insight:** Phase 1 is scaffolding -- every component has a well-established Go library. The value is in correct wiring, not novel implementation.

## Common Pitfalls

### Pitfall 1: xray-core Import Side Effects
**What goes wrong:** Importing `github.com/xtls/xray-core/core` alone is insufficient. Xray-core uses Go's `init()` registration pattern -- protocol handlers, transports, and config loaders must be imported for their side effects.
**Why it happens:** Xray-core's modular architecture registers features via `init()` functions. Without the right imports, `core.LoadConfig` and `core.New` fail with mysterious errors about missing handlers.
**How to avoid:** Include blank imports for required features. At minimum for Phase 1 (where we only need the config loader, not actual proxy operation):
```go
import (
    "github.com/xtls/xray-core/core"
    // Import config format loaders
    _ "github.com/xtls/xray-core/main/distro/all"
)
```
**Warning signs:** `core.LoadConfig` returns "unknown config format" errors; `core.New` returns "missing handler" errors.

### Pitfall 2: koanf Key Delimiter Mismatch
**What goes wrong:** Nested config keys don't resolve correctly, returning zero values.
**Why it happens:** koanf uses a configurable delimiter (default `.`). If your YAML has nested keys like `proxy.socks_port` but you initialized koanf with a different delimiter, lookups fail silently.
**How to avoid:** Always initialize with `koanf.New(".")` and use consistent dot-notation in `confmap.Provider` defaults. Verify with `k.Print()` during development.
**Warning signs:** `k.Int("proxy.socks_port")` returns `0` even though the YAML file has the value.

### Pitfall 3: Config File Doesn't Exist on First Run
**What goes wrong:** App crashes on first run because config file doesn't exist yet.
**Why it happens:** `file.Provider` returns an error if the file doesn't exist. New installations have no config file.
**How to avoid:** Check `os.Stat()` before loading. If file doesn't exist, load only defaults. Create file on first write (e.g., `azad config set` or on graceful first-run).
**Warning signs:** First run after install panics or shows file-not-found error.

### Pitfall 4: Cleanup Command Runs Before Config Loads
**What goes wrong:** `--cleanup` flag triggers in `PersistentPreRun` but needs config data (e.g., which ports were configured) that hasn't loaded yet.
**Why it happens:** cobra's `PersistentPreRun` fires before `Run`. If cleanup logic depends on config, the ordering matters.
**How to avoid:** Load config in `PersistentPreRunE` before checking cleanup flags. Or: make cleanup self-contained by reading state from a lockfile/pidfile rather than config.
**Warning signs:** Cleanup reports "no config found" or uses wrong port numbers.

### Pitfall 5: Binary Size Surprise from xray-core
**What goes wrong:** Final binary is 40-60MB, surprising for a "simple" CLI tool.
**Why it happens:** xray-core brings in all protocol implementations (VLESS, VMess, Trojan, Shadowsocks, WireGuard, Hysteria), gRPC, quic-go, crypto libraries, etc.
**How to avoid:** Accept this as a known tradeoff (documented in STATE.md). Use `-ldflags="-s -w"` and `-trimpath` to strip ~20-30% of size. Consider UPX compression for distribution if needed. Validate with a build spike early in the phase.
**Warning signs:** `go build` produces unexpectedly large binary; CI/CD storage or download times increase.

### Pitfall 6: macOS System Proxy Left Set After Crash
**What goes wrong:** If the app crashes while system proxy is enabled, the user's system routes all traffic through a dead proxy -- effectively killing their internet.
**Why it happens:** `networksetup` commands set global system state. A crash bypasses cleanup.
**How to avoid:** Write a state file (e.g., `~/.config/azad/.proxy-state`) when proxy is set. `--cleanup` reads this file and runs the reverse `networksetup` commands. Check for stale state on startup.
**Warning signs:** User reports "no internet after crash"; `networksetup -getwebproxy Wi-Fi` shows proxy still set to localhost.

## Code Examples

Verified patterns from official sources:

### Creating and Starting xray-core Instance
```go
// Source: Context7 /xtls/xray-core - programmatic Go usage
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/xtls/xray-core/core"
    _ "github.com/xtls/xray-core/main/distro/all"  // register all features
)

func main() {
    // Load configuration from file
    config, err := core.LoadConfig("json", []string{"config.json"})
    if err != nil {
        log.Fatal("Failed to load config:", err)
    }

    // Create new Xray instance (not started yet)
    server, err := core.New(config)
    if err != nil {
        log.Fatal("Failed to create instance:", err)
    }
    defer server.Close()

    // Start the server
    if err := server.Start(); err != nil {
        log.Fatal("Failed to start:", err)
    }

    // Wait for interrupt signal
    osSignals := make(chan os.Signal, 1)
    signal.Notify(osSignals, os.Interrupt, syscall.SIGTERM)
    <-osSignals

    log.Println("Shutting down...")
}
```

### koanf Load Defaults + File + Unmarshal
```go
// Source: Context7 /knadh/koanf - README
package config

import (
    "github.com/knadh/koanf/v2"
    "github.com/knadh/koanf/parsers/yaml"
    "github.com/knadh/koanf/providers/confmap"
    "github.com/knadh/koanf/providers/file"
)

var k = koanf.New(".")

func LoadConfig(path string) (*AppConfig, error) {
    // 1. Load defaults
    k.Load(confmap.Provider(map[string]interface{}{
        "proxy.socks_port": 1080,
        "proxy.http_port":  8080,
    }, "."), nil)

    // 2. Load YAML file (merges on top of defaults)
    k.Load(file.Provider(path), yaml.Parser())

    // 3. Unmarshal to struct
    var cfg AppConfig
    if err := k.Unmarshal("", &cfg); err != nil {
        return nil, err
    }
    return &cfg, nil
}
```

### koanf Marshal Back to YAML for Save
```go
// Source: Context7 /knadh/koanf - README marshal example
// Marshal the koanf instance back to YAML bytes
b, err := k.Marshal(yaml.Parser())
if err != nil {
    return err
}
// Write to disk
return os.WriteFile(configPath, b, 0600)
```

### Cobra Root + Subcommands
```go
// Source: Context7 /spf13/cobra - user guide
func main() {
    rootCmd := &cobra.Command{
        Use:   "azad",
        Short: "Beautiful terminal VPN client",
    }

    connectCmd := &cobra.Command{
        Use:   "connect [server]",
        Short: "Connect to a VPN server",
        Run: func(cmd *cobra.Command, args []string) {
            // Phase 3 implementation
            fmt.Println("connect: not yet implemented")
        },
    }

    serversCmd := &cobra.Command{
        Use:   "servers",
        Short: "Manage VPN servers",
        Run: func(cmd *cobra.Command, args []string) {
            // Phase 4 implementation
            fmt.Println("servers: not yet implemented")
        },
    }

    configCmd := &cobra.Command{
        Use:   "config",
        Short: "View and modify configuration",
        Run: func(cmd *cobra.Command, args []string) {
            // Show current config
        },
    }

    rootCmd.AddCommand(connectCmd, serversCmd, configCmd)
    rootCmd.Execute()
}
```

### Graceful Shutdown with signal.NotifyContext
```go
// Source: Go stdlib signal.NotifyContext + community patterns
func main() {
    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer cancel()

    // Start application...

    // Block until signal
    <-ctx.Done()

    // Cleanup with timeout
    cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cleanupCancel()

    if err := shutdown(cleanupCtx); err != nil {
        log.Printf("cleanup error: %v", err)
    }
}
```

### Terminal State Save and Restore
```go
// Source: golang.org/x/term package documentation
import "golang.org/x/term"

// Save terminal state on startup
oldState, err := term.GetState(int(os.Stdin.Fd()))
if err != nil {
    log.Printf("warning: could not save terminal state: %v", err)
}

// Restore on cleanup (deferred or in signal handler)
if oldState != nil {
    term.Restore(int(os.Stdin.Fd()), oldState)
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Shell out to `./xray` binary | Embed xray-core as Go library via `core.New()` | Always available; v2ray-core had this pattern | Single binary, no IPC, programmatic control |
| Viper for config | koanf v2 for config | koanf v2 stable since 2023 | No key-lowercasing bug, lighter deps, clean marshal/unmarshal |
| `signal.Notify` + raw channels | `signal.NotifyContext` | Go 1.16 (2021) | Integrates with context cancellation; less boilerplate |
| `$HOME/.appname/config` | `os.UserConfigDir()` + app subdir | Go 1.13 (2019) | Cross-platform XDG compliance |
| cobra + Viper tight coupling | cobra + koanf (or any config lib) | Community trend 2024+ | Decoupled; cobra handles CLI, koanf handles config independently |

**Deprecated/outdated:**
- `ioutil.WriteFile`: Deprecated since Go 1.16; use `os.WriteFile` instead
- `github.com/knadh/koanf` (v1): Use `github.com/knadh/koanf/v2` -- v1 has different import paths
- `golang.org/x/crypto/ssh/terminal`: Deprecated; use `golang.org/x/term` instead

## Open Questions

1. **xray-core Go 1.26 requirement**
   - What we know: xray-core's go.mod specifies `go 1.26`. This is a very recent Go version.
   - What's unclear: Whether this is a hard requirement or if it builds with Go 1.22+. The `go` directive in go.mod since Go 1.21 controls language features and toolchain compatibility.
   - Recommendation: Use Go 1.26 toolchain to match xray-core's requirement. Verify with `go build` early.

2. **koanf structs provider for write-back**
   - What we know: koanf's `Marshal()` method serializes the koanf instance to bytes. The `structs` provider can load a struct into koanf.
   - What's unclear: Whether round-tripping (load YAML -> modify struct -> save YAML) preserves comments and field ordering.
   - Recommendation: Accept that comments are lost on write-back (standard YAML marshal behavior). Document this limitation. Keep config file simple enough that comment loss is not problematic.

3. **Proxy state file vs. PID file for crash detection**
   - What we know: The existing bash tool uses `data/proxy.pid` to track the xray process. The new app needs crash detection for `--cleanup`.
   - What's unclear: Best approach -- PID file (check if process alive), state file (record what proxy state was set), or both.
   - Recommendation: Use a state file at `~/.config/azad/.state.json` recording `{"proxy_set": true, "socks_port": 1080, "http_port": 8080, "pid": 12345, "network_service": "Wi-Fi"}`. On `--cleanup`, read this file and reverse proxy settings. On clean exit, delete the file.

## Sources

### Primary (HIGH confidence)
- Context7 `/xtls/xray-core` - Go programmatic API (core.New, instance.Start, LoadConfig)
- Context7 `/knadh/koanf` - Config load/save patterns, YAML parser, struct binding, marshal
- Context7 `/spf13/cobra` - Command structure, flags, subcommands, help generation
- [pkg.go.dev/github.com/xtls/xray-core/core](https://pkg.go.dev/github.com/xtls/xray-core/core) - Full API documentation (Instance, Server, LoadConfig, New, Start, Close)
- [pkg.go.dev/github.com/xtls/xray-core](https://pkg.go.dev/github.com/xtls/xray-core) - Module info (v1.260206.0, Go 1.26, MPL-2.0)
- xray-core go.mod (fetched raw from GitHub main branch) - Go 1.26 requirement, dependency list

### Secondary (MEDIUM confidence)
- [Go graceful shutdown patterns](https://victoriametrics.com/blog/go-graceful-shutdown/) - signal.NotifyContext best practices
- [Go project structure](https://www.glukhov.org/post/2025/12/go-project-structure/) - cmd/internal layout patterns 2025
- [Go XDG config](https://github.com/adrg/xdg) - XDG spec compliance context (confirmed stdlib sufficient)
- [Koanf + Cobra integration gist](https://gist.github.com/jxsl13/52127961c2cd2d2798cd340b4032218c) - Integration pattern reference
- [xray-core build flags](https://github.com/XTLS/Xray-core) - CGO_ENABLED=0, -trimpath, -ldflags="-s -w" build optimization

### Tertiary (LOW confidence)
- Terminal reset patterns (stty sane, term.Restore) - community knowledge, needs validation with bubbletea in Phase 4
- xray-core binary size (~40-60MB) - reported in GitHub issues, needs validation with actual build spike

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All libraries verified via Context7 and official docs; versions confirmed on pkg.go.dev
- Architecture: HIGH - Go cmd/internal pattern is well-established community standard; cobra command organization follows official user guide
- Pitfalls: HIGH - xray-core import side effects confirmed via Context7 code examples; koanf patterns verified via README; signal handling via Go stdlib docs
- Build considerations: MEDIUM - Binary size concern documented but not yet validated with actual build; Go 1.26 requirement from go.mod but not tested

**Research date:** 2026-02-25
**Valid until:** 2026-03-25 (stable ecosystem, slow-moving dependencies)

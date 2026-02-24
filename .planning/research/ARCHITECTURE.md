# Architecture Research

**Domain:** Go TUI VPN client wrapping Xray-core
**Researched:** 2026-02-24
**Confidence:** HIGH

## Recommended Architecture

### Pattern: Layered TUI Application with Embedded Proxy Engine

```
┌─────────────────────────────────────────────┐
│  CLI Layer (cobra)                          │
│  azad / azad connect / azad --cleanup       │
├─────────────────────────────────────────────┤
│  TUI Layer (bubbletea + lipgloss + bubbles) │
│  Views: Dashboard, ServerList, Ping,        │
│         Settings, Connection Stats          │
├─────────────────────────────────────────────┤
│  Application Core (pure Go)                 │
│  Config, Server Store, Subscription Parser, │
│  Protocol URI Parser, State Machine         │
├─────────────────────────────────────────────┤
│  Proxy Engine (xray-core as Go library)     │
│  core.New() / instance.Start/Close()        │
│  VLESS, VMess, Trojan, Shadowsocks          │
├─────────────────────────────────────────────┤
│  Platform Layer (OS abstractions)           │
│  System proxy, Keychain, Network detection  │
└─────────────────────────────────────────────┘
```

## Major Components

### 1. CLI Entry Point (`cmd/`)

**Responsibility:** Parse CLI flags, route to TUI or headless mode.

- `azad` — launches TUI (default)
- `azad connect` — headless quick-connect
- `azad servers` — list/manage servers without TUI
- `azad --cleanup` — unset system proxy, kill orphaned processes
- `azad --reset-terminal` — restore terminal state

Uses cobra for command routing. The TUI is the primary interface; CLI subcommands enable scripting and recovery.

### 2. TUI Layer (`internal/tui/`)

**Responsibility:** All user interaction via bubbletea Elm Architecture.

**Root Model Pattern:**
```go
type rootModel struct {
    activeView  viewType    // dashboard, servers, ping, settings
    dashboard   dashboardModel
    serverList  serverListModel
    pingView    pingModel
    settings    settingsModel
    statusBar   statusBarModel
    // shared state
    app         *app.App    // application core reference
    width       int
    height      int
}
```

**View switching** via parent model that routes messages to the active child. Each view is a separate bubbletea model that implements `Init()`, `Update()`, `View()`.

**Key patterns:**
- Parent model owns shared state, child views receive copies
- All I/O through `tea.Cmd` — never block `Update()` or `View()`
- Use `tea.Batch()` for concurrent operations (e.g., ping all servers)
- Status bar always visible showing connection state, proxy port, current server

### 3. Application Core (`internal/app/`)

**Responsibility:** Business logic, state management, no UI awareness.

**Sub-components:**
- `config` — koanf-based config load/save, YAML format, XDG paths
- `server` — Server data model, CRUD operations, JSON persistence
- `subscription` — Fetch, decode (base64/base64url), parse into server list
- `protocol` — URI parsers for vless://, vmess://, trojan://, ss://
- `proxy` — Xray-core lifecycle wrapper (configure, start, stop, status)
- `ping` — Concurrent TCP ping with latency measurement
- `stats` — Connection statistics (uptime, latency, data transfer)

**The App struct** is the glue:
```go
type App struct {
    Config       *config.Config
    ServerStore  *server.Store
    ProxyEngine  *proxy.Engine
    SubManager   *subscription.Manager
}
```

The TUI layer holds a reference to `App` and calls its methods via `tea.Cmd` functions.

### 4. Proxy Engine (`internal/proxy/`)

**Responsibility:** Wrap xray-core library for lifecycle management.

```go
type Engine struct {
    instance *core.Instance
    status   Status  // disconnected, connecting, connected, error
    config   *ProxyConfig
    mu       sync.RWMutex
}

func (e *Engine) Start(ctx context.Context, cfg *ProxyConfig) error
func (e *Engine) Stop() error
func (e *Engine) Status() Status
func (e *Engine) Stats() *ConnectionStats
```

**Critical:** Import xray-core as Go library, not external binary:
- Import `github.com/xtls/xray-core/core`
- Side-effect imports for JSON config: `_ "github.com/xtls/xray-core/main/json"`
- Side-effect imports for protocols: `_ "github.com/xtls/xray-core/proxy/vless/inbound"` etc.
- Build config programmatically or via `core.LoadConfig("json", reader)`

### 5. Platform Layer (`internal/platform/`)

**Responsibility:** OS-specific operations behind interfaces.

```go
type SystemProxy interface {
    Set(socksAddr, httpAddr string) error
    Unset() error
    Status() (bool, error)
}
```

Implementations:
- `platform_darwin.go` — macOS via sysproxy/networksetup
- `platform_linux.go` — Linux via env vars + gsettings
- `platform_windows.go` — Windows via sysproxy/registry

## Data Flow

### Connection Flow
```
User presses "Connect"
  → TUI sends ConnectMsg to rootModel.Update()
  → Update() returns tea.Cmd that calls app.ProxyEngine.Start()
  → Goroutine starts xray-core instance
  → Returns ConnectedMsg or ErrorMsg
  → Update() receives msg, updates status bar
  → View() re-renders with new connection state
```

### Subscription Import Flow
```
User enters subscription URL
  → TUI sends FetchSubMsg to Update()
  → Update() returns tea.Cmd that calls app.SubManager.Fetch(url)
  → Goroutine: HTTP fetch → base64 decode → parse URIs → extract servers
  → Returns SubFetchedMsg{servers: []Server}
  → Update() adds servers to ServerStore, re-renders list
```

### Ping Flow
```
User navigates to Ping view
  → TUI sends PingAllMsg
  → Update() returns tea.Batch of ping Cmds (one per server, max 20 concurrent)
  → Each goroutine: net.DialTimeout → measure latency → return PingResultMsg{server, latency}
  → Update() receives each result, updates server latency, re-renders progressively
```

## Build Order (dependency chain)

### Phase 1: Foundation
Build order: config → protocol parsers → server store → proxy engine spike
- These have no UI dependency
- Validates xray-core library import works
- Unit testable in isolation

### Phase 2: Core TUI Shell
Build order: cobra CLI → root model → view switching → status bar → basic server list view
- Depends on: config (for layout preferences), server store (for data)
- Establishes the TUI framework all other views build on

### Phase 3: Connection Engine
Build order: proxy engine (full) → connect/disconnect → system proxy → connection stats
- Depends on: Phase 1 (engine), Phase 2 (TUI to show status)
- This is where the app becomes functional

### Phase 4: Intelligence
Build order: concurrent ping → smart server selection → subscription management → quick connect
- Depends on: Phase 3 (need working connections to test against)
- This is where the app becomes delightful

### Phase 5: Polish
Build order: live stats → auto-reconnect → themes → search/filter → keyboard help
- Depends on: all previous phases
- This is where the app becomes professional

## Key Architectural Decisions

### 1. Xray-core as Library (not binary)
Import as Go module. Single binary. No IPC, no PID files, no binary distribution.
Trade-off: ~40-60MB binary. Acceptable.

### 2. Parent Model with Active View
Root model owns shared state and routes to child views. Not a model stack — we need persistent state (connection status) visible across all views.

### 3. Application Core Separated from TUI
Business logic in `internal/app/` with no bubbletea imports. TUI in `internal/tui/` wraps app calls in `tea.Cmd`. This enables headless CLI usage and unit testing without TUI.

### 4. Platform Abstraction from Day One
All OS-specific code behind interfaces. Build tags (`//go:build darwin`) for implementations. Never `runtime.GOOS` checks scattered through business logic.

### 5. JSON for Server Storage
Replace `name|link` flat file with `servers.json`. Supports rich metadata (latency, last connected, tags, subscription source) without delimiter conflicts.

## Sources

- [Bubbletea v2 Documentation](https://pkg.go.dev/charm.land/bubbletea/v2) — Elm Architecture, Cmd/Msg pattern
- [Managing Nested Models in Bubbletea](https://donderom.com/posts/managing-nested-models-with-bubble-tea/) — Parent-child model patterns
- [Multi-View Interfaces in Bubbletea](https://shi.foo/weblog/multi-view-interfaces-in-bubble-tea/) — View routing
- [Xray-core Library API](https://pkg.go.dev/github.com/xtls/xray-core/core) — core.New, instance.Start/Close
- [GoXRay Project](https://github.com/goxray) — Validates xray-core-as-library approach
- [xray-knife](https://github.com/lilendian0x00/xray-knife) — URI parsing reference implementation

---
*Architecture research for: Go TUI VPN client wrapping Xray-core (Azad)*
*Researched: 2026-02-24*

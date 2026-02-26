# Phase 5: Quick Connect - Research

**Researched:** 2026-02-26
**Domain:** CLI auto-connect behavior, TUI auto-connect on startup, config persistence for server preferences
**Confidence:** HIGH

## Summary

Phase 5 delivers the "one command" promise: running `azad` with no arguments should auto-connect to the best available server, and `azad connect` should do the same headlessly. This phase is primarily a wiring and integration phase, not a new-library phase. All the building blocks already exist: the `engine.Engine` manages Xray-core lifecycle, `serverstore.Store` provides CRUD with atomic persistence, `config.Config` already has `Server.LastUsed` field, `sysproxy` sets/unsets system proxy, and the TUI renders connection state. The work is connecting these pieces with auto-connect logic and persisting preferences.

Three requirements drive this phase: (1) QCON-01: TUI auto-connect on launch (select last-used or fastest server, initiate connection automatically), (2) QCON-02: headless `azad connect` without TUI (connect, print status, block on signal), (3) QCON-03: persist server preference and last-used selection between sessions. The headless connect command (`connect.go`) already exists with most of the QCON-02 logic but does not yet save `LastUsed` after connecting or handle the "fastest server" fallback when no history exists. The TUI currently does NOT initiate connections at all -- `enter` only selects a server in the detail panel. Both the TUI and CLI need a connection-initiation flow, and both need to persist `LastUsed` after successful connections.

The primary technical challenges are: (1) implementing auto-connect as a Bubble Tea `Init` command that fires on TUI startup (before user interaction), (2) adding a "fastest server" selection strategy when no `LastUsed` is set (ping-then-connect or use stored latency), (3) saving `LastUsed` to config after successful connection from both TUI and CLI paths, and (4) ensuring the TUI connection flow (start engine, set system proxy, verify IP, write proxy state) runs correctly as async `tea.Cmd` commands without blocking the UI.

**Primary recommendation:** Build a shared `quickconnect` package (or inline the logic) that resolves the "best server" using the existing `findServer` pattern (last-used > lowest-stored-latency > first-in-list). Wire this into both `root.go` (TUI path via `Init` command) and `connect.go` (headless path). After successful connection in either path, save `cfg.Server.LastUsed = server.ID` and `server.LastConnected = time.Now()` to persist preferences.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| QCON-01 | Running `azad` (no args) launches TUI and auto-connects to last-used or fastest server | TUI model's `Init()` returns a `tea.Cmd` that resolves best server and initiates connection; `connectResultMsg` already handled in `Update`; auto-select server in list to match connected server |
| QCON-02 | Running `azad connect` connects headlessly without TUI, printing status to stdout | `connect.go` already implements most of this via `runConnect`; needs: (a) save `LastUsed` on success, (b) "fastest server" fallback when no args and no LastUsed, (c) update `LastConnected` on server |
| QCON-03 | Server preference and last-used selection persist between sessions | `config.Config.Server.LastUsed` already exists with `config.Save()`; `protocol.Server.LastConnected` field exists; need to call `config.Save()` after connection and update `LastConnected` in server store |
</phase_requirements>

## Standard Stack

### Core (already in project)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Bubble Tea | v2.0.0 (`charm.land/bubbletea/v2`) | TUI framework | Already wired; `Init()` returns `tea.Cmd` for auto-connect on startup |
| cobra | v1.9.1 | CLI routing | Root command (TUI) and `connect` subcommand (headless) already exist |
| koanf | v2.3.2 | Config read/write | `config.Save()` already tested and working for `LastUsed` persistence |
| engine | internal | Xray-core lifecycle | `Engine.Start()`/`Stop()` already handle connection; TUI needs to call these via `tea.Cmd` |
| serverstore | internal | Server CRUD | `Store.List()`/`FindByID()` for server resolution; needs new `UpdateServer()` for `LastConnected` |
| sysproxy | internal | System proxy | `SetSystemProxy()`/`UnsetSystemProxy()` already used in headless connect |
| lifecycle | internal | Cleanup and signals | `ProxyState` write for crash recovery already exists in CLI path |

### Supporting (no new dependencies)
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| config | internal | Paths + load/save | Save `LastUsed` after successful connection |
| protocol | internal | Server struct | `LastConnected` field already exists on `Server` struct |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Ping-then-connect for "fastest" | Use stored `LatencyMs` from last ping session | Avoids startup delay; stored latency may be stale but is good enough for UX |
| New preferences file | Extend existing `config.yaml` | Keeps things simple; `ServerConfig` already has `LastUsed`; add more fields as needed |

**Installation:**
No new dependencies. All existing packages suffice.

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── cli/
│   ├── root.go           # TUI launch + auto-connect via Init
│   ├── connect.go        # Headless connect + LastUsed persistence
│   └── ...
├── config/
│   └── config.go         # Config struct (already has Server.LastUsed)
├── engine/
│   └── engine.go         # Xray-core lifecycle (unchanged)
├── serverstore/
│   └── store.go          # Add UpdateServer() for LastConnected
├── tui/
│   ├── app.go            # Init() returns auto-connect cmd
│   ├── connect_cmd.go    # NEW: tea.Cmd functions for TUI connection flow
│   └── messages.go       # Add autoConnectMsg
└── ...
```

### Pattern 1: Auto-Connect via Bubble Tea Init Command
**What:** The TUI model's `Init()` method returns a `tea.Cmd` that resolves the best server and initiates a connection. This fires immediately when the TUI starts, before any user interaction.
**When to use:** QCON-01 -- TUI auto-connect on launch
**Example:**
```go
// Source: Context7 / charmbracelet/bubbletea tutorials
func (m model) Init() tea.Cmd {
    return tea.Batch(
        tickCmd(),
        m.autoConnectCmd(),
    )
}

func (m model) autoConnectCmd() tea.Cmd {
    // Capture values needed by the goroutine (not model itself)
    store := m.store
    cfg := m.cfg
    eng := m.engine

    return func() tea.Msg {
        server, err := resolveServer(store, cfg)
        if err != nil {
            return connectResultMsg{Err: err}
        }

        if err := eng.Start(context.Background(), *server, cfg.Proxy.SOCKSPort, cfg.Proxy.HTTPPort); err != nil {
            return connectResultMsg{Err: err}
        }

        // Set system proxy, write proxy state, verify IP (best-effort)
        // ... (same flow as connect.go)

        return connectResultMsg{Err: nil}
    }
}
```

### Pattern 2: Shared Server Resolution Logic
**What:** Extract `findServer` from `connect.go` into a shared function (or keep in `cli` package) that both TUI and headless paths use. The resolution order is: (1) explicit server name/ID if provided, (2) `cfg.Server.LastUsed` if set and server still exists, (3) server with lowest stored `LatencyMs` > 0, (4) first server in list.
**When to use:** Both QCON-01 and QCON-02
**Example:**
```go
// resolveServer picks the best server to connect to.
// Priority: lastUsed > lowest latency > first server
func resolveServer(store *serverstore.Store, cfg *config.Config) (*protocol.Server, error) {
    servers := store.List()
    if len(servers) == 0 {
        return nil, fmt.Errorf("no servers available")
    }

    // Try last-used
    if cfg.Server.LastUsed != "" {
        if srv, ok := store.FindByID(cfg.Server.LastUsed); ok {
            return srv, nil
        }
    }

    // Try fastest (lowest positive LatencyMs)
    var fastest *protocol.Server
    for i := range servers {
        if servers[i].LatencyMs > 0 {
            if fastest == nil || servers[i].LatencyMs < fastest.LatencyMs {
                s := servers[i]
                fastest = &s
            }
        }
    }
    if fastest != nil {
        return fastest, nil
    }

    // Fall back to first
    return &servers[0], nil
}
```

### Pattern 3: Persist LastUsed After Connection
**What:** After a successful connection (engine started, proxy working), save `cfg.Server.LastUsed = server.ID` to config and update `server.LastConnected` in the store. This happens in both the TUI (via a follow-up `tea.Cmd`) and the headless CLI (inline after `eng.Start`).
**When to use:** QCON-03
**Example:**
```go
// In headless connect (connect.go), after eng.Start succeeds:
cfg.Server.LastUsed = server.ID
if cfgPath, err := config.FilePath(); err == nil {
    _ = config.Save(cfg, cfgPath)
}

// Update LastConnected on server in store
server.LastConnected = time.Now()
_ = store.UpdateServer(*server)
```

### Pattern 4: TUI Connection Flow as tea.Cmd
**What:** The full connection flow (start engine, detect network service, write proxy state, set system proxy, verify IP) runs inside a `tea.Cmd` function. This keeps the UI responsive while the connection is established. The result message triggers status bar updates and LastUsed persistence.
**When to use:** QCON-01 (TUI connections)
**Example:**
```go
func connectServerCmd(srv protocol.Server, eng *engine.Engine, cfg *config.Config, store *serverstore.Store) tea.Cmd {
    return func() tea.Msg {
        ctx := context.Background()
        if err := eng.Start(ctx, srv, cfg.Proxy.SOCKSPort, cfg.Proxy.HTTPPort); err != nil {
            return connectResultMsg{Err: err}
        }

        // System proxy (best-effort)
        svc, svcErr := sysproxy.DetectNetworkService()
        if svcErr == nil {
            writeProxyState(svc, cfg.Proxy.SOCKSPort, cfg.Proxy.HTTPPort)
            _ = sysproxy.SetSystemProxy(svc, cfg.Proxy.SOCKSPort, cfg.Proxy.HTTPPort)
        }

        // Persist preferences
        cfg.Server.LastUsed = srv.ID
        if cfgPath, err := config.FilePath(); err == nil {
            _ = config.Save(cfg, cfgPath)
        }
        srv.LastConnected = time.Now()
        _ = store.UpdateServer(srv)

        // Verify IP (best-effort, don't fail on this)
        _, _ = engine.VerifyIP(cfg.Proxy.SOCKSPort)

        return connectResultMsg{Err: nil}
    }
}
```

### Pattern 5: TUI Disconnect Flow
**What:** When user presses `q`/`ctrl+c` while connected, the TUI should stop the engine, unset system proxy, and remove state file before quitting. This needs a pre-quit cleanup sequence.
**When to use:** TUI quit while connected
**Example:**
```go
case "q", "ctrl+c":
    // Check if connected, clean up before quitting
    status, _, _ := m.engine.Status()
    if status == engine.StatusConnected {
        return m, tea.Sequence(
            disconnectCmd(m.engine),
            tea.Quit,
        )
    }
    return m, tea.Quit
```

### Anti-Patterns to Avoid
- **Blocking Init:** Never block inside `Init()` or `Update()` directly. Always return a `tea.Cmd` for I/O work. The connection flow (engine start, system proxy, verify IP) takes 2-10 seconds and must not block rendering.
- **Goroutine leaks in TUI:** Don't spawn raw goroutines from the model. Use `tea.Cmd` functions which Bubble Tea manages. The engine's `context.Background()` is fine since `engine.Stop()` handles cleanup.
- **Shared mutable config:** The `cfg` pointer is shared between root model and commands. `tea.Cmd` functions capture `cfg` from the model but run in a goroutine. Since only one connection happens at a time and config writes are idempotent, this is safe in practice, but be aware of the pattern.
- **Stale latency data:** Don't block startup with a full ping sweep. Use stored `LatencyMs` from previous sessions for "fastest" selection. Users can manually `p` (ping all) in the TUI to refresh.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Server resolution logic | Separate "smart selector" | Simple priority chain (lastUsed > fastestStored > first) | YAGNI; complex selection algorithms add no value for 10-100 servers |
| Config file format | Custom preferences file | Extend existing `config.yaml` via koanf | Already tested and working; `LastUsed` field already exists |
| Connection state machine | New state machine library | Engine's `ConnectionStatus` enum + TUI view state | Existing pattern works; 4 states (disconnected/connecting/connected/error) cover all cases |
| Signal handling for headless | Custom signal code | Existing `lifecycle.WithShutdown` + cobra context | Already provides SIGINT/SIGTERM handling via `<-ctx.Done()` |

**Key insight:** Phase 5 is an integration phase. Every piece already exists. The danger is over-engineering what is essentially "wire existing things together and save a preference."

## Common Pitfalls

### Pitfall 1: TUI Auto-Connect With Empty Server Store
**What goes wrong:** App launches, `Init()` fires auto-connect, but server store is empty (first run). The connection fails with "no servers available" and user sees an error flash.
**Why it happens:** Auto-connect assumes servers exist.
**How to avoid:** Check server count before auto-connecting. If empty, skip auto-connect silently. The TUI should show the empty server list with instructions to add servers (`a` key).
**Warning signs:** `connectResultMsg{Err: "no servers available"}` appearing on fresh installs.

### Pitfall 2: Config Save Race Between TUI and Engine
**What goes wrong:** `config.Save()` is called from a `tea.Cmd` goroutine while the main model also reads `cfg`. If a second connection attempt happens before the first save completes, config could be corrupted.
**Why it happens:** Shared `*config.Config` pointer between model and commands.
**How to avoid:** Config saves are idempotent (write full struct). The koanf Save() creates a fresh instance. Concurrent saves of the same data are harmless. For extra safety, save config from the `Update` handler (main goroutine) after receiving `connectResultMsg`, not from the command goroutine.
**Warning signs:** Truncated or empty config.yaml after connection.

### Pitfall 3: System Proxy Left Set After TUI Crash
**What goes wrong:** TUI auto-connects (sets system proxy), then crashes or is killed. System proxy remains set, routing all traffic through a dead proxy.
**Why it happens:** TUI crash skips cleanup.
**How to avoid:** This is already handled by the `lifecycle.ProxyState` + `--cleanup` mechanism. Ensure the TUI connection flow writes `.state.json` BEFORE setting system proxy (same pattern as `connect.go`). Also detect dirty state on startup via `PersistentPreRunE` or at TUI init.
**Warning signs:** `config.StateFilePath()` file exists on startup.

### Pitfall 4: Headless Connect Not Saving LastUsed
**What goes wrong:** User runs `azad connect`, connects successfully, but `LastUsed` is never saved. Next `azad` launch doesn't remember the server.
**Why it happens:** The existing `connect.go` doesn't call `config.Save()` after connection.
**How to avoid:** Add `config.Save()` call after successful `eng.Start()` in `runConnect`.
**Warning signs:** `azad config` always shows empty `last_used`.

### Pitfall 5: Server Deleted Between Sessions
**What goes wrong:** `LastUsed` references a server ID that was deleted (user ran `D` to clear all, or subscription refresh removed it). Auto-connect fails to find the server.
**Why it happens:** ID stored in config no longer exists in server store.
**How to avoid:** `findServer`/`resolveServer` already handles this -- `store.FindByID` returns false, and we fall through to the next strategy (fastest, then first). Just ensure the fallback chain is complete.
**Warning signs:** Always falling back to first server despite having `LastUsed` set.

### Pitfall 6: Enter Key Ambiguity in TUI
**What goes wrong:** `enter` currently does `syncDetail()` (show detail). Phase 5 changes it to "connect to selected server." Users expecting detail-view get a connection instead.
**Why it happens:** Overloading the `enter` key.
**How to avoid:** The keybindings already define `enter/c` for "connect." Change `enter` to initiate connection (start engine for selected server). Detail panel auto-updates on list navigation (j/k) anyway, so `enter` for connect is the natural meaning.
**Warning signs:** Confusing UX feedback on `enter` press.

## Code Examples

Verified patterns from codebase analysis:

### Existing findServer in connect.go (to be extracted/shared)
```go
// Source: internal/cli/connect.go lines 134-169
func findServer(store *serverstore.Store, cfg *config.Config, args []string) (*protocol.Server, error) {
    servers := store.List()
    if len(servers) == 0 {
        return nil, fmt.Errorf("no servers available")
    }
    // Try explicit name, then LastUsed, then first server
    if cfg.Server.LastUsed != "" {
        if srv, ok := store.FindByID(cfg.Server.LastUsed); ok {
            return srv, nil
        }
    }
    return &servers[0], nil
}
```

### Config Save Pattern (already tested)
```go
// Source: internal/config/config.go + config_test.go
cfg.Server.LastUsed = server.ID
configPath, _ := config.FilePath()
config.Save(cfg, configPath)
```

### Bubble Tea Init With Auto-Connect
```go
// Source: Context7 / charmbracelet/bubbletea - Init with commands
func (m model) Init() tea.Cmd {
    return tea.Batch(
        tickCmd(),           // existing uptime ticker
        m.autoConnectCmd(),  // new: resolve + connect on startup
    )
}
```

### Updating Server LastConnected in Store
```go
// New method needed on serverstore.Store
func (s *Store) UpdateServer(updated protocol.Server) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    for i := range s.servers {
        if s.servers[i].ID == updated.ID {
            s.servers[i] = updated
            return s.save()
        }
    }
    return fmt.Errorf("server %q not found", updated.ID)
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `enter` in TUI shows detail | `enter` in TUI connects | Phase 5 | Detail auto-syncs on j/k navigation; `enter` becomes connect action |
| No auto-connect on TUI launch | `Init()` returns auto-connect cmd | Phase 5 | TUI connects immediately on startup |
| `connect.go` doesn't save LastUsed | `connect.go` saves LastUsed after success | Phase 5 | Headless connect remembers server for next session |
| First server fallback only | LastUsed > lowest LatencyMs > first | Phase 5 | Smarter server selection |

**Deprecated/outdated:**
- None. This phase extends existing patterns without replacing anything.

## Open Questions

1. **Should auto-connect show a "Connecting..." state in TUI before server list renders?**
   - What we know: The `Init()` command runs async so the TUI will render immediately with "Disconnected" status, then transition to "Connecting" then "Connected" as messages arrive. This is natural and provides visual feedback.
   - What's unclear: Whether users will notice the brief "Disconnected" flash on fast connections.
   - Recommendation: Accept the flash. It's brief (< 1 second on fast servers) and consistent with how the TUI works for all other async operations. Could optionally set initial status to "Connecting" in `New()` if auto-connect will fire.

2. **Should `azad connect` without arguments also be an alias for the TUI?**
   - What we know: Requirements say `azad` (no args) = TUI + auto-connect, `azad connect` = headless. These are distinct behaviors.
   - What's unclear: Whether `azad connect --tui` or similar flag is needed.
   - Recommendation: Keep them distinct. `azad` = TUI, `azad connect` = headless. No flags needed for v1.

3. **Should we ping-then-connect for "fastest" on first run?**
   - What we know: First run has no `LastUsed` and no stored `LatencyMs`. Falling back to first-in-list is arbitrary.
   - What's unclear: Whether users expect a ping sweep on first connect.
   - Recommendation: Fall back to first server on first run. Users can manually ping (`p`) and re-connect. A startup ping sweep adds 5+ seconds of delay and complexity. Not worth it for v1.

## Sources

### Primary (HIGH confidence)
- Codebase analysis of all `internal/` packages (direct file reads)
- Context7 `/charmbracelet/bubbletea` - Init commands, tea.Batch, tea.Sequence patterns
- Context7 `/spf13/cobra` - RunE, PersistentPreRunE, subcommand routing

### Secondary (MEDIUM confidence)
- Bubble Tea tutorials (README.md, commands tutorial) via Context7

### Tertiary (LOW confidence)
- None. All findings based on codebase analysis and verified library documentation.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - No new dependencies; all packages already in project and working
- Architecture: HIGH - Patterns extrapolated directly from existing working code (connect.go, TUI app.go)
- Pitfalls: HIGH - Based on direct codebase analysis of existing flows and shared state patterns

**Research date:** 2026-02-26
**Valid until:** 2026-03-28 (30 days - stable domain, no external API changes expected)

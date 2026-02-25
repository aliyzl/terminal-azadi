# Phase 4: TUI and Server Interaction - Research

**Researched:** 2026-02-25
**Domain:** Terminal User Interface (TUI) with Bubble Tea v2, Lip Gloss v2, Bubbles v2
**Confidence:** HIGH

## Summary

Phase 4 builds the interactive terminal user interface for Azad, replacing the current placeholder CLI commands with a full split-pane TUI. The Charmbracelet v2 stack (Bubble Tea, Lip Gloss, Bubbles) was chosen as a locked decision in earlier phases and has reached stable v2.0.0 release (February 24, 2025) with the new `charm.land/*` module paths. The v2 release introduced significant breaking changes from v1: `View()` now returns `tea.View` (not `string`), key messages changed from `tea.KeyMsg` to `tea.KeyPressMsg`, and terminal features are declared on the View struct rather than via commands.

The existing codebase provides a strong foundation: `engine.Engine` manages Xray-core lifecycle with mutex-protected state, `serverstore.Store` handles CRUD with atomic persistence, `subscription.Fetch` downloads and parses server lists, and `protocol.Server` carries rich metadata including latency. The TUI will compose these existing packages into an interactive experience using Bubble Tea's Model-View-Update architecture with child model composition for the split-pane layout.

The primary technical challenges are: (1) wiring concurrent server pinging through Bubble Tea commands (not raw goroutines), (2) managing focus between list panel, detail panel, and modal overlays (help, add-server forms), and (3) ensuring the layout adapts to terminal resizing while remaining readable in both dark and light terminals via Lip Gloss adaptive colors.

**Primary recommendation:** Build a root model that owns child models (server list, detail panel, status bar, help overlay) and routes messages via a focus enum. Use `charm.land/bubbles/v2/list` for the server list with built-in fuzzy filtering. Use `tea.Batch` for concurrent pings, sending individual `pingResultMsg` messages back to Update.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| TUI-01 | Split-pane layout: server list panel, detail panel, status bar | Lip Gloss `JoinHorizontal`/`JoinVertical` for compositing panels; `lipgloss.Width()`/`lipgloss.Height()` for measurement; `tea.WindowSizeMsg` for responsive sizing |
| TUI-02 | Vim-style navigation (j/k, Enter, Esc, q) | `tea.KeyPressMsg` with `.String()` matching; `key.NewBinding` from `charm.land/bubbles/v2/key` for structured keybindings |
| TUI-03 | Fuzzy-search/filter by name, country, protocol | `bubbles/list` has built-in fuzzy filtering via `SetFilteringEnabled(true)`; `FilterValue()` on list items controls searchable content |
| TUI-04 | Status bar: connection state, current server, proxy port, uptime | Custom status bar component reading from `engine.Engine.Status()` and a `time.Time` connection start tracker; `tea.Tick` for uptime updates |
| TUI-05 | Help overlay via ? key | `bubbles/help` component with `ShortHelp()/FullHelp()` interface on a `keyMap` struct; toggle with `?` keybinding |
| TUI-06 | Adaptive layout for terminal size, minimum-size message | `tea.WindowSizeMsg` broadcasts dimensions; check minimum (e.g., 60x20); show fallback message if too small |
| TUI-07 | Consistent color palette via lipgloss, dark/light readability | `lipgloss.AdaptiveColor{Light: "...", Dark: "..."}` for all colors; define a `theme` struct with named styles |
| SRVR-01 | View server list with name, protocol, latency | Server list panel using `bubbles/list` with custom `list.DefaultItem` showing name, protocol badge, latency |
| SRVR-02 | Add server by pasting URI | Text input modal using `bubbles/textinput`; parse via existing `protocol.ParseURI()`; add via `serverstore.Store.Add()` |
| SRVR-03 | Add servers from subscription URL | Text input modal for URL; fetch via existing `subscription.Fetch()`; bulk add to store |
| SRVR-04 | Refresh subscription | Iterate stored subscription URLs, call `store.ReplaceBySource()`; show spinner during fetch |
| SRVR-05 | Remove individual servers or clear all | Keybinding `d` for delete selected, `D` for clear all with confirmation; use `store.Remove(id)` / `store.Clear()` |
| SRVR-06 | Ping all servers concurrently with progress, sort by latency | `tea.Batch` of ping commands using `net.DialTimeout`; `progress.Model` for visual feedback; sort and re-set list items on completion |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Module Path | Purpose | Why Standard |
|---------|---------|-------------|---------|--------------|
| Bubble Tea | v2.0.0 | `charm.land/bubbletea/v2` | TUI framework (Model-View-Update) | Only production-grade Go TUI framework; Elm architecture; handles terminal I/O, rendering, event loop |
| Lip Gloss | v2.0.0 | `charm.land/lipgloss/v2` | Terminal styling and layout | CSS-like styling; `JoinHorizontal`/`JoinVertical`/`Place` for layout; `AdaptiveColor` for dark/light |
| Bubbles | v2.0.0 | `charm.land/bubbles/v2` | Pre-built TUI components | list, textinput, viewport, spinner, progress, help, key -- all battle-tested Bubble Tea components |

### Supporting (already in project)
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| cobra | v1.9.1 | CLI routing | Already wired; root command will launch TUI when run without args |
| serverstore | internal | Server CRUD | Already exists at `internal/serverstore`; TUI reads/writes through this |
| engine | internal | Xray-core proxy lifecycle | Already exists at `internal/engine`; TUI calls `Start()`/`Stop()`/`Status()` |
| subscription | internal | Subscription fetch/decode | Already exists at `internal/subscription`; TUI triggers via commands |
| protocol | internal | URI parsing, Server struct | Already exists at `internal/protocol`; TUI uses `ParseURI()` and `Server` type |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| bubbles/list | Custom list component | list bubble has built-in fuzzy filtering, pagination, status bar, help -- building custom would duplicate all of this |
| lipgloss layout | bubblelayout (third-party) | bubblelayout adds declarative layout but is a third-party dep; lipgloss Join/Place is sufficient for a two-panel layout |
| net.DialTimeout for pings | ICMP ping library | ICMP requires root/raw socket; TCP dial to server port is sufficient and works without privileges |

**Installation:**
```bash
go get charm.land/bubbletea/v2@v2.0.0
go get charm.land/lipgloss/v2@v2.0.0
go get charm.land/bubbles/v2@v2.0.0
```

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── tui/                   # TUI package (new)
│   ├── app.go             # Root model, Init/Update/View, program entry
│   ├── theme.go           # Color palette, named styles (AdaptiveColor)
│   ├── keys.go            # keyMap struct, all keybindings, help interface
│   ├── serverlist.go      # Server list panel (wraps bubbles/list)
│   ├── detail.go          # Detail panel (server info display)
│   ├── statusbar.go       # Status bar (connection state, uptime)
│   ├── help.go            # Help overlay (wraps bubbles/help)
│   ├── input.go           # Input modals (add server, add subscription)
│   ├── ping.go            # Ping command logic (tea.Cmd producers)
│   └── messages.go        # Custom message types (pingResultMsg, etc.)
├── cli/                   # Existing CLI package
│   ├── root.go            # Modified: RunE launches TUI instead of help
│   └── connect.go         # Existing headless connect (kept for Phase 5)
├── engine/                # Existing (unchanged)
├── serverstore/           # Existing (unchanged)
├── protocol/              # Existing (unchanged)
├── subscription/          # Existing (unchanged)
├── config/                # Existing (unchanged)
├── sysproxy/              # Existing (unchanged)
└── lifecycle/             # Existing (unchanged)
```

### Pattern 1: Root Model with Child Composition
**What:** A single root model owns child models and routes messages via a focus/view state enum.
**When to use:** Any multi-panel Bubble Tea application.
**Example:**
```go
// Source: Charmbracelet community pattern, verified via official discussions
package tui

import (
    tea "charm.land/bubbletea/v2"
    "charm.land/bubbles/v2/list"
    "charm.land/lipgloss/v2"
)

type viewState int
const (
    viewNormal viewState = iota
    viewHelp
    viewAddServer
    viewAddSubscription
    viewConfirmClear
)

type model struct {
    // Child models
    serverList  list.Model
    detail      detailModel
    statusBar   statusBarModel
    help        helpModel
    input       inputModel

    // State
    view        viewState
    width       int
    height      int
    ready       bool

    // Shared references (read from, write via commands)
    store       *serverstore.Store
    engine      *engine.Engine
    cfg         *config.Config
}

func (m model) View() tea.View {
    if !m.ready {
        return tea.NewView("Initializing...")
    }
    if m.width < minWidth || m.height < minHeight {
        return tea.NewView("Terminal too small. Resize to at least 60x20.")
    }

    // Build layout
    listPanel := m.renderListPanel()
    detailPanel := m.renderDetailPanel()
    mainContent := lipgloss.JoinHorizontal(lipgloss.Top, listPanel, detailPanel)
    statusBar := m.statusBar.View()
    content := lipgloss.JoinVertical(lipgloss.Left, mainContent, statusBar)

    // Overlay help if active
    if m.view == viewHelp {
        content = m.help.Overlay(content)
    }

    v := tea.NewView(content)
    v.AltScreen = true
    return v
}
```

### Pattern 2: Message-Based Async Operations (Pinging)
**What:** Use `tea.Batch` to fire concurrent commands; each returns a message processed by `Update`.
**When to use:** Concurrent server pinging, subscription fetching, connection verification.
**Example:**
```go
// Source: Bubble Tea commands documentation (charm.land/blog/commands-in-bubbletea)
type pingResultMsg struct {
    ServerID  string
    LatencyMs int
    Err       error
}

type pingCompleteMsg struct{}

func pingAllCmd(servers []protocol.Server) tea.Cmd {
    cmds := make([]tea.Cmd, len(servers))
    for i, srv := range servers {
        srv := srv // capture
        cmds[i] = func() tea.Msg {
            start := time.Now()
            conn, err := net.DialTimeout("tcp",
                fmt.Sprintf("%s:%d", srv.Address, srv.Port),
                5*time.Second)
            if err != nil {
                return pingResultMsg{ServerID: srv.ID, Err: err}
            }
            conn.Close()
            latency := int(time.Since(start).Milliseconds())
            return pingResultMsg{ServerID: srv.ID, LatencyMs: latency}
        }
    }
    return tea.Batch(cmds...)
}
```

### Pattern 3: Adaptive Color Theme
**What:** Define all colors as `lipgloss.AdaptiveColor` so the UI works on both dark and light terminals.
**When to use:** Every styled element in the TUI.
**Example:**
```go
// Source: Lip Gloss v2 docs (charm.land/lipgloss/v2)
package tui

import "charm.land/lipgloss/v2"

type Theme struct {
    Primary       lipgloss.AdaptiveColor
    Secondary     lipgloss.AdaptiveColor
    Accent        lipgloss.AdaptiveColor
    Muted         lipgloss.AdaptiveColor
    Success       lipgloss.AdaptiveColor
    Warning       lipgloss.AdaptiveColor
    Error         lipgloss.AdaptiveColor
    Border        lipgloss.AdaptiveColor
    StatusBar     lipgloss.AdaptiveColor
    StatusBarText lipgloss.AdaptiveColor
}

var DefaultTheme = Theme{
    Primary:       lipgloss.AdaptiveColor{Light: "235", Dark: "252"},
    Secondary:     lipgloss.AdaptiveColor{Light: "241", Dark: "245"},
    Accent:        lipgloss.AdaptiveColor{Light: "63", Dark: "86"},
    Muted:         lipgloss.AdaptiveColor{Light: "250", Dark: "238"},
    Success:       lipgloss.AdaptiveColor{Light: "34", Dark: "78"},
    Warning:       lipgloss.AdaptiveColor{Light: "208", Dark: "214"},
    Error:         lipgloss.AdaptiveColor{Light: "196", Dark: "204"},
    Border:        lipgloss.AdaptiveColor{Light: "245", Dark: "240"},
    StatusBar:     lipgloss.AdaptiveColor{Light: "235", Dark: "236"},
    StatusBarText: lipgloss.AdaptiveColor{Light: "252", Dark: "252"},
}
```

### Pattern 4: Focus Management Between Panels
**What:** Track which panel has focus; route key messages only to the focused panel.
**When to use:** Multi-panel layouts where components share keybindings.
**Example:**
```go
type focusedPanel int
const (
    focusList focusedPanel = iota
    focusDetail
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyPressMsg:
        // Global keys always handled regardless of focus
        switch msg.String() {
        case "q", "ctrl+c":
            return m, tea.Quit
        case "?":
            m.view = viewHelp
            return m, nil
        case "tab":
            if m.focused == focusList {
                m.focused = focusDetail
            } else {
                m.focused = focusList
            }
            return m, nil
        }

    case tea.WindowSizeMsg:
        // Broadcast to all children
        m.width = msg.Width
        m.height = msg.Height
        m.serverList.SetSize(m.listWidth(), m.listHeight())
        m.detail.SetSize(m.detailWidth(), m.detailHeight())
        return m, nil
    }

    // Route to focused panel
    var cmd tea.Cmd
    switch m.focused {
    case focusList:
        m.serverList, cmd = m.serverList.Update(msg)
    case focusDetail:
        // detail panel update logic
    }
    return m, cmd
}
```

### Anti-Patterns to Avoid
- **Raw goroutines in Bubble Tea:** Never spawn goroutines directly. Always use `tea.Cmd` functions. Bubble Tea runs commands in isolated goroutines internally.
- **Shared mutable state across commands:** Commands execute concurrently. Never read/write shared model fields from within a command function. Pass data as function parameters, return results as messages.
- **Hardcoded terminal dimensions:** Always derive panel sizes from `tea.WindowSizeMsg`. Never assume 80x24 or any fixed size.
- **Single monolithic Update function:** As the TUI grows, route messages to child model Update functions. Keep the root Update thin -- global keys + routing only.
- **String-building in Update:** Do all rendering in `View()`, never in `Update()`. Update should only change state; View reads state.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Fuzzy-filtered server list | Custom list with filter loop | `bubbles/list` with `SetFilteringEnabled(true)` | Built-in fuzzy matching, pagination, status messages, keyboard navigation, delegated rendering |
| Text input for URI/URL | Raw rune handling | `bubbles/textinput` | Cursor movement, clipboard, character limits, placeholder, blink animation |
| Scrollable detail view | Manual scroll offset tracking | `bubbles/viewport` | Mouse wheel, keyboard scrolling, percentage tracking, resize handling |
| Loading spinners | Frame-based ASCII animation | `bubbles/spinner` | 12+ built-in styles, tick-based animation, lipgloss styling |
| Progress indication | Manual progress bar rendering | `bubbles/progress` | Gradient fills, smooth animation, auto-width, ViewAs for static rendering |
| Help overlay | Manual keybinding listing | `bubbles/help` + `bubbles/key` | Auto-generates from keyMap interface, short/full modes, responsive width |
| Terminal color detection | Manual ANSI escape sequences | `lipgloss.AdaptiveColor` + renderer | Auto-detects dark/light background, degrades across color profiles |
| Layout composition | Manual string padding/alignment | `lipgloss.JoinHorizontal/JoinVertical/Place` | Handles multi-line blocks, alignment, measurement |

**Key insight:** The Bubbles v2 library provides production-quality components for every UI primitive needed. Hand-rolling any of these would take weeks and miss edge cases that Bubbles handles (terminal resize, mouse events, accessibility, color degradation).

## Common Pitfalls

### Pitfall 1: Using v1 API Patterns with v2 Imports
**What goes wrong:** Code compiles against v1 patterns but v2 has breaking changes: `View()` returns `tea.View` not `string`, key messages are `tea.KeyPressMsg` not `tea.KeyMsg`, space bar is `"space"` not `" "`.
**Why it happens:** Most online tutorials and AI training data show v1 patterns. The v2 release is from February 2025.
**How to avoid:** Always reference v2-specific API: `func (m model) View() tea.View`, `case tea.KeyPressMsg:`, `case "space":`. Use `tea.NewView(s)` to wrap string content.
**Warning signs:** Compilation errors about "cannot use string as tea.View", "tea.KeyMsg undefined".

### Pitfall 2: Blocking the Event Loop with Synchronous I/O
**What goes wrong:** Calling `net.DialTimeout`, `subscription.Fetch`, or `engine.Start` directly in `Update()` freezes the UI.
**Why it happens:** Update runs on the main event loop thread. Any blocking call stalls rendering and input.
**How to avoid:** Wrap ALL I/O in `tea.Cmd` functions. Return commands from Update, never block.
**Warning signs:** UI freezes during operations, spinner stops animating, keys buffer up.

### Pitfall 3: Not Broadcasting WindowSizeMsg to All Children
**What goes wrong:** Child components retain stale dimensions after terminal resize, causing layout overflow or truncation.
**Why it happens:** Only the root model receives `WindowSizeMsg` from Bubble Tea. Children don't get it unless you forward it.
**How to avoid:** In root Update, always recalculate and call `SetSize()` on every child component when `WindowSizeMsg` arrives.
**Warning signs:** Layout breaks on resize, text wraps unexpectedly, panels overflow.

### Pitfall 4: Not Handling the "Not Ready" State
**What goes wrong:** View panics or renders garbage before the first `WindowSizeMsg` arrives.
**Why it happens:** Bubble Tea sends `WindowSizeMsg` after Init, but View may be called before it arrives.
**How to avoid:** Add a `ready bool` field. Return a loading placeholder from View until the first WindowSizeMsg sets dimensions.
**Warning signs:** Panic on first render, zero-width/height calculations.

### Pitfall 5: Forgetting to Set AltScreen on the View Struct
**What goes wrong:** TUI renders inline, mixing with shell output, scrollback pollution.
**Why it happens:** In v2, AltScreen is a field on `tea.View`, not a program option or command.
**How to avoid:** Always set `v.AltScreen = true` on the returned `tea.View` in your root model's `View()` method.
**Warning signs:** TUI appears at bottom of terminal, previous shell output visible above.

### Pitfall 6: Mutating Model State Inside Commands
**What goes wrong:** Race conditions, stale data, non-deterministic behavior.
**Why it happens:** Commands run concurrently in goroutines. If a command closure captures a model pointer, it races with the event loop.
**How to avoid:** Pass all needed data as value parameters to command functions. Return results as messages. Never close over model fields.
**Warning signs:** Intermittent data corruption, test flakes, "concurrent map read and map write" panics.

### Pitfall 7: Conflicting Keybindings Between List and Custom Keys
**What goes wrong:** Pressing `/` for custom action triggers list's built-in filter mode instead.
**Why it happens:** `bubbles/list` has built-in keybindings (/, esc, q) that conflict with app-level bindings.
**How to avoid:** Disable list bindings you want to override: `l.KeyMap.Filter.SetEnabled(false)` if redefining `/`. Or use the list's built-in filtering (recommended for this project since we want fuzzy search).
**Warning signs:** Keys do unexpected things, filter mode activates unintentionally.

## Code Examples

Verified patterns from official sources:

### Bubble Tea v2 Model Interface (complete skeleton)
```go
// Source: charm.land/bubbletea/v2 official tutorial
package tui

import (
    "fmt"
    tea "charm.land/bubbletea/v2"
    "charm.land/lipgloss/v2"
)

type model struct {
    width, height int
    ready         bool
    // ... child models
}

func (m model) Init() tea.Cmd {
    return nil // or tea.Batch(loadServers, tickUptime)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyPressMsg:     // NOTE: KeyPressMsg, not KeyMsg (v2)
        switch msg.String() {
        case "q", "ctrl+c":
            return m, tea.Quit
        }
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        m.ready = true
    }
    return m, nil
}

func (m model) View() tea.View {  // NOTE: returns tea.View, not string (v2)
    if !m.ready {
        return tea.NewView("Loading...")
    }
    content := fmt.Sprintf("Terminal: %dx%d", m.width, m.height)
    v := tea.NewView(content)
    v.AltScreen = true
    return v
}
```

### Split-Pane Layout with Lip Gloss
```go
// Source: charm.land/lipgloss/v2 layout documentation
func (m model) renderLayout() string {
    listWidth := m.width / 3
    detailWidth := m.width - listWidth - 1  // -1 for border

    listStyle := lipgloss.NewStyle().
        Width(listWidth).
        Height(m.height - statusBarHeight).
        BorderRight(true).
        BorderStyle(lipgloss.NormalBorder()).
        BorderForeground(theme.Border)

    detailStyle := lipgloss.NewStyle().
        Width(detailWidth).
        Height(m.height - statusBarHeight)

    listPanel := listStyle.Render(m.serverList.View())
    detailPanel := detailStyle.Render(m.detail.View())

    main := lipgloss.JoinHorizontal(lipgloss.Top, listPanel, detailPanel)

    statusBar := m.statusBar.View()

    return lipgloss.JoinVertical(lipgloss.Left, main, statusBar)
}
```

### Server List Item Implementation
```go
// Source: charm.land/bubbles/v2/list documentation
type serverItem struct {
    server protocol.Server
}

func (s serverItem) Title() string {
    return s.server.Name
}

func (s serverItem) Description() string {
    proto := string(s.server.Protocol)
    if s.server.LatencyMs > 0 {
        return fmt.Sprintf("%s | %dms", proto, s.server.LatencyMs)
    }
    return proto
}

func (s serverItem) FilterValue() string {
    // Enables fuzzy search by name, address, and protocol
    return s.server.Name + " " + s.server.Address + " " + string(s.server.Protocol)
}
```

### Concurrent Ping with Progress
```go
// Source: Bubble Tea commands pattern (charm.land/blog/commands-in-bubbletea)
type pingStartMsg struct{ total int }
type pingResultMsg struct {
    serverID  string
    latencyMs int
    err       error
}
type allPingsCompleteMsg struct{}

func pingAllServersCmd(servers []protocol.Server) tea.Cmd {
    cmds := make([]tea.Cmd, 0, len(servers)+1)
    for _, srv := range servers {
        srv := srv
        cmds = append(cmds, func() tea.Msg {
            start := time.Now()
            conn, err := net.DialTimeout("tcp",
                fmt.Sprintf("%s:%d", srv.Address, srv.Port),
                5*time.Second,
            )
            if err != nil {
                return pingResultMsg{serverID: srv.ID, err: err}
            }
            conn.Close()
            return pingResultMsg{
                serverID:  srv.ID,
                latencyMs: int(time.Since(start).Milliseconds()),
            }
        })
    }
    return tea.Batch(cmds...)
}

// In Update:
// case pingResultMsg: update server latency in store, increment progress counter
// When all received: sort list by latency, update list items
```

### Status Bar with Uptime Ticker
```go
// Source: Bubble Tea tick pattern
type tickMsg time.Time

func tickCmd() tea.Cmd {
    return tea.Tick(time.Second, func(t time.Time) tea.Msg {
        return tickMsg(t)
    })
}

type statusBarModel struct {
    connectionStatus engine.ConnectionStatus
    serverName       string
    socksPort        int
    connectedAt      time.Time
}

func (s statusBarModel) View() string {
    status := statusStyle.Render(s.connectionStatus.String())
    server := serverStyle.Render(s.serverName)
    port := portStyle.Render(fmt.Sprintf("SOCKS5:%d", s.socksPort))

    var uptime string
    if s.connectionStatus == engine.StatusConnected {
        d := time.Since(s.connectedAt).Truncate(time.Second)
        uptime = uptimeStyle.Render(d.String())
    }

    return statusBarStyle.Render(
        lipgloss.JoinHorizontal(lipgloss.Center, status, " | ", server, " | ", port, " | ", uptime),
    )
}
```

### Help Overlay with KeyMap
```go
// Source: charm.land/bubbles/v2/help + key documentation
type keyMap struct {
    Up          key.Binding
    Down        key.Binding
    Select      key.Binding
    Back        key.Binding
    Quit        key.Binding
    Help        key.Binding
    Filter      key.Binding
    PingAll     key.Binding
    AddServer   key.Binding
    AddSub      key.Binding
    RefreshSub  key.Binding
    Delete      key.Binding
    ClearAll    key.Binding
    Connect     key.Binding
}

func defaultKeyMap() keyMap {
    return keyMap{
        Up:         key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("j/k", "navigate")),
        Down:       key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("j/k", "navigate")),
        Select:     key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "connect")),
        Back:       key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
        Quit:       key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
        Help:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
        Filter:     key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
        PingAll:    key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "ping all")),
        AddServer:  key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add server")),
        AddSub:     key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "add subscription")),
        RefreshSub: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh subscriptions")),
        Delete:     key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
        ClearAll:   key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "clear all")),
        Connect:    key.NewBinding(key.WithKeys("enter", "c"), key.WithHelp("enter/c", "connect")),
    }
}

func (k keyMap) ShortHelp() []key.Binding {
    return []key.Binding{k.Help, k.Quit, k.Select, k.Filter}
}

func (k keyMap) FullHelp() [][]key.Binding {
    return [][]key.Binding{
        {k.Up, k.Down, k.Select, k.Back},
        {k.Filter, k.PingAll, k.Connect},
        {k.AddServer, k.AddSub, k.RefreshSub},
        {k.Delete, k.ClearAll, k.Quit, k.Help},
    }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `View() string` | `View() tea.View` | Bubble Tea v2.0.0 (Feb 2025) | Must use `tea.NewView(s)` wrapper; View struct enables declarative alt screen, mouse, cursor |
| `tea.KeyMsg` | `tea.KeyPressMsg` / `tea.KeyReleaseMsg` | Bubble Tea v2.0.0 (Feb 2025) | Type assertion must use `tea.KeyPressMsg`; key release events now available |
| `" "` for space | `"space"` for space | Bubble Tea v2.0.0 (Feb 2025) | Space bar string comparison changed |
| `tea.EnterAltScreen` command | `v.AltScreen = true` on View struct | Bubble Tea v2.0.0 (Feb 2025) | No more commands for terminal modes; set fields on `tea.View` |
| `tea.WithAltScreen()` program option | `v.AltScreen = true` on View struct | Bubble Tea v2.0.0 (Feb 2025) | Program options simplified; terminal modes are per-view |
| `github.com/charmbracelet/*` | `charm.land/*` | v2.0.0 (Feb 2025) | All imports must use `charm.land/` vanity domain for v2 |
| Exported fields on bubbles | Getter/setter methods | Bubbles v2.0.0 (Feb 2025) | Use `l.SetSize()` not `l.Width = x`; use functional options for constructors |
| `list.New(items, delegate, w, h)` | `list.New(items)` | Bubbles v2.0.0 (Feb 2025) | Constructor takes only items; use `SetSize()` and `SetDelegate()` separately |

**Deprecated/outdated:**
- `github.com/charmbracelet/bubbletea` (v1): Use `charm.land/bubbletea/v2` instead
- `github.com/charmbracelet/lipgloss` (v1): Use `charm.land/lipgloss/v2` instead
- `github.com/charmbracelet/bubbles` (v1): Use `charm.land/bubbles/v2` instead
- `tea.WithAltScreen()` program option: Set `v.AltScreen = true` on View struct instead
- `tea.EnterAltScreen` / `tea.ExitAltScreen` commands: Set field on View struct instead
- `tea.EnableMouseCellMotion()` command: Set `v.MouseMode` on View struct instead

## Open Questions

1. **bubbles/list v2 constructor signature**
   - What we know: v2 changed from `list.New(items, delegate, w, h)` to `list.New(items)` based on release notes mentioning functional options. The delegate is set separately.
   - What's unclear: The exact functional options API for list.New in v2.0.0 (Context7 data may show v1 patterns).
   - Recommendation: Run `go doc charm.land/bubbles/v2/list New` after installing to verify the constructor signature. Fall back to `list.New(items)` followed by `l.SetDelegate(d)` and `l.SetSize(w, h)`.

2. **tea.View interaction with bubbles component View() methods**
   - What we know: Root model returns `tea.View`. Child bubbles components (list, viewport) still return `string` from their `View()`.
   - What's unclear: Whether bubbles v2 components also return `tea.View` or still return `string`.
   - Recommendation: Only the root model needs to return `tea.View`. Child component `View()` strings are composed into the root's content string, then wrapped in `tea.NewView()`. Verify at implementation time.

3. **Lip Gloss v2 renderer integration with Bubble Tea v2**
   - What we know: Lip Gloss v2 makes renderers explicit. Bubble Tea v2 handles color profile detection.
   - What's unclear: Whether `lipgloss.NewStyle()` works standalone in v2 or requires a renderer from the Bubble Tea program.
   - Recommendation: Start with `lipgloss.NewStyle()` (default renderer). If color detection fails, investigate renderer integration. The v2 release notes suggest Bubble Tea handles this automatically.

## Sources

### Primary (HIGH confidence)
- Context7 `/charmbracelet/bubbletea` - Model interface, Update patterns, key handling, batch commands, program options, WindowSizeMsg
- Context7 `/charmbracelet/lipgloss` - Style creation, layout utilities (JoinHorizontal/JoinVertical/Place), adaptive colors, measurement
- Context7 `/charmbracelet/bubbles` - list component, textinput, viewport, spinner, progress, help, key bindings
- [pkg.go.dev/charm.land/bubbletea/v2](https://pkg.go.dev/charm.land/bubbletea/v2) - v2.0.0 API reference, types, interfaces
- [pkg.go.dev/charm.land/lipgloss/v2](https://pkg.go.dev/charm.land/lipgloss/v2) - v2.0.0 layout functions, style API
- [pkg.go.dev/charm.land/bubbles/v2](https://pkg.go.dev/charm.land/bubbles/v2) - v2.0.0 sub-packages listing

### Secondary (MEDIUM confidence)
- [Bubble Tea v2 Discussion #1374](https://github.com/charmbracelet/bubbletea/discussions/1374) - v2 migration guide, breaking changes, View struct details
- [Bubble Tea v2.0.0 Release](https://github.com/charmbracelet/bubbletea/releases/tag/v2.0.0) - Release notes, feature list, API changes
- [Lip Gloss v2 Discussion #506](https://github.com/charmbracelet/lipgloss/discussions/506) - v2 changes, renderer model
- [Commands in Bubbletea](https://charm.land/blog/commands-in-bubbletea/) - tea.Cmd patterns, tea.Batch, concurrency rules
- [Tips for building Bubble Tea programs](https://leg100.github.io/en/posts/building-bubbletea-programs/) - Root model composition, child routing, focus management

### Tertiary (LOW confidence)
- [bubblelayout](https://github.com/winder/bubblelayout) - Third-party declarative layout (not recommended for this project but shows patterns)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All three libraries confirmed at v2.0.0 stable via pkg.go.dev, release pages, and Context7
- Architecture: HIGH - Model-View-Update composition is well-documented; split-pane pattern verified across multiple sources
- Pitfalls: HIGH - v2 breaking changes documented in official migration guide; I/O-in-commands rule is core Bubble Tea doctrine
- Code examples: MEDIUM - v2 API verified for core patterns; some bubbles v2 constructor details may need verification at implementation time

**Research date:** 2026-02-25
**Valid until:** 2026-03-25 (stable libraries, 30-day validity)

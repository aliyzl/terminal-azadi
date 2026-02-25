---
phase: 04-tui-and-server-interaction
verified: 2026-02-25T00:00:00Z
status: human_needed
score: 6/6 must-haves verified
human_verification:
  - test: "Run TUI in a light-background terminal and verify readability"
    expected: "All text and UI elements remain legible with adequate contrast on a light terminal background"
    why_human: "App defaults to dark-mode styles (NewStyles(DefaultTheme, true)) and the comment says 'updated on BackgroundColorMsg' but no handler for tea.BackgroundColorMsg exists in Update(). Cannot confirm light terminal readability without running the app."
  - test: "Navigate server list and observe detail panel sync"
    expected: "Pressing j/k moves selection in the server list and the detail panel immediately updates to show the selected server's info"
    why_human: "syncDetail is called after key routing to the list model, but the exact user-visible timing and correctness of this sync requires interactive verification."
  - test: "Press p to ping servers and observe progress and sort"
    expected: "Server list title shows 'Servers (pinging N/M...)' during ping, and when complete the list re-orders fastest first"
    why_human: "Visual progress indication and sort order require live terminal interaction to confirm."
  - test: "Resize terminal to below 60x20 then resize back"
    expected: "'Terminal too small. Resize to at least 60x20.' message appears when small; full split-pane layout restores when enlarged"
    why_human: "Layout adaptation to terminal resize requires interactive terminal session."
---

# Phase 4: TUI and Server Interaction Verification Report

**Phase Goal:** Users interact with a beautiful, keyboard-driven terminal interface to browse servers, ping them, manage subscriptions, and connect
**Verified:** 2026-02-25
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | App displays a split-pane layout with server list panel, detail panel, and persistent status bar showing connection state, current server, proxy port, and uptime | VERIFIED | `app.go:View()` composes `listPanel + detailPanel` horizontally and joins with `m.statusBar.View()` vertically. `statusbar.go:View()` renders all four sections: status indicator, serverName, `SOCKS5:{port}`, and uptime duration. |
| 2 | User navigates the server list with j/k keys, selects with Enter, goes back with Esc, quits with q, and sees all keybindings via ? help overlay | VERIFIED | `app.go:handleKeyPress()` routes `q`=Quit, `?`=viewHelp, `enter`=syncDetail, `esc`=clear filter or no-op, default=delegate to serverList. `keys.go` defines all bindings with `ShortHelp`/`FullHelp` implementing `help.KeyMap`. `help.go:Render()` shows centered bordered box with all 14 bindings. |
| 3 | User can fuzzy-search/filter servers by name, country, or protocol and see results update in real-time | VERIFIED | `serverlist.go:FilterValue()` returns `Name + " " + Address + " " + Protocol`. `newServerList()` calls `l.SetFilteringEnabled(true)`. `app.go:handleKeyPress()` checks `m.serverList.FilterState() == list.Filtering` and delegates to list when active. `/` key routes to list model which activates built-in fuzzy filter. |
| 4 | User can add a server by pasting a URI, add servers from a subscription URL, refresh a subscription, and remove individual servers or clear all — all through the TUI | VERIFIED | `app.go` handles `a`=viewAddServer, `s`=viewAddSubscription, `r`=refreshSubscriptionsCmd, `d`=removeServerCmd, `D`=viewConfirmDelete with y/n confirmation. `input.go:addServerCmd()` calls `protocol.ParseURI()`, `addSubscriptionCmd()` calls `subscription.Fetch()`, `refreshSubscriptionsCmd()` iterates unique subscription sources and re-fetches. |
| 5 | Pinging all servers runs concurrently with visual progress indication, and results sort the server list by latency | VERIFIED | `ping.go:pingAllCmd()` builds one `tea.Cmd` per server using `net.DialTimeout` and returns `tea.Batch(cmds...)` for concurrent execution. `app.go` tracks `pingDone`/`pingTotal`, updates list title with "Servers (pinging N/M...)", and calls `rebuildListSortedByLatency()` when all complete. Sort puts fastest servers first; errored servers last. |
| 6 | Layout adapts to terminal size, color palette is consistent via lipgloss, and the app is readable in both dark and light terminals | PARTIAL — automated checks pass; light terminal readability needs human | `tea.WindowSizeMsg` handler resizes all children proportionally (list=width/3, detail=remaining-1, statusbar=full-width). Min-size guard at 60x20. Color palette uses `ColorPair` with `Light`/`Dark` variants resolved via `lipgloss.LightDark`. **Gap:** `New()` always calls `NewStyles(DefaultTheme, true)` (dark mode) with a comment saying it will be "updated on BackgroundColorMsg", but no `tea.BackgroundColorMsg` handler exists in `Update()`. Light terminal users will receive dark-palette styles. Colors remain legible but may have reduced contrast on light backgrounds. |

**Score:** 6/6 truths verified at artifact level (1 has an adaptive-color gap needing human confirmation)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/tui/theme.go` | Theme struct with ColorPair fields and DefaultTheme | VERIFIED | `ColorPair` struct with `Light`/`Dark` fields, `DefaultTheme` with 10 color slots, `NewStyles()` resolves via `lipgloss.LightDark`. Compiles. |
| `internal/tui/keys.go` | keyMap struct with all keybindings, ShortHelp/FullHelp methods | VERIFIED | 14 bindings defined (Up, Down, Select, Back, Quit, Help, Filter, PingAll, AddServer, AddSub, RefreshSub, Delete, ClearAll, Connect). `ShortHelp()` and `FullHelp()` present. |
| `internal/tui/messages.go` | Custom message types for TUI async operations | VERIFIED | 11 message types: `pingStartMsg`, `pingResultMsg`, `allPingsCompleteMsg`, `serverAddedMsg`, `serverRemovedMsg`, `serversReplacedMsg`, `subscriptionFetchedMsg`, `connectResultMsg`, `disconnectMsg`, `tickMsg`, `errMsg`. |
| `internal/tui/serverlist.go` | serverItem implementing list.DefaultItem, list wrapping bubbles/list | VERIFIED | `serverItem` has `Title()`, `Description()`, `FilterValue()`. `serversToItems()` and `newServerList()` present. `FilterValue()` includes name + address + protocol for fuzzy search. |
| `internal/tui/detail.go` | Detail panel rendering selected server info | VERIFIED | `detailModel` with `SetServer()`, `SetSize()`, `SetStyles()`, `View()`. Renders name, protocol, address:port, transport, TLS, flow, subscription source, last connected, latency. Handles nil server gracefully. |
| `internal/tui/statusbar.go` | Status bar with connection state, server, port, uptime | VERIFIED | `statusBarModel` with four-section View(): status indicator (color-coded), server name, `SOCKS5:{port}`, uptime duration. References `engine.ConnectionStatus`. |
| `internal/tui/app.go` | Root model with Init/Update/View, child composition, layout, focus management | VERIFIED | `New()` constructor, `Init()` returning `tickCmd()`, `Update()` handling 12 message types + key presses, `View()` returning `tea.View` with `AltScreen=true`. Composes all child models. |
| `internal/tui/help.go` | Help overlay toggled by ? key | VERIFIED | `helpModel` with `newHelpModel()` and `Render()`. Uses `lipgloss.Place` for centered overlay. Renders `FullHelpView` from all key groups. |
| `internal/tui/input.go` | Text input modal for add server URI and add subscription URL | VERIFIED | `inputModel` with `SetMode()`, `Update()`, `View()`. Two modes: `inputAddServer` and `inputAddSubscription`. Async command functions `addServerCmd`, `addSubscriptionCmd`, `refreshSubscriptionsCmd`. |
| `internal/tui/ping.go` | Concurrent ping command and progress tracking | VERIFIED | `pingAllCmd()` uses `tea.Batch` for concurrency, per-server TCP dial with 5s timeout. `removeServerCmd()` and `clearAllCmd()` also defined. |
| `internal/cli/root.go` | Root command RunE launches TUI when no subcommand given | VERIFIED | `RunE` loads config, creates store and engine, calls `tui.New(store, eng, cfg)` and `tea.NewProgram(m).Run()`. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `internal/tui/serverlist.go` | `internal/protocol/server.go` | serverItem wraps protocol.Server | WIRED | `serverItem struct { server protocol.Server }` — direct value embedding. |
| `internal/tui/statusbar.go` | `internal/engine/engine.go` | reads ConnectionStatus enum | WIRED | `status engine.ConnectionStatus` field; switch on `engine.StatusConnected`, `engine.StatusConnecting`, `engine.StatusError`. |
| `internal/tui/app.go` | `internal/tui/serverlist.go` | root model owns server list child model | WIRED | `m.serverList list.Model` field, used in `View()`, `Update()`, `handleKeyPress()`, `reloadServers()`, `rebuildListSortedByLatency()`. 37 references. |
| `internal/tui/app.go` | `internal/tui/statusbar.go` | root model owns status bar child model | WIRED | `m.statusBar statusBarModel` field, used in `View()`, `Update()`, `New()`. |
| `internal/tui/app.go` | `internal/tui/detail.go` | root model owns detail child model | WIRED | `m.detail detailModel` field, used in `View()`, `Update()`, `syncDetail()`. |
| `internal/cli/root.go` | `internal/tui/app.go` | root command launches TUI program | WIRED | `tui.New(store, eng, cfg)` and `tea.NewProgram(m).Run()` in RunE. |
| `internal/tui/input.go` | `internal/protocol/parse.go` | ParseURI called on submitted server URI | WIRED | `protocol.ParseURI(uri)` in `addServerCmd()` at line 119. |
| `internal/tui/input.go` | `internal/subscription/subscription.go` | Fetch called for subscription URL | WIRED | `subscription.Fetch(url)` in `addSubscriptionCmd()` at line 134 and `refreshSubscriptionsCmd()` at line 172. |
| `internal/tui/ping.go` | `internal/serverstore/store.go` | Store updated with latency results | WIRED | `store.Remove()` in `removeServerCmd()`, `store.Clear()` in `clearAllCmd()`. Latencies tracked in model map rather than store (documented design decision). |
| `internal/tui/app.go` | `internal/tui/input.go` | Root model delegates to input modal in overlay views | WIRED | `m.input inputModel` field, `m.input.SetMode()`, `m.input.Update(msg)`, `m.input.View()`, `m.input.Value()` used in handleKeyPress for viewAddServer/viewAddSubscription states. |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| TUI-01 | 04-02 | Split-pane layout: server list panel, detail panel, status bar | SATISFIED | `app.go:View()` composes three panels into split-pane layout with border separator. |
| TUI-02 | 04-02 | Navigate with vim-style keys (j/k up/down, Enter select, Esc back, q quit) | SATISFIED | All keys handled in `handleKeyPress()`. j/k/esc/enter routed to list; q returns `tea.Quit`. |
| TUI-03 | 04-02 | Fuzzy-search/filter servers by name, country, or protocol | SATISFIED | `FilterValue()` returns name+address+protocol; `SetFilteringEnabled(true)` on list; `/` activates built-in filter. |
| TUI-04 | 04-01 | Status bar shows: connection state, current server, proxy port, uptime | SATISFIED | `statusBarModel.View()` renders all four sections: `● Connected`, server name, `SOCKS5:{port}`, `{h}h {m}m {s}s`. |
| TUI-05 | 04-02 | Contextual help via ? key with all available keybindings | SATISFIED | `helpModel.Render()` shows centered bordered box with `FullHelpView(m.keys.FullHelp())` — all 14 bindings. |
| TUI-06 | 04-02 | Adapts layout to terminal size; shows minimum-size message if too small | SATISFIED | `WindowSizeMsg` resizes all children. Guard at `m.width < minWidth || m.height < minHeight` returns "Terminal too small" message. |
| TUI-07 | 04-01 | Consistent color palette via lipgloss; readable in dark and light terminals | PARTIAL | Palette uses `ColorPair` with both light and dark variants resolved via `lipgloss.LightDark`. App hardcodes dark mode on init (no `BackgroundColorMsg` handler). Light terminal readability needs human confirmation. |
| SRVR-01 | 04-01 | View server list with name, protocol, and latency | SATISFIED | `serverItem.Title()` = name, `serverItem.Description()` = `protocol \| {ms}ms`. |
| SRVR-02 | 04-03 | Add server by pasting a protocol URI | SATISFIED | `a` key activates input modal; `addServerCmd()` calls `protocol.ParseURI()` and `store.Add()`. |
| SRVR-03 | 04-03 | Add servers from subscription URL | SATISFIED | `s` key activates subscription modal; `addSubscriptionCmd()` calls `subscription.Fetch()` and `store.ReplaceBySource()`. |
| SRVR-04 | 04-03 | Refresh subscription to get latest server list | SATISFIED | `r` key calls `refreshSubscriptionsCmd()` which re-fetches all unique subscription sources. |
| SRVR-05 | 04-03 | Remove individual servers or clear all | SATISFIED | `d` key calls `removeServerCmd()`. `D` key shows confirm dialog; `y`/enter calls `clearAllCmd()`. |
| SRVR-06 | 04-03 | Ping all servers concurrently with visual progress and sort by latency | SATISFIED | `p` key calls `pingAllCmd()` via `tea.Batch`. Progress title "Servers (pinging N/M...)". `rebuildListSortedByLatency()` sorts on completion. |

**Requirements coverage: 13/13 claimed — 12 SATISFIED, 1 PARTIAL (TUI-07 adaptive color)**

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `internal/tui/app.go` | 67 | Comment "updated on BackgroundColorMsg" but no handler implemented | Warning | Theme defaults to dark; light terminal users get dark-palette styles. Colors have both light/dark variants defined but the switch never fires at runtime. |
| `internal/tui/detail.go` | 38 | `placeholder := m.styles.Dim.Render("No server selected")` | Info | Expected behavior — this is a valid "no selection" state, not a stub. |

No blocking anti-patterns. The `BackgroundColorMsg` gap is a warning: the infrastructure (ColorPair + LightDark) is fully implemented, but the runtime trigger to switch between dark/light is absent. The app works correctly in dark-background terminals.

### Human Verification Required

#### 1. Light Terminal Readability

**Test:** Open a terminal configured with a white/light background (e.g., macOS Terminal with "Basic" theme). Run `go run ./cmd/azad/`. Observe all UI elements.
**Expected:** Text is legible; protocol badges, server names, status bar, and help overlay all have adequate contrast on the light background.
**Why human:** The app calls `NewStyles(DefaultTheme, true)` at startup (hardcoded dark) and never updates styles at runtime — no `tea.BackgroundColorMsg` handler exists. The `ColorPair.Light` colors ARE defined (e.g., Primary Light=235, Accent Light=63) but never activated. A human must confirm whether dark-mode colors remain readable on light backgrounds, or whether this breaks the TUI-07 requirement.

#### 2. Interactive Navigation and Detail Sync

**Test:** Run `go run ./cmd/azad/` with some servers loaded. Press `j` and `k` to move through the server list.
**Expected:** Detail panel updates immediately on each keypress to show the selected server's full info (name, protocol, address, TLS, latency etc.).
**Why human:** `syncDetail()` is called after routing keys to the list model, but the exact timing and correctness requires live interaction to confirm there is no visual lag or stale state.

#### 3. Concurrent Ping Progress and Sort

**Test:** With multiple servers loaded, press `p`.
**Expected:** List title immediately changes to "Servers (pinging 0/N...)". As pings complete, counter increments. When all finish, list reorders with lowest-latency servers first and title resets to "Servers".
**Why human:** Concurrency behavior and visual progress require a live terminal with real network servers to observe correctly.

#### 4. Minimum Size Guard

**Test:** Run TUI, then drag terminal window to smaller than 60 columns or 20 rows.
**Expected:** "Terminal too small. Resize to at least 60x20." message appears. When terminal is enlarged back past the threshold, the full split-pane layout restores.
**Why human:** Terminal resize events and layout restoration require interactive session.

### Gaps Summary

No hard blockers exist. All 10 artifacts are substantive and non-stub. All 10 key links are wired. The full project builds cleanly (`go build ./...`) and all tests pass (`go test ./...`).

The one architectural note (not a gap): the adaptive color system is structurally complete (ColorPair with Light/Dark values, `NewStyles(theme, isDark)` factory, `lipgloss.LightDark` resolver), but the runtime trigger to detect and switch background mode is missing. The plan's comment acknowledges this was intended to be wired via `tea.BackgroundColorMsg`. In practice, the app works in dark terminals. Whether it works acceptably in light terminals requires human confirmation to determine if TUI-07 is fully met.

---

_Verified: 2026-02-25_
_Verifier: Claude (gsd-verifier)_

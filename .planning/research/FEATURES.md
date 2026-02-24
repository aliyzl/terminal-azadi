# Feature Landscape

**Domain:** Terminal VPN/proxy client with TUI (wrapping Xray-core)
**Researched:** 2026-02-24
**Confidence:** MEDIUM-HIGH (based on analysis of v2rayN, Clash Verge Rev, IRBox, sing-box clients, proton-tui, vortix, lazygit, btop, and k9s UX patterns)

## Table Stakes

Features users expect from a professional terminal proxy client. Missing any of these and users leave for v2rayN or Clash Verge Rev.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **Multi-protocol support (VLESS, VMess, Trojan, SS)** | Every competing client supports these. VLESS-only is a non-starter for public release. | Med | Xray-core handles the protocols; the work is parsing URI formats and generating correct configs for each |
| **Subscription URL management** | Standard distribution method for proxy servers. Users get a URL from their provider and expect the client to fetch/parse/update it. | Med | Must handle base64-encoded and plaintext lists, multiple subscription sources, and periodic refresh |
| **Server list with latency display** | v2rayN, Clash, IRBox all show ping times next to servers. Users need to see which servers are fast before connecting. | Med | Concurrent TCP ping (existing bash version is sequential -- a key pain point from CONCERNS.md) |
| **One-command connect** | Users expect `azad` to connect them with minimal friction. Quick connect to last-used or fastest server. | Low | Remember last server, auto-select fastest if no preference |
| **System proxy integration** | All desktop clients set system proxy. Users should not have to manually configure their OS. | Med | macOS networksetup exists; need Linux (gsettings/env vars) and Windows (registry) equivalents |
| **Keyboard-driven navigation** | This is a TUI -- vim-style j/k/h/l, Enter to confirm, Esc to cancel, ? for help. Lazygit and k9s set this expectation. | Low | Bubbletea keymap pattern with help.Model for discoverability |
| **Connection status display** | Users need to see: connected/disconnected, which server, how long, current IP. Not knowing your state is unacceptable. | Low | Status bar component always visible at top or bottom |
| **Graceful connect/disconnect lifecycle** | Clean proxy start, PID management, graceful shutdown, no orphan xray processes. Existing bash version has fragile PID handling. | Med | Context-based lifecycle with proper signal handling in Go |
| **Configuration persistence** | Settings survive between sessions: preferred servers, proxy ports, UI preferences. | Low | YAML or TOML config file in XDG-compliant location |
| **Import from clipboard/URI** | Paste a vless:// or vmess:// link and have it parsed immediately. v2rayN supports Ctrl+V import. | Low | Parse protocol URIs from clipboard, add to server list |
| **Help system / keybinding discovery** | Lazygit shows contextual help with ?. k9s has a full keybinding reference. Users must be able to discover what keys do what. | Low | Bubbles help.Model with ShortHelp() and FullHelp() |
| **Cross-platform support (macOS, Linux, Windows)** | Go cross-compiles trivially. Not supporting Linux is leaving the largest target audience behind. | Med | Platform-specific: system proxy, Xray binary download, shell integration. Design abstractions early. |
| **Colored, readable output** | btop, lazygit, k9s are all visually striking. A plain-text TUI in 2026 looks broken. | Low | Lipgloss styles, consistent color palette, adaptive to terminal capabilities |
| **Error handling with actionable messages** | "Connection failed" is useless. "Connection to server X timed out after 5s -- try a different server (press j/k to navigate)" is useful. | Med | Structured error types with user-facing messages and suggested actions |

## Differentiators

Features that make Azad stand out from the crowd. No existing tool does the "beautiful terminal proxy client" well -- this is a green field.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Live connection statistics (speed, data transferred, uptime)** | No terminal proxy client shows real-time bandwidth. Vortix does it for WireGuard; nobody does it for Xray. This is the "btop for your VPN" moment. | High | Requires reading Xray stats API or monitoring network interface traffic. Visual sparkline/graph display via lipgloss. |
| **Concurrent visual ping sweep** | Show all servers pinging simultaneously with a progress indicator, then sort by latency. Existing bash version does this sequentially (painful). IRBox and v2rayN do concurrent ping but without the visual drama of watching results arrive live. | Med | Goroutines + tea.Cmd for async results. Bubble table with live-updating latency column. |
| **Smart server selection** | Auto-pick fastest server, remember favorites, suggest based on history. Goes beyond "auto select" in other clients by learning preferences. | Med | Score = f(latency, last_used, user_preference, success_rate). Store per-server history. |
| **Split-pane TUI layout** | Lazygit-style multi-panel layout: server list on left, connection details on right, status bar at bottom. No proxy client has this level of TUI sophistication. | High | Bubbletea layout management with dynamic panel sizing and focus management. Handle terminal resize gracefully. |
| **Fuzzy search/filter for servers** | With dozens or hundreds of servers from subscriptions, users need to quickly filter by country, name, or protocol. Lazygit has filtering; k9s has / search. | Med | Bubbles textinput with fuzzy matching against server name, country, protocol type |
| **Theme support (dark/light/custom)** | btop has multiple themes. A themeable proxy client signals quality and maturity. | Med | Lipgloss adaptive colors with theme definitions. Start with dark theme (terminal default), add light and high-contrast. |
| **Auto-reconnect on drop** | Connection monitoring that detects proxy failure and reconnects automatically, trying next-fastest server if current fails. Proton VPN roadmap lists this; most terminal tools lack it. | High | Health check goroutine polling connection status. Backoff retry with server failover. |
| **Server grouping by country/provider** | Visual tree or grouped list showing servers organized by country (like proton-tui's tree view). Makes large server lists navigable. | Med | Collapsible groups in table component. Parse country from server name or geoip lookup. |
| **Export/share server config** | Copy a server's URI to clipboard, or display as QR code in terminal (for mobile client setup). v2rayN has this; no TUI client does. | Low | URI generation from stored config. Terminal QR code rendering via library. |
| **Toast notifications and event log** | Vortix has toast notifications for status changes. An event log panel showing "Connected to Tokyo-01 at 14:32" etc. provides audit trail. | Med | Notification queue rendered in a dedicated area. Scrollable log view accessible via hotkey. |
| **Profile/config switching** | Multiple saved configurations (e.g., "Work" with split routing, "Travel" with full tunnel). Switch between them instantly. | Med | Named profiles in config file. Each profile stores server preference, routing mode, ports. |
| **One-key quick actions** | Like vortix's 1-9 quick-slot connections. Pin favorite servers to number keys for instant connect. | Low | Number key bindings in server list for first 9 pinned servers |

## Anti-Features

Features to explicitly NOT build. These are traps that add complexity without value for the target audience.

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| **Built-in routing rules editor** | Xray routing (geoip, geosite, domain rules) is deeply complex. v2rayN has a whole routing configuration system. Building a TUI editor for this is a multi-month project that serves power users who already use v2rayN. | Provide sensible defaults (direct for private IPs, proxy everything else). Support loading custom Xray routing config from a JSON file for power users. |
| **DNS configuration UI** | Xray DNS settings (DoH, DoT, QUIC, domain strategies) are expert-level. A TUI form for this adds enormous complexity. | Use sane DNS defaults. Allow overriding via config file. |
| **TUN mode / transparent proxy** | TUN mode requires root/admin privileges, platform-specific tun device management, and complex routing table manipulation. Clash Verge and sing-box have this but it took years to stabilize. | Stick to SOCKS5 + HTTP proxy model. System proxy integration covers 95% of use cases. Document TUN as a future possibility. |
| **Custom protocol implementation** | Do not implement VLESS/VMess/Trojan directly. Xray-core is actively maintained by the XTLS team and handles all protocol details. | Always wrap Xray-core. Stay as a config generator and process manager. |
| **GUI/web interface** | v2rayA went web-based. That is a different product. Terminal-first means terminal-only. | Invest all UI effort in the TUI. The constraint is the differentiator. |
| **Mobile/embedded support** | Terminal apps do not run on phones. Do not try to support iOS/Android. | Build for macOS, Linux, Windows. That is the full scope. |
| **Multi-language / i18n** | v2rayN supports 7 languages. For an open-source terminal tool, English-first is fine. i18n adds string management overhead that slows development. | English only for v1. Accept community PRs for translations later if demand exists. |
| **Proxy chaining / cascading** | Running traffic through multiple proxies sequentially. Niche use case, significant complexity. | Single proxy hop. Power users can chain externally. |
| **Ad blocking / content filtering** | Some clients integrate ad-block rules via routing. This is scope creep for a connection client. | Users who want ad blocking can use it alongside a DNS-based blocker. |
| **Auto-update mechanism for Azad itself** | Self-updating binaries are a security concern and complex to implement correctly. | Distribute via go install, Homebrew, GitHub releases. Users update through their package manager. |
| **Server-side management** | Setting up V2Ray/Xray servers. This is a client-only tool. | Link to documentation for server setup. Stay in the client lane. |

## Feature Dependencies

```
Configuration persistence --> All other features (everything needs config to work)
Multi-protocol URI parsing --> Subscription management (subs contain mixed protocol links)
Multi-protocol URI parsing --> Import from clipboard
Subscription management --> Server list display
Server list display --> Concurrent ping sweep
Server list display --> Fuzzy search/filter
Server list display --> Server grouping
Concurrent ping sweep --> Smart server selection
Smart server selection --> One-command quick connect
Smart server selection --> Auto-reconnect
Xray process lifecycle --> Connection status display
Xray process lifecycle --> Live connection statistics
Connection status display --> Auto-reconnect
System proxy integration --> Connection status display (show proxy mode in status)
Help system --> Keyboard navigation (help shows available keys)
Split-pane layout --> All visual features (layout is the shell everything lives in)
Theme support --> Split-pane layout (themes apply to all visual components)
```

## MVP Recommendation

Prioritize these for first usable release, in dependency order:

1. **Configuration persistence** (YAML config, XDG paths) -- foundation for everything
2. **Multi-protocol URI parsing** (VLESS, VMess, Trojan, SS link formats) -- core data model
3. **Subscription management** (fetch, decode, parse, store, refresh) -- primary server source
4. **Split-pane TUI layout** (server list + detail panel + status bar) -- the visual shell
5. **Server list with concurrent ping** (display servers, ping all, sort by latency) -- primary interaction
6. **Xray process lifecycle** (start/stop/restart with proper cleanup) -- core function
7. **Connection status display** (connected state, current server, IP, uptime) -- essential feedback
8. **System proxy integration** (macOS first, Linux second) -- makes it actually useful
9. **Keyboard navigation + help system** (vim keys, ? help, contextual hints) -- discoverability
10. **One-command quick connect** (azad connect, auto-fastest) -- the magic moment

Defer to post-MVP:
- **Live connection statistics**: High complexity, requires Xray stats API research
- **Auto-reconnect**: Needs robust connection health detection first
- **Theme support**: Dark theme default is fine for launch
- **Server grouping**: Flat list with search is sufficient for MVP
- **Profile switching**: Single config is fine initially
- **Export/QR code**: Nice to have, not blocking
- **Toast notifications / event log**: Console-style feedback is sufficient initially

## Competitive Landscape Summary

| Capability | v2rayN | Clash Verge | IRBox | proton-tui | vortix | **Azad (target)** |
|------------|--------|-------------|-------|------------|--------|-------------------|
| Multi-protocol | Yes (8+) | Yes (Clash) | Yes (8) | WireGuard only | WG+OpenVPN | Yes (4 via Xray) |
| GUI/TUI | Desktop GUI | Desktop GUI | Desktop GUI | TUI (Rust) | TUI (Rust) | **TUI (Go)** |
| Live stats | Traffic counters | Minimal | No | No | Yes (excellent) | **Yes (target)** |
| Subscription mgmt | Yes (auto) | Yes (YAML) | Yes | No (ProtonVPN) | No | **Yes** |
| Concurrent ping | Yes | Yes (auto) | Yes | No | No | **Yes (visual)** |
| Server search/filter | Basic | Rule-based | No | Country/city | No | **Fuzzy search** |
| Routing rules | Full editor | Full YAML | Presets | N/A | N/A | **Sane defaults + file override** |
| Cross-platform | Win/Linux/Mac | Win/Linux/Mac | Windows | Linux | Linux/Mac | **Mac/Linux/Win** |
| QR code support | Yes | No | No | No | No | **Yes (terminal)** |
| Auto-reconnect | No | Failover groups | Auto-select | No | Kill switch | **Yes** |
| Open source | Yes | Yes | Yes | Yes | Yes | **Yes** |

**The gap Azad fills:** No existing tool combines multi-protocol Xray support with a beautiful, lazygit-quality TUI. proton-tui and vortix prove the terminal VPN TUI concept works, but they are locked to WireGuard/OpenVPN. v2rayN and Clash Verge have the protocol support but are desktop GUIs. Azad bridges these worlds.

## Sources

- [v2rayN features and architecture](https://deepwiki.com/2dust/v2rayN) -- MEDIUM confidence (DeepWiki analysis)
- [Clash Verge Rev official site](https://clash-verge.org/) -- MEDIUM confidence (official marketing)
- [IRBox GitHub](https://github.com/frank-vpl/IRBox) -- MEDIUM confidence (GitHub README)
- [proton-tui GitHub](https://github.com/cdump/proton-tui) -- HIGH confidence (direct source)
- [vortix GitHub](https://github.com/Harry-kp/vortix) -- HIGH confidence (direct source)
- [Xray-core routing documentation](https://xtls.github.io/en/config/routing.html) -- HIGH confidence (official docs)
- [Xray-core DNS documentation](https://xtls.github.io/en/config/dns.html) -- HIGH confidence (official docs)
- [lazygit GitHub and UX patterns](https://github.com/jesseduffield/lazygit) -- HIGH confidence (direct source)
- [Bubbletea best practices](https://leg100.github.io/en/posts/building-bubbletea-programs/) -- MEDIUM confidence (community expert)
- [Charmbracelet bubbles components](https://github.com/charmbracelet/bubbles) -- HIGH confidence (official source)
- [sing-box features](https://sing-box.sagernet.org/configuration/) -- HIGH confidence (official docs)
- [v2rayA web GUI client](https://github.com/v2rayA/v2rayA) -- HIGH confidence (direct source)

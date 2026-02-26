# Roadmap: Azad

## Overview

Azad is a full rewrite of a bash-based VLESS terminal tool into a professional Go TUI application wrapping Xray-core. The roadmap follows the natural data flow: foundation and config system first, then the protocol parsing data pipeline, then the Xray-core connection engine, then the full TUI experience with server management, then the "one command" quick connect promise, and finally cross-platform distribution. Each phase delivers a verifiable capability that the next phase builds on.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Foundation** - Go module, config system, CLI skeleton, signal handling, cleanup
- [x] **Phase 2: Protocol Parsing** - Multi-protocol URI parsers, subscription fetcher, server data store
- [x] **Phase 3: Connection Engine** - Xray-core proxy lifecycle, system proxy, connection verification
- [x] **Phase 4: TUI and Server Interaction** - Full interactive terminal UI with server list, ping, navigation, and server management
- [x] **Phase 5: Quick Connect** - Zero-argument TUI launch, headless connect, session memory
- [ ] **Phase 6: Distribution** - Cross-platform builds, geo asset management, recovery commands
- [ ] **Phase 7: Kill Switch** - Firewall-based traffic blocking, terminal close guard, crash-safe recovery

## Phase Details

### Phase 1: Foundation
**Goal**: A working Go binary with config persistence, CLI routing, and crash-safe lifecycle management
**Depends on**: Nothing (first phase)
**Requirements**: FNDN-01, FNDN-02, FNDN-03, FNDN-04
**Success Criteria** (what must be TRUE):
  1. Running `azad --help` shows available subcommands (connect, servers, config) with descriptions
  2. App reads and writes configuration to `~/.config/azad/config.yaml` (or XDG equivalent), preserving values between runs
  3. Sending SIGTERM or SIGINT to the running process exits cleanly without leaving orphaned state
  4. Running `azad --cleanup` removes any dirty proxy state from a previous crash, and `azad --reset-terminal` restores terminal to usable state
**Plans**: 2 plans

Plans:
- [x] 01-01-PLAN.md — Go module init, xray-core dependency, project structure, cobra CLI skeleton with subcommands
- [x] 01-02-PLAN.md — koanf config system with XDG paths, signal handling, cleanup and reset-terminal commands

### Phase 2: Protocol Parsing
**Goal**: All four protocol URIs parse correctly and subscriptions fetch into a persistent server store
**Depends on**: Phase 1
**Requirements**: PROT-01, PROT-02, PROT-03, PROT-04, PROT-05, PROT-06
**Success Criteria** (what must be TRUE):
  1. Pasting a vless://, vmess://, trojan://, or ss:// URI produces a valid server entry with correct name, address, port, and protocol-specific parameters
  2. Fetching a subscription URL decodes the response and extracts all server URIs regardless of base64/base64url encoding variations
  3. Server entries persist in JSON format with rich metadata (name, protocol, latency, last connected, subscription source) and survive app restarts
  4. Malformed URIs produce clear error messages identifying what went wrong, rather than silent failures or panics
**Plans**: 2 plans

Plans:
- [x] 02-01-PLAN.md — Server struct, 4 protocol parsers (VLESS/VMess/Trojan/SS), ParseURI dispatcher with TDD
- [x] 02-02-PLAN.md — Server JSON store with atomic writes, subscription fetcher with base64 decoding

### Phase 3: Connection Engine
**Goal**: The app can start and stop an Xray-core proxy, route traffic through it, and manage system proxy state safely
**Depends on**: Phase 2
**Requirements**: CONN-01, CONN-02, CONN-03, CONN-04, CONN-05, CONN-06
**Success Criteria** (what must be TRUE):
  1. App starts Xray-core proxy on configurable SOCKS5 and HTTP ports using a parsed server config, and traffic routed through those ports exits via the remote server
  2. App stops the proxy cleanly (instance.Close) with no orphaned Xray goroutines or leaked ports
  3. Connection status transitions (disconnected -> connecting -> connected -> error) are tracked and observable through the CLI
  4. After connecting, app verifies the external IP through the proxy and confirms it differs from the direct IP
  5. On macOS, app sets system proxy on connect and unsets on disconnect, and detects/cleans dirty proxy state on startup
**Plans**: 3 plans

Plans:
- [ ] 03-01-PLAN.md — Xray JSON config builder (TDD): Server struct to Xray JSON for all 4 protocols
- [ ] 03-02-PLAN.md — System proxy management: macOS networksetup detect/set/unset + cleanup upgrade
- [ ] 03-03-PLAN.md — Engine lifecycle + IP verify + CLI connect: Start/Stop state machine, IP check, connect command wiring

### Phase 4: TUI and Server Interaction
**Goal**: Users interact with a beautiful, keyboard-driven terminal interface to browse servers, ping them, manage subscriptions, and connect
**Depends on**: Phase 3
**Requirements**: TUI-01, TUI-02, TUI-03, TUI-04, TUI-05, TUI-06, TUI-07, SRVR-01, SRVR-02, SRVR-03, SRVR-04, SRVR-05, SRVR-06
**Success Criteria** (what must be TRUE):
  1. App displays a split-pane layout with server list panel, detail panel, and persistent status bar showing connection state, current server, proxy port, and uptime
  2. User navigates the server list with j/k keys, selects with Enter, goes back with Esc, quits with q, and sees all keybindings via ? help overlay
  3. User can fuzzy-search/filter servers by name, country, or protocol and see results update in real-time
  4. User can add a server by pasting a URI, add servers from a subscription URL, refresh a subscription, and remove individual servers or clear all -- all through the TUI
  5. Pinging all servers runs concurrently with visual progress indication, and results sort the server list by latency
  6. Layout adapts to terminal size, color palette is consistent via lipgloss, and the app is readable in both dark and light terminals
**Plans**: 4 plans

Plans:
- [x] 04-01-PLAN.md — TUI visual foundation: theme, keybindings, messages, server list panel, detail panel, status bar
- [x] 04-02-PLAN.md — Root model with split-pane layout, navigation, help overlay, CLI wiring
- [x] 04-03-PLAN.md — Server management actions: add server/subscription, refresh, remove, ping all with progress
- [x] 04-04-PLAN.md — Gap closure: fix paste routing in input modals (tea.PasteMsg + view-aware default fallthrough)

### Phase 5: Quick Connect
**Goal**: The "one command" promise -- launch azad with no arguments and be connected to the best server instantly
**Depends on**: Phase 4
**Requirements**: QCON-01, QCON-02, QCON-03
**Success Criteria** (what must be TRUE):
  1. Running `azad` with no arguments launches the TUI and auto-connects to the last-used server (or fastest if no history)
  2. Running `azad connect` connects headlessly without launching the TUI, printing status to stdout
  3. Server preference and last-used selection persist between sessions, so repeated launches connect to the user's preferred server without re-selection
**Plans**: 2 plans

Plans:
- [x] 05-01-PLAN.md — Server resolution with latency fallback + headless connect persistence
- [x] 05-02-PLAN.md — TUI connection lifecycle: auto-connect, Enter/c connect, quit cleanup

### Phase 6: Distribution
**Goal**: Users on macOS and Linux can install a single binary that handles first-run setup automatically
**Depends on**: Phase 5
**Requirements**: DIST-01, DIST-02, DIST-03
**Success Criteria** (what must be TRUE):
  1. App builds as a single binary for macOS (amd64, arm64) and Linux (amd64, arm64) via GoReleaser
  2. On first run, app auto-downloads geoip.dat and geosite.dat to the data directory without user intervention
  3. Recovery commands (--cleanup, --reset-terminal) work correctly on both macOS and Linux platforms
**Plans**: TBD

Plans:
- [ ] 06-01: TBD

### Phase 7: Kill Switch
**Goal**: When enabled, all non-VPN traffic is blocked at the firewall level — if VPN drops or terminal closes, nothing leaks. User can always recover via `azad` or `azad --cleanup`.
**Depends on**: Phase 5
**Requirements**: KILL-01, KILL-02, KILL-03, KILL-04, KILL-05
**Success Criteria** (what must be TRUE):
  1. With kill switch enabled, closing the terminal or crashing the app blocks ALL internet traffic (no leak)
  2. Running `azad` after a crash/close resumes the VPN session (or offers to reconnect), restoring internet through the VPN
  3. Running `azad --cleanup` disables the kill switch firewall rules and restores normal internet, even if VPN state is broken
  4. macOS shows a confirmation dialog when user tries to close the terminal window while kill switch is active
  5. Kill switch is a manual toggle (TUI keybinding or `azad connect --kill-switch`), not automatic
**Plans**: 2 plans

Plans:
- [x] 07-01-PLAN.md — killswitch package (pf rules, privilege escalation, enable/disable API) + ProxyState extension + cleanup upgrade
- [ ] 07-02-PLAN.md — Wire kill switch into CLI (--kill-switch flag), TUI (K keybinding toggle), startup recovery, status bar

## Progress

**Execution Order:**
Phases execute in numeric order: 1 -> 2 -> 3 -> 4 -> 5 -> 6

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Foundation | 2/2 | Complete    | 2026-02-24 |
| 2. Protocol Parsing | 2/2 | Complete    | 2026-02-24 |
| 3. Connection Engine | 3/3 | Complete    | 2026-02-25 |
| 4. TUI and Server Interaction | 4/4 | Complete | 2026-02-26 |
| 5. Quick Connect | 2/2 | Complete    | 2026-02-26 |
| 6. Distribution | 0/? | Not started | - |
| 7. Kill Switch | 1/2 | In progress | - |

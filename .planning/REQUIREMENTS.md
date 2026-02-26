# Requirements: Azad

**Defined:** 2026-02-24
**Core Value:** One command to connect to the fastest VPN server through a stunning terminal interface

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### Foundation

- [x] **FNDN-01**: App initializes Go module with xray-core as library dependency (not external binary)
- [x] **FNDN-02**: App reads/writes YAML config from XDG-compliant path (~/.config/azad/config.yaml)
- [x] **FNDN-03**: App provides cobra CLI with subcommands (connect, servers, config, --cleanup, --reset-terminal)
- [x] **FNDN-04**: App handles SIGTERM/SIGINT gracefully, cleaning up proxy and terminal state

### Protocol Support

- [x] **PROT-01**: App parses vless:// URIs into server configurations
- [x] **PROT-02**: App parses vmess:// URIs (base64-encoded JSON) into server configurations
- [x] **PROT-03**: App parses trojan:// URIs into server configurations
- [x] **PROT-04**: App parses ss:// (Shadowsocks) URIs into server configurations
- [x] **PROT-05**: App fetches subscription URLs, decodes base64/base64url content, and extracts all protocol URIs
- [x] **PROT-06**: App stores servers in JSON format with rich metadata (name, protocol, latency, last connected, subscription source)

### Connection

- [x] **CONN-01**: App starts Xray-core proxy via Go library API (core.New/instance.Start) on configurable SOCKS5 and HTTP ports
- [x] **CONN-02**: App stops Xray-core proxy cleanly via instance.Close()
- [x] **CONN-03**: App displays connection status (disconnected/connecting/connected/error) with current server name
- [x] **CONN-04**: App verifies connection works by checking external IP through the proxy
- [x] **CONN-05**: App sets/unsets macOS system proxy (SOCKS + HTTP) via networksetup or sysproxy
- [x] **CONN-06**: App detects and cleans up dirty proxy state on startup (previous crash left system proxy set)

### Server Management

- [x] **SRVR-01**: User can view server list with name, protocol, and latency
- [x] **SRVR-02**: User can add server by pasting a protocol URI
- [x] **SRVR-03**: User can add servers from subscription URL
- [x] **SRVR-04**: User can refresh subscription to get latest server list
- [x] **SRVR-05**: User can remove individual servers or clear all
- [x] **SRVR-06**: App pings all servers concurrently with visual progress and sorts by latency

### TUI

- [x] **TUI-01**: App displays a split-pane layout: server list panel, detail panel, status bar
- [x] **TUI-02**: User navigates with vim-style keys (j/k up/down, Enter select, Esc back, q quit)
- [x] **TUI-03**: User can fuzzy-search/filter servers by name, country, or protocol
- [x] **TUI-04**: Status bar shows: connection state, current server, proxy port, uptime
- [x] **TUI-05**: App shows contextual help via ? key with all available keybindings
- [x] **TUI-06**: App adapts layout to terminal size and shows minimum-size message if too small
- [x] **TUI-07**: App uses consistent color palette via lipgloss with readable output in both dark and light terminals

### Quick Connect

- [x] **QCON-01**: User can run `azad` with no arguments to launch TUI and connect to last-used or fastest server
- [x] **QCON-02**: User can run `azad connect` for headless quick-connect (no TUI, just connect and show status)
- [x] **QCON-03**: App remembers last-used server and user preferences between sessions

### Distribution

- [ ] **DIST-01**: App builds as single binary for macOS (amd64, arm64) and Linux (amd64, arm64)
- [ ] **DIST-02**: App auto-downloads geoip.dat and geosite.dat on first run if not present
- [ ] **DIST-03**: App provides --cleanup and --reset-terminal recovery commands

### Kill Switch

- [x] **KILL-01**: App blocks all non-VPN traffic via macOS packet filter (pfctl) when kill switch is enabled
- [x] **KILL-02**: Firewall rules persist if terminal closes or app crashes — no traffic leaks until user explicitly recovers
- [x] **KILL-03**: Running `azad` after crash/close resumes VPN or offers reconnect, restoring internet through VPN
- [x] **KILL-04**: Running `azad --cleanup` removes kill switch firewall rules and restores normal internet even if VPN state is broken
- [x] **KILL-05**: macOS shows confirmation dialog when user tries to close terminal while kill switch is active (process detection)

### Split Tunneling

- [x] **SPLT-01**: Inclusive mode routes only listed IPs/hostnames through VPN, everything else goes direct
- [x] **SPLT-02**: Exclusive mode routes all traffic through VPN except listed IPs/hostnames which go direct
- [x] **SPLT-03**: Rules support single IPs, CIDR ranges, hostnames, and wildcard domains (*.example.com)
- [ ] **SPLT-04**: User can add/remove rules, switch modes, and toggle split tunneling through TUI settings menu
- [x] **SPLT-05**: User can manage split tunnel rules via `azad split-tunnel` CLI subcommand (add/remove/list/mode/enable/disable/clear)
- [x] **SPLT-06**: Split tunneling coordinates with kill switch — bypass IPs allowed through pf firewall rules

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Advanced Features

- **ADV-01**: Live connection statistics (bandwidth, data transferred, latency graph)
- **ADV-02**: Auto-reconnect on connection drop with server failover
- **ADV-03**: Theme support (dark/light/custom color schemes)
- **ADV-04**: Server grouping by country or subscription provider
- **ADV-05**: Profile/config switching (Work, Travel, etc.)
- **ADV-06**: Export server as URI or terminal QR code
- **ADV-07**: Toast notifications and scrollable event log
- **ADV-08**: Windows support (system proxy, distribution)
- **ADV-09**: One-key quick slots (1-9 for pinned servers)

## Out of Scope

| Feature | Reason |
|---------|--------|
| Routing rules editor | Expert-level Xray config; provide sane defaults + file override for power users |
| DNS configuration UI | Expert-level; use sensible defaults, allow config file override |
| TUN mode / transparent proxy | Requires root, years of platform-specific work; SOCKS5+HTTP covers 95% of use cases |
| Custom protocol implementation | Xray-core handles all protocols; we are a config generator and lifecycle manager |
| GUI / web interface | Terminal-first is the differentiator; building a GUI is a different product |
| Mobile support | Terminal apps don't run on phones |
| Multi-language / i18n | English-first; accept community translations later |
| Proxy chaining | Niche use case; users can chain externally |
| Ad blocking / content filtering | Scope creep; use alongside DNS-based blocker |
| Self-update mechanism | Security concern; distribute via package managers |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| FNDN-01 | Phase 1 | Complete |
| FNDN-02 | Phase 1 | Complete |
| FNDN-03 | Phase 1 | Complete |
| FNDN-04 | Phase 1 | Complete |
| PROT-01 | Phase 2 | Complete |
| PROT-02 | Phase 2 | Complete |
| PROT-03 | Phase 2 | Complete |
| PROT-04 | Phase 2 | Complete |
| PROT-05 | Phase 2 | Complete |
| PROT-06 | Phase 2 | Complete |
| CONN-01 | Phase 3 | Complete |
| CONN-02 | Phase 3 | Complete |
| CONN-03 | Phase 3 | Complete |
| CONN-04 | Phase 3 | Complete |
| CONN-05 | Phase 3 | Complete |
| CONN-06 | Phase 3 | Complete |
| SRVR-01 | Phase 4 | Complete |
| SRVR-02 | Phase 4 | Complete |
| SRVR-03 | Phase 4 | Complete |
| SRVR-04 | Phase 4 | Complete |
| SRVR-05 | Phase 4 | Complete |
| SRVR-06 | Phase 4 | Complete |
| TUI-01 | Phase 4 | Complete |
| TUI-02 | Phase 4 | Complete |
| TUI-03 | Phase 4 | Complete |
| TUI-04 | Phase 4 | Complete |
| TUI-05 | Phase 4 | Complete |
| TUI-06 | Phase 4 | Complete |
| TUI-07 | Phase 4 | Complete |
| QCON-01 | Phase 5 | Complete |
| QCON-02 | Phase 5 | Complete |
| QCON-03 | Phase 5 | Complete |
| DIST-01 | Phase 6 | Pending |
| DIST-02 | Phase 6 | Pending |
| DIST-03 | Phase 6 | Pending |
| KILL-01 | Phase 7 | Complete |
| KILL-02 | Phase 7 | Complete |
| KILL-03 | Phase 7 | Complete |
| KILL-04 | Phase 7 | Complete |
| KILL-05 | Phase 7 | Complete |
| SPLT-01 | Phase 8 | Complete |
| SPLT-02 | Phase 8 | Complete |
| SPLT-03 | Phase 8 | Complete |
| SPLT-04 | Phase 8 | Pending |
| SPLT-05 | Phase 8 | Complete |
| SPLT-06 | Phase 8 | Complete |

**Coverage:**
- v1 requirements: 46 total
- Mapped to phases: 46
- Unmapped: 0

---
*Requirements defined: 2026-02-24*
*Last updated: 2026-02-26 after phase 8 planning*

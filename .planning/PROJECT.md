# Azad

## What This Is

Azad is a beautiful, professional terminal VPN client built in Go. It wraps Xray-core to provide multi-protocol support (VLESS, VMess, Trojan, Shadowsocks) through a gorgeous bubbletea/lipgloss TUI. Single binary, cross-platform, open source — designed to be the best terminal VPN app available.

## Core Value

One command to connect to the fastest VPN server through a stunning terminal interface that feels like a real application, not a script.

## Requirements

### Validated

- ✓ VLESS protocol support via Xray-core — existing
- ✓ Subscription URL fetching and parsing — existing
- ✓ Server list management (add/remove/clear) — existing
- ✓ TCP ping for server speed testing — existing
- ✓ Background/foreground proxy modes — existing
- ✓ System proxy toggle (macOS) — existing
- ✓ Connection verification (IP check) — existing
- ✓ Shell helper installation (proxy_on/proxy_off) — existing

### Active

- [ ] Full rewrite in Go with bubbletea/lipgloss TUI
- [ ] Multi-protocol support (VLESS, VMess, Trojan, Shadowsocks)
- [ ] Beautiful interactive terminal UI (like lazygit/btop)
- [ ] One-command quick connect with auto-fastest server
- [ ] Live connection stats (speed, latency, uptime, data transferred)
- [ ] Concurrent server ping with visual results
- [ ] Subscription management with auto-refresh
- [ ] Cross-platform support (macOS, Linux, Windows)
- [ ] Single binary distribution (embed or auto-download Xray-core)
- [ ] Keyboard-driven navigation (vim-style keys)
- [ ] Persistent configuration (YAML/TOML config file)
- [ ] Graceful error handling with helpful messages
- [ ] Auto-reconnect on connection drop
- [ ] Smart server selection (remember last, auto-pick fastest)

- [ ] Split tunneling with inclusive/exclusive modes (Windscribe-style)
  - SPLT-01: Inclusive mode — only listed IPs/hostnames route through VPN, rest goes direct
  - SPLT-02: Exclusive mode — all traffic through VPN except listed IPs/hostnames which go direct
  - SPLT-03: Rule types — single IPs, CIDR ranges (e.g. 10.0.0.0/8), hostnames, wildcard domains (*.example.com)
  - SPLT-04: TUI management — add/remove rules, switch modes, view active rules through the TUI
  - SPLT-05: CLI management — `azad split-tunnel` subcommand for headless rule management
  - SPLT-06: Kill switch coordination — split tunnel rules integrate cleanly with kill switch firewall rules

### Out of Scope

- GUI/desktop application — terminal-first, always
- Custom proxy protocol implementation — rely on Xray-core
- VPN server setup/provisioning — client only
- Mobile support — terminal app, desktop platforms only

## Context

This is a rewrite of an existing bash-based VLESS terminal tool. The current version works but feels amateur — printf menus, sequential pings, fragile string parsing, macOS-only. The codebase map in `.planning/codebase/` documents the current state.

The name "azad" means "free" in Persian — fitting for a VPN tool. The existing user base expects the `azad` command to keep working.

Xray-core is the proven proxy engine — we wrap it, not replace it. The Go ecosystem has excellent TUI libraries (bubbletea, lipgloss, bubbles) that can produce professional-grade terminal interfaces.

## Constraints

- **Proxy engine**: Xray-core (proven, multi-protocol, active development)
- **Language**: Go (bubbletea ecosystem, single binary, cross-compile)
- **TUI framework**: bubbletea + lipgloss + bubbles (charmbracelet stack)
- **Distribution**: Single binary via `go install`, Homebrew, GitHub releases
- **Compatibility**: Must parse existing vless:// and subscription URL formats

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Full rewrite in Go | Bash can't produce professional TUI; Go gives single binary + bubbletea | — Pending |
| Wrap Xray-core | Don't reinvent proxy protocols; Xray is proven and actively maintained | — Pending |
| Charmbracelet TUI stack | bubbletea/lipgloss/bubbles is the gold standard for Go terminal UIs | — Pending |
| Multi-protocol from v1 | Xray already supports them; limiting to VLESS only wastes capability | — Pending |
| Cross-platform from v1 | Go cross-compiles easily; design for it early, not as afterthought | — Pending |

---
*Last updated: 2026-02-24 after initialization*

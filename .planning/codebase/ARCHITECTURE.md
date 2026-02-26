# Architecture

## Overview

VLESS Terminal is a **single-layer CLI wrapper** around Xray-core that provides an interactive terminal-based VLESS VPN client for macOS. There is no client-server architecture, no API layer, and no persistent state beyond flat files.

## Pattern

**Script-per-concern** with a central menu orchestrator.

- `menu.sh` — Main entry point; interactive menu loop that dispatches to functions and child scripts
- `link-to-full-config.sh` — VLESS URI parser; generates Xray JSON config from a `vless://` link
- `vless-link-to-config.sh` — Standalone VLESS URI parser (prints outbound JSON only, used for reference)
- `run.sh` — Thin wrapper that launches `xray run -config config.json`
- `install.sh` — Downloads Xray-core binary from GitHub releases
- `setup-azad.sh` — One-time setup: installs Xray + adds shell aliases to `~/.zshrc`

## Layers

```
┌─────────────────────────────────────┐
│  User Terminal (interactive menu)   │  ← menu.sh
├─────────────────────────────────────┤
│  Config Generation                  │  ← link-to-full-config.sh
├─────────────────────────────────────┤
│  Xray-core (VLESS proxy binary)    │  ← ./xray (downloaded binary)
├─────────────────────────────────────┤
│  macOS Network Services             │  ← networksetup commands
└─────────────────────────────────────┘
```

## Data Flow

1. **Server input** → User pastes VLESS link or subscription URL
2. **Storage** → `data/servers.txt` (format: `name|vless://...` per line)
3. **Selection** → User picks server number → `link-to-full-config.sh` generates `config.json`
4. **Proxy execution** → `xray run -config config.json` opens SOCKS5 (1080) + HTTP (8080) on localhost
5. **System integration** → Optional macOS system proxy via `networksetup` commands

## Entry Points

- `menu.sh` — Primary entry point (interactive TUI menu, 18 options)
- `run.sh` — Direct proxy runner (non-interactive)
- `setup-azad.sh` — One-time bootstrap (installs + sets up shell)
- `install.sh` — Standalone Xray installer
- `link-to-full-config.sh` — Config generator (called by menu.sh and usable standalone)
- `vless-link-to-config.sh` — Standalone outbound JSON printer

## Key Abstractions

- **Server list** — Flat file `data/servers.txt` with `name|link` format; functions read/write sequentially
- **Config generation** — Shell-based VLESS URI parser builds JSON via heredoc/echo (no jq dependency)
- **Proxy lifecycle** — PID file (`data/proxy.pid`) tracks background xray process; `kill` for stop
- **Ping/connectivity** — TCP connect via `nc -z` with python3 timing; IP check via `curl` to external services
- **System proxy** — macOS `networksetup` commands to set/unset SOCKS + HTTP proxy on active network interface

## Error Handling

- `set -e` in helper scripts (install, setup, link-to-full-config)
- `menu.sh` does NOT use `set -e` (interactive loop must survive errors)
- Individual functions validate inputs (link format, file existence, server count) and print colored error messages
- No structured error codes; relies on printf with color-coded `msg_ok`/`msg_err`/`msg_info` helpers

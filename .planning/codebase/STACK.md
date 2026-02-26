# Technology Stack

**Analysis Date:** 2026-02-24

## Languages

**Primary:**
- Bash (shell scripting) - All interactive menus, server management, proxy control, and configuration parsing
- JSON - Configuration file format for Xray proxy settings

**Secondary:**
- Python3 - Used minimally for timestamp calculations in ping operations (`menu.sh` line 82, 85)

## Runtime

**Environment:**
- Xray-core v26.2.6 - Compiled binary proxy application for macOS (x86_64 and arm64 architectures)
- Bash shell - Script execution environment
- macOS (Darwin) - Primary target platform for shell scripts

**Package Manager:**
- No traditional package manager used
- Manual binary download from GitHub releases for Xray-core
- Shell-based installation via curl and unzip

## Frameworks

**Proxy & Networking:**
- Xray-core v26.2.6 - VLESS protocol implementation and proxy gateway
  - Provides SOCKS5 (port 1080) and HTTP proxy (port 8080) inbound listeners
  - Supports VLESS protocol with TLS, REALITY, WebSocket, and TCP transport

**Testing & Connectivity:**
- `nc` (netcat) - TCP connectivity testing for ping functionality (`menu.sh` lines 84, 452, 568)
- `curl` - HTTP requests for subscription fetching, IP detection, and connectivity testing

**Build/Dev:**
- No build framework - Uses pre-compiled Xray-core binary from XTLS GitHub releases

## Key Dependencies

**Critical:**
- Xray-core (`/Users/lee/vless-terminal/xray`) - Core proxy engine
  - Downloaded from: `https://github.com/XTLS/Xray-core/releases/download/v26.2.6/Xray-macos-{arch}.zip`
  - Provides VLESS protocol support with multiple stream security options

**Infrastructure:**
- curl - HTTP client for downloading Xray releases, fetching subscription URLs, IP lookups
- unzip - Archive extraction for Xray release packages
- base64 - Decoding subscription content (often base64-encoded)
- python3 - Time measurement for ping latency calculation
- networksetup (macOS) - System proxy configuration for browser/app-level VPN

**Data Files:**
- geoip.dat (20MB) - GeoIP database for geographic IP-based routing rules
- geosite.dat (10MB) - Domain/site database for domain-based routing rules

## Configuration

**Environment:**
- Shell configuration: `~/.zshrc` - Shell aliases and environment variables for `proxy_on`/`proxy_off`
- Runtime config: `config.json` - Xray proxy configuration (generated or user-provided)
- Defaults provided: `config.template.json` - Template for manual configuration

**Required configurations:**
- VLESS server details (UUID, address, port, protocol parameters)
- Subscription URLs for server lists (base64 or plaintext formatted)
- Network interface selection (Wi-Fi or Ethernet on macOS)

**Build:**
- No build configuration files (pre-built binary distribution)
- Install script: `install.sh` - Downloads and extracts Xray-core binary

## Platform Requirements

**Development:**
- macOS (tested on Darwin 24.6.0)
- Bash 4.0+ (for shell scripting features used)
- curl with HTTPS support
- unzip utility
- Python3 (for time operations in ping)
- netcat (nc) with connect timeout support
- networksetup utility (macOS-specific for system proxy)

**Production:**
- Deployment target: macOS (both Intel x86_64 and Apple Silicon arm64)
- Requires network connectivity for subscription updates and proxy operation
- Requires permission to modify system proxy settings (for options 15-16)

---

*Stack analysis: 2026-02-24*

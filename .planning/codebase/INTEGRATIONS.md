# External Integrations

**Analysis Date:** 2026-02-24

## APIs & External Services

**Xray GitHub Releases:**
- Service: GitHub Releases API / CDN
- What it's used for: Download Xray-core binaries for installation
  - SDK/Client: `curl` HTTP client
  - URL: `https://github.com/XTLS/Xray-core/releases/download/v26.2.6/Xray-macos-{arm64|64}.zip`
  - Trigger: `install.sh` during first-time setup (option 14 in menu)

**VPN Subscription Services:**
- Service: User-provided subscription URLs (via `add_subscription` function, `menu.sh` line 125-160)
- What it's used for: Fetch list of VLESS proxy servers
  - Client: `curl` with 15-second timeout
  - Content format: Base64-encoded or plaintext VLESS links
  - Processing: Decodes base64 if detected, parses `vless://` URLs
  - Storage: `data/servers.txt` (pipe-delimited name|link format)

**IP Detection Services:**
- Services queried (in order): `https://icanhazip.com`, `https://api.ipify.org`, `https://ifconfig.me/ip`
- What it's used for: Determine if VPN is active by comparing direct IP vs proxied IP
  - Client: `curl` with 8-10 second timeout
  - Used by: Test full tunnel (option 17), test connection (option 11)
  - Proxy support: Requests can be routed through SOCKS5 proxy at 127.0.0.1:1080

## Data Storage

**Databases:**
- Not applicable - No persistent database used

**File Storage:**
- Local filesystem only - All data stored in project directory:
  - `data/servers.txt` - List of VLESS proxy servers (pipe-delimited name|link)
  - `data/subscriptions.txt` - List of subscription URLs used
  - `data/current.txt` - Index of currently selected server
  - `data/last_sub_url.txt` - Last-used subscription URL (for refresh)
  - `data/proxy.pid` - Process ID of background Xray proxy process
  - `data/xray.log` - Xray process output log

**Caching:**
- No explicit caching layer - Subscriptions must be manually refreshed (option 3)

## Authentication & Identity

**Auth Provider:**
- Custom VLESS protocol with UUID
- Implementation approach:
  - User provides VLESS server link containing UUID (unique identifier)
  - UUID is parsed from link (`link-to-full-config.sh` line 21)
  - Stored in config.json outbounds.settings.vnext[0].users[0].id
  - Xray-core handles VLESS authentication with server
  - No centralized auth service - fully decentralized per-server authentication

**Encryption:**
- Optional encryption field in VLESS protocol (typically "none" for xtls/tls scenarios)
- TLS/REALITY security configured per server based on stream settings
- REALITY protocol support with public key, short ID, and fingerprint parameters

## Monitoring & Observability

**Error Tracking:**
- Not detected - No error tracking service integrated

**Logs:**
- Standard output logging:
  - Xray-core logs to `data/xray.log` (started with `nohup`, `menu.sh` line 324)
  - Log level configurable in config.json: `"log": { "loglevel": "warning" }`
  - Available levels: debug, info, warning, error
  - Menu script outputs colored status messages to terminal

## CI/CD & Deployment

**Hosting:**
- Xray-core hosted on GitHub: `https://github.com/XTLS/Xray-core`
- Releases available as compiled binaries for macOS (x86_64 and arm64)

**CI Pipeline:**
- Not applicable - Project is a client-side tool, not a service
- Installation is manual download of pre-compiled binary from GitHub releases

## Environment Configuration

**Required env vars:**
- None required for core functionality
- Optional shell environment variables after setup (option 13):
  - `all_proxy=socks5://127.0.0.1:1080` - Terminal proxy setting for curl/wget/git
  - `ALL_PROXY=socks5://127.0.0.1:1080` - Alternative uppercase variant

**Secrets location:**
- Secrets/credentials not stored in this project
- VLESS UUIDs passed via:
  - User input (option 1 - manual link entry)
  - Subscription URLs (option 2 - fetched from user-provided sources)
  - Environment variables or files outside this project scope
- System proxy credentials: None (SOCKS5 on localhost requires no auth)

## Webhooks & Callbacks

**Incoming:**
- None detected

**Outgoing:**
- None detected - Project is a pure VPN client with no outbound integrations

## Network Protocol Implementations

**Inbound (Xray):**
- SOCKS5 proxy at 127.0.0.1:1080 (UDP enabled)
- HTTP proxy at 127.0.0.1:8080
- Configuration: `config.json` inbounds array (lines 61-63)

**Outbound (Xray):**
- VLESS protocol over:
  - TCP (default)
  - WebSocket (ws)
  - TCP with XTLS Vision
  - TCP with REALITY (advanced)
- Security modes:
  - None (direct)
  - TLS (with SNI)
  - REALITY (anti-GFW with fingerprint spoofing)
- Routing:
  - Domain strategy: IPIfNonMatch
  - Rule: Direct connection for private IPs (`geoip:private`)

---

*Integration audit: 2026-02-24*

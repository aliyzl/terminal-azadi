# Phase 3: Connection Engine - Research

**Researched:** 2026-02-25
**Domain:** Xray-core Go library lifecycle, system proxy management, connection state machine
**Confidence:** HIGH

## Summary

Phase 3 implements the core proxy engine lifecycle: starting/stopping Xray-core as an embedded Go library, managing macOS system proxy state via `networksetup` commands, verifying connectivity through the proxy, and tracking connection state transitions. The existing codebase already provides the foundation: `protocol.Server` struct with all protocol fields, `config.Config` with SOCKS/HTTP port settings, `lifecycle.ProxyState` struct with crash-recovery state, and `lifecycle.RunCleanup` with placeholder comments for Phase 3 proxy reversal.

The recommended approach builds an Xray JSON config in memory from a `protocol.Server` struct, feeds it to `serial.LoadJSONConfig` to get a `*core.Config` protobuf, then calls `core.New` + `instance.Start` to launch the proxy. System proxy is managed directly via `os/exec` calls to macOS `networksetup` (no third-party library needed). Connection verification uses Go's `net/http` with a SOCKS5 proxy dialer to fetch external IP.

**Primary recommendation:** Use JSON-based config building with `infra/conf/serial.LoadJSONConfig` rather than constructing protobuf structs directly -- JSON mirrors the well-documented Xray config format, is easier to debug, and matches the existing bash implementation's config structure exactly.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| CONN-01 | Start Xray-core proxy via Go library API (core.New/instance.Start) on configurable SOCKS5 and HTTP ports | Xray-core Go API: `serial.LoadJSONConfig` -> `core.New` -> `instance.Start`. JSON config with SOCKS + HTTP inbounds and protocol-specific outbound. Config builder converts `protocol.Server` to JSON. |
| CONN-02 | Stop Xray-core proxy cleanly via instance.Close() | `instance.Close()` shuts down all features. Must nil the instance reference. ProxyState file removed on clean shutdown. |
| CONN-03 | Display connection status (disconnected/connecting/connected/error) with current server name | State machine with `ConnectionStatus` iota enum. Observable via method on engine struct. CLI prints transitions. |
| CONN-04 | Verify connection works by checking external IP through the proxy | HTTP GET to `https://icanhazip.com` via SOCKS5 proxy dialer (`golang.org/x/net/proxy`). Compare with direct IP. |
| CONN-05 | Set/unset macOS system proxy (SOCKS + HTTP) via networksetup | `networksetup -setsocksfirewallproxy`, `-setwebproxy`, `-setsecurewebproxy` commands via `os/exec`. Detect active network service. |
| CONN-06 | Detect and clean up dirty proxy state on startup (previous crash left system proxy set) | Existing `lifecycle.ProxyState` JSON + `.state.json` file. Phase 3 adds actual `networksetup` reversal commands to `RunCleanup`. |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/xtls/xray-core/core` | v1.260206.0 (already in go.mod) | Proxy instance lifecycle (New, Start, Close) | The proxy engine -- already a dependency |
| `github.com/xtls/xray-core/infra/conf` | (same module) | JSON-friendly config structs with Build() method | Converts JSON config to protobuf core.Config |
| `github.com/xtls/xray-core/infra/conf/serial` | (same module) | `LoadJSONConfig(io.Reader)` -> `*core.Config` | Official JSON config loading pipeline |
| `golang.org/x/net/proxy` | (stdlib extended) | SOCKS5 proxy dialer for IP verification | Standard Go library for SOCKS5 client connections |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `encoding/json` | stdlib | Build Xray JSON config in memory | Config generation from protocol.Server |
| `os/exec` | stdlib | Run `networksetup` commands for system proxy | macOS proxy set/unset/detect |
| `net/http` | stdlib | HTTP client for IP verification | Fetching external IP through proxy |
| `sync` | stdlib | Mutex for connection state | Thread-safe state transitions |
| `context` | stdlib | Cancellation for proxy lifecycle | Ties into existing lifecycle.WithShutdown |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| JSON config string -> LoadJSONConfig | Direct protobuf struct construction via `conf.Config{}.Build()` | Protobuf construction is more type-safe but much more verbose, harder to debug, and poorly documented. JSON approach mirrors existing bash config exactly. |
| Direct `networksetup` commands | `github.com/getlantern/sysproxy` library | sysproxy extracts embedded helper binaries, last updated ~2016, adds unnecessary complexity. `networksetup` is 6 lines of exec.Command. |
| `golang.org/x/net/proxy` for SOCKS5 | `net/http` with `http_proxy` env var | env var approach is global; proxy.SOCKS5 dialer is scoped to specific HTTP client |

**Installation:**
```bash
go get golang.org/x/net/proxy
```
Note: `xray-core` and its subpackages are already in `go.mod`. The `golang.org/x/net` module is already an indirect dependency via xray-core.

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── engine/              # NEW: Xray-core proxy lifecycle
│   ├── engine.go        # Engine struct, Start/Stop, state machine
│   ├── config.go        # Build Xray JSON config from protocol.Server
│   └── verify.go        # IP verification through proxy
├── sysproxy/            # NEW: macOS system proxy management
│   ├── sysproxy.go      # Set/Unset/Detect/Clean system proxy
│   └── detect.go        # Detect active network service
├── config/              # EXISTS
├── lifecycle/           # EXISTS (cleanup.go gets Phase 3 upgrade)
├── protocol/            # EXISTS
├── serverstore/         # EXISTS
├── subscription/        # EXISTS
└── cli/                 # EXISTS (connect.go gets Phase 3 implementation)
```

### Pattern 1: JSON Config Builder
**What:** Convert a `protocol.Server` struct into an Xray-compatible JSON config string, then load via `serial.LoadJSONConfig`.
**When to use:** Every time a connection is started.
**Example:**
```go
// Source: xray-core pkg.go.dev + existing bash link-to-full-config.sh pattern
import (
    "bytes"
    "encoding/json"
    "github.com/xtls/xray-core/infra/conf/serial"
)

// XrayConfig represents the JSON config structure matching Xray's format.
type XrayConfig struct {
    Log       LogConfig            `json:"log"`
    Inbounds  []InboundConfig      `json:"inbounds"`
    Outbounds []OutboundConfig     `json:"outbounds"`
    Routing   RoutingConfig        `json:"routing"`
}

func BuildConfig(srv protocol.Server, socksPort, httpPort int) (*core.Config, error) {
    cfg := XrayConfig{
        Log: LogConfig{LogLevel: "warning"},
        Inbounds: []InboundConfig{
            {Tag: "socks-in", Listen: "127.0.0.1", Port: socksPort, Protocol: "socks",
             Settings: json.RawMessage(`{"udp":true}`)},
            {Tag: "http-in", Listen: "127.0.0.1", Port: httpPort, Protocol: "http"},
        },
        Outbounds: buildOutbound(srv),
        Routing: defaultRouting(),
    }

    jsonBytes, err := json.Marshal(cfg)
    if err != nil {
        return nil, fmt.Errorf("marshaling xray config: %w", err)
    }

    return serial.LoadJSONConfig(bytes.NewReader(jsonBytes))
}
```

### Pattern 2: Engine Lifecycle with State Machine
**What:** A struct that owns the Xray instance, tracks connection state, and provides Start/Stop methods.
**When to use:** Central connection management.
**Example:**
```go
// Source: xray-core core package API + Go state machine patterns
type ConnectionStatus int

const (
    StatusDisconnected ConnectionStatus = iota
    StatusConnecting
    StatusConnected
    StatusError
)

type Engine struct {
    mu       sync.Mutex
    instance *core.Instance
    status   ConnectionStatus
    server   *protocol.Server
    err      error
}

func (e *Engine) Start(ctx context.Context, srv protocol.Server, socksPort, httpPort int) error {
    e.mu.Lock()
    defer e.mu.Unlock()

    e.setStatus(StatusConnecting)

    config, err := BuildConfig(srv, socksPort, httpPort)
    if err != nil {
        e.setStatus(StatusError)
        return fmt.Errorf("building config: %w", err)
    }

    instance, err := core.New(config)
    if err != nil {
        e.setStatus(StatusError)
        return fmt.Errorf("creating xray instance: %w", err)
    }

    if err := instance.Start(); err != nil {
        instance.Close()
        e.setStatus(StatusError)
        return fmt.Errorf("starting xray instance: %w", err)
    }

    e.instance = instance
    e.server = &srv
    e.setStatus(StatusConnected)
    return nil
}

func (e *Engine) Stop() error {
    e.mu.Lock()
    defer e.mu.Unlock()

    if e.instance != nil {
        err := e.instance.Close()
        e.instance = nil
        e.server = nil
        e.setStatus(StatusDisconnected)
        return err
    }
    return nil
}
```

### Pattern 3: Protocol-Specific Outbound Builder
**What:** Switch on `protocol.Server.Protocol` to build the correct outbound JSON structure.
**When to use:** Inside the config builder when constructing outbound config.
**Example:**
```go
func buildOutbound(srv protocol.Server) []OutboundConfig {
    var outbound OutboundConfig

    switch srv.Protocol {
    case protocol.ProtocolVLESS:
        outbound = buildVLESSOutbound(srv)
    case protocol.ProtocolVMess:
        outbound = buildVMessOutbound(srv)
    case protocol.ProtocolTrojan:
        outbound = buildTrojanOutbound(srv)
    case protocol.ProtocolShadowsocks:
        outbound = buildShadowsocksOutbound(srv)
    }

    return []OutboundConfig{
        outbound,
        {Tag: "direct", Protocol: "freedom"},
    }
}
```

### Pattern 4: System Proxy via networksetup
**What:** Use `os/exec` to call macOS `networksetup` for proxy management.
**When to use:** On connect (set) and disconnect (unset).
**Example:**
```go
// Source: macOS networksetup man page + existing menu.sh set_system_proxy/unset_system_proxy
func SetSystemProxy(service string, socksPort, httpPort int) error {
    cmds := [][]string{
        {"networksetup", "-setsocksfirewallproxy", service, "127.0.0.1", strconv.Itoa(socksPort)},
        {"networksetup", "-setsocksfirewallproxystate", service, "on"},
        {"networksetup", "-setwebproxy", service, "127.0.0.1", strconv.Itoa(httpPort)},
        {"networksetup", "-setsecurewebproxy", service, "127.0.0.1", strconv.Itoa(httpPort)},
        {"networksetup", "-setwebproxystate", service, "on"},
        {"networksetup", "-setsecurewebproxystate", service, "on"},
    }
    for _, args := range cmds {
        if err := exec.Command(args[0], args[1:]...).Run(); err != nil {
            return fmt.Errorf("running %v: %w", args, err)
        }
    }
    return nil
}

func UnsetSystemProxy(service string) error {
    cmds := [][]string{
        {"networksetup", "-setsocksfirewallproxystate", service, "off"},
        {"networksetup", "-setwebproxystate", service, "off"},
        {"networksetup", "-setsecurewebproxystate", service, "off"},
    }
    for _, args := range cmds {
        if err := exec.Command(args[0], args[1:]...).Run(); err != nil {
            return fmt.Errorf("running %v: %w", args, err)
        }
    }
    return nil
}
```

### Anti-Patterns to Avoid
- **Building protobuf configs directly:** The `conf.Config{}.Build()` approach requires constructing complex nested protobuf structs with `serial.ToTypedMessage()`. JSON approach is simpler, debuggable, and matches Xray documentation.
- **Global xray instance:** Xray docs say "at most one Server instance running" but don't enforce it. Always track the instance in a struct with proper mutex protection.
- **Forgetting to nil the instance after Close:** `instance.Close()` does not nil the reference. A closed instance cannot be restarted. Always set `e.instance = nil` after closing.
- **Setting system proxy without tracking state:** Always write ProxyState to `.state.json` BEFORE setting system proxy, so `--cleanup` can reverse it after a crash.
- **Hardcoding "Wi-Fi" as network service:** Must detect the active network service dynamically -- users may be on Ethernet, Thunderbolt, USB tethering, etc.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Xray JSON config parsing | Custom JSON->protobuf converter | `serial.LoadJSONConfig(reader)` | Handles all protocol validation, type registration, and protobuf conversion |
| SOCKS5 proxy dialing | Raw TCP connection through SOCKS5 | `golang.org/x/net/proxy.SOCKS5()` | SOCKS5 handshake has edge cases (auth, UDP, IPv6) |
| Protocol outbound JSON structures | Guessing JSON field names | Reference existing `link-to-full-config.sh` and Xray docs | Field names must match exactly what xray-core expects |
| Xray config format registration | Manual protobuf config building | Import `_ "github.com/xtls/xray-core/main/distro/all"` or register JSON loader | Required blank import to register all protocol handlers and config formats |

**Key insight:** The Xray JSON config format is the stable API. The Go protobuf types change between versions. Build JSON configs programmatically, let `serial.LoadJSONConfig` handle the protobuf conversion.

## Common Pitfalls

### Pitfall 1: JSON Config Format Not Registered
**What goes wrong:** `core.StartInstance("json", bytes)` or `serial.LoadJSONConfig` fails with "unknown config format" or protocol-not-found errors.
**Why it happens:** Xray-core only registers protobuf format by default. JSON support and protocol handlers require importing initialization packages.
**How to avoid:** Import the distro/all package or individual protocol packages with blank imports:
```go
import _ "github.com/xtls/xray-core/main/distro/all"
```
This registers JSON config loader, all protocols (VLESS, VMess, Trojan, Shadowsocks, freedom, blackhole), all transports (TCP, WS, gRPC, etc.), and all security (TLS, REALITY).
**Warning signs:** "unknown config format: json" error, or "unknown protocol: vless" during config build.

### Pitfall 2: Port Already in Use
**What goes wrong:** `instance.Start()` fails because SOCKS5 or HTTP port is already bound (previous instance not closed, or another process).
**Why it happens:** No port availability check before starting, or previous instance leaked.
**How to avoid:** Check port availability before starting. Always ensure previous instance is fully closed before starting a new one. Use the state file to detect stale state.
**Warning signs:** "bind: address already in use" error from Start().

### Pitfall 3: System Proxy Left Set After Crash
**What goes wrong:** User's entire system routes through a dead proxy after app crash, breaking all internet.
**Why it happens:** App set system proxy, then crashed/killed before unsetting.
**How to avoid:** Write ProxyState BEFORE setting proxy. On startup, check for dirty state and clean it. The existing `lifecycle.ProxyState` and `.state.json` pattern handles this -- Phase 3 adds actual networksetup reversal.
**Warning signs:** Internet stops working after the app crashes. `networksetup -getsocksfirewallproxy "Wi-Fi"` shows proxy still set.

### Pitfall 4: Network Service Detection Fails
**What goes wrong:** `networksetup` commands fail because the network service name is wrong or doesn't exist.
**Why it happens:** Hardcoding "Wi-Fi" when user is on Ethernet, or locale-dependent service names.
**How to avoid:** Detect active network service by iterating `networksetup -listallnetworkservices` and checking which has an active interface (matching the existing `get_network_service` pattern from menu.sh).
**Warning signs:** "networksetup: command failed" or proxy appears set but doesn't work.

### Pitfall 5: Xray Instance Can Only Start Once
**What goes wrong:** Attempting to reconnect after disconnect fails.
**Why it happens:** A `core.Instance` can be started only once. After `Close()`, it cannot be restarted.
**How to avoid:** Create a new `core.Instance` for each connection. Never try to call `Start()` on a previously closed instance.
**Warning signs:** "instance already started" or similar error on reconnect.

### Pitfall 6: GeoIP/GeoSite Files Not Found
**What goes wrong:** Routing rules referencing `geoip:private` fail because `geoip.dat`/`geosite.dat` are not in the expected location.
**Why it happens:** Xray-core looks for these files in specific locations (working directory, or configured path).
**How to avoid:** The project already has `geoip.dat` and `geosite.dat` in the root directory. Set the xray asset path environment variable: `os.Setenv("XRAY_LOCATION_ASSET", dataDir)` before building config, or ensure the working directory contains these files.
**Warning signs:** "failed to open file: geoip.dat" errors during config build.

## Code Examples

Verified patterns from official sources:

### Building Xray JSON Config for VLESS Outbound
```go
// Source: Existing link-to-full-config.sh + xray-core JSON docs
// VLESS outbound JSON structure
{
    "tag": "proxy",
    "protocol": "vless",
    "settings": {
        "vnext": [{
            "address": "server.example.com",
            "port": 443,
            "users": [{
                "id": "uuid-here",
                "encryption": "none",
                "flow": "xtls-rprx-vision"
            }]
        }]
    },
    "streamSettings": {
        "network": "tcp",
        "security": "reality",
        "realitySettings": {
            "serverName": "www.microsoft.com",
            "fingerprint": "chrome",
            "publicKey": "publickey",
            "shortId": "shortid"
        }
    }
}
```

### Building Xray JSON Config for VMess Outbound
```go
// Source: xray-core docs, Context7
{
    "tag": "proxy",
    "protocol": "vmess",
    "settings": {
        "vnext": [{
            "address": "server.example.com",
            "port": 443,
            "users": [{
                "id": "uuid-here",
                "alterId": 0,
                "security": "auto"
            }]
        }]
    },
    "streamSettings": {
        "network": "ws",
        "security": "tls",
        "wsSettings": {
            "path": "/vmess"
        },
        "tlsSettings": {
            "serverName": "example.com"
        }
    }
}
```

### Building Xray JSON Config for Trojan Outbound
```go
// Source: xray-core docs, Context7
{
    "tag": "proxy",
    "protocol": "trojan",
    "settings": {
        "servers": [{
            "address": "server.example.com",
            "port": 443,
            "password": "password-here"
        }]
    },
    "streamSettings": {
        "network": "tcp",
        "security": "tls",
        "tlsSettings": {
            "serverName": "server.example.com"
        }
    }
}
```

### Building Xray JSON Config for Shadowsocks Outbound
```go
// Source: xray-core docs, Context7
{
    "tag": "proxy",
    "protocol": "shadowsocks",
    "settings": {
        "servers": [{
            "address": "server.example.com",
            "port": 8388,
            "method": "aes-256-gcm",
            "password": "password-here"
        }]
    }
}
```

### Xray Instance Lifecycle in Go
```go
// Source: https://pkg.go.dev/github.com/xtls/xray-core/core (Context7)
import (
    "github.com/xtls/xray-core/core"
    _ "github.com/xtls/xray-core/main/distro/all" // Register all protocols + JSON loader
)

// Load config from JSON bytes
config, err := serial.LoadJSONConfig(bytes.NewReader(jsonBytes))
if err != nil {
    return fmt.Errorf("loading config: %w", err)
}

// Create instance (does not start yet)
instance, err := core.New(config)
if err != nil {
    return fmt.Errorf("creating instance: %w", err)
}

// Start proxy
if err := instance.Start(); err != nil {
    instance.Close() // Clean up on failure
    return fmt.Errorf("starting proxy: %w", err)
}

// ... proxy is running, SOCKS5/HTTP ports are listening ...

// Stop proxy
if err := instance.Close(); err != nil {
    log.Printf("error closing instance: %v", err)
}
```

### IP Verification Through SOCKS5 Proxy
```go
// Source: golang.org/x/net/proxy docs + Go net/http patterns
import (
    "net/http"
    "golang.org/x/net/proxy"
)

func VerifyIP(socksPort int) (proxyIP string, err error) {
    dialer, err := proxy.SOCKS5("tcp",
        fmt.Sprintf("127.0.0.1:%d", socksPort),
        nil, // no auth
        proxy.Direct,
    )
    if err != nil {
        return "", fmt.Errorf("creating SOCKS5 dialer: %w", err)
    }

    transport := &http.Transport{
        Dial: dialer.Dial,
    }
    client := &http.Client{
        Transport: transport,
        Timeout:   10 * time.Second,
    }

    resp, err := client.Get("https://icanhazip.com")
    if err != nil {
        return "", fmt.Errorf("fetching IP: %w", err)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }

    return strings.TrimSpace(string(body)), nil
}
```

### Detect Active Network Service (macOS)
```go
// Source: macOS networksetup man page + existing menu.sh get_network_service pattern
func DetectNetworkService() (string, error) {
    out, err := exec.Command("networksetup", "-listallnetworkservices").Output()
    if err != nil {
        return "", fmt.Errorf("listing network services: %w", err)
    }

    lines := strings.Split(string(out), "\n")
    // First pass: prefer Wi-Fi or Ethernet
    for _, line := range lines {
        svc := strings.TrimSpace(line)
        if svc == "" || strings.HasPrefix(svc, "An asterisk") || strings.Contains(svc, "*") {
            continue
        }
        if svc == "Wi-Fi" || strings.Contains(svc, "Ethernet") {
            return svc, nil
        }
    }
    // Second pass: return first non-disabled service
    for _, line := range lines {
        svc := strings.TrimSpace(line)
        if svc == "" || strings.HasPrefix(svc, "An asterisk") || strings.Contains(svc, "*") {
            continue
        }
        return svc, nil
    }
    return "", fmt.Errorf("no active network service found")
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Spawning xray binary as child process | Embedding xray-core as Go library via `core.New`/`instance.Start` | Xray-core has supported library mode since v1.0 | No external binary, no IPC, single process |
| protobuf-only config | JSON config via `serial.LoadJSONConfig` | Added in xray-core early versions | JSON is the standard config format, protobuf is internal |
| `getlantern/sysproxy` with embedded helpers | Direct `networksetup` calls via `os/exec` | N/A | sysproxy is unmaintained (2016), networksetup is stable macOS API |
| `v2ray-core` library | `xray-core` (XTLS fork) | 2020 | Active development, REALITY protocol, better performance |

**Deprecated/outdated:**
- `getlantern/sysproxy`: Last meaningful update ~2016, embeds platform-specific binaries, overly complex for macOS-only use case.
- `v2ray-core`: Xray-core is the actively maintained fork with REALITY and XTLS support.
- Building configs via protobuf structs directly: Fragile, poorly documented, changes between versions. JSON is the stable config interface.

## Open Questions

1. **Blank import size impact**
   - What we know: `_ "github.com/xtls/xray-core/main/distro/all"` registers all protocols, transports, and config formats.
   - What's unclear: Whether this import significantly increases binary size beyond what's already pulled in by the xray-core dependency.
   - Recommendation: Use the full distro/all import for simplicity. Binary size is already ~44MB (noted in STATE.md). Selective imports would save minimal size but add maintenance burden. This can be optimized in Phase 6 (Distribution) if needed.

2. **GeoIP/GeoSite asset path**
   - What we know: The project has `geoip.dat` and `geosite.dat` in the root directory. Xray-core looks for them via `XRAY_LOCATION_ASSET` env var or working directory.
   - What's unclear: Where these files will live in the installed binary distribution.
   - Recommendation: For Phase 3, set `XRAY_LOCATION_ASSET` to the data directory path (from `config.DataDir()`). Phase 6 handles auto-download and proper placement.

3. **macOS permissions for networksetup**
   - What we know: `networksetup` requires admin privileges on macOS. The existing bash tool runs without sudo.
   - What's unclear: Whether newer macOS versions (Ventura/Sonoma/Sequoia) restrict `networksetup` further.
   - Recommendation: Try without sudo first (existing bash tool works). If permission errors occur, document that user may need to grant Terminal full disk access or run with sudo. This matches behavior of other proxy tools.

## Sources

### Primary (HIGH confidence)
- [xray-core/core package - Go Packages](https://pkg.go.dev/github.com/xtls/xray-core/core) - Instance lifecycle API (New, Start, Close, StartInstance)
- [xray-core/infra/conf package - Go Packages](https://pkg.go.dev/github.com/xtls/xray-core/infra/conf) - JSON config struct types and Build() method
- [xray-core/infra/conf/serial package - Go Packages](https://pkg.go.dev/github.com/xtls/xray-core/infra/conf/serial) - LoadJSONConfig, DecodeJSONConfig functions
- Context7 /xtls/xray-core - Xray instance creation, config examples, protocol configurations
- Context7 /websites/xtls_github_io - VLESS, VMess, Trojan, Shadowsocks config structures, transport settings
- [macOS networksetup man page](https://ss64.com/mac/networksetup.html) - System proxy commands
- [golang.org/x/net/proxy package](https://pkg.go.dev/golang.org/x/net/proxy) - SOCKS5 proxy dialer
- Existing codebase: `menu.sh` (set_system_proxy, unset_system_proxy, get_network_service, test_full_tunnel, fetch_ip)
- Existing codebase: `link-to-full-config.sh` (JSON config structure for VLESS with REALITY/TLS/WS)
- Existing codebase: `lifecycle/cleanup.go` (ProxyState struct, RunCleanup placeholder)

### Secondary (MEDIUM confidence)
- [Set Mac OS X SOCKS proxy - GitHub Gist](https://gist.github.com/jordelver/3073101) - networksetup examples verified against man page
- [Go and Proxy Servers: Part 3 - SOCKS proxies](https://eli.thegreenplace.net/2022/go-and-proxy-servers-part-3-socks-proxies/) - SOCKS5 dialer pattern in Go
- [getlantern/sysproxy GitHub](https://github.com/getlantern/sysproxy) - Evaluated and rejected (unmaintained)

### Tertiary (LOW confidence)
- [State Machine Patterns in Go](https://www.codingexplorations.com/blog/state-machine-patterns-in-go) - General Go state machine patterns (well-known pattern, low risk)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - xray-core is already a dependency, API verified via Go package docs and Context7
- Architecture: HIGH - JSON config builder pattern proven by existing bash implementation, Go API verified
- Pitfalls: HIGH - Identified from official docs (instance lifecycle constraints) and existing bash tool (proxy cleanup pattern)

**Research date:** 2026-02-25
**Valid until:** 2026-03-25 (xray-core API is stable; networksetup is stable macOS API)

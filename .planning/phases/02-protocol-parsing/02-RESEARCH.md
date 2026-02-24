# Phase 2: Protocol Parsing - Research

**Researched:** 2026-02-25
**Domain:** Multi-protocol URI parsing (VLESS/VMess/Trojan/Shadowsocks), subscription fetching, server persistence
**Confidence:** HIGH

## Summary

Phase 2 transforms raw protocol URIs and subscription URLs into a persistent server store. The four protocols (VLESS, VMess, Trojan, Shadowsocks) each use distinct URI formats: VLESS/Trojan/Shadowsocks follow RFC 3986-style `scheme://userinfo@host:port?params#fragment`, while VMess uses `vmess://base64(JSON)`. Subscription URLs return base64-encoded text where each line is a protocol URI. Go's standard `net/url` package handles the RFC 3986 protocols directly; VMess requires base64-decode-then-JSON-unmarshal.

The critical output of this phase is not Xray-core JSON configs (that's Phase 3), but rather a normalized `Server` struct that captures all protocol-specific parameters alongside rich metadata (name, protocol type, latency, last connected timestamp, subscription source). These structs serialize to a `servers.json` file in the app's config directory. The existing `config.DataDir()` from Phase 1 already provides the storage path.

The biggest parsing pitfall is the lack of formal standardization -- each protocol's URI format evolved through community convention rather than an RFC. VMess in particular has no single standard (the "v2rayN format" with base64-encoded JSON is the de facto standard). Robust parsing must handle edge cases: missing fragments, URL-encoded characters, IPv6 addresses in brackets, padded/unpadded base64, and base64url vs standard base64 variants.

**Primary recommendation:** Build a `internal/protocol` package with per-protocol parser functions (`ParseVLESS`, `ParseVMess`, `ParseTrojan`, `ParseShadowsocks`) that each return a unified `Server` struct. Add `internal/subscription` for HTTP fetching and base64 decoding. Add `internal/serverstore` for JSON persistence with file locking. Use Go's `net/url.Parse` for VLESS/Trojan/SS, custom base64+JSON for VMess. No external parsing libraries needed -- stdlib is sufficient.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| PROT-01 | App parses vless:// URIs into server configurations | `net/url.Parse` handles `vless://uuid@host:port?params#name` directly. Query params: `type`, `security`, `flow`, `sni`, `fp`, `pbk`, `sid`, `spx`, `path`, `host`, `serviceName`, `alpn`, `allowInsecure`. Fragment = server name. |
| PROT-02 | App parses vmess:// URIs (base64-encoded JSON) into server configurations | Strip `vmess://` prefix, base64-decode remainder, JSON-unmarshal into struct with fields: `v`, `ps`, `add`, `port`, `id`, `aid`, `net`, `type`, `host`, `path`, `tls`, `sni`, `alpn`, `fp`. Must handle both padded and unpadded base64. |
| PROT-03 | App parses trojan:// URIs into server configurations | `net/url.Parse` handles `trojan://password@host:port?params#name`. Query params: `type`, `security`, `sni`, `fp`, `path`, `host`, `serviceName`, `alpn`, `flow`, `pbk`, `sid`, `spx`. Password from `url.User.Username()`. |
| PROT-04 | App parses ss:// URIs (Shadowsocks SIP002) into server configurations | SIP002 format: `ss://base64(method:password)@host:port#name` or `ss://method:percent-encoded-password@host:port#name` (AEAD-2022). Must detect base64 vs plaintext userinfo. Plugin params in query string. |
| PROT-05 | App fetches subscription URLs, decodes base64/base64url content, and extracts all protocol URIs | HTTP GET with User-Agent header. Response body is base64-encoded text. Decode (handling both standard and URL-safe base64, with/without padding), split by newlines, parse each line as protocol URI. Filter empty lines and unrecognized schemes. |
| PROT-06 | App stores servers in JSON format with rich metadata (name, protocol, latency, last connected, subscription source) | `Server` struct with JSON tags, `ServerStore` wrapping `[]Server` with `Load`/`Save` methods. File: `servers.json` in `config.DataDir()`. Atomic writes (write to temp, rename) for crash safety. `encoding/json` with `json.MarshalIndent`. |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| net/url (stdlib) | Go 1.25 | URI parsing for VLESS, Trojan, SS | Handles RFC 3986 URIs with custom schemes, extracts userinfo/host/port/query/fragment |
| encoding/base64 (stdlib) | Go 1.25 | Base64 decoding for VMess URIs and subscriptions | Supports both StdEncoding and URLEncoding, with/without padding |
| encoding/json (stdlib) | Go 1.25 | VMess JSON parsing + server store persistence | Standard Go JSON marshal/unmarshal with struct tags |
| net/http (stdlib) | Go 1.25 | Subscription URL fetching | Standard HTTP client with timeout support |
| os (stdlib) | Go 1.25 | File I/O for server store | Atomic write pattern with os.CreateTemp + os.Rename |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| sync (stdlib) | Go 1.25 | Mutex for server store concurrent access | When multiple goroutines read/write server list |
| strings (stdlib) | Go 1.25 | URI splitting, base64 padding normalization | Splitting subscription response by newlines |
| strconv (stdlib) | Go 1.25 | Port string to int conversion | Parsing port numbers from URI components |
| time (stdlib) | Go 1.25 | Timestamps for server metadata | LastConnected, subscription fetch timestamps |
| fmt (stdlib) | Go 1.25 | Error message formatting | Descriptive parse error messages |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Hand-written parsers | github.com/cnlangzi/proxyclient | Third-party lib does parsing + xray-core lifecycle; too heavy a dependency for just parsing, ties us to their abstractions |
| net/url for VMess | Custom URI parser | VMess uses base64(JSON) not RFC 3986; net/url is wrong tool for VMess specifically |
| encoding/json for store | github.com/knadh/koanf with JSON | koanf is designed for config not data stores; encoding/json is simpler and more appropriate for structured data arrays |
| os.Rename atomic write | Write directly to file | Direct write risks corruption on crash mid-write; atomic write is standard safety pattern |

**Installation:**
```bash
# No new dependencies needed -- Phase 2 uses only Go stdlib
# All required packages are already in the Go standard library
```

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── protocol/           # URI parsing (new in Phase 2)
│   ├── server.go       # Server struct, Protocol enum, shared types
│   ├── vless.go        # ParseVLESS(uri string) (*Server, error)
│   ├── vmess.go        # ParseVMess(uri string) (*Server, error)
│   ├── trojan.go       # ParseTrojan(uri string) (*Server, error)
│   ├── shadowsocks.go  # ParseShadowsocks(uri string) (*Server, error)
│   ├── parse.go        # ParseURI(uri string) (*Server, error) dispatcher
│   ├── parse_test.go   # Table-driven tests for all protocols
│   ├── vless_test.go   # VLESS-specific edge case tests
│   ├── vmess_test.go   # VMess-specific edge case tests
│   ├── trojan_test.go  # Trojan-specific edge case tests
│   └── shadowsocks_test.go  # SS-specific edge case tests
├── subscription/       # Subscription fetching (new in Phase 2)
│   ├── fetch.go        # Fetch(url string) ([]Server, error)
│   ├── decode.go       # Base64 decode utilities
│   └── fetch_test.go   # Tests with mock HTTP server
├── serverstore/        # Server persistence (new in Phase 2)
│   ├── store.go        # Store struct with Load/Save/Add/Remove/List
│   └── store_test.go   # Persistence round-trip tests
├── config/             # (existing from Phase 1)
├── cli/                # (existing from Phase 1)
└── lifecycle/          # (existing from Phase 1)
```

### Pattern 1: Unified Server Struct
**What:** A single `Server` struct captures all protocol-specific parameters as a flat structure with optional fields, rather than protocol-specific sub-structs.
**When to use:** When different protocols share many common fields (address, port, TLS settings, transport type) and the differences are sparse optional fields.
**Example:**
```go
// Source: Derived from xray-core config structures + proxyclient patterns
package protocol

import "time"

// Protocol represents a supported proxy protocol.
type Protocol string

const (
    ProtocolVLESS       Protocol = "vless"
    ProtocolVMess       Protocol = "vmess"
    ProtocolTrojan      Protocol = "trojan"
    ProtocolShadowsocks Protocol = "shadowsocks"
)

// Server represents a parsed proxy server with rich metadata.
type Server struct {
    // Identity
    ID       string   `json:"id"`        // Unique ID (generated UUID)
    Name     string   `json:"name"`      // Display name (from URI fragment)
    Protocol Protocol `json:"protocol"`

    // Connection
    Address string `json:"address"`
    Port    int    `json:"port"`

    // Authentication (protocol-dependent)
    UUID     string `json:"uuid,omitempty"`     // VLESS, VMess
    Password string `json:"password,omitempty"` // Trojan, Shadowsocks
    AlterID  int    `json:"alter_id,omitempty"` // VMess

    // Encryption
    Encryption string `json:"encryption,omitempty"` // VLESS: "none"; VMess: "auto"/"aes-128-gcm"/etc
    Method     string `json:"method,omitempty"`     // Shadowsocks cipher method
    Security   string `json:"security,omitempty"`   // VMess: "auto"/"aes-128-gcm"/etc

    // Transport
    Network     string `json:"network,omitempty"`      // tcp/ws/grpc/kcp/quic/httpupgrade/xhttp
    Path        string `json:"path,omitempty"`          // WebSocket path, HTTP path
    Host        string `json:"host,omitempty"`          // HTTP Host header
    ServiceName string `json:"service_name,omitempty"` // gRPC service name

    // TLS
    TLS         string `json:"tls,omitempty"`          // none/tls/reality
    SNI         string `json:"sni,omitempty"`           // Server Name Indication
    ALPN        string `json:"alpn,omitempty"`          // ALPN protocols (comma-separated)
    Fingerprint string `json:"fingerprint,omitempty"`  // uTLS fingerprint
    AllowInsecure bool `json:"allow_insecure,omitempty"`

    // REALITY-specific
    PublicKey string `json:"public_key,omitempty"` // REALITY public key
    ShortID   string `json:"short_id,omitempty"`   // REALITY short ID
    SpiderX   string `json:"spider_x,omitempty"`   // REALITY spider X

    // Flow
    Flow string `json:"flow,omitempty"` // XTLS flow: xtls-rprx-vision

    // VMess-specific
    Type string `json:"type,omitempty"` // VMess camouflage type (none/http/srtp/utp/wechat-video)

    // Metadata
    SubscriptionSource string    `json:"subscription_source,omitempty"` // URL this server came from
    LatencyMs          int       `json:"latency_ms,omitempty"`          // Last measured latency
    LastConnected      time.Time `json:"last_connected,omitempty"`      // Last successful connection
    AddedAt            time.Time `json:"added_at"`                      // When server was added
    RawURI             string    `json:"raw_uri,omitempty"`             // Original URI for re-parsing
}
```

### Pattern 2: Scheme-Dispatched Parser
**What:** A single `ParseURI` function detects the scheme prefix and dispatches to protocol-specific parsers.
**When to use:** When the caller has a raw URI string and wants a unified parsing interface.
**Example:**
```go
// Source: Standard Go pattern, similar to proxyclient/xray
package protocol

import (
    "fmt"
    "strings"
)

// ParseURI detects the protocol scheme and delegates to the appropriate parser.
func ParseURI(uri string) (*Server, error) {
    uri = strings.TrimSpace(uri)
    if uri == "" {
        return nil, fmt.Errorf("empty URI")
    }

    switch {
    case strings.HasPrefix(uri, "vless://"):
        return ParseVLESS(uri)
    case strings.HasPrefix(uri, "vmess://"):
        return ParseVMess(uri)
    case strings.HasPrefix(uri, "trojan://"):
        return ParseTrojan(uri)
    case strings.HasPrefix(uri, "ss://"):
        return ParseShadowsocks(uri)
    default:
        return nil, fmt.Errorf("unsupported protocol scheme in URI: %q", uri)
    }
}
```

### Pattern 3: Atomic File Store
**What:** Write to a temporary file then atomically rename, preventing corruption on crash.
**When to use:** Any persistent data file that must survive process crashes.
**Example:**
```go
// Source: Standard Go atomic write pattern
package serverstore

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "sync"
)

type Store struct {
    mu      sync.RWMutex
    servers []protocol.Server
    path    string
}

func (s *Store) Save() error {
    s.mu.RLock()
    defer s.mu.RUnlock()

    data, err := json.MarshalIndent(s.servers, "", "  ")
    if err != nil {
        return fmt.Errorf("marshaling servers: %w", err)
    }

    dir := filepath.Dir(s.path)
    tmp, err := os.CreateTemp(dir, "servers-*.json.tmp")
    if err != nil {
        return fmt.Errorf("creating temp file: %w", err)
    }
    defer os.Remove(tmp.Name()) // clean up on error

    if _, err := tmp.Write(data); err != nil {
        tmp.Close()
        return fmt.Errorf("writing temp file: %w", err)
    }
    if err := tmp.Close(); err != nil {
        return fmt.Errorf("closing temp file: %w", err)
    }

    if err := os.Rename(tmp.Name(), s.path); err != nil {
        return fmt.Errorf("renaming temp to store: %w", err)
    }
    return nil
}
```

### Pattern 4: Base64 Decode with Fallback
**What:** Try multiple base64 encodings to handle real-world variation in how subscriptions and VMess links are encoded.
**When to use:** Any base64 decoding of subscription content or VMess URIs.
**Example:**
```go
// Source: Common pattern in proxy client implementations
package subscription

import (
    "encoding/base64"
    "strings"
)

// decodeBase64 tries standard, URL-safe, and padded/unpadded variants.
func decodeBase64(s string) ([]byte, error) {
    // Normalize: trim whitespace
    s = strings.TrimSpace(s)

    // Try standard encoding (with padding)
    if decoded, err := base64.StdEncoding.DecodeString(s); err == nil {
        return decoded, nil
    }

    // Try standard encoding without padding
    if decoded, err := base64.RawStdEncoding.DecodeString(s); err == nil {
        return decoded, nil
    }

    // Try URL-safe encoding (with padding)
    if decoded, err := base64.URLEncoding.DecodeString(s); err == nil {
        return decoded, nil
    }

    // Try URL-safe encoding without padding
    if decoded, err := base64.RawURLEncoding.DecodeString(s); err == nil {
        return decoded, nil
    }

    return nil, fmt.Errorf("failed to decode base64: not valid in any encoding variant")
}
```

### Anti-Patterns to Avoid
- **Single base64 decoder:** Real subscriptions mix base64 variants. Always try multiple encodings with fallback.
- **Panicking on malformed input:** URIs from subscriptions are untrusted input. Every parser MUST return `error`, never panic.
- **Shared mutable state without locking:** The server store is read/written from multiple goroutines (fetch, UI, save). Use `sync.RWMutex`.
- **Storing Xray JSON configs instead of parsed structs:** Phase 2 stores the parsed server metadata. Phase 3 generates Xray JSON configs from the Server struct at connection time. Storing Xray configs directly couples the store format to Xray-core's config schema.
- **Validating parsed URIs by connecting:** Parsing and validation are separate from connection testing. A valid parse means all required fields are present and well-formed, not that the server is reachable.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| RFC 3986 URI parsing | Custom regex parser | `net/url.Parse()` | Handles escaping, IPv6, userinfo, query params correctly; battle-tested |
| Base64 encoding variants | Custom base64 decoder | `encoding/base64` with 4 encoding variants | Standard library handles padding, URL-safe, etc. |
| JSON serialization | Custom file format | `encoding/json` with struct tags | Struct tags give precise control; MarshalIndent for readability |
| HTTP client with timeout | Raw TCP connection | `net/http.Client{Timeout: time.Duration}` | Handles redirects, TLS, connection pooling |
| UUID generation for server IDs | Custom random string | `crypto/rand` + UUID format | Collision-resistant unique identifiers |
| URL percent-encoding | Manual character replacement | `url.PathUnescape()` / `url.QueryUnescape()` | Correct handling of all percent-encoded sequences |

**Key insight:** Protocol URI parsing is 90% standard URL parsing that Go's stdlib handles well. The remaining 10% (VMess base64+JSON, SS SIP002 userinfo detection, base64 fallback) is custom logic that's straightforward once the edge cases are mapped.

## Common Pitfalls

### Pitfall 1: VMess Base64 Padding Inconsistency
**What goes wrong:** VMess links from different clients use inconsistent base64 padding. Some include `=` padding, some don't. Some use standard base64, others use URL-safe.
**Why it happens:** No formal VMess URI standard exists. The "v2rayN format" became de facto but implementations vary.
**How to avoid:** Use the base64 fallback chain: try StdEncoding, then RawStdEncoding, then URLEncoding, then RawURLEncoding. Accept the first successful decode.
**Warning signs:** Base64 decode errors on valid-looking VMess links.

### Pitfall 2: VMess JSON Field Types Are Strings Not Numbers
**What goes wrong:** The VMess JSON `port` and `aid` (alterId) fields are often encoded as strings ("443") not numbers (443), causing `json.Unmarshal` to fail when the target struct uses `int`.
**Why it happens:** JavaScript-originated format where numbers and strings are interchangeable.
**How to avoid:** Use a custom type (like `proxyclient.Int`) that implements `json.Unmarshaler` to handle both string and number representations, or use `json.Number` / unmarshal to `interface{}` and type-assert.
**Warning signs:** Parse failures on `port` or `aid` fields from specific subscription providers.

### Pitfall 3: Shadowsocks SIP002 Dual Userinfo Encoding
**What goes wrong:** SIP002 allows two formats for userinfo: base64-encoded `method:password` (legacy/AEAD) or plain `method:password` with percent-encoding (AEAD-2022). Parsing only one format breaks the other.
**Why it happens:** SIP002 evolved to accommodate AEAD-2022 ciphers which contain base64 characters in passwords, making base64-wrapping ambiguous.
**How to avoid:** Check if userinfo contains a colon -- if yes, it's plaintext `method:password`. If no colon, try base64-decoding it first, then check for colon in the result.
**Warning signs:** Shadowsocks connections fail with "unknown method" errors because method and password were not correctly split.

### Pitfall 4: Subscription Response Encoding Issues
**What goes wrong:** Some subscription providers return content with BOM markers, mixed line endings (`\r\n` vs `\n`), trailing whitespace, or double-encoded base64.
**Why it happens:** No formal subscription format standard. Different panel software generates different output.
**How to avoid:** Normalize the subscription response before parsing: strip BOM, normalize line endings to `\n`, trim whitespace from each line, skip empty lines. If first decode produces more base64-looking content, try decoding again.
**Warning signs:** Parse errors on subscription content that looks valid, or servers list is unexpectedly empty.

### Pitfall 5: IPv6 Address Handling in URIs
**What goes wrong:** IPv6 addresses in URIs must be wrapped in brackets (`[::1]`), but the parsed hostname from `url.Parse` strips the brackets. If you reconstruct the URI or use the hostname directly, you may lose the port delimiter.
**Why it happens:** IPv6 and port use the same `:` delimiter; brackets disambiguate in URIs but aren't part of the address itself.
**How to avoid:** Use `url.Hostname()` (strips brackets) for the address and `url.Port()` for the port. Never split `url.Host` manually with `:`.
**Warning signs:** Parse failures or wrong port numbers when processing IPv6 servers.

### Pitfall 6: Missing Fragment Means Empty Server Name
**What goes wrong:** URIs without a `#fragment` produce servers with empty names, making them indistinguishable in the server list.
**Why it happens:** The fragment (remark) is technically optional in all protocol URIs.
**How to avoid:** Fall back to `address:port` as the server name when the fragment is empty. For VMess, use the `ps` (remark) field from the JSON.
**Warning signs:** Server list shows blank names or multiple identical unnamed entries.

## Code Examples

Verified patterns from official sources and community implementations:

### VLESS URI Parsing
```go
// Source: Derived from Xray docs + proxyclient/xray patterns
func ParseVLESS(uri string) (*Server, error) {
    u, err := url.Parse(uri)
    if err != nil {
        return nil, fmt.Errorf("invalid VLESS URI: %w", err)
    }
    if u.Scheme != "vless" {
        return nil, fmt.Errorf("expected vless:// scheme, got %s://", u.Scheme)
    }

    uuid := u.User.Username()
    if uuid == "" {
        return nil, fmt.Errorf("VLESS URI missing UUID")
    }

    host := u.Hostname()
    portStr := u.Port()
    if host == "" || portStr == "" {
        return nil, fmt.Errorf("VLESS URI missing host or port")
    }
    port, err := strconv.Atoi(portStr)
    if err != nil {
        return nil, fmt.Errorf("VLESS URI invalid port %q: %w", portStr, err)
    }

    q := u.Query()
    name := u.Fragment
    if name == "" {
        name = fmt.Sprintf("%s:%d", host, port)
    }

    return &Server{
        Name:        name,
        Protocol:    ProtocolVLESS,
        Address:     host,
        Port:        port,
        UUID:        uuid,
        Encryption:  q.Get("encryption"),
        Flow:        q.Get("flow"),
        Network:     defaultString(q.Get("type"), "tcp"),
        TLS:         defaultString(q.Get("security"), "none"),
        SNI:         q.Get("sni"),
        Fingerprint: q.Get("fp"),
        PublicKey:   q.Get("pbk"),
        ShortID:     q.Get("sid"),
        SpiderX:     q.Get("spx"),
        Path:        q.Get("path"),
        Host:        q.Get("host"),
        ServiceName: q.Get("serviceName"),
        ALPN:        q.Get("alpn"),
        RawURI:      uri,
    }, nil
}
```

### VMess URI Parsing
```go
// Source: Derived from v2rayN format + proxyclient/xray patterns
type vmessJSON struct {
    V    jsonFlexInt `json:"v"`
    PS   string     `json:"ps"`   // Server name/remark
    Add  string     `json:"add"`  // Address
    Port jsonFlexInt `json:"port"`
    ID   string     `json:"id"`   // UUID
    Aid  jsonFlexInt `json:"aid"`  // AlterID
    Net  string     `json:"net"`  // Network (tcp/ws/grpc/etc)
    Type string     `json:"type"` // Camouflage type
    Host string     `json:"host"`
    Path string     `json:"path"`
    TLS  string     `json:"tls"`
    SNI  string     `json:"sni"`
    ALPN string     `json:"alpn"`
    Fp   string     `json:"fp"`   // Fingerprint
}

// jsonFlexInt handles both string "443" and number 443 in JSON.
type jsonFlexInt int

func (f *jsonFlexInt) UnmarshalJSON(data []byte) error {
    // Try number first
    var n int
    if err := json.Unmarshal(data, &n); err == nil {
        *f = jsonFlexInt(n)
        return nil
    }
    // Try string
    var s string
    if err := json.Unmarshal(data, &s); err == nil {
        n, err := strconv.Atoi(s)
        if err != nil {
            return fmt.Errorf("cannot convert %q to int: %w", s, err)
        }
        *f = jsonFlexInt(n)
        return nil
    }
    return fmt.Errorf("jsonFlexInt: cannot unmarshal %s", string(data))
}

func ParseVMess(uri string) (*Server, error) {
    raw := strings.TrimPrefix(uri, "vmess://")
    data, err := decodeBase64(raw)
    if err != nil {
        return nil, fmt.Errorf("VMess URI base64 decode failed: %w", err)
    }

    var v vmessJSON
    if err := json.Unmarshal(data, &v); err != nil {
        return nil, fmt.Errorf("VMess URI JSON parse failed: %w", err)
    }

    if v.Add == "" {
        return nil, fmt.Errorf("VMess URI missing server address")
    }
    port := int(v.Port)
    if port == 0 {
        return nil, fmt.Errorf("VMess URI missing or zero port")
    }

    name := v.PS
    if name == "" {
        name = fmt.Sprintf("%s:%d", v.Add, port)
    }

    return &Server{
        Name:        name,
        Protocol:    ProtocolVMess,
        Address:     v.Add,
        Port:        port,
        UUID:        v.ID,
        AlterID:     int(v.Aid),
        Security:    "auto",
        Network:     defaultString(v.Net, "tcp"),
        Type:        v.Type,
        Host:        v.Host,
        Path:        v.Path,
        TLS:         v.TLS,
        SNI:         v.SNI,
        ALPN:        v.ALPN,
        Fingerprint: v.Fp,
        RawURI:      uri,
    }, nil
}
```

### Trojan URI Parsing
```go
// Source: Derived from trojan-go URL spec + Xray trojan outbound docs
func ParseTrojan(uri string) (*Server, error) {
    u, err := url.Parse(uri)
    if err != nil {
        return nil, fmt.Errorf("invalid Trojan URI: %w", err)
    }
    if u.Scheme != "trojan" {
        return nil, fmt.Errorf("expected trojan:// scheme, got %s://", u.Scheme)
    }

    password := u.User.Username()
    if password == "" {
        return nil, fmt.Errorf("Trojan URI missing password")
    }

    host := u.Hostname()
    portStr := u.Port()
    if host == "" {
        return nil, fmt.Errorf("Trojan URI missing host")
    }
    port := 443 // default
    if portStr != "" {
        port, err = strconv.Atoi(portStr)
        if err != nil {
            return nil, fmt.Errorf("Trojan URI invalid port: %w", err)
        }
    }

    q := u.Query()
    name := u.Fragment
    if name == "" {
        name = fmt.Sprintf("%s:%d", host, port)
    }

    return &Server{
        Name:        name,
        Protocol:    ProtocolTrojan,
        Address:     host,
        Port:        port,
        Password:    password,
        Flow:        q.Get("flow"),
        Network:     defaultString(q.Get("type"), "tcp"),
        TLS:         defaultString(q.Get("security"), "tls"),
        SNI:         q.Get("sni"),
        Fingerprint: q.Get("fp"),
        Path:        q.Get("path"),
        Host:        q.Get("host"),
        ServiceName: q.Get("serviceName"),
        ALPN:        q.Get("alpn"),
        PublicKey:   q.Get("pbk"),
        ShortID:     q.Get("sid"),
        SpiderX:     q.Get("spx"),
        RawURI:      uri,
    }, nil
}
```

### Shadowsocks (SIP002) URI Parsing
```go
// Source: Derived from SIP002 spec (shadowsocks.org/doc/sip002.html)
func ParseShadowsocks(uri string) (*Server, error) {
    u, err := url.Parse(uri)
    if err != nil {
        return nil, fmt.Errorf("invalid Shadowsocks URI: %w", err)
    }
    if u.Scheme != "ss" {
        return nil, fmt.Errorf("expected ss:// scheme, got %s://", u.Scheme)
    }

    host := u.Hostname()
    portStr := u.Port()
    if host == "" || portStr == "" {
        return nil, fmt.Errorf("Shadowsocks URI missing host or port")
    }
    port, err := strconv.Atoi(portStr)
    if err != nil {
        return nil, fmt.Errorf("Shadowsocks URI invalid port: %w", err)
    }

    // Detect userinfo format: base64 or plaintext
    var method, password string
    userinfo := u.User.String()
    if strings.Contains(userinfo, ":") {
        // Plaintext format: method:password (AEAD-2022 or percent-encoded)
        method = u.User.Username()
        password, _ = u.User.Password()
    } else {
        // Base64-encoded format: base64(method:password)
        decoded, err := decodeBase64(userinfo)
        if err != nil {
            return nil, fmt.Errorf("Shadowsocks URI userinfo decode failed: %w", err)
        }
        parts := strings.SplitN(string(decoded), ":", 2)
        if len(parts) != 2 {
            return nil, fmt.Errorf("Shadowsocks URI: decoded userinfo missing method:password")
        }
        method = parts[0]
        password = parts[1]
    }

    name := u.Fragment
    if name == "" {
        name = fmt.Sprintf("%s:%d", host, port)
    }

    return &Server{
        Name:     name,
        Protocol: ProtocolShadowsocks,
        Address:  host,
        Port:     port,
        Method:   method,
        Password: password,
        RawURI:   uri,
    }, nil
}
```

### Subscription Fetching
```go
// Source: V2RayN subscription format convention
func Fetch(subscriptionURL string) ([]*protocol.Server, error) {
    client := &http.Client{Timeout: 30 * time.Second}
    req, err := http.NewRequest("GET", subscriptionURL, nil)
    if err != nil {
        return nil, fmt.Errorf("creating request: %w", err)
    }
    req.Header.Set("User-Agent", "Azad/1.0")

    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("fetching subscription: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("subscription returned HTTP %d", resp.StatusCode)
    }

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("reading subscription body: %w", err)
    }

    // Decode base64 (the entire body is base64-encoded)
    decoded, err := decodeBase64(string(body))
    if err != nil {
        return nil, fmt.Errorf("decoding subscription body: %w", err)
    }

    // Split into individual URIs
    lines := strings.Split(string(decoded), "\n")
    var servers []*protocol.Server
    for _, line := range lines {
        line = strings.TrimSpace(line)
        if line == "" {
            continue
        }
        server, err := protocol.ParseURI(line)
        if err != nil {
            // Log warning but continue parsing other lines
            continue
        }
        server.SubscriptionSource = subscriptionURL
        servers = append(servers, server)
    }

    if len(servers) == 0 {
        return nil, fmt.Errorf("subscription contained no valid server URIs")
    }

    return servers, nil
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| VMess with alterId > 0 | VMess with alterId = 0 (AEAD) | V2Ray 4.28.1 (2020) | alterId=0 is default; non-zero is legacy |
| Shadowsocks stream ciphers | Shadowsocks AEAD + AEAD-2022 (2022-blake3-*) | SS 2022 spec | AEAD-2022 ciphers use plain userinfo in SIP002, not base64 |
| `tcp` network name | `raw` network name in Xray v24.9.30+ | Sept 2024 | Xray renamed "tcp" to "raw" internally; share links still use "type=tcp" |
| No REALITY | REALITY as TLS replacement | Xray v1.8.0 (2023) | New security type with pbk/sid/spx parameters in share links |
| VMess-only subscriptions | Multi-protocol subscriptions | ~2021 | Subscriptions now contain mixed vless/vmess/trojan/ss URIs |

**Deprecated/outdated:**
- VMess alterId > 0: Legacy, creates unnecessary overhead. Default to 0.
- Shadowsocks stream ciphers (rc4-md5, aes-256-cfb, etc.): Insecure, removed from modern implementations. Focus on AEAD and AEAD-2022 methods.
- `type=tcp` in Xray config: Internally renamed to `raw` in Xray v24.9.30+, but share links still use `tcp`. Parser should accept both and normalize.

## Open Questions

1. **Subscription metadata headers**
   - What we know: Some VLESS subscriptions include headers like `profile-title`, `profile-update-interval`, `subscription-userinfo` before the base64 body
   - What's unclear: Whether these headers are in the HTTP response headers or in the decoded body; how widespread this practice is
   - Recommendation: Parse HTTP response headers for metadata but don't rely on them. The base64 body is the primary data source.

2. **SSR (ShadowsocksR) support**
   - What we know: Some subscriptions include `ssr://` links alongside `ss://` links. SSR is a fork of Shadowsocks with obfuscation plugins.
   - What's unclear: Whether xray-core supports SSR natively
   - Recommendation: Out of scope for Phase 2. Skip `ssr://` links during subscription parsing with a log warning. Requirements specify `ss://` (Shadowsocks) only.

3. **Server deduplication**
   - What we know: Subscriptions may contain duplicate servers (same address:port:uuid). Refreshing a subscription may re-add existing servers.
   - What's unclear: Best deduplication strategy (by address+port? by full config? by raw URI?)
   - Recommendation: Deduplicate by `address + port + protocol + uuid/password` tuple. When refreshing a subscription, replace all servers from that subscription source rather than appending.

## Sources

### Primary (HIGH confidence)
- Xray-core official documentation (xtls.github.io) - VLESS, VMess, Trojan, Shadowsocks outbound configs, StreamSettings, REALITY
- SIP002 URI Scheme specification (shadowsocks.org/doc/sip002.html) - Shadowsocks URI format
- Trojan-Go URL scheme draft (azadzadeh.github.io/trojan-go/en/developer/url/) - Trojan URI format
- Go standard library documentation (pkg.go.dev/net/url, encoding/base64, encoding/json) - Parsing and serialization APIs
- Context7 /websites/xtls_github_io - Xray outbound config structures, stream settings

### Secondary (MEDIUM confidence)
- proxyclient/xray Go package (pkg.go.dev/github.com/cnlangzi/proxyclient/xray) - Reference implementation of protocol parsing in Go
- v2rayN subscription format (liolok.com/v2ray-subscription-parse/) - De facto subscription format standard
- VMess URI format (github.com/v2ray/v2ray-core/issues/1487, boypt/vmess2json) - VMess share link JSON structure
- Xray Checker subscription docs (xray-checker.kutovoy.dev/configuration/subscription/) - Subscription format variations

### Tertiary (LOW confidence)
- Individual proxy config repos (v2ray-configs, v2ray-free) - Real-world URI examples but not authoritative specifications
- docsbot.ai AI-generated prompts - URI breakdown examples, useful for field enumeration but not authoritative

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All stdlib, no external dependencies needed. Go's net/url and encoding packages are well-documented and stable.
- Architecture: HIGH - Pattern of per-protocol parser functions dispatched by scheme is well-established in multiple Go proxy client implementations (proxyclient, v2rayN, sing-box).
- Pitfalls: HIGH - Base64 variants, VMess JSON quirks, SIP002 dual encoding, and subscription format issues are well-documented through community bug reports and multiple independent implementations.
- Protocol URI formats: MEDIUM - No formal RFC exists for any of these protocols. The formats are documented through community convention, client implementations, and GitHub issues/discussions. VLESS and Trojan are the most consistent; VMess is the most variable.

**Research date:** 2026-02-25
**Valid until:** 2026-03-25 (30 days - stable domain, protocol formats change slowly)

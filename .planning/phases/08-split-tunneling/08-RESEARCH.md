# Phase 8: Split Tunneling - Research

**Researched:** 2026-02-26
**Domain:** Xray-core routing rules, macOS pf firewall coordination, DNS resolution for hostname rules
**Confidence:** HIGH

## Summary

Split tunneling in Azad is best implemented **entirely at the Xray-core routing layer**, not at the OS network layer. Since Azad uses a SOCKS5/HTTP system proxy (not TUN mode), all application traffic already flows through Xray-core's internal routing engine. Xray natively supports routing rules based on domains, IPs, and CIDR ranges, with built-in support for directing matched traffic to either a `proxy` or `direct` (freedom) outbound. This means the core split tunneling logic requires no new dependencies -- only modifications to the existing `BuildConfig` function to inject user-defined routing rules into the Xray JSON configuration.

The main complexity lies in three areas: (1) translating user-friendly rule syntax (like `*.google.com` or `10.0.0.0/8`) into Xray routing rule format, (2) coordinating with the kill switch pf rules so that "direct" traffic from the freedom outbound is allowed through the firewall, and (3) persisting rules in the YAML config and providing TUI/CLI management interfaces. DNS resolution for hostname rules is handled by Xray's built-in DNS resolver when using `domainStrategy: "IPIfNonMatch"`, which resolves domains to IPs for a second-pass match when no domain rule matches directly.

**Primary recommendation:** Implement split tunneling as Xray routing rules injected into `BuildConfig`, with a new `splittunnel` package for rule parsing/validation, config persistence via koanf, and kill switch pf rule generation that allows resolved direct-traffic IPs through the firewall.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| SPLT-01 | Inclusive mode -- only listed IPs/hostnames route through VPN, rest goes direct | Xray routing rules: listed items get `outboundTag: "proxy"`, default first outbound set to `direct` (freedom). Rules evaluated top-to-bottom; unmatched traffic falls to first outbound. |
| SPLT-02 | Exclusive mode -- all traffic through VPN except listed IPs/hostnames which go direct | Xray routing rules: listed items get `outboundTag: "direct"`, default first outbound stays `proxy`. This is the simpler mode since it matches existing routing structure. |
| SPLT-03 | Rule types -- single IPs, CIDR ranges, hostnames, wildcard domains | Xray `ip` field supports single IPs (`1.2.3.4`) and CIDR (`10.0.0.0/8`). Xray `domain` field supports exact match (`full:example.com`), subdomain/wildcard match (`domain:example.com` matches `*.example.com`), and keyword match. Direct mapping from user syntax to Xray syntax. |
| SPLT-04 | TUI management -- add/remove rules, switch modes, view active rules | New TUI view state for split tunnel management, accessible from the settings menu (m key). Follow existing modal/overlay patterns from kill switch and server management. |
| SPLT-05 | CLI management -- `azad split-tunnel` subcommand | New cobra subcommand with `add`, `remove`, `list`, `mode`, `clear` sub-subcommands. Follows existing CLI patterns from `servers` command. |
| SPLT-06 | Kill switch coordination -- split tunnel rules integrate with kill switch firewall rules | In exclusive mode (default), kill switch pf rules need no changes (all traffic goes through VPN except direct items which exit locally). In inclusive mode, pf rules must additionally allow direct traffic to bypass the firewall. `GenerateRules` must accept optional bypass IPs for the split tunnel direct destinations. |
</phase_requirements>

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Xray-core routing | v1.260206.0 (current) | Traffic routing by domain/IP rules | Already a dependency; native routing engine handles domain matching, IP/CIDR matching, and outbound selection |
| koanf | v2.3.2 (current) | Config persistence for split tunnel rules | Already the project's config library; add SplitTunnel section to Config struct |
| cobra | v1.9.1 (current) | CLI subcommand for split-tunnel management | Already the project's CLI framework |
| bubbletea/lipgloss/bubbles v2 | v2.0.0 (current) | TUI split tunnel rule management | Already the project's TUI stack |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| net (stdlib) | Go 1.25 | IP/CIDR validation via `net.ParseCIDR`, `net.ParseIP` | Rule validation before persisting to config |
| encoding/json (stdlib) | Go 1.25 | Xray config JSON generation | Already used in `engine/config.go` |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Xray routing rules | macOS pf-level routing | pf cannot route by domain, only IP; would require external DNS resolution and pf table management. Xray routing is the correct layer since traffic already flows through the SOCKS5 proxy. |
| Manual DNS resolution for hostnames | Xray `domainStrategy: "IPIfNonMatch"` | Xray's built-in DNS handles domain-to-IP resolution transparently. Manual resolution (e.g. `net.LookupHost`) would duplicate work and miss DNS changes. |
| Third-party DNS cache (rs/dnscache) | Xray built-in DNS | Xray already has a DNS module with caching, TTL support, and parallel queries. No need for external DNS library. |

## Architecture Patterns

### Recommended Project Structure

```
internal/
├── splittunnel/          # NEW: Rule types, validation, Xray rule generation
│   ├── rule.go           # Rule struct, parsing, validation
│   ├── rule_test.go      # Rule validation tests
│   ├── xray.go           # Convert rules to Xray RoutingRule format
│   └── xray_test.go      # Xray rule generation tests
├── config/
│   └── config.go         # MODIFY: Add SplitTunnel section to Config struct
├── engine/
│   └── config.go         # MODIFY: BuildConfig accepts split tunnel rules, injects into routing
├── killswitch/
│   └── rules.go          # MODIFY: GenerateRules accepts bypass IPs for split tunnel coordination
├── cli/
│   ├── split_tunnel.go   # NEW: azad split-tunnel subcommand
│   └── connect.go        # MODIFY: Pass split tunnel config to engine
├── tui/
│   ├── app.go            # MODIFY: New view states for split tunnel management
│   ├── split_tunnel.go   # NEW: Split tunnel TUI views (rule list, add modal, mode toggle)
│   ├── keys.go           # MODIFY: Add split tunnel keybinding
│   └── connect_cmd.go    # MODIFY: Pass split tunnel config when building Xray config
```

### Pattern 1: Xray Routing Rule Injection

**What:** Split tunnel rules are translated to Xray routing rules and injected into the Xray JSON config at build time.
**When to use:** Every time `BuildConfig` is called (connect, auto-connect, reconnect).

**How it works:**

For **exclusive mode** (bypass list -- default, most common):
- First outbound remains `proxy` (VPN)
- User's bypass rules become routing rules with `outboundTag: "direct"`
- Existing `geoip:private -> direct` rule is preserved
- Unmatched traffic goes through VPN (first outbound = proxy)

For **inclusive mode** (VPN-only list):
- First outbound changes to `direct` (freedom) -- unmatched traffic goes direct
- User's VPN rules become routing rules with `outboundTag: "proxy"`
- Private IPs still go direct (safety)

```go
// Source: Xray routing docs https://xtls.github.io/en/config/routing.html
// Exclusive mode example (bypass google.com and 10.0.0.0/8)
{
    "routing": {
        "domainStrategy": "IPIfNonMatch",
        "rules": [
            {
                "type": "field",
                "domain": ["domain:google.com"],
                "outboundTag": "direct"
            },
            {
                "type": "field",
                "ip": ["10.0.0.0/8"],
                "outboundTag": "direct"
            },
            {
                "type": "field",
                "ip": ["geoip:private"],
                "outboundTag": "direct"
            }
        ]
    }
}
```

### Pattern 2: Rule Type Discrimination and Validation

**What:** Each user rule is parsed, classified (IP, CIDR, hostname, wildcard domain), validated, and stored.
**When to use:** When user adds a rule via TUI or CLI.

```go
// Rule represents a single split tunnel entry.
type Rule struct {
    Value string   `koanf:"value" yaml:"value"`       // Original user input
    Type  RuleType `koanf:"type"  yaml:"type"`        // ip, cidr, domain, wildcard
}

type RuleType string

const (
    RuleTypeIP       RuleType = "ip"       // Single IP: 1.2.3.4
    RuleTypeCIDR     RuleType = "cidr"     // CIDR range: 10.0.0.0/8
    RuleTypeDomain   RuleType = "domain"   // Exact domain: example.com
    RuleTypeWildcard RuleType = "wildcard" // Wildcard: *.example.com
)

// ParseRule classifies and validates a user input string.
func ParseRule(input string) (Rule, error) {
    input = strings.TrimSpace(input)

    // Try IP first
    if ip := net.ParseIP(input); ip != nil {
        return Rule{Value: input, Type: RuleTypeIP}, nil
    }

    // Try CIDR
    if _, _, err := net.ParseCIDR(input); err == nil {
        return Rule{Value: input, Type: RuleTypeCIDR}, nil
    }

    // Try wildcard domain (*.example.com)
    if strings.HasPrefix(input, "*.") {
        domain := input[2:] // strip "*."
        if isValidDomain(domain) {
            return Rule{Value: input, Type: RuleTypeWildcard}, nil
        }
        return Rule{}, fmt.Errorf("invalid wildcard domain: %s", input)
    }

    // Plain domain
    if isValidDomain(input) {
        return Rule{Value: input, Type: RuleTypeDomain}, nil
    }

    return Rule{}, fmt.Errorf("invalid rule: %s (expected IP, CIDR, domain, or *.domain)", input)
}
```

### Pattern 3: Xray Rule Translation

**What:** Translate parsed rules into Xray RoutingRule structs for injection into the config.
**When to use:** Inside BuildConfig when split tunnel rules exist.

```go
// ToXrayRules converts split tunnel rules to Xray routing rules.
// The outboundTag depends on the mode:
//   - Exclusive mode: rules -> "direct" (bypass VPN)
//   - Inclusive mode: rules -> "proxy" (use VPN)
func ToXrayRules(rules []Rule, mode Mode) []RoutingRule {
    var domains []string
    var ips []string

    for _, r := range rules {
        switch r.Type {
        case RuleTypeIP:
            ips = append(ips, r.Value)
        case RuleTypeCIDR:
            ips = append(ips, r.Value)
        case RuleTypeDomain:
            // full: prefix for exact domain match
            domains = append(domains, "full:"+r.Value)
            // Also match subdomains: domain:example.com matches *.example.com
            domains = append(domains, "domain:"+r.Value)
        case RuleTypeWildcard:
            // *.example.com -> domain:example.com in Xray
            base := strings.TrimPrefix(r.Value, "*.")
            domains = append(domains, "domain:"+base)
        }
    }

    tag := "direct"
    if mode == ModeInclusive {
        tag = "proxy"
    }

    var xrayRules []RoutingRule
    if len(domains) > 0 {
        xrayRules = append(xrayRules, RoutingRule{
            Type:        "field",
            Domain:      domains,
            OutboundTag: tag,
        })
    }
    if len(ips) > 0 {
        xrayRules = append(xrayRules, RoutingRule{
            Type:        "field",
            IP:          ips,
            OutboundTag: tag,
        })
    }

    return xrayRules
}
```

### Pattern 4: Config Persistence

**What:** Split tunnel rules and mode persist in the YAML config file alongside proxy settings.
**When to use:** Adding/removing rules, switching modes.

```yaml
# ~/.config/azad/config.yaml
proxy:
  socks_port: 1080
  http_port: 8080
server:
  last_used: "abc123"
split_tunnel:
  enabled: false
  mode: "exclusive"     # "exclusive" (bypass list) or "inclusive" (VPN-only list)
  rules:
    - value: "10.0.0.0/8"
      type: "cidr"
    - value: "*.google.com"
      type: "wildcard"
    - value: "192.168.1.100"
      type: "ip"
    - value: "example.com"
      type: "domain"
```

### Pattern 5: Kill Switch Coordination

**What:** When kill switch is active alongside split tunneling, the pf rules must allow direct-bound traffic to exit.
**When to use:** When both kill switch and split tunnel are active simultaneously.

**Key insight:** In SOCKS5 proxy mode, "direct" traffic from the Xray freedom outbound exits the machine directly (not through the VPN tunnel). If the kill switch blocks all non-VPN traffic, direct split tunnel traffic gets blocked too.

**Solution:** When split tunneling is active in exclusive mode, resolve IP rules and pass them to `GenerateRules` as additional allowed destinations. For domain rules, Xray handles DNS resolution internally, but the resolved IPs need to be allowed through pf. This requires:

1. For IP/CIDR rules: add them directly to the pf allow list
2. For domain rules: resolve them at connection time and add resolved IPs to pf. Accept that pf won't dynamically update when DNS changes (domain routing still works at the Xray level, but pf bypass requires static IPs).

```go
// Modified GenerateRules signature
func GenerateRules(serverIP string, serverPort int, bypassIPs []string) string {
    // ... existing rules ...
    // Add bypass IPs for split tunnel direct traffic
    for _, ip := range bypassIPs {
        rules += fmt.Sprintf("pass out quick from any to %s\n", ip)
    }
    // ... block rules ...
}
```

**In inclusive mode with kill switch:** All non-listed traffic goes direct, which means the kill switch effectively blocks it (desired behavior -- only VPN-listed traffic works). No special coordination needed.

### Anti-Patterns to Avoid

- **OS-level routing for SOCKS proxy split tunneling:** Since traffic flows through Xray via SOCKS5/HTTP, OS routing tables and pf rules cannot selectively route by domain. Xray's routing engine is the correct layer.
- **Restarting Xray instance on rule change:** Rules only take effect when the Xray config is built. Changing rules while connected requires a reconnect (stop + start with new config). Do not try to hot-reload routing rules into a running instance.
- **Resolving all domains at config build time:** Domain rules should use Xray's `domain:` matching. DNS resolution happens inside Xray. Do not pre-resolve domains to IPs for the Xray routing rules -- this defeats the purpose of domain-based matching and breaks when IPs change.
- **Separate domain and IP into different rule arrays in config:** Store all rules in a single array with type discrimination. Xray routing rules are separated by domain vs IP, but the user-facing data model should be unified.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Domain matching with wildcards | Custom regex matching engine | Xray's `domain:` prefix routing | Xray handles subdomain matching, regex, keyword matching natively; handles millions of requests efficiently |
| DNS resolution for hostname rules | Periodic DNS cache with TTL tracking | Xray `domainStrategy: "IPIfNonMatch"` | Xray has built-in DNS with caching, parallel queries, DoH support. Adding external DNS adds complexity and potential for resolution conflicts |
| IP/CIDR validation | Manual string parsing | `net.ParseIP()` and `net.ParseCIDR()` from stdlib | Edge cases in CIDR notation (invalid masks, IPv6) are already handled |
| Domain validation | Regex-based domain validator | Simple hostname check (no leading dot, has at least one dot, valid chars) | Full RFC compliance is overkill; simple validation catches user errors |

**Key insight:** The Xray routing engine IS the split tunneling engine. We are configuring it, not building a competing router.

## Common Pitfalls

### Pitfall 1: Inclusive Mode Outbound Ordering
**What goes wrong:** In inclusive mode, the first outbound must be `direct` (freedom), not `proxy`. If the first outbound stays as `proxy`, unmatched traffic goes through VPN instead of direct -- defeating the purpose.
**Why it happens:** Xray sends unmatched traffic to the first outbound by default. The current `BuildConfig` always puts `proxy` first.
**How to avoid:** When mode is inclusive, swap outbound ordering: `[direct, proxy]` instead of `[proxy, direct]`.
**Warning signs:** All traffic goes through VPN even with inclusive mode enabled.

### Pitfall 2: Kill Switch Blocks Direct Split Tunnel Traffic
**What goes wrong:** User enables both kill switch and exclusive-mode split tunneling. The bypass rules route traffic to `direct` in Xray, but pf blocks all non-VPN traffic, so direct traffic is dropped.
**Why it happens:** Kill switch pf rules only allow traffic to the VPN server IP. Direct traffic exits to different IPs.
**How to avoid:** When both features are active, resolve IP/CIDR bypass rules and inject them into the pf allow list. For domain rules, resolve at connection time.
**Warning signs:** Split tunnel bypass rules don't work when kill switch is on.

### Pitfall 3: Domain Rules Need DomainStrategy Change
**What goes wrong:** Domain-based rules don't match traffic because `domainStrategy` is `"AsIs"`.
**Why it happens:** With `"AsIs"`, Xray only matches the domain from the connection request. If the app connects by IP (not domain), no domain match occurs. `"IPIfNonMatch"` resolves the domain for a second-pass IP match.
**How to avoid:** Change `domainStrategy` to `"IPIfNonMatch"` when split tunnel rules include domain rules. Keep `"AsIs"` when rules are IP-only (for lower latency).
**Warning signs:** Domain rules work in browser (which uses domain names) but not in apps that connect by IP.

### Pitfall 4: Rule Changes Require Reconnection
**What goes wrong:** User adds a split tunnel rule while connected, but traffic routing doesn't change.
**Why it happens:** Xray routing rules are compiled into the core.Config at instance creation time. They cannot be modified on a running instance.
**How to avoid:** After rule changes while connected, automatically trigger a reconnect (disconnect + connect with new config). Show a confirmation: "Applying changes requires reconnecting. Reconnect now?"
**Warning signs:** Rules appear saved but have no effect until manual reconnect.

### Pitfall 5: Wildcard Domain Syntax Confusion
**What goes wrong:** User enters `*.google.com` but expects `google.com` itself to also match.
**Why it happens:** Xray's `domain:google.com` matches both `google.com` AND `*.google.com`, which is what users expect from `*.google.com`. But if we only use `domain:`, then entering just `google.com` also matches wildcards, which may not be expected.
**How to avoid:** For `RuleTypeDomain` (plain `google.com`): use both `full:google.com` AND `domain:google.com` to match the exact domain and all subdomains. For `RuleTypeWildcard` (`*.google.com`): use only `domain:google.com` (matches subdomains but not the bare domain). This gives intuitive behavior.
**Warning signs:** Users report that some domain patterns don't match as expected.

### Pitfall 6: Private IP Rule Conflicts
**What goes wrong:** User adds a private IP range (e.g., `192.168.0.0/16`) to VPN-only list in inclusive mode, but it still goes direct.
**Why it happens:** The existing `geoip:private -> direct` rule in BuildConfig fires before user rules (rules evaluate top-to-bottom).
**How to avoid:** When split tunneling is active, place user rules BEFORE the `geoip:private` rule in the routing rules array.
**Warning signs:** Private IPs always go direct regardless of split tunnel mode.

## Code Examples

### BuildConfig with Split Tunnel Support

```go
// Modified BuildConfig signature
func BuildConfig(srv protocol.Server, socksPort, httpPort int, splitCfg *splittunnel.Config) (*XrayConfig, *core.Config, error) {
    outbound, err := buildOutbound(srv)
    if err != nil {
        return nil, nil, err
    }

    sniffing := &SniffingConfig{
        Enabled:      true,
        DestOverride: []string{"http", "tls"},
    }

    // Determine outbound ordering based on split tunnel mode
    var outbounds []OutboundConfig
    if splitCfg != nil && splitCfg.Enabled && splitCfg.Mode == splittunnel.ModeInclusive {
        // Inclusive: direct first (default for unmatched), proxy for listed
        outbounds = []OutboundConfig{
            {Tag: "direct", Protocol: "freedom"},
            outbound,
        }
    } else {
        // Normal / Exclusive: proxy first (default for unmatched), direct for listed
        outbounds = []OutboundConfig{
            outbound,
            {Tag: "direct", Protocol: "freedom"},
        }
    }

    // Build routing rules
    domainStrategy := "AsIs"
    var rules []RoutingRule

    if splitCfg != nil && splitCfg.Enabled && len(splitCfg.Rules) > 0 {
        // User split tunnel rules go FIRST (highest priority)
        rules = append(rules, splittunnel.ToXrayRules(splitCfg.Rules, splitCfg.Mode)...)

        // Change domain strategy if any domain rules exist
        if splittunnel.HasDomainRules(splitCfg.Rules) {
            domainStrategy = "IPIfNonMatch"
        }
    }

    // Private IPs always direct (safety net)
    rules = append(rules, RoutingRule{
        Type:        "field",
        OutboundTag: "direct",
        IP:          []string{"geoip:private"},
    })

    cfg := &XrayConfig{
        Log:       LogConfig{LogLevel: "warning"},
        Inbounds:  []InboundConfig{/* same as before */},
        Outbounds: outbounds,
        Routing: RoutingConfig{
            DomainStrategy: domainStrategy,
            Rules:          rules,
        },
    }

    // ... marshal and load as before
}
```

### Config Struct Extension

```go
// In config/config.go
type Config struct {
    Proxy       ProxyConfig       `koanf:"proxy"`
    Server      ServerConfig      `koanf:"server"`
    SplitTunnel SplitTunnelConfig `koanf:"split_tunnel"`
}

type SplitTunnelConfig struct {
    Enabled bool              `koanf:"enabled"`
    Mode    string            `koanf:"mode"`     // "exclusive" or "inclusive"
    Rules   []SplitTunnelRule `koanf:"rules"`
}

type SplitTunnelRule struct {
    Value string `koanf:"value"`
    Type  string `koanf:"type"`
}
```

### Kill Switch with Split Tunnel Bypass

```go
// In killswitch/rules.go -- modified to accept bypass IPs
func GenerateRules(serverIP string, serverPort int, bypassIPs []string) string {
    var builder strings.Builder

    builder.WriteString("# Azad Kill Switch - generated rules\n")
    builder.WriteString("set block-policy drop\n\n")
    builder.WriteString("pass quick on lo0 all\n")
    builder.WriteString(fmt.Sprintf("pass out quick proto {tcp, udp} from any to %s port %d\n", serverIP, serverPort))
    builder.WriteString("pass quick proto {tcp, udp} from any port 67:68 to any port 67:68\n")
    builder.WriteString("pass out quick proto {tcp, udp} from any to any port 53\n")

    // Split tunnel bypass IPs
    for _, ip := range bypassIPs {
        builder.WriteString(fmt.Sprintf("pass out quick from any to %s\n", ip))
    }

    builder.WriteString("block out all\n")
    builder.WriteString("block in all\n")
    builder.WriteString("block out inet6 all\n")
    builder.WriteString("block in inet6 all\n")

    return builder.String()
}
```

### CLI Subcommand

```go
// In cli/split_tunnel.go
func newSplitTunnelCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "split-tunnel",
        Short: "Manage split tunnel rules",
        Aliases: []string{"st"},
    }

    cmd.AddCommand(
        newSTAddCmd(),      // azad split-tunnel add <rule>
        newSTRemoveCmd(),   // azad split-tunnel remove <rule>
        newSTListCmd(),     // azad split-tunnel list
        newSTModeCmd(),     // azad split-tunnel mode <exclusive|inclusive>
        newSTEnableCmd(),   // azad split-tunnel enable/disable
        newSTClearCmd(),    // azad split-tunnel clear
    )

    return cmd
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| TUN mode split tunneling (route table manipulation) | SOCKS proxy with Xray routing rules | N/A (Azad design decision) | Eliminates need for root/admin for basic split tunneling; kill switch still needs admin for pf |
| Manual /etc/pf.conf editing for bypass rules | Programmatic pf anchor with bypass IPs | Phase 7 (kill switch) | Clean anchor-based approach extends naturally to split tunnel bypass |
| External DNS resolution for hostname rules | Xray built-in DNS with domainStrategy | Xray-core native | No external DNS cache needed; Xray handles resolution, caching, and re-resolution |

**Note on approach:** Most commercial VPN apps (Windscribe, Mullvad, ExpressVPN) use TUN mode for split tunneling, which allows OS-level per-app and per-route control. Azad's SOCKS5 proxy approach provides domain-level and IP-level routing (which covers the requirements) but cannot do per-app splitting. This is an intentional trade-off documented in REQUIREMENTS.md (TUN mode is out of scope).

## Open Questions

1. **Should rule changes auto-reconnect or require manual reconnect?**
   - What we know: Xray config is compiled at instance creation and cannot be hot-reloaded
   - What's unclear: Whether auto-reconnecting after every rule change is acceptable UX, or if a "pending changes" indicator with manual apply is better
   - Recommendation: Auto-reconnect with confirmation dialog in TUI; in CLI, show warning that reconnect is needed

2. **Domain resolution for kill switch bypass -- how fresh?**
   - What we know: When kill switch + split tunnel are both active, domain bypass rules need IP addresses for pf rules. Xray handles domain routing internally, but pf only works with IPs.
   - What's unclear: Whether stale DNS resolution for pf bypass causes practical issues (the Xray routing still works correctly; only the pf bypass might be stale)
   - Recommendation: Resolve domains once at connection time for pf bypass. Document that domain-based bypass with kill switch may not track DNS changes until reconnection. This is an acceptable limitation since the Xray routing layer always handles it correctly -- the pf bypass is just an optimization.

3. **Maximum number of rules?**
   - What we know: Xray routing rules have no documented limit. pf tables support thousands of entries.
   - What's unclear: Performance impact of hundreds of routing rules on Xray
   - Recommendation: Soft limit of 100 rules with a warning; no hard limit. Real-world usage will be 5-20 rules.

## Sources

### Primary (HIGH confidence)
- Xray-core routing docs (https://xtls.github.io/en/config/routing.html) -- complete routing rule schema, domain matching syntax, IP/CIDR matching, domainStrategy options
- Xray-core DNS docs (https://xtls.github.io/en/config/dns.html) -- built-in DNS configuration, caching, parallel queries
- Context7 /websites/xtls_github_io -- verified routing rule examples, domain/IP syntax, outbound ordering behavior
- Existing codebase analysis -- `internal/engine/config.go`, `internal/killswitch/rules.go`, `internal/config/config.go`

### Secondary (MEDIUM confidence)
- macOS pf documentation (https://murusfirewall.com/Documentation/OS%20X%20PF%20Manual.pdf) -- pf anchor rules, table syntax for IP lists
- ZorroVPN pf kill switch guide (https://zorrovpn.com/articles/osx-pf-vpn-only?lang=en) -- pf VPN-only rules with bypass
- Xray-core discussions #3047, #3621, #4653 -- community-verified domain routing patterns

### Tertiary (LOW confidence)
- Mullvad split tunneling deep wiki (https://deepwiki.com/mullvad/mullvadvpn-app/3.5-split-tunneling) -- reference for how commercial VPNs implement split tunneling (uses different architecture: TUN mode)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- no new dependencies needed; Xray routing is well-documented and already integrated
- Architecture: HIGH -- Xray routing rule injection is the canonical approach for SOCKS proxy split tunneling; verified with official docs and community examples
- Pitfalls: HIGH -- identified through analysis of existing codebase (outbound ordering, pf coordination, domainStrategy) and Xray community discussions
- Kill switch coordination: MEDIUM -- the pf bypass for domain rules is an edge case that needs validation during implementation

**Research date:** 2026-02-26
**Valid until:** 2026-03-26 (stable domain; Xray routing API has not changed in years)

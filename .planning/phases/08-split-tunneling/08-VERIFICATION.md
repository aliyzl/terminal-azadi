---
phase: 08-split-tunneling
verified: 2026-02-26T19:50:00Z
status: passed
score: 20/20 must-haves verified
re_verification: false
gaps: []
human_verification:
  - test: "Connect to a server with exclusive mode split tunnel rules configured, then enable kill switch. Verify that traffic to bypass IPs flows directly while VPN traffic is routed through the proxy."
    expected: "Split tunnel routing takes effect — traffic destined for listed IPs/CIDRs bypasses VPN. Kill switch pf rules allow direct traffic to listed IPs while blocking all other non-VPN traffic."
    why_human: "End-to-end routing behavior requires actual pf/xray runtime execution and network inspection tools (e.g., traceroute). Cannot be verified programmatically via static analysis."
  - test: "Open TUI, press m for settings menu, press t for split tunnel. Add a rule via a key, delete via d key, toggle mode via t key, toggle enable/disable via e key."
    expected: "Split tunnel overlay renders with rule list, status (ENABLED/DISABLED), mode (Exclusive/Inclusive). All operations persist to config. Status bar shows SPLIT indicator when enabled with rules."
    why_human: "TUI rendering and interactive input require visual inspection of a running terminal."
  - test: "Run 'azad split-tunnel list' (empty), 'azad split-tunnel add 10.0.0.0/8', 'azad split-tunnel add *.google.com', 'azad split-tunnel list', 'azad split-tunnel mode inclusive', 'azad split-tunnel remove 10.0.0.0/8', 'azad split-tunnel clear'."
    expected: "Each command produces the documented output message. Config file is updated correctly after each mutation command."
    why_human: "CLI output formatting and config file mutation correctness require running the binary."
---

# Phase 8: Split Tunneling Verification Report

**Phase Goal:** Windscribe-style split tunneling — users define which traffic goes through the VPN and which bypasses it, using IP addresses and hostnames in either inclusive (only listed routes go through VPN) or exclusive (everything except listed routes goes through VPN) mode

**Verified:** 2026-02-26T19:50:00Z
**Status:** PASSED
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | IP addresses (single and CIDR) are validated and classified correctly | VERIFIED | `ParseRule` in `internal/splittunnel/rule.go` uses `net.ParseIP` and `net.ParseCIDR`; 12 table-driven tests pass covering IPv4, IPv6, CIDR /8 and /24 |
| 2 | Hostnames and wildcard domains are validated and classified correctly | VERIFIED | `isValidDomain` validates plain domains; `*.` prefix triggers wildcard path; tests for `example.com`, `sub.example.com`, `*.google.com` all pass |
| 3 | Invalid rule inputs produce clear error messages | VERIFIED | `ParseRule` returns descriptive `fmt.Errorf` messages for empty, garbage, and invalid wildcard inputs; test cases for all three pass |
| 4 | Exclusive mode routes listed rules to direct outbound, unmatched to proxy | VERIFIED | `BuildConfig` places proxy outbound first when exclusive; `ToXrayRules` sets `outboundTag: "direct"` for exclusive mode; `TestBuildConfigSplitTunnel` confirms |
| 5 | Inclusive mode routes listed rules to proxy outbound, unmatched to direct | VERIFIED | `BuildConfig` swaps outbounds (direct first) for inclusive mode; `ToXrayRules` sets `outboundTag: "proxy"`; test `inclusive_mode_with_IP_rule` passes |
| 6 | Domain rules change domainStrategy to IPIfNonMatch | VERIFIED | `BuildConfig` calls `splittunnel.HasDomainRules` and sets `domainStrategy = "IPIfNonMatch"` when domain/wildcard rules present; test `exclusive_mode_with_domain_rule` confirms |
| 7 | User split tunnel rules are placed before geoip:private rule | VERIFIED | `BuildConfig` appends split tunnel rules first, then appends `geoip:private` rule last; comment says "always last (safety net)" |
| 8 | Config struct persists split tunnel rules and mode via koanf | VERIFIED | `internal/config/config.go` has `SplitTunnelConfig` and `SplitTunnelRule` types with `koanf:"split_tunnel"` tag; `Config.SplitTunnel SplitTunnelConfig` field present |
| 9 | GenerateRules accepts bypass IPs for split tunnel direct destinations | VERIFIED | `internal/killswitch/rules.go` `GenerateRules(serverIP string, serverPort int, bypassIPs []string)` iterates bypassIPs and emits `pass out quick from any to <ip>` rules |
| 10 | Engine.Start accepts split tunnel config and passes it to BuildConfig | VERIFIED | `internal/engine/engine.go` uses variadic `splitCfg ...*splittunnel.Config`; extracts `sc` and passes to `BuildConfig(srv, socksPort, httpPort, sc)` |
| 11 | azad split-tunnel add/remove/list/mode/enable/disable/clear subcommands work correctly | VERIFIED | `internal/cli/split_tunnel.go` exports `newSplitTunnelCmd()` with all 7 subcommands; each calls `config.Load`/`config.Save`; registered in `root.go` line 101 |
| 12 | azad connect passes split tunnel config from config file to engine | VERIFIED | `internal/cli/connect.go` builds `splitCfg` from `cfg.SplitTunnel` when enabled and passes to `eng.Start(ctx, *server, ..., splitCfg)` at line 84 |
| 13 | Kill switch pf rules allow direct traffic to bypass IPs when split tunnel is active | VERIFIED | CLI `connect.go` extracts bypassIPs from exclusive-mode rules (lines 108-125), passes to `killswitch.Enable(resolvedIP, server.Port, bypassIPs)` at line 144 |
| 14 | User can view split tunnel rules and mode from the TUI settings menu | VERIFIED | `renderSplitTunnelView` in `internal/tui/split_tunnel.go` renders status, mode, numbered rule list with type annotations; called from `View()` at line 780 |
| 15 | User can add a split tunnel rule via input modal in the TUI | VERIFIED | `viewAddSplitRule` case in `handleKeyPress` calls `splittunnel.ParseRule`, appends to `m.cfg.SplitTunnel.Rules`, calls `saveSplitTunnelCmd` |
| 16 | User can remove a split tunnel rule from the TUI | VERIFIED | `viewSplitTunnel` "d" key handler splices rule from `m.cfg.SplitTunnel.Rules` and calls `saveSplitTunnelCmd` |
| 17 | User can toggle split tunnel mode between exclusive and inclusive | VERIFIED | `viewSplitTunnel` "t" key handler toggles `m.cfg.SplitTunnel.Mode` between "inclusive" and "exclusive" and saves |
| 18 | User can enable/disable split tunneling from the TUI | VERIFIED | `viewSplitTunnel` "e" key handler toggles `m.cfg.SplitTunnel.Enabled`, updates status bar, saves config |
| 19 | Status bar shows SPLIT indicator when split tunneling is enabled | VERIFIED | `statusBarModel.splitTunnel bool` field; `SetSplitTunnel` method; `View()` renders `" SPLIT "` in accent color when `m.splitTunnel == true` |
| 20 | TUI connect commands pass split tunnel config to engine | VERIFIED | `connectServerCmd` and `autoConnectCmd` in `connect_cmd.go` both accept `splitCfg *splittunnel.Config` and pass to `eng.Start(..., splitCfg)` |

**Score:** 20/20 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/splittunnel/rule.go` | Rule struct, RuleType enum, ParseRule validation, Mode type | VERIFIED | Exports: `Rule`, `RuleType`, `RuleTypeIP`, `RuleTypeCIDR`, `RuleTypeDomain`, `RuleTypeWildcard`, `Mode`, `ModeExclusive`, `ModeInclusive`, `ParseRule`, `HasDomainRules` — all present |
| `internal/splittunnel/xray.go` | ToXrayRules converting split tunnel rules to Xray RoutingRule format | VERIFIED | `XrayRoutingRule` struct and `ToXrayRules(rules []Rule, mode Mode) []XrayRoutingRule` implemented; 6 table-driven tests pass |
| `internal/config/config.go` | SplitTunnelConfig and SplitTunnelRule types in Config struct | VERIFIED | `Config.SplitTunnel SplitTunnelConfig`, `SplitTunnelConfig{Enabled bool, Mode string, Rules []SplitTunnelRule}`, `SplitTunnelRule{Value, Type string}` — all present with koanf tags |
| `internal/engine/config.go` | BuildConfig accepts optional split tunnel config and injects routing rules | VERIFIED | `func BuildConfig(srv protocol.Server, socksPort, httpPort int, splitCfg *splittunnel.Config) (*XrayConfig, *core.Config, error)` — signature correct; domain field, outbound ordering, rule injection all implemented |
| `internal/killswitch/rules.go` | GenerateRules with bypass IPs parameter | VERIFIED | `func GenerateRules(serverIP string, serverPort int, bypassIPs []string) string` — iterates bypassIPs, emits `pass out quick from any to <ip>` before block rules |
| `internal/engine/engine.go` | Engine.Start accepts split tunnel config | VERIFIED | `func (e *Engine) Start(ctx context.Context, srv protocol.Server, socksPort, httpPort int, splitCfg ...*splittunnel.Config) error` — variadic, extracts `sc` and passes to `BuildConfig` |
| `internal/cli/split_tunnel.go` | azad split-tunnel subcommand with add/remove/list/mode/enable/disable/clear | VERIFIED | All 7 subcommands implemented; `loadConfig` helper factored out; registered via `newSplitTunnelCmd()` |
| `internal/cli/connect.go` | Connect flow loads and passes split tunnel config | VERIFIED | Builds `splitCfg` from `cfg.SplitTunnel`, passes to `eng.Start`, extracts bypass IPs, passes to `killswitch.Enable` |
| `internal/tui/split_tunnel.go` | Split tunnel TUI view rendering and helper functions | VERIFIED | `renderSplitTunnelView`, `buildSplitTunnelConfig`, `saveSplitTunnelCmd` all implemented |
| `internal/tui/app.go` | viewSplitTunnel and viewAddSplitRule view states, split tunnel key handling | VERIFIED | Both view states in enum; `splitTunnelIdx int` field; full key handling for a/d/e/t/j/k/esc in viewSplitTunnel; enter/esc in viewAddSplitRule |
| `internal/tui/keys.go` | Menu keybinding (SplitTunnel accessible from menu) | VERIFIED | `Menu` key binding ("m") present; split tunnel accessed via "t" from viewMenu |
| `internal/tui/connect_cmd.go` | Connect commands pass split tunnel config from cfg to engine | VERIFIED | `connectServerCmd` and `autoConnectCmd` both accept and pass `splitCfg`; `extractBypassIPs` helper; `enableKillSwitchCmd` accepts bypass IPs |
| `internal/tui/statusbar.go` | SPLIT indicator in status bar | VERIFIED | `splitTunnel bool` field, `SetSplitTunnel(active bool)` method, `" SPLIT "` rendered in accent color |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `internal/splittunnel/xray.go` | `internal/engine/config.go` | `ToXrayRules called in BuildConfig` | WIRED | Line 149: `for _, xr := range splittunnel.ToXrayRules(splitCfg.Rules, splitCfg.Mode)` — called and result iterated |
| `internal/config/config.go` | `internal/splittunnel/rule.go` | `SplitTunnelRule mirrors splittunnel.Rule for persistence` | WIRED | Both define `Value string` and `Type string` fields; conversion is done at CLI/TUI call sites |
| `internal/engine/config.go` | RoutingRule | `Domain field added to RoutingRule struct` | WIRED | Line 108: `Domain []string \`json:"domain,omitempty"\`` — field present in `RoutingRule` struct |
| `internal/cli/connect.go` | `internal/engine/engine.go` | `Engine.Start receives split tunnel config` | WIRED | Line 84: `eng.Start(ctx, *server, cfg.Proxy.SOCKSPort, cfg.Proxy.HTTPPort, splitCfg)` |
| `internal/killswitch/rules.go` | `internal/cli/connect.go` | `Connect passes bypass IPs to kill switch Enable` | WIRED | Lines 108-125 extract bypassIPs; line 144: `killswitch.Enable(resolvedIP, server.Port, bypassIPs)` |
| `internal/cli/split_tunnel.go` | `internal/config/config.go` | `CLI reads/writes SplitTunnelConfig` | WIRED | Each subcommand calls `loadConfig()` then `config.Save(cfg, configPath)` after modifying `cfg.SplitTunnel` |
| `internal/tui/app.go` | `internal/tui/split_tunnel.go` | `View() renders split tunnel overlay from view state` | WIRED | Line 780: `case viewSplitTunnel: content = renderSplitTunnelView(m)` |
| `internal/tui/connect_cmd.go` | `internal/engine/engine.go` | `Engine.Start called with split tunnel config` | WIRED | Lines 28 and 105: `eng.Start(context.Background(), ..., splitCfg)` in both `connectServerCmd` and `autoConnectCmd` |
| `internal/tui/app.go` | `internal/config/config.go` | `Split tunnel rules read/written to cfg.SplitTunnel` | WIRED | 14 occurrences of `m.cfg.SplitTunnel` across key handlers; `saveSplitTunnelCmd` persists on every mutation |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| SPLT-01 | 08-01 | Inclusive mode routes only listed IPs/hostnames through VPN, everything else goes direct | SATISFIED | `BuildConfig` swaps outbound ordering (direct first) for inclusive mode; `ToXrayRules` sets `outboundTag: "proxy"` for listed rules; `TestBuildConfigSplitTunnel/inclusive_mode_with_IP_rule` passes |
| SPLT-02 | 08-01 | Exclusive mode routes all traffic through VPN except listed IPs/hostnames which go direct | SATISFIED | `BuildConfig` keeps proxy first for exclusive mode; `ToXrayRules` sets `outboundTag: "direct"` for listed rules; `TestBuildConfigSplitTunnel/exclusive_mode_with_domain_rule` passes |
| SPLT-03 | 08-01 | Rules support single IPs, CIDR ranges, hostnames, and wildcard domains (*.example.com) | SATISFIED | `ParseRule` handles all 4 types via `net.ParseIP`, `net.ParseCIDR`, `*.` prefix check, and `isValidDomain`; 12 table-driven test cases all pass |
| SPLT-04 | 08-03 | User can add/remove rules, switch modes, and toggle split tunneling through TUI settings menu | SATISFIED | TUI `viewSplitTunnel` implements a/d/e/t key handlers; `viewAddSplitRule` modal; status bar SPLIT indicator; settings menu shows split tunnel row with t keybind |
| SPLT-05 | 08-02 | User can manage split tunnel rules via `azad split-tunnel` CLI subcommand (add/remove/list/mode/enable/disable/clear) | SATISFIED | `internal/cli/split_tunnel.go` implements all 7 subcommands; registered in `root.go` |
| SPLT-06 | 08-02 | Split tunneling coordinates with kill switch — bypass IPs allowed through pf firewall rules | SATISFIED | `GenerateRules` accepts `bypassIPs []string` and emits `pass out quick from any to <ip>`; CLI and TUI both extract bypass IPs from exclusive-mode rules and pass to `killswitch.Enable` |

All 6 requirements (SPLT-01 through SPLT-06) are SATISFIED. No orphaned requirements detected.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `internal/tui/ping.go` | 21 | `address format "%s:%d" does not work with IPv6` (go vet warning) | Info | Pre-existing issue from Phase 04 (`2177215`). Not introduced by Phase 08. No functional impact on split tunneling. |

No stubs, placeholder returns, TODO comments, or incomplete handlers found in any Phase 08 files.

### Human Verification Required

#### 1. End-to-end split tunnel routing with kill switch

**Test:** Configure 2-3 exclusive mode rules (e.g., `10.0.0.0/8`, `example.com`). Connect via `azad connect --kill-switch`. Run `traceroute 10.0.0.1` and `traceroute 8.8.8.8`.
**Expected:** Traffic to `10.0.0.1` goes direct (not through SOCKS proxy). Traffic to `8.8.8.8` routes through VPN. Kill switch pf anchor shows bypass pass rule for 10.0.0.0/8.
**Why human:** Requires live pf firewall execution, actual network traffic routing inspection, and Xray-core runtime behavior — not verifiable via static analysis.

#### 2. TUI split tunnel management flow

**Test:** Launch TUI (`azad`). Press m, then t. Add rule `192.168.1.0/24` via a key. Delete it via d. Toggle mode via t key. Toggle enable via e key. Verify status bar SPLIT indicator appears when enabled.
**Expected:** Overlay renders correctly with lipgloss borders. All key handlers work. Status bar updates immediately. Config file reflects each change.
**Why human:** Requires visual inspection of terminal rendering and interactive input flow.

#### 3. Inclusive mode end-to-end

**Test:** Configure inclusive mode with `8.8.8.8` as a rule. Connect. Verify that `curl --socks5 127.0.0.1:1080 https://8.8.8.8` routes through VPN while other traffic goes direct.
**Expected:** Only listed destinations use VPN; default route is direct (freedom outbound first).
**Why human:** Requires live Xray-core execution and network traffic verification.

### Gaps Summary

No gaps found. All automated checks passed.

**Build:** `go build ./...` — succeeds with zero errors.
**Tests:** `go test ./internal/splittunnel/... ./internal/engine/...` — 20 tests pass across both packages.
**Vet:** All Phase 08 packages (splittunnel, engine, killswitch, config) pass `go vet` with zero issues. The single vet warning in `internal/tui/ping.go` is a pre-existing issue from Phase 04, not introduced by Phase 08, and does not affect split tunneling functionality.

---

_Verified: 2026-02-26T19:50:00Z_
_Verifier: Claude (gsd-verifier)_

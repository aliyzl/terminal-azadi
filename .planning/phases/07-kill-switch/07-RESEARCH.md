# Phase 7: Kill Switch - Research

**Researched:** 2026-02-26
**Domain:** macOS packet filter (pfctl/pf), firewall rule management, process signal handling, crash-safe state persistence
**Confidence:** MEDIUM

## Summary

The kill switch prevents traffic leaks by using macOS's built-in packet filter (pf/pfctl) to block all outbound traffic except: (1) traffic to the VPN server's IP address on the proxy ports, and (2) loopback/localhost traffic for the local SOCKS5/HTTP proxy. When the kill switch is active and the app crashes or the terminal closes, the pf rules persist in the kernel, blocking all internet until the user runs `azad` (which reconnects) or `azad --cleanup` (which removes the rules).

This phase is architecturally simpler than a traditional VPN kill switch because Azad uses a SOCKS5/HTTP proxy (not a tunnel interface like utun0). The pf rules only need to allow traffic to the remote server IP while blocking everything else. The app already has a crash-recovery state system (`.state.json` + `--cleanup`) that will be extended to include kill switch state.

**Primary recommendation:** Use macOS pfctl with a named anchor (`com.azad.killswitch`) to load/flush firewall rules. Require `sudo` via `osascript -e 'do shell script "..." with administrator privileges'` for the GUI password prompt. Persist kill switch state in the existing `.state.json` file so `--cleanup` can always recover.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| KILL-01 | Block all non-VPN traffic via macOS pfctl when kill switch enabled | pf anchor rules with `block all` + `pass` for VPN server IP + loopback; loaded via `sudo pfctl -a com.azad.killswitch -f -` |
| KILL-02 | Firewall rules persist if terminal closes or app crashes | pf rules are kernel-level; once loaded they survive process death. State file records `kill_switch_active: true` for recovery |
| KILL-03 | Running `azad` after crash resumes VPN or offers reconnect | On startup, detect `.state.json` with `kill_switch_active`, reconnect to stored server, or prompt user |
| KILL-04 | `azad --cleanup` removes kill switch rules and restores internet | Flush anchor: `sudo pfctl -a com.azad.killswitch -F all`; restore default pf state; remove state file |
| KILL-05 | macOS shows confirmation dialog when closing terminal while active | Terminal.app built-in behavior: shows "processes are running" dialog when a foreground process (azad) is still alive. Works automatically. |
</phase_requirements>

## Standard Stack

### Core
| Tool | Version | Purpose | Why Standard |
|------|---------|---------|--------------|
| macOS pfctl | Built-in | Packet filter firewall CLI | Only firewall tool on macOS; used by all VPN kill switch implementations |
| pf anchors | Built-in | Namespaced rule sets | Isolates our rules from system rules; clean load/flush without touching `/etc/pf.conf` |
| os/exec (Go stdlib) | 1.21+ | Execute pfctl commands | Already used in the codebase for `networksetup`; same pattern applies |
| osascript | Built-in | GUI privilege escalation | Standard macOS way to prompt user for admin password from a terminal app |

### Supporting
| Tool | Version | Purpose | When to Use |
|------|---------|---------|-------------|
| SIGHUP handler | Go stdlib | Detect terminal close | Trap SIGHUP to know terminal is closing; log but intentionally let pf rules persist |
| `.state.json` | Existing | Crash recovery state | Extended with `KillSwitchActive` field to track kill switch state across crashes |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| pfctl anchors | Modifying /etc/pf.conf directly | Anchor approach is safer -- no risk of corrupting system pf config; macOS updates won't overwrite |
| osascript privilege prompt | Requiring user to run `sudo azad` | osascript gives native macOS password dialog; running whole app as root is dangerous and unnecessary |
| Network Extension / NEFilterProvider | pfctl | Network Extension requires Apple Developer signing, entitlements, and app notarization; massive overkill for a CLI tool |

## Architecture Patterns

### Recommended Module Structure
```
internal/
├── killswitch/
│   ├── killswitch.go     # Enable/Disable/IsActive/Cleanup public API
│   ├── rules.go          # pf rule generation (templated per server)
│   └── privilege.go      # osascript/sudo privilege escalation
```

### Pattern 1: pf Anchor-Based Kill Switch
**What:** Load kill switch rules into a dedicated pf anchor; flush the anchor to disable.
**When to use:** Every time kill switch is toggled on/off.
**Example pf rules (generated dynamically per connection):**
```
# Azad Kill Switch Rules
# Loaded into anchor: com.azad.killswitch

# Block policy: drop silently (no RST/ICMP unreachable)
set block-policy drop

# Allow all loopback traffic (required for local SOCKS5/HTTP proxy)
pass quick on lo0 all

# Allow traffic to the VPN server IP (the remote Xray server)
pass out quick proto {tcp, udp} from any to <SERVER_IP> port <SERVER_PORT>

# Allow DHCP (required to maintain network connectivity)
pass quick proto {tcp, udp} from any port 67:68 to any port 67:68

# Allow DNS to resolve the VPN server hostname (if needed)
# Only if server address is a hostname, not an IP
pass out quick proto {tcp, udp} from any to any port 53

# Block everything else
block out all
block in all
```

**pfctl commands:**
```bash
# Enable kill switch (load rules into anchor)
echo "$RULES" | sudo pfctl -a com.azad.killswitch -f -
sudo pfctl -E    # Enable pf if not already enabled

# Disable kill switch (flush anchor rules)
sudo pfctl -a com.azad.killswitch -F all

# Check if anchor has rules
sudo pfctl -a com.azad.killswitch -sr
```

### Pattern 2: Privilege Escalation via osascript
**What:** Use AppleScript to prompt for admin password and run pfctl with elevated privileges.
**When to use:** When enabling or disabling kill switch (any pfctl operation).
**Example:**
```go
// Source: macOS osascript documentation + Apple Developer Forums
func runPrivileged(command string) error {
    script := fmt.Sprintf(
        `do shell script "%s" with administrator privileges`,
        strings.ReplaceAll(command, `"`, `\"`),
    )
    cmd := exec.Command("osascript", "-e", script)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("privileged command failed: %w: %s", err, output)
    }
    return nil
}
```

### Pattern 3: Extended State File for Crash Recovery
**What:** Extend the existing ProxyState in `.state.json` with kill switch fields.
**When to use:** Every time kill switch state changes.
**Example:**
```go
type ProxyState struct {
    ProxySet         bool   `json:"proxy_set"`
    SOCKSPort        int    `json:"socks_port"`
    HTTPPort         int    `json:"http_port"`
    NetworkService   string `json:"network_service"`
    PID              int    `json:"pid"`
    // New fields for kill switch
    KillSwitchActive bool   `json:"kill_switch_active"`
    ServerAddress    string `json:"server_address"`  // Remote server IP for rule regeneration
    ServerPort       int    `json:"server_port"`     // Remote server port
}
```

### Pattern 4: Startup Recovery Flow
**What:** On startup, detect kill switch state and handle recovery.
**When to use:** In the root command's PersistentPreRunE, before TUI launch.
**Flow:**
```
1. Read .state.json
2. If kill_switch_active == true:
   a. Internet is currently blocked (pf rules are active)
   b. If running `azad` (TUI mode): auto-reconnect to stored server
   c. If running `azad --cleanup`: flush anchor rules, remove state file
   d. If running `azad connect`: connect to specified/stored server
3. If kill_switch_active == false or no state file:
   a. Normal startup
```

### Anti-Patterns to Avoid
- **Modifying /etc/pf.conf directly:** macOS updates overwrite this file. Use anchors instead.
- **Flushing all pf rules on disable:** `pfctl -F all` destroys Apple's system rules (AirDrop, Application Firewall). Only flush our anchor: `pfctl -a com.azad.killswitch -F all`.
- **Running the entire app as root:** Only pfctl commands need elevation. Use osascript for targeted privilege escalation.
- **Blocking loopback traffic:** The local SOCKS5/HTTP proxy listens on 127.0.0.1. Blocking loopback breaks the proxy entirely.
- **Blocking DNS unconditionally:** If the VPN server address was resolved from a hostname, DNS must work at least once. Consider resolving the server IP before enabling the kill switch.
- **Forgetting IPv6:** Must block IPv6 too (`block out inet6 all`) or traffic can leak via IPv6.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Firewall rule management | Custom iptables/nftables equivalent | macOS pfctl (built-in) | pfctl is the only packet filter on macOS; it's stable, well-documented, kernel-level |
| Privilege escalation | Custom auth framework | osascript + "with administrator privileges" | Native macOS password dialog; handles keychain, Touch ID, etc. |
| Anchor management | File-based pf.conf editing | Pipe rules via stdin to `pfctl -a anchor -f -` | No file management, no cleanup, no risk of orphaned temp files |
| Process detection for terminal close | Custom process monitoring | Terminal.app's built-in "processes are running" dialog | macOS Terminal (and iTerm2) already detect foreground processes and show confirmation |

**Key insight:** macOS provides all the building blocks natively (pfctl for firewall, osascript for privilege escalation, Terminal.app for close detection). The implementation is primarily about orchestrating these tools correctly and ensuring crash-safe state management.

## Common Pitfalls

### Pitfall 1: Locking the User Out of Their Own Machine
**What goes wrong:** Kill switch blocks all traffic, user cannot reach the internet at all, and `azad --cleanup` also fails because it needs sudo (which needs a password dialog).
**Why it happens:** If osascript is not available or the user is SSH'd in, privilege escalation fails.
**How to avoid:** `azad --cleanup` should detect if it's already running as root (via `sudo azad --cleanup`) and skip osascript. Provide clear documentation: "If locked out, run `sudo pfctl -a com.azad.killswitch -F all && sudo pfctl -d` to manually recover."
**Warning signs:** User reports "internet doesn't work and azad --cleanup fails."

### Pitfall 2: Not Flushing Rules on macOS Reboot
**What goes wrong:** Actually a non-issue -- pf rules loaded into anchors do NOT persist across reboots. pf is disabled by default on macOS. This is actually desirable: if the machine reboots, normal internet is restored automatically.
**Why it matters:** Understanding this means we don't need a launchd agent or any boot-time persistence. The kill switch is session-based by design.
**How to avoid:** Document this clearly: "Rebooting your Mac always restores normal internet."

### Pitfall 3: IPv6 Leak
**What goes wrong:** pf rules block IPv4 but not IPv6. Traffic leaks via IPv6.
**Why it happens:** Rules only specify `inet` (IPv4) family.
**How to avoid:** Include explicit `block out inet6 all` rule to block all IPv6 traffic when kill switch is active.
**Warning signs:** IP leak test shows IPv6 address despite kill switch being active.

### Pitfall 4: DNS Leak Before VPN Connection
**What goes wrong:** DNS resolution for the VPN server hostname happens through the unprotected connection.
**Why it happens:** Server address in config is a hostname, not an IP.
**How to avoid:** Resolve the server hostname to an IP address BEFORE enabling the kill switch. Store the resolved IP in the pf rules, not the hostname. The `pass out` rule should reference a numeric IP address.
**Warning signs:** DNS queries visible in network traffic before VPN connects.

### Pitfall 5: Breaking Apple's pf Anchors
**What goes wrong:** AirDrop, Application Firewall, and other Apple services stop working.
**Why it happens:** Flushing all pf rules (`pfctl -F all` or `pfctl -f custom.conf`) instead of flushing just our anchor.
**How to avoid:** Always use anchor-scoped operations: `pfctl -a com.azad.killswitch -F all` to flush only our rules. Never touch the root ruleset or Apple's anchors.
**Warning signs:** AirDrop, iCloud, or other Apple services stop working after enabling/disabling kill switch.

### Pitfall 6: Orphaned pf State After App Update
**What goes wrong:** User updates azad binary while kill switch is active. New version doesn't know about old state.
**Why it happens:** State file format changed or state file was not migrated.
**How to avoid:** Kill switch state format should be stable and backwards-compatible. Always check for state file on startup regardless of version.

## Code Examples

### Generating pf Rules for a Connection
```go
// Source: vpn-kill-switch.com/post/pf/ + OpenBSD pf.conf(5) man page
func GenerateRules(serverIP string, serverPort int) string {
    return fmt.Sprintf(`# Azad Kill Switch - generated rules
# Anchor: com.azad.killswitch

set block-policy drop

# Allow loopback (local proxy)
pass quick on lo0 all

# Allow traffic to VPN server
pass out quick proto {tcp, udp} from any to %s port %d

# Allow DHCP
pass quick proto {tcp, udp} from any port 67:68 to any port 67:68

# Allow DNS (for initial resolution)
pass out quick proto {tcp, udp} from any to any port 53

# Block everything else (IPv4 and IPv6)
block out all
block in all
`, serverIP, serverPort)
}
```

### Enabling Kill Switch with Privilege Escalation
```go
// Source: Apple Developer Forums + osascript documentation
func Enable(rules string) error {
    // Pipe rules to pfctl via osascript privilege escalation
    // Use base64 to safely pass rules through shell escaping
    encoded := base64.StdEncoding.EncodeToString([]byte(rules))
    command := fmt.Sprintf(
        "echo %s | base64 -d | /sbin/pfctl -a com.azad.killswitch -f - && /sbin/pfctl -E",
        encoded,
    )
    return runPrivileged(command)
}

func Disable() error {
    command := "/sbin/pfctl -a com.azad.killswitch -F all"
    return runPrivileged(command)
}
```

### Checking Kill Switch State on Startup
```go
// Source: Existing lifecycle/cleanup.go pattern in codebase
func RecoverKillSwitch(state *lifecycle.ProxyState) error {
    if !state.KillSwitchActive {
        return nil
    }
    // Kill switch was active when app last ran.
    // pf rules are still loaded in the kernel.
    // Options: reconnect or cleanup.
    return nil // Caller decides: reconnect or cleanup
}
```

### SIGHUP Handler for Terminal Close
```go
// Source: Go os/signal package documentation
func handleSIGHUP() {
    c := make(chan os.Signal, 1)
    signal.Notify(c, syscall.SIGHUP)
    go func() {
        <-c
        // Terminal is closing. Kill switch pf rules will persist
        // in the kernel (this is the desired behavior).
        // Log but don't clean up -- that's the whole point.
        log.Println("Terminal closing. Kill switch rules remain active.")
        // Clean up proxy state but keep kill switch state
        os.Exit(0)
    }()
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Modify /etc/pf.conf directly | Use named anchors (pfctl -a) | macOS 10.7+ | No risk of system config corruption; survives macOS updates |
| ipfw firewall | pf/pfctl | macOS 10.7 (Lion) | ipfw was removed; pfctl is the only option |
| Require `sudo` on CLI | osascript with admin privileges | Always available on macOS | Native password dialog with Touch ID support |
| launchd for rule persistence | Session-based rules (non-persistent) | Design choice | Reboot always restores normal internet; safer for users |

**Deprecated/outdated:**
- ipfw: Removed from macOS since Lion (10.7). Not available.
- `/etc/pf.conf` modification: macOS updates overwrite this file. Use anchors.

## Open Questions

1. **DNS Rule Scope**
   - What we know: DNS (port 53) must be allowed for the VPN server hostname to resolve. After connection, DNS should go through the proxy.
   - What's unclear: Should we allow DNS permanently or only during connection establishment? Blocking DNS after connection could cause issues if the proxy DNS fails.
   - Recommendation: Allow DNS always. The system proxy routes DNS through the SOCKS5 proxy when connected anyway. The DNS `pass` rule is a safety net, not a leak vector.

2. **pf Enable State**
   - What we know: `pfctl -E` enables pf. `pfctl -d` disables pf entirely. macOS may already have pf enabled (for Application Firewall).
   - What's unclear: If pf is already enabled by macOS, should we call `pfctl -E` again? Is it safe to call `pfctl -d` on cleanup (it would disable ALL pf, including Apple's rules)?
   - Recommendation: Use `pfctl -E` to enable (it's idempotent -- just increments a reference count). On cleanup, only flush our anchor (`pfctl -a com.azad.killswitch -F all`); do NOT call `pfctl -d`. This preserves Apple's pf state.

3. **Kill Switch + Headless Connect**
   - What we know: `azad connect --kill-switch` should enable kill switch in headless mode.
   - What's unclear: In headless mode, `osascript` may not work (no GUI for password dialog). SSH sessions won't have a display.
   - Recommendation: In headless mode, require the user to run `sudo azad connect --kill-switch` or handle the `osascript` failure gracefully with a message: "Kill switch requires GUI access for privilege escalation. Use `sudo azad connect --kill-switch` in headless mode."

4. **TUI Keybinding for Kill Switch Toggle**
   - What we know: KILL-05 says "manual toggle (TUI keybinding or `azad connect --kill-switch`)".
   - What's unclear: Which key? Should it be a two-step confirmation (since it requires admin password)?
   - Recommendation: Use `K` (uppercase) as the toggle key. Show a confirmation overlay first ("Enable kill switch? This requires admin privileges. [y/n]"), then trigger osascript on confirmation.

## Sources

### Primary (HIGH confidence)
- [OpenBSD pf.conf(5) man page](https://man.openbsd.org/pf.conf) - pf rule syntax, user keyword, anchor system, block/pass semantics
- [macOS pf.conf(5) man page](https://www.unix.com/man-page/osx/5/pf.conf/) - Confirmed macOS pf supports `user` keyword for per-UID filtering
- [Go os/signal package](https://pkg.go.dev/os/signal) - SIGHUP handling for terminal close detection
- Existing codebase: `internal/lifecycle/cleanup.go`, `internal/sysproxy/`, `internal/cli/root.go` - Established patterns for state management and privilege execution

### Secondary (MEDIUM confidence)
- [vpn-kill-switch.com/post/pf/](https://vpn-kill-switch.com/post/pf/) - Verified pf kill switch rule patterns with anchor approach
- [Neil Sabol's pfctl blog](https://blog.neilsabol.site/post/quickly-easily-adding-pf-packet-filter-firewall-rules-macos-osx/) - macOS-specific pfctl usage, anchor persistence behavior
- [Apple Support: Terminal close behavior](https://support.apple.com/guide/terminal/open-or-quit-terminal-apd5265185d-f365-44cb-8b09-71a064a42125/mac) - Terminal.app shows "processes running" dialog on close
- [Apple Developer Forums: BSD privilege escalation](https://developer.apple.com/forums/thread/708765) - osascript with administrator privileges approach

### Tertiary (LOW confidence)
- [GitHub: VPN-Kill-Switch-MacOS](https://github.com/Aybee2k8/VPN-Kill-Switch-MacOS) - Community kill switch implementation (not verified in detail)
- [GitHub: mullvad/pfctl-rs](https://github.com/mullvad/pfctl-rs) - Mullvad's Rust pfctl library (reference architecture, not directly usable)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - pfctl is the only firewall option on macOS; well-documented, stable API
- Architecture: MEDIUM - Anchor approach is well-established; SOCKS5 proxy kill switch (vs VPN tunnel) is less commonly documented but the pf rules are straightforward
- Pitfalls: HIGH - Common pitfalls (IPv6 leak, breaking Apple anchors, DNS leak) are well-known in the VPN community

**Research date:** 2026-02-26
**Valid until:** 2026-03-26 (stable domain -- pfctl/pf hasn't changed significantly in years)

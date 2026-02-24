# Pitfalls Research

**Domain:** Go TUI VPN client wrapping Xray-core
**Researched:** 2026-02-24
**Confidence:** HIGH (verified against official docs, multiple community sources, and real-world projects)

## Critical Pitfalls

### Pitfall 1: Blocking the Bubbletea Event Loop

**What goes wrong:**
Performing I/O operations (network pings, subscription fetches, Xray-core lifecycle calls, file reads) directly inside `Update()` or `View()` freezes the entire TUI. The UI becomes unresponsive -- no keypresses register, no rendering updates, the app appears crashed. This is the single most common mistake in Bubbletea applications.

**Why it happens:**
Bubbletea processes messages sequentially through `Update()` and `View()`. Any blocking call in either function stalls the entire event loop. Developers coming from imperative UI frameworks expect to "just call the function" and update state afterward. The Elm Architecture pattern requires a different mental model: dispatch a `tea.Cmd`, receive the result as a `tea.Msg`.

**How to avoid:**
- Every I/O operation must be a `tea.Cmd` that returns a `tea.Msg`.
- Use `tea.Batch()` to fire multiple concurrent commands (e.g., pinging all servers simultaneously).
- Use `tea.Sequence()` when command ordering matters.
- Set explicit timeouts on all HTTP clients and network operations inside commands -- Bubbletea has no built-in command timeout, so a hung HTTP request blocks its goroutine until program termination.
- Pattern: `func pingServer(addr string) tea.Cmd { return func() tea.Msg { ... } }`

**Warning signs:**
- UI freezes when connecting, pinging, or fetching subscriptions.
- Keypresses queue up and fire all at once after an operation completes.
- `View()` function calls anything that touches the network or filesystem.

**Phase to address:**
Phase 1 (Core Architecture). Establish the Cmd/Msg pattern from the very first line of TUI code. Retrofitting this later means rewriting every feature.

---

### Pitfall 2: Orphaned Xray-core Processes on Crash/Exit

**What goes wrong:**
When the Go app crashes, panics, gets SIGKILL'd, or the user closes the terminal, the Xray-core process (whether external binary or in-process goroutines) keeps running. The SOCKS/HTTP proxy ports remain bound. On next launch, the app cannot bind the same ports and fails. Users end up with invisible proxy processes consuming resources and holding network state.

**Why it happens:**
Go's `defer` does not run on SIGKILL or `os.Exit()`. Child processes created with `exec.Command` are not automatically killed when the parent dies. Bubbletea catches panics for terminal restoration but does not clean up application-specific resources. On macOS and Linux, orphaned child processes get reparented to init/launchd, not killed.

**How to avoid:**
- Use process groups (`cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}`) on Unix to kill the entire group on shutdown.
- Register signal handlers for SIGTERM and SIGINT that explicitly stop Xray and unset system proxy before exiting.
- Implement a PID file / lock file mechanism: on startup, check if a previous instance's Xray process is still running and kill it.
- If using xray-core as a Go library (recommended), use `context.Context` with cancellation to propagate shutdown to the Xray instance. Call `instance.Close()` in cleanup.
- Add a startup health check: attempt to connect to the proxy port; if something is already bound, offer to kill it.

**Warning signs:**
- "Address already in use" errors on startup.
- `lsof -i :1080` shows an xray process from a previous session.
- System proxy still set after the app has exited.

**Phase to address:**
Phase 1 (Core Architecture). Process lifecycle management must be designed into the foundation, not bolted on.

---

### Pitfall 3: Xray-core as External Binary vs. Go Library -- Wrong Choice

**What goes wrong:**
Choosing to wrap Xray as an external binary (exec.Command) seems simpler but creates a cascade of problems: binary distribution per platform/arch, version synchronization, asset file management (geoip.dat, geosite.dat), IPC complexity, and the orphaned process problem above. Conversely, importing xray-core as a Go library bloats the binary and pulls in a massive dependency tree but eliminates most of these issues.

**Why it happens:**
The existing bash tool runs xray as an external binary, so the natural instinct is to replicate this pattern. The xray-core Go library API is not well-documented for embedding use cases. Developers underestimate the operational complexity of bundling and managing external binaries across platforms.

**How to avoid:**
Import xray-core as a Go library. The API is straightforward:
```go
config, err := core.LoadConfig("json", configBytes)
instance, err := core.New(config)
err = instance.Start()
// ... later ...
instance.Close()
```
This gives you: single binary distribution, in-process lifecycle control, no IPC, no version sync issues, shared Go context for cancellation. The binary size increase (~20-30MB) is acceptable for a desktop tool and can be reduced with `go build -ldflags="-s -w"` and UPX compression.

You still need geoip.dat and geosite.dat files. Handle this with: (a) embed them via `//go:embed` if size is acceptable (~10MB each), or (b) auto-download on first run to `~/.config/azad/` with integrity verification.

**Warning signs:**
- Planning to "download xray binary on first run" or "expect user to install xray."
- Writing IPC/gRPC code to communicate with an external xray process.
- Architecture has "xray manager" that shells out with `exec.Command`.

**Phase to address:**
Phase 1 (Core Architecture). This is the most consequential architectural decision. Library import must be validated in a spike before committing to it.

---

### Pitfall 4: System Proxy State Left Dirty

**What goes wrong:**
The app sets the OS system proxy (macOS `networksetup`, Linux `gsettings`/env vars, Windows registry) but crashes or is force-killed before unsetting it. The user's entire system routes through a dead proxy. Web browsing, package managers, git -- everything breaks. Users who do not understand proxy settings are completely stuck.

**Why it happens:**
System proxy modification is a stateful side effect with no automatic rollback. The existing bash tool already has this problem (documented in CONCERNS.md). Signal handlers and defer statements do not cover all exit paths (SIGKILL, power loss, kernel panic).

**How to avoid:**
- Store proxy state in a file (`~/.config/azad/proxy-state.json`) with timestamp. On startup, check if state file indicates proxy was set but app is not running -- offer to clean up.
- Implement a "watchdog" pattern: a separate lightweight goroutine or even a launchd/systemd service that monitors the main process and cleans up proxy settings if it dies.
- Simpler approach: on startup, always unset system proxy first, then set it only when actively connecting. Document clearly that `azad --cleanup` can fix a dirty state.
- Show the current proxy state in the TUI status bar so users always know.
- Consider NOT setting system proxy by default. Default to SOCKS/HTTP proxy on localhost and let the user opt-in to system proxy. This limits blast radius.

**Warning signs:**
- Users report "internet stopped working after closing azad."
- No `--cleanup` or `--reset` command exists.
- System proxy toggle has no confirmation prompt.

**Phase to address:**
Phase 2 (Connection Management). Must be solved before any system proxy feature ships. Provide cleanup command even before the full TUI is ready.

---

### Pitfall 5: Race Conditions from Mutating Model Outside Update()

**What goes wrong:**
Goroutines spawned by commands directly modify the model struct instead of sending messages back through the Bubbletea event loop. This causes data races, intermittent crashes, and corrupted state that is extremely difficult to reproduce and debug.

**Why it happens:**
Go makes it easy to close over model fields in goroutines. The compiler does not prevent it. The bug manifests only under specific timing conditions, so tests pass on CI but fail in production. Developers who are not deeply familiar with the Elm Architecture pattern naturally reach for shared mutable state.

**How to avoid:**
- Strict rule: goroutines spawned by `tea.Cmd` functions must NEVER reference or modify model fields. They receive input as function parameters and return results as `tea.Msg` values.
- Run `go test -race ./...` in CI on every commit. The race detector catches most of these.
- Use the pattern: `func doThing(input string) tea.Cmd { return func() tea.Msg { result := ...; return thingDoneMsg{result} } }`
- Code review checklist: "Does any Cmd closure capture a pointer to the model?"

**Warning signs:**
- Intermittent panics with "concurrent map read and map write" or similar.
- Tests pass locally but fail on CI (or vice versa).
- `-race` flag in `go test` reports data races.

**Phase to address:**
Phase 1 (Core Architecture). Establish patterns and CI enforcement from day one.

---

### Pitfall 6: Subscription/URI Parsing Fragility

**What goes wrong:**
VLESS, VMess, Trojan, and Shadowsocks URI schemes are not formally standardized (despite RFC3986 proposals). Real-world subscription services emit URIs with: base64url vs standard base64 encoding, missing padding, percent-encoded vs raw Unicode fragments, IPv6 addresses in brackets, unusual query parameter ordering, mixed protocols in one subscription, non-standard extensions. A parser that works with one subscription provider breaks with another.

**Why it happens:**
Developers test against one or two subscription providers and call it done. The bash tool already has this problem (documented in CONCERNS.md: "VLESS URI parsing uses string manipulation... edge cases may break"). The VMess URI scheme is particularly problematic as it uses a base64-encoded JSON blob with no formal spec.

**How to avoid:**
- Use Go's `net/url` package for URI parsing, not string splitting.
- Handle both `base64.StdEncoding` and `base64.RawURLEncoding` for subscription decoding. Try one, fall back to the other.
- Build a comprehensive test suite with real-world URIs from at least 5 different subscription providers.
- Parse defensively: every field extraction must handle "not present" gracefully.
- VMess parsing: decode base64 -> parse JSON -> validate all fields exist before using.
- VLESS parsing: handle fragment (`#name`) separately, URL-decode it, sanitize for display.
- Use `|` as delimiter for server storage? The existing tool does this and it breaks on server names containing `|`. Use a proper serialization format (JSON, TOML) instead.

**Warning signs:**
- Users report "failed to parse subscription" with specific providers.
- Base64 decode errors on some subscriptions but not others.
- Server names display as garbled text (encoding issue).

**Phase to address:**
Phase 2 (Protocol Support). Build the parser with extensive test fixtures before integrating into the TUI.

---

### Pitfall 7: Terminal Restoration Failure

**What goes wrong:**
The TUI uses Bubbletea's alternate screen buffer and raw terminal mode. If the app panics inside a `tea.Cmd` goroutine (not the main event loop), or if cleanup is interrupted, the terminal is left in raw mode: no echo, no line editing, broken cursor. Users must type `reset` blindly to recover. This creates a terrible first impression and erodes trust.

**Why it happens:**
Bubbletea's built-in panic recovery (`CatchPanics`, on by default) only catches panics in the main goroutine. Panics in Cmd goroutines are caught by the runtime but not by Bubbletea's recovery. Also, if the app sets up the alternate screen but crashes before `Program.Quit()` sends the cleanup escape sequences, the terminal stays in alternate screen mode.

**How to avoid:**
- Wrap every `tea.Cmd` function body in a `defer func() { if r := recover(); r != nil { ... } }()` that returns an error message instead of panicking.
- Add a `recover()` wrapper utility: `func safeCmd(fn func() tea.Msg) tea.Cmd` that catches panics and converts them to error messages.
- Ship a `azad --reset-terminal` flag that just outputs terminal reset escape sequences and exits.
- Test the panic-in-Cmd path explicitly: inject a panic in a test Cmd and verify the terminal recovers.

**Warning signs:**
- After a crash, typing in the terminal produces no visible output.
- Users report having to close and reopen their terminal.
- `CatchPanics` is disabled (never do this in production).

**Phase to address:**
Phase 1 (Core Architecture). The `safeCmd` wrapper should be one of the first utilities written.

---

## Technical Debt Patterns

Shortcuts that seem reasonable but create long-term problems.

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Hardcoded ports (1080/8080) | No config complexity | Conflicts with other tools, cannot run multiple instances | Never -- use config with sensible defaults from day one |
| Storing servers in flat file with delimiter | Simple to implement | Breaks on special characters in names, no schema evolution | Never -- use JSON/TOML from day one |
| Shelling out to `networksetup` for macOS proxy | Works immediately | Requires parsing command output, breaks on macOS updates, no Linux/Windows | MVP only -- abstract behind an interface for cross-platform |
| Skipping geoip/geosite asset management | Smaller binary, simpler startup | Routing rules do not work, users get confused by missing features | Never -- either embed or auto-download with clear error messages |
| Global mutable state for connection status | Easy to access from anywhere | Race conditions, untestable, impossible to support multiple connections later | Never -- pass state through model |
| Ignoring Windows support initially | Faster development | Windows users are the majority of VPN tool users; adding it later means restructuring all OS-specific code | Acceptable for alpha, but abstract OS operations behind interfaces from the start |

## Integration Gotchas

Common mistakes when connecting to external services and system APIs.

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| Xray-core library | Importing without registering JSON config loader (only protobuf works by default) | Import `_ "github.com/xtls/xray-core/main/json"` to register the JSON config format |
| Xray-core library | Not importing protocol packages (xray cannot handle VLESS/VMess without protocol registration) | Import all needed protocol packages as side-effect imports: `_ "github.com/xtls/xray-core/proxy/vless/inbound"`, etc. |
| Xray-core geoip/geosite | Assuming files are optional -- routing rules silently fail without them | Either embed via `//go:embed` or download on first run; verify presence at startup with clear error |
| Subscription URLs | Trusting subscription content blindly after HTTPS fetch | Validate parsed URIs structurally; limit server count per subscription (DoS via huge subscription); sanitize all display strings |
| macOS system proxy | Using `networksetup -setwebproxy` without checking which network service is active | Query active network service first with `networksetup -listnetworkserviceorder`; set proxy on the active service only |
| HTTP/SOCKS proxy ports | Binding to ports without checking availability first | Use `net.Listen("tcp", ":0")` to find available port, or check with `net.DialTimeout` before binding |

## Performance Traps

Patterns that work at small scale but fail as server count or usage grows.

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Sequential server pinging | UI frozen for minutes with 50+ servers (3s timeout each = 150s) | Use `tea.Batch()` to ping all servers concurrently; cap concurrent goroutines with a semaphore (e.g., 20 at a time) | >10 servers |
| Re-rendering entire server list on every tick | High CPU, flickering, dropped frames in the TUI | Only re-render changed components; use viewport for long lists; throttle tick rate for stats updates to 1-2 Hz | >50 servers in list |
| Polling connection stats by shelling out | Each poll spawns a process, parses output | Use xray-core library's gRPC stats API or in-process stats collection | When stats update rate > 1 Hz |
| Loading full geoip.dat into memory on every config change | Memory spikes of 50-100MB, GC pauses | Load once at startup, cache in memory, reuse across config reloads | Always (these files are large) |
| Synchronous subscription fetch blocking server list display | User stares at empty screen while subscriptions download | Show cached server list immediately; fetch subscriptions in background Cmd; show progress indicator | Any network latency > 500ms |

## Security Mistakes

Domain-specific security issues beyond general web security.

| Mistake | Risk | Prevention |
|---------|------|------------|
| Storing UUID/credentials in plaintext config | Credential theft if config file is exposed (git commit, backup, screen share) | Use OS keychain where available (macOS Keychain, Linux secret-service); at minimum set file permissions to 0600; add config path to default .gitignore |
| Not validating subscription URL before fetching | SSRF-like attacks: crafted subscription URL could probe internal network | Validate URL scheme (https only), reject private IP ranges, set fetch timeout, limit response size |
| Passing unsanitized VLESS fragment (#name) to display | Terminal escape sequence injection: crafted server name could execute terminal commands or corrupt display | Strip ANSI escape sequences and control characters from all display strings; use lipgloss rendering which escapes by default |
| Running xray-core as root/admin unnecessarily | Full system compromise if xray has a vulnerability | Never require root. Bind to ports >1024. If system proxy needs elevated permissions, use a minimal privilege-escalation helper |
| Not verifying geoip/geosite file integrity | Tampered routing files could redirect traffic through malicious servers | Verify SHA256 checksums on download; pin expected checksums per release |
| Logging sensitive data (UUIDs, server addresses) | Credential leakage via log files | Never log UUIDs or full server URIs; log only server tags/names; make debug logging opt-in |

## UX Pitfalls

Common user experience mistakes in terminal VPN client applications.

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| No feedback during long operations (connecting, pinging) | User thinks app is frozen, presses Ctrl+C, kills connection mid-setup | Show spinner/progress bar for every operation >200ms; show elapsed time for pings |
| Cryptic error messages from xray-core | "transport: authentication handshake failed" means nothing to users | Map common xray errors to human-readable messages: "Server rejected connection -- check your credentials" |
| Auto-connecting without showing what is happening | User does not know which server was selected or why | Show "Connecting to [server name] ([country])..." with the selection reason (fastest, last-used) |
| Vim keybindings with no documentation | Users press random keys, trigger unexpected actions | Show key hints in the footer/status bar; support `?` for help overlay; support both vim and arrow key navigation |
| No way to cancel in-progress operations | User starts pinging 100 servers and cannot stop it | Every long operation must be cancellable (Esc key sends cancel message, propagated via context cancellation) |
| Terminal too small for the layout | Rendering breaks, text overlaps, components disappear | Detect minimum terminal size on startup; show "resize your terminal to at least 80x24" message; handle `tea.WindowSizeMsg` to adapt layout |

## "Looks Done But Isn't" Checklist

Things that appear complete but are missing critical pieces.

- [ ] **Connection established:** Often missing verification -- ping a known IP-check service (e.g., ifconfig.me) through the proxy to confirm traffic actually routes through the VPN, not just that xray started
- [ ] **Server list populated:** Often missing deduplication -- same server from multiple subscriptions appears multiple times; implement URI-based deduplication
- [ ] **System proxy set:** Often missing DNS configuration -- HTTP/SOCKS proxy is set but DNS still leaks to ISP; configure DNS-over-HTTPS or route DNS through the proxy
- [ ] **Cross-platform support:** Often missing Windows terminal compatibility -- ANSI escape codes, terminal resize, raw mode all behave differently on Windows; test in cmd.exe, PowerShell, and Windows Terminal
- [ ] **Config file migration:** Often missing version field -- config format changes between versions but no migration path exists; include a `version` field from v1 and write migration logic
- [ ] **Graceful disconnect:** Often missing proxy cleanup -- xray stops but system proxy is still set, or SOCKS port is unbound but HTTP proxy port lingers; verify ALL state is cleaned up
- [ ] **Error handling:** Often missing retry logic -- a single failed ping marks server as "dead" permanently; implement retry with backoff for transient network errors
- [ ] **Single instance:** Often missing mutex -- user launches azad twice, both instances fight over the same proxy ports; use a file lock or Unix socket to enforce single instance

## Recovery Strategies

When pitfalls occur despite prevention, how to recover.

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Orphaned xray process | LOW | Ship `azad --kill` that finds and kills any xray processes started by azad (use PID file); document in troubleshooting guide |
| Dirty system proxy | LOW | Ship `azad --cleanup` that unconditionally unsets system proxy on all interfaces; run automatically on startup if state file indicates dirty shutdown |
| Corrupted terminal | LOW | Ship `azad --reset-terminal` that outputs reset escape sequences; document "type reset and press Enter" in README |
| Corrupted config file | MEDIUM | Keep automatic backups of last-known-good config; validate config on load and fall back to backup; ship `azad --reset-config` |
| Binary size too large from xray-core library | MEDIUM | Use build tags to exclude unused protocols; apply `-ldflags="-s -w"`; consider UPX compression; accept that 30-40MB is normal for this type of tool |
| Race condition in production | HIGH | Cannot recover at runtime; requires code fix. Prevention (race detector in CI) is the only viable strategy |

## Pitfall-to-Phase Mapping

How roadmap phases should address these pitfalls.

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Blocking event loop | Phase 1: Core Architecture | All I/O goes through tea.Cmd; no network/file calls in Update() or View(); enforce via code review |
| Orphaned processes | Phase 1: Core Architecture | Kill azad with SIGTERM, verify xray process also dies; kill with SIGKILL, restart, verify cleanup happens |
| Xray integration choice (library vs binary) | Phase 1: Core Architecture | Spike validates: library import compiles, cross-compiles, binary size is acceptable, Start/Close lifecycle works |
| System proxy dirty state | Phase 2: Connection Management | Force-kill azad during active connection; verify system proxy is clean on restart; test on macOS and Linux |
| Race conditions | Phase 1: Core Architecture | CI runs `go test -race ./...` on every commit; zero race conditions allowed |
| URI parsing fragility | Phase 2: Protocol Support | Test suite includes URIs from 5+ subscription providers; both base64 and base64url; IPv6; Unicode names; VMess JSON blobs |
| Terminal restoration | Phase 1: Core Architecture | Inject panic in a tea.Cmd; verify terminal recovers; test `--reset-terminal` flag |
| Responsive layout | Phase 1: Core Architecture | Test at 80x24 (minimum), 120x40 (normal), 200x60 (large); verify no rendering artifacts on resize |
| Cross-platform proxy | Phase 3: Cross-Platform | Test system proxy set/unset on macOS, Ubuntu, Windows; test with and without active network |
| Antivirus false positives | Phase 4: Distribution | Sign Windows binaries with code signing certificate; submit to Microsoft for analysis; test with common AV tools |
| Geoip/geosite management | Phase 2: Protocol Support | Start with no dat files present; verify auto-download works; verify routing rules function correctly after download |

## Sources

- [Tips for building Bubble Tea programs](https://leg100.github.io/en/posts/building-bubbletea-programs/) -- Comprehensive pitfalls guide, HIGH confidence
- [Commands in Bubble Tea (official)](https://charm.land/blog/commands-in-bubbletea/) -- Cmd/Msg pattern, HIGH confidence
- [Bubbletea GitHub - terminal settings not restored after panics](https://github.com/charmbracelet/bubbletea/issues/1459) -- Terminal restoration issue, HIGH confidence
- [Bubbletea DeepWiki - Core Components](https://deepwiki.com/charmbracelet/bubbletea/2-core-components) -- Concurrency architecture, MEDIUM confidence
- [XTLS/Xray-core DeepWiki](https://deepwiki.com/XTLS/Xray-core) -- Architecture overview, MEDIUM confidence
- [XTLS/Xray-core pkg.go.dev](https://pkg.go.dev/github.com/xtls/xray-core/core) -- Library API, HIGH confidence
- [XTLS/libXray](https://github.com/XTLS/libXray) -- Official library wrapper, API stability concerns, MEDIUM confidence
- [Kill child processes and all children in Go](https://medium.com/@felixge/killing-a-child-process-and-all-of-its-children-in-go-54079af94773) -- Process group management, HIGH confidence
- [golang-nuts: Kill child processes on parent termination](https://groups.google.com/g/golang-nuts/c/nIt7DDTMAlA) -- Cross-platform process cleanup, HIGH confidence
- [getlantern/sysproxy](https://github.com/getlantern/sysproxy) -- Cross-platform system proxy library, MEDIUM confidence
- [VMess URI scheme proposal](https://github.com/v2ray/v2ray-core/issues/1487) -- Lack of standardized URI scheme, HIGH confidence
- [Go binaries antivirus false positives](https://github.com/golang/go/issues/44323) -- Distribution challenge, HIGH confidence
- [Xray-core GeoIP and GeoSite Data](https://deepwiki.com/XTLS/Xray-core/6.3-geoip-and-geosite-data) -- Asset management, MEDIUM confidence
- .planning/codebase/CONCERNS.md -- Existing codebase pitfalls, HIGH confidence (firsthand)

---
*Pitfalls research for: Go TUI VPN client wrapping Xray-core (Azad)*
*Researched: 2026-02-24*

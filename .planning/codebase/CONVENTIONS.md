# Coding Conventions

**Analysis Date:** 2026-02-24

## Naming Patterns

**Files:**
- Snake case with `.sh` extension: `menu.sh`, `install.sh`, `link-to-full-config.sh`
- Descriptive names matching function: `vless-link-to-config.sh` for VLESS parsing, `setup-azad.sh` for shell setup
- Entry point/main script: `menu.sh` (743 lines, primary interactive CLI)

**Functions:**
- Snake case: `add_vless()`, `ping_server()`, `get_host_port()`, `draw_box_top()`
- Purpose-driven names: `get_network_service()`, `set_system_proxy()`, `test_full_tunnel()`
- Prefix patterns:
  - `get_*`: Fetch/retrieve data
  - `set_*`: Configure or enable
  - `unset_*`: Disable or clear
  - `is_*`: Boolean check
  - `do_*`: Execute primary action
  - `fetch_*`: Remote retrieval

**Variables:**
- Upper case for constants and configuration: `DIR`, `DATA`, `SERVERS`, `SUBS`, `PROXY_PID_FILE`, `PING_TIMEOUT=3`
- Lower case for local/temporary variables: `link`, `name`, `url`, `choice`, `pid`
- Descriptive: `LAST_SUB_URL`, `PROXY_PID_FILE`, `direct_ip`, `vpn_ip`
- Color codes as short caps: `R` (reset), `B` (bold), `D` (dim), `C` (cyan), `G` (green), `Y` (yellow), `M` (magenta), `RED`, `BL` (blue)

**Types/Data Structures:**
- Pipe-delimited files for structured data: `name|link` format in `servers.txt` and parsed with `${line%%|*}` and `${line#*|}`
- Temporary files via `mktemp`: Used in `remove_server()` to atomically replace file contents

## Code Style

**Formatting:**
- No automated formatter detected (no `.prettierrc`, `eslintrc`, or similar)
- Manual formatting conventions observed:
  - 2-space indentation in functions
  - Comments on separate lines or inline with `#`
  - One statement per line (no chained commands in most cases)

**Linting:**
- No linting tools detected (no eslint, shellcheck config files)
- Style enforced via `set -e` for error exit on failure
- `#!/usr/bin/env bash` shebang for portability

**Error Handling:**
- `set -e` at script start: Causes immediate exit on any command failure (see `menu.sh` line 3)
- Explicit `|| return` or `|| { ... ; return; }` patterns for recoverable errors:
  - Example in `get_host_port()` line 66: `[[ "$link" =~ ^vless:// ]] || return 1`
  - Example in `add_subscription()` lines 139-140: `raw=$(curl -fsSL ... 2>/dev/null) || { msg_err "..."; return; }`
- Suppressed stderr for optional operations: `2>/dev/null` for non-critical commands
- Exit codes used: `return 0` for success, `return 1` for failure in functions; `exit 1` in scripts
- Inline error messages with `msg_err()` helper before return/exit

## Comment Style

**Guidelines observed:**
- Shebang + single-line purpose at file top: `#!/usr/bin/env bash\n# Interactive menu: Add VLESS, Add subscription...`
- Function-level comments before definition: `# Get host:port from vless link (for ping)` (line 63)
- Inline comments for non-obvious logic: `# macOS nc: -z connect only, -w timeout seconds; Linux nc: -z -w or -G` (line 83)
- Explanation comments for complex behavior: `# Strip vless:// and split` (line 13 in vless-link-to-config.sh)
- User-facing comments in interactive prompts: `${D}Paste your server link — it usually starts with vless://${R}` (line 104)
- Intent comments for temporary files: `local tmp; tmp=$(mktemp)` with implicit "replace atomically" pattern

**JSDoc/Documentation:**
- No JSDoc or formal documentation in code
- Comments focus on operation intent, not parameter/return documentation

## Import Organization

**No traditional imports in shell scripts.**

**Configuration loading pattern:**
- Hard-coded directory structure: `DIR="$(cd "$(dirname "$0")" && pwd)"` (line 4)
- Derived paths: `DATA="$DIR/data"`, `SERVERS="$DATA/servers.txt"` (lines 5-6)
- Files sourced or executed (not imported): `"$DIR/run.sh"` called directly in `run_proxy()` (line 302)

**External command usage:**
- POSIX tools: `curl`, `nc` (netcat), `python3`, `grep`, `base64`, `unzip`
- macOS-specific: `networksetup` for proxy configuration (lines 388-395)
- Xray binary: `./xray run -config "$CONFIG"`

## Module Design

**Function organization in `menu.sh` (743 lines):**
- Utility functions first: `line()`, `draw_box_top()`, `msg_ok()`, `msg_err()`, `msg_info()` (lines 40-61)
- Core logic functions: `get_host_port()`, `ping_server()`, `count_servers()` (lines 63-98)
- User-facing features: `add_vless()`, `add_subscription()`, `do_ping()`, `select_server()` (lines 100-281)
- System operations: `run_proxy()`, `run_proxy_bg()`, `stop_proxy()` (lines 283-353)
- Platform-specific: `get_network_service()`, `set_system_proxy()`, `unset_system_proxy()` (lines 355-412)
- Diagnostic functions: `test_full_tunnel()`, `show_status()`, `test_connection()` (lines 439-580)
- Setup functions: `install_shell_helpers()`, `quick_connect()` (lines 638-562)
- Main loop: Infinite `while true` menu dispatcher (lines 666-743)

**Single-responsibility in config generation scripts:**
- `vless-link-to-config.sh`: Parse VLESS link → output JSON fragment
- `link-to-full-config.sh`: Parse VLESS link → write complete config.json
- `setup-azad.sh`: One-time shell integration setup
- `run.sh`: Minimal wrapper to invoke Xray with config

**Exports/Public API:**
- No explicit exports; all scripts are executable entry points
- Functions available only within their defining script; no sourcing pattern observed
- Data files as pseudo-API: `servers.txt`, `subscriptions.txt`, `current.txt`, `last_sub_url.txt`, `proxy.pid`

## Validation and Input Sanitization

**Regex matching for validation:**
- VLESS link validation: `[[ "$link" =~ ^vless:// ]]` (multiple places)
- IP address check: `[[ "$raw" =~ ^[0-9.]+$ ]] || [[ "$raw" =~ ^[0-9a-fA-F:]+$ ]]` (line 416)
- URL validation: `[[ "$url" =~ ^https?:// ]]` (line 185)

**Whitespace handling:**
- `tr -d '[:space:]'` to strip all whitespace from user input (lines 108, 132, 183)
- `tr -d '\r'` to remove carriage returns from subscription lines (line 150)

**Data extraction with parameter expansion:**
- Safe parsing without regex: `${variable%%delimiter}` and `${variable#*delimiter}` patterns
- Example: `HOST="${REST%%:*}"` extracts before colon, `PORT="${REST#*:}"` extracts after (lines 28-29 in vless-link-to-config.sh)

## Performance Patterns

**Subprocess minimization:**
- Pipeline operations for parsing: `grep -c . "$SERVERS" 2>/dev/null` for count (line 96)
- Command substitution: `$(...)` used, but not excessively
- Conditional command execution: `2>/dev/null &&` pattern to suppress errors

**Looping and streaming:**
- `while IFS= read -r line; do ... done < "$FILE"` pattern for file processing (line 226 in do_ping)
- Avoids loading entire file into memory for large server lists

**Background execution:**
- `nohup ... &` for background VPN process (line 324)
- PID stored in file for later control (line 325)

## Bash-specific Features Used

- Arithmetic: `$(( end - start ))` for time delta (line 86)
- String manipulation: `${var%%pattern}`, `${var#*pattern}`, `${var:0:28}` substring (line 232)
- Array-like structures: Pipe-delimited text files instead of arrays
- Process control: `kill -0 $pid` to check if process exists (line 318)
- Conditional expressions: `[[ ... ]]` for tests (recommended over `[ ... ]`)

---

*Convention analysis: 2026-02-24*

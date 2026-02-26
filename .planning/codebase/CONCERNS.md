# Concerns

## Security

### High Priority
- **Unsanitized URL input** — Subscription URLs are passed directly to `curl -fsSL` without validation beyond basic `https?://` check. Malicious URLs could trigger unintended behavior.
- **Shell injection via VLESS links** — Server names extracted from VLESS fragment (`#name`) are used unquoted in some contexts. Crafted link fragments with shell metacharacters could be problematic.
- **Config contains credentials** — `config.json` contains UUID and server details in plaintext. No `.gitignore` present to prevent accidental commits.

### Medium Priority
- **System proxy modification** — `set_system_proxy` and `unset_system_proxy` modify macOS network settings globally. No confirmation prompt before making system-wide changes.
- **No TLS certificate validation** — `curl -fsSL` uses default cert validation, but subscription content is trusted blindly after fetch.

## Technical Debt

### Monolithic menu.sh
- `menu.sh` is 743 lines containing all 18 menu functions inline. No modularity — every function lives in one file.
- Duplicated VLESS parsing logic between `link-to-full-config.sh` and `vless-link-to-config.sh` — the two scripts do nearly the same thing with slightly different output.

### Hardcoded Values
- Xray version `v26.2.6` hardcoded in `install.sh:4` — no auto-update mechanism
- Port numbers `1080` (SOCKS) and `8080` (HTTP) hardcoded throughout `menu.sh`, `run.sh`, `link-to-full-config.sh`, and `config.template.json`
- Ping timeout `3` seconds hardcoded in `menu.sh:11`
- `curl --max-time` values vary (8, 10, 15) across different functions

### Fragile Parsing
- VLESS URI parsing uses string manipulation (`${var%%pattern}`, `${var#*pattern}`) instead of proper URL parsing. Edge cases (IPv6 addresses, encoded characters, unusual query params) may break.
- Subscription content detection (`grep -q '^[A-Za-z0-9+/=]*$'` for base64) is a heuristic that can misidentify content format.
- Server file format (`name|link`) uses `|` as delimiter — server names containing `|` would corrupt the data.

## Performance

- **Sequential ping** — `do_ping` tests servers one at a time with 3-second timeout each. With many servers, this blocks for minutes. No parallel execution.
- **Python dependency for timing** — `ping_server` shells out to `python3 -c "import time; ..."` twice per server for millisecond timing. Could use bash `$SECONDS` or `date +%s%N` instead.

## Missing Features

- **No .gitignore** — `config.json`, `xray`, `data/`, `*.dat`, and `path/` should all be gitignored
- **No update mechanism** — No way to update Xray version without editing `install.sh`
- **No server import/export** — Cannot backup or share server lists
- **No logging** — Menu interactions and errors not logged; only xray background logs exist
- **macOS only** — Hardcoded `networksetup` commands and macOS binary downloads; no Linux/Windows support

## Leftover/Unused

- `path/to/venv/` — Python virtual environment directory containing only pip. Appears unused by the project; likely leftover from development.
- `README.md` — Contains upstream Xray-core README, not project-specific documentation.

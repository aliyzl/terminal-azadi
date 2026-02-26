# Deferred Items - Phase 08 Split Tunneling

## Pre-existing Build Breakage: TUI split tunnel files

**Found during:** 08-02 Task 1 verification
**Description:** Untracked/uncommitted TUI files from plan 08-03 break `go build ./...`:
- `internal/tui/split_tunnel.go` references `m.splitTunnelIdx` (undefined field) and `splitTunnelSavedMsg` (undefined type)
- `internal/tui/app.go` (modified) imports splittunnel unused, references `SetSplitTunnel` (undefined method)
- `internal/tui/keys.go`, `internal/tui/statusbar.go`, `internal/tui/input.go`, `internal/tui/messages.go` also modified

**Impact:** `go build ./...` fails. Package-specific builds (`killswitch`, `engine`, `splittunnel`, `config`) succeed.
**Resolution:** Will be resolved when 08-03 (TUI integration) is executed properly.

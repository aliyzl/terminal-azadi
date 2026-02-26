---
status: complete
phase: 06-distribution
source: [06-01-SUMMARY.md, 06-02-SUMMARY.md, 06-03-SUMMARY.md, 06-04-SUMMARY.md]
started: 2026-02-26T17:30:00Z
updated: 2026-02-26T17:30:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Binary builds and reports version
expected: `go build -ldflags "-X main.version=test" ./cmd/azad` compiles without errors. Running `./azad --version` prints a version string.
result: pass

### 2. GoReleaser config validates
expected: `.goreleaser.yaml` contains all required sections: version 2 header, builds (darwin/linux x amd64/arm64), archives, checksum (sha256), sboms, changelog, release, brews (homebrew-tap), nfpms (deb/rpm), aurs (azad-bin), snapcrafts (classic).
result: pass

### 3. Geo asset unit tests pass
expected: `go test ./internal/geoasset/ -v` passes all 3 tests: download-when-missing, skip-existing, checksum-mismatch.
result: pass

### 4. Geo pre-flight integrated in engine
expected: `internal/engine/engine.go` calls `geoasset.EnsureAssets(dataDir)` BEFORE `core.New(coreConfig)`. This prevents Xray panics when geo files are missing on first run.
result: pass

### 5. Platform-gated cleanup
expected: `internal/lifecycle/cleanup.go` wraps sysproxy and killswitch calls in `runtime.GOOS == "darwin"` checks. Linux gets informational messages instead of calls to macOS-only binaries.
result: pass

### 6. Install script is valid POSIX
expected: `sh -n scripts/install.sh` passes with no syntax errors. Script uses `#!/bin/sh`, no bashisms. Contains uname-based OS/arch detection, SHA256 checksum verification, and /usr/local/bin → ~/.local/bin fallback.
result: pass

### 7. Release workflow structure
expected: `.github/workflows/release.yml` triggers on `v*` tag push, uses `actions/checkout@v4` with `fetch-depth: 0`, `actions/setup-go@v5`, and `goreleaser/goreleaser-action@v6` with `release --clean`.
result: pass

## Summary

total: 7
passed: 7
issues: 0
pending: 0
skipped: 0

## Gaps

[none yet]

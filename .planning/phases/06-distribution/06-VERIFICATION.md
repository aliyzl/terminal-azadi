---
phase: 06-distribution
verified: 2026-02-26T21:30:00Z
status: passed
score: 7/7 must-haves verified
re_verification: false
human_verification:
  - test: "curl -sSL <install-url> | sh on a fresh macOS machine"
    expected: "Script detects darwin/arm64 or darwin/amd64, downloads correct binary, verifies checksum, installs to /usr/local/bin or ~/.local/bin, and prints 'Run azad --help to get started'"
    why_human: "Requires a real GitHub Release to exist and a fresh machine without azad pre-installed; cannot simulate live GitHub API in static analysis"
  - test: "brew install leejooy96/tap/azad on macOS with Homebrew"
    expected: "Formula installs binary and shell completions (bash/zsh/fish) to Homebrew directories; azad --version succeeds after install"
    why_human: "Requires the leejooy96/homebrew-tap repo to exist and a tagged release to have been published via GoReleaser; cannot verify without a live release"
  - test: "First-run geo asset download on a machine with no geoip.dat/geosite.dat"
    expected: "App prints 'Downloading geoip.dat...' and 'Downloading geosite.dat...' before connecting; after download, subsequent runs skip download"
    why_human: "Requires running the binary on a machine without existing geo assets and an active network connection to github.com/Loyalsoldier"
---

# Phase 6: Distribution Verification Report

**Phase Goal:** Users on macOS and Linux can install azad with a single command (brew, curl, or package manager) and the binary handles all first-run setup automatically
**Verified:** 2026-02-26T21:30:00Z
**Status:** passed
**Re-verification:** No — initial verification

---

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | App builds as a single binary for macOS (amd64, arm64) and Linux (amd64, arm64) via GoReleaser with `-ldflags="-s -w"` for size optimization | VERIFIED | `.goreleaser.yaml` has `version: 2`, `goos: [darwin, linux]`, `goarch: [amd64, arm64]`, `ldflags: [-s -w -X main.version={{.Version}}]`, `-trimpath`, `CGO_ENABLED=0`; `go build ./cmd/azad` compiles successfully |
| 2 | On first run, app auto-downloads geoip.dat and geosite.dat to the data directory without user intervention | VERIFIED | `internal/geoasset/geoasset.go` exports `EnsureAssets`; `internal/engine/engine.go` calls `geoasset.EnsureAssets(dataDir)` at line 77, BEFORE `core.New` at line 96; all 3 tests pass (download, skip-existing, checksum-mismatch) |
| 3 | Recovery commands (--cleanup, --reset-terminal) work correctly on both macOS and Linux platforms | VERIFIED | `internal/lifecycle/cleanup.go` wraps `sysproxy.UnsetSystemProxy` and `killswitch.Cleanup` in `runtime.GOOS == "darwin"` guards; Linux branch prints informational messages; `RunResetTerminal` uses `stty sane` which is POSIX-portable |
| 4 | `curl -sSL <install-url> \| bash` detects OS/arch, downloads the correct binary, and places it in PATH | VERIFIED (automated portion) | `scripts/install.sh` uses `uname -s` and `uname -m` for detection; handles darwin/linux and amd64/arm64/aarch64; SHA256 verification via `shasum -a 256` (macOS) and `sha256sum` (Linux); falls back to `~/.local/bin`; passes `sh -n` POSIX syntax check; no bashisms found |
| 5 | Homebrew tap (`brew install azad`) installs the binary with proper formula including dependencies and completions | VERIFIED (config present) | `.goreleaser.yaml` `brews:` section with `repository: leejooy96/homebrew-tap`, `directory: Formula`, `extra_install` for bash/zsh/fish completions, `test: #{bin}/azad --version`, `skip_upload` gated on IsSnapshot |
| 6 | GitHub Releases contain platform binaries, SHA256 checksums, and SBOM for each release | VERIFIED | `.goreleaser.yaml` has `checksum: {name_template: checksums.txt, algorithm: sha256}` and `sboms: [{artifacts: archive}]`; `.github/workflows/release.yml` triggers on `v*` tags via `goreleaser/goreleaser-action@v6` with `args: release --clean` |
| 7 | Linux packages available for major distros: .deb (APT), .rpm (DNF/YUM), AUR PKGBUILD, and snap | VERIFIED | `.goreleaser.yaml` has `nfpms:` with `formats: [deb, rpm]` and completion paths; `aurs:` with `azad-bin` and SSH push to `aur.archlinux.org`; `snapcrafts:` with `confinement: classic` for VPN network access |

**Score:** 7/7 truths verified

---

### Required Artifacts

| Artifact | Plan | Status | Details |
|----------|------|--------|---------|
| `.goreleaser.yaml` | 01, 03, 04 | VERIFIED | 119 lines; `version: 2`; all sections: `builds`, `archives`, `checksum`, `sboms`, `changelog`, `release`, `brews`, `nfpms`, `aurs`, `snapcrafts` |
| `.github/workflows/release.yml` | 01 | VERIFIED | Triggers on `v*` tag push; `goreleaser/goreleaser-action@v6`; `args: release --clean`; `fetch-depth: 0`; proper permissions |
| `scripts/completions.sh` | 01 | VERIFIED | Executable (`-rwxr-xr-x`); generates bash/zsh/fish completions via `go run ./cmd/azad completion` |
| `internal/geoasset/geoasset.go` | 02 | VERIFIED | Exports `EnsureAssets(dataDir string) error`; SHA256 verification; atomic rename; 5-minute HTTP timeout; package-level `httpClient` and `Assets` vars for testability |
| `internal/geoasset/geoasset_test.go` | 02 | VERIFIED | 3 tests via `httptest.NewServer`: download-when-missing, skip-existing, checksum-mismatch; all pass |
| `internal/engine/engine.go` | 02 | VERIFIED | Imports `geoasset`; calls `geoasset.EnsureAssets(dataDir)` at line 77, before `BuildConfig` (line 88) and `core.New` (line 96) |
| `internal/lifecycle/cleanup.go` | 02 | VERIFIED | Imports `runtime`; `runtime.GOOS == "darwin"` gates `sysproxy.UnsetSystemProxy` and `killswitch.Cleanup`; Linux branch prints informational messages |
| `scripts/install.sh` | 03 | VERIFIED | Executable (`-rwxr-xr-x`); POSIX `#!/bin/sh`; passes `sh -n`; OS/arch detection via `uname`; SHA256 verification; `/usr/local/bin` + `~/.local/bin` fallback; PATH warning |

---

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `.github/workflows/release.yml` | `.goreleaser.yaml` | `goreleaser-action` invokes `release --clean` | WIRED | `args: release --clean` confirmed at line 28 |
| `.goreleaser.yaml` | `scripts/completions.sh` | `before.hooks` runs script | WIRED | `./scripts/completions.sh` at line 9 |
| `.goreleaser.yaml` | `cmd/azad/main.go` | `ldflags` injects `main.version` | WIRED | `main.version={{.Version}}` in ldflags; `var version = "dev"` in `cmd/azad/main.go` line 19 |
| `internal/engine/engine.go` | `internal/geoasset/geoasset.go` | `Engine.Start` calls `EnsureAssets` before `core.New` | WIRED | Call at line 77; `core.New` at line 96; ordering confirmed |
| `internal/lifecycle/cleanup.go` | `internal/sysproxy` | `runtime.GOOS == "darwin"` gate before `sysproxy` call | WIRED | Lines 59-67 confirmed; Linux falls through to informational print |
| `internal/lifecycle/cleanup.go` | `internal/killswitch` | `runtime.GOOS == "darwin"` gate before `pfctl` calls | WIRED | Lines 74-83 confirmed; Linux fallback message present |
| `scripts/install.sh` | GitHub Releases | Downloads via `https://github.com/${REPO}/releases/download/${VERSION}` | WIRED | `REPO="leejooy96/azad"` variable; full URL constructed as `DOWNLOAD_BASE`; pattern uses variable substitution (not literal string match, but semantically equivalent) |
| `.goreleaser.yaml nfpms` | `scripts/completions.sh` | `nfpms contents` reference `completions/` generated by `before.hooks` | WIRED | `contents` lists `completions/azad.bash`, `.zsh`, `.fish`; `before.hooks` generates them |
| `.goreleaser.yaml aurs` | AUR git repo | GoReleaser pushes PKGBUILD to `ssh://aur@aur.archlinux.org/azad-bin.git` | WIRED | `azad-bin` at line 95; `git_url` at line 102 |

---

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|---------------|-------------|--------|----------|
| DIST-01 | 06-01 | Single binary for macOS (amd64, arm64) and Linux (amd64, arm64) via GoReleaser with size optimization | SATISFIED | `.goreleaser.yaml`: 4 platform targets, `-s -w -trimpath`, `CGO_ENABLED=0` |
| DIST-02 | 06-02 | Auto-download geoip.dat and geosite.dat on first run with progress display and integrity check | SATISFIED | `geoasset.EnsureAssets` with `fmt.Printf` progress, SHA256 verification, called from `Engine.Start` |
| DIST-03 | 06-02 | `--cleanup` and `--reset-terminal` recovery commands work on both macOS and Linux | SATISFIED | `RunCleanup` with `runtime.GOOS` gating; `RunResetTerminal` uses POSIX `stty sane` |
| DIST-04 | 06-03 | curl-pipe install script detects OS/arch, downloads correct binary, places in PATH | SATISFIED | `scripts/install.sh`: `uname -s`/`uname -m`, tar.gz download, checksum verify, PATH placement with warning |
| DIST-05 | 06-03 | Homebrew tap formula installs binary with proper metadata and completions | SATISFIED | `.goreleaser.yaml` `brews:` section with `extra_install` for shell completions, `test` block |
| DIST-06 | 06-01 | GitHub Releases include platform binaries, SHA256 checksums, and SBOM | SATISFIED | `checksum: {algorithm: sha256}`, `sboms: [{artifacts: archive}]`, `release.yml` tag-triggered |
| DIST-07 | 06-04 | Linux packages: .deb (APT), .rpm (DNF/YUM), AUR PKGBUILD, snap | SATISFIED | `nfpms: [deb, rpm]`, `aurs: [azad-bin]`, `snapcrafts: [classic]` all present in `.goreleaser.yaml` |

No orphaned requirements — all 7 DIST requirements claimed by plans and verified in code.

---

### Anti-Patterns Found

None. Scanned all 8 phase artifact files for TODO, FIXME, placeholder comments, empty implementations, and stub returns. No anti-patterns detected.

---

### Human Verification Required

The following items cannot be verified without a live GitHub Release and real machines:

#### 1. curl-pipe install on fresh machine

**Test:** On a fresh macOS (arm64) and fresh Linux (x86_64) machine without azad installed, run:
```sh
curl -sSL https://raw.githubusercontent.com/leejooy96/azad/master/scripts/install.sh | sh
```
**Expected:** Script prints detected platform, downloads correct `.tar.gz`, verifies checksum, installs to `/usr/local/bin/azad` or `~/.local/bin/azad`, prints success message and `Run 'azad --help' to get started`
**Why human:** Requires a real GitHub Release to exist (the repo has no published releases yet) and an internet connection to the GitHub API and Releases CDN

#### 2. Homebrew tap install

**Test:** After a tagged release is published, run `brew tap leejooy96/tap && brew install leejooy96/tap/azad` on macOS
**Expected:** Formula installs successfully; `azad --version` prints the release version; shell completions are available
**Why human:** Requires the `leejooy96/homebrew-tap` repository to exist on GitHub and a real tagged release to have triggered GoReleaser's formula push

#### 3. First-run geo asset auto-download

**Test:** On a machine without existing geo data files, run `azad` (connecting to any configured server)
**Expected:** Terminal prints `Downloading geoip.dat...` then `Downloaded geoip.dat (verified)` then `Downloading geosite.dat...` then `Downloaded geosite.dat (verified)` before the connection proceeds; on subsequent runs, download is skipped
**Why human:** Requires deleting the data directory to simulate first run, an active internet connection to Loyalsoldier's GitHub release CDN, and an actual VPN server configuration

---

### Gaps Summary

None. All 7 success criteria are verifiable in the codebase. All 8 planned artifacts exist and are substantive (no stubs). All key links are wired. All 7 DIST requirements are satisfied by concrete implementation. The binary compiles cleanly. All 3 geoasset tests pass.

The three human verification items are operational deployment checks that require live infrastructure (GitHub Releases, Homebrew tap repo, real machines) — they are not gaps in the implementation.

---

## Commit Verification

All 8 task commits referenced in summaries confirmed in git log:
- `39a21b0` — GoReleaser v2 config and completions script
- `94b84af` — GitHub Actions release workflow
- `b236be4` — geoasset package with SHA256 verification
- `46a63f0` — engine pre-flight and platform-gated cleanup
- `8efff12` — POSIX curl-pipe install script
- `ee1af7e` — Homebrew tap configuration
- `1396472` — nFPM .deb/.rpm configuration
- `c9f0e63` — AUR PKGBUILD and snap configuration

---

_Verified: 2026-02-26T21:30:00Z_
_Verifier: Claude (gsd-verifier)_

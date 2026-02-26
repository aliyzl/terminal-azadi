---
phase: 06-distribution
plan: 03
subsystem: infra
tags: [curl-pipe-installer, homebrew-tap, posix-shell, goreleaser, distribution]

# Dependency graph
requires:
  - phase: 06-distribution
    provides: "GoReleaser v2 config with archives, checksums, and release workflow"
provides:
  - "POSIX curl-pipe install script with OS/arch detection and SHA256 checksum verification"
  - "Homebrew tap formula auto-generation via GoReleaser brews section"
  - "Shell completions (bash/zsh/fish) installed via Homebrew formula"
affects: [06-distribution]

# Tech tracking
tech-stack:
  added: [posix-sh, homebrew-formula]
  patterns: [curl-pipe-install, checksum-verification, homebrew-tap-auto-push]

key-files:
  created:
    - scripts/install.sh
  modified:
    - .goreleaser.yaml

key-decisions:
  - "POSIX sh (not bash) for maximum portability across macOS and Linux"
  - "Fallback from /usr/local/bin to ~/.local/bin when no sudo available"
  - "Conditional skip_upload in brews for snapshot builds only"

patterns-established:
  - "Curl-pipe installer: uname-based OS/arch detection with checksum verification before install"
  - "Homebrew formula with extra_install for shell completions from archive"

requirements-completed: [DIST-04, DIST-05]

# Metrics
duration: 2min
completed: 2026-02-26
---

# Phase 6 Plan 3: Install Script and Homebrew Tap Summary

**POSIX curl-pipe installer with SHA256 verification and GoReleaser Homebrew tap formula with shell completions**

## Performance

- **Duration:** 2 min
- **Started:** 2026-02-26T17:13:24Z
- **Completed:** 2026-02-26T17:14:55Z
- **Tasks:** 2
- **Files created/modified:** 2

## Accomplishments
- POSIX-compatible install script that detects OS (darwin/linux) and arch (amd64/arm64) via uname, downloads from GitHub Releases, and verifies SHA256 checksum before installing
- Graceful installation path: tries /usr/local/bin with sudo fallback, then ~/.local/bin with PATH warning
- GoReleaser brews configuration auto-generates Homebrew formula and pushes to leejooy96/homebrew-tap on release
- Homebrew formula includes shell completions (bash/zsh/fish) via extra_install and a version test

## Task Commits

Each task was committed atomically:

1. **Task 1: Create POSIX curl-pipe install script** - `8efff12` (feat)
2. **Task 2: Add Homebrew tap configuration to GoReleaser** - `ee1af7e` (feat)

## Files Created/Modified
- `scripts/install.sh` - POSIX curl-pipe installer with OS/arch detection, SHA256 checksum verification, sudo/non-sudo install, and cleanup
- `.goreleaser.yaml` - Added brews section for Homebrew tap formula auto-generation with shell completions

## Decisions Made
- Used `#!/bin/sh` (POSIX) instead of bash for maximum portability -- no bashisms like [[ ]] or arrays
- Install path fallback: /usr/local/bin (writable or sudo) -> ~/.local/bin (no privilege needed) with PATH warning
- GoReleaser v2 naming: `repository` (not `tap`), `directory` (not `folder`) per v2 spec
- `skip_upload` conditional on IsSnapshot so formula only updates on actual releases

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None - the install script works against public GitHub Releases. The GH_PAT secret (documented in Plan 06-01) must have write access to the homebrew-tap repo for formula push to succeed, but that was already documented.

## Next Phase Readiness
- Install script ready for use once first release is published to GitHub Releases
- Homebrew formula will auto-generate on next tagged release via GoReleaser
- The homebrew-tap repository (leejooy96/homebrew-tap) must exist on GitHub before the first release

## Self-Check: PASSED

All files exist on disk and all commits verified in git log.

---
*Phase: 06-distribution*
*Completed: 2026-02-26*

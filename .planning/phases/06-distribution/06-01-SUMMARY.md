---
phase: 06-distribution
plan: 01
subsystem: infra
tags: [goreleaser, github-actions, ci-cd, cross-compilation, sbom, checksums]

# Dependency graph
requires:
  - phase: 01-foundation
    provides: "cobra CLI with cmd/azad entry point and var version = dev"
provides:
  - "GoReleaser v2 config for 4-platform builds (darwin/linux x amd64/arm64)"
  - "GitHub Actions release workflow triggered on v* tags"
  - "Shell completions generation script (bash/zsh/fish)"
  - "SHA256 checksums and SBOM for all release archives"
affects: [06-distribution]

# Tech tracking
tech-stack:
  added: [goreleaser-v2, goreleaser-action-v6, syft]
  patterns: [tag-triggered-release, declarative-build-config]

key-files:
  created:
    - .goreleaser.yaml
    - .github/workflows/release.yml
    - scripts/completions.sh
  modified: []

key-decisions:
  - "Single ldflags line with -s -w and version injection for simplicity"
  - "GH_PAT secret instead of GITHUB_TOKEN to prepare for Homebrew tap cross-repo push in Plan 03"
  - "tar.gz archive format for both macOS and Linux"

patterns-established:
  - "GoReleaser v2 declarative config: all build/release configuration in .goreleaser.yaml"
  - "Tag-triggered releases: push v* tag to trigger full release pipeline"
  - "Cobra completions via before.hooks: scripts/completions.sh generates bash/zsh/fish"

requirements-completed: [DIST-01, DIST-06]

# Metrics
duration: 2min
completed: 2026-02-26
---

# Phase 6 Plan 1: GoReleaser and Release Automation Summary

**GoReleaser v2 config for 4-platform builds with SHA256 checksums, SBOM, and GitHub Actions tag-triggered release workflow**

## Performance

- **Duration:** 2 min
- **Started:** 2026-02-26T17:07:48Z
- **Completed:** 2026-02-26T17:09:21Z
- **Tasks:** 2
- **Files created:** 3

## Accomplishments
- GoReleaser v2 configuration driving cross-platform builds for darwin/linux on amd64/arm64 with CGO_ENABLED=0
- ldflags -s -w for binary size optimization plus -trimpath for path privacy, with version injection into main.version
- Automated release pipeline: push v* tag triggers GitHub Actions workflow running goreleaser release --clean
- Release artifacts include SHA256 checksums, SBOM (via Syft), changelog, and shell completions (bash/zsh/fish)

## Task Commits

Each task was committed atomically:

1. **Task 1: Create GoReleaser configuration and completions script** - `39a21b0` (feat)
2. **Task 2: Create GitHub Actions release workflow** - `94b84af` (feat)

## Files Created/Modified
- `.goreleaser.yaml` - GoReleaser v2 configuration: builds, archives, checksums, SBOM, changelog, release
- `.github/workflows/release.yml` - GitHub Actions workflow for tag-triggered releases via goreleaser-action@v6
- `scripts/completions.sh` - Shell completion generation script (bash/zsh/fish) using cobra

## Decisions Made
- Single ldflags line (`-s -w -X main.version={{.Version}}`) keeps config concise while providing both size optimization and version injection
- Used GH_PAT secret name (not GITHUB_TOKEN) to prepare for Homebrew tap cross-repo push in Plan 03; documented fallback in workflow comments
- tar.gz format for all archives since both macOS and Linux handle it natively

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required. GH_PAT secret will need to be configured in the GitHub repository settings before the first release, but that is documented in the workflow file and will be needed at release time.

## Next Phase Readiness
- GoReleaser config is ready for extension with nFPM (Linux packages), Homebrew tap, and AUR in subsequent plans
- Release workflow will work as-is once GH_PAT secret is added to the repository
- Shell completions script can be tested locally with `./scripts/completions.sh` (requires Go and the project to compile)

---
*Phase: 06-distribution*
*Completed: 2026-02-26*

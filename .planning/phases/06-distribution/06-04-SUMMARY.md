---
phase: 06-distribution
plan: 04
subsystem: infra
tags: [goreleaser, nfpm, deb, rpm, aur, snap, linux-packages]

# Dependency graph
requires:
  - phase: 06-distribution-01
    provides: "GoReleaser base config with builds, archives, changelog, release"
  - phase: 06-distribution-03
    provides: "Homebrew brews section and completions.sh hook"
provides:
  - "nFPM .deb and .rpm package generation with shell completions"
  - "AUR PKGBUILD for azad-bin with SSH-based push"
  - "Snap manifest with classic confinement for VPN capabilities"
affects: [ci-cd, release-workflow]

# Tech tracking
tech-stack:
  added: [nfpm, aur, snapcraft]
  patterns: [goreleaser-linux-packages, conditional-skip-upload]

key-files:
  created: []
  modified: [".goreleaser.yaml"]

key-decisions:
  - "ConventionalFileName template for standard .deb/.rpm naming"
  - "azad-bin AUR name follows binary package convention"
  - "Classic snap confinement for VPN network control"

patterns-established:
  - "nFPM contents install completions to standard Linux FHS paths"
  - "AUR package block uses install -Dm755/644 for correct permissions"
  - "YAML comments document CI secret requirements inline"

requirements-completed: [DIST-07]

# Metrics
duration: 1min
completed: 2026-02-26
---

# Phase 6 Plan 4: Linux Packages Summary

**GoReleaser nFPM (.deb/.rpm), AUR PKGBUILD (azad-bin), and snap (classic confinement) configuration for native Linux package manager installation**

## Performance

- **Duration:** 1 min
- **Started:** 2026-02-26T17:17:43Z
- **Completed:** 2026-02-26T17:18:52Z
- **Tasks:** 2
- **Files modified:** 1

## Accomplishments
- nFPM configuration generates both .deb and .rpm packages with shell completions installed to standard Linux paths
- AUR PKGBUILD configured as azad-bin with SSH key auth, correct file permissions, and conditional upload skip
- Snap manifest configured with classic confinement required for VPN network operations

## Task Commits

Each task was committed atomically:

1. **Task 1: Add nFPM configuration for .deb and .rpm packages** - `1396472` (feat)
2. **Task 2: Add AUR PKGBUILD and snap configuration** - `c9f0e63` (feat)

## Files Created/Modified
- `.goreleaser.yaml` - Added nfpms, aurs, and snapcrafts sections for complete Linux package distribution

## Decisions Made
- ConventionalFileName template for standard package naming (azad_1.0.0_amd64.deb, azad-1.0.0-1.x86_64.rpm)
- azad-bin AUR package name follows convention for pre-built binary packages (vs source-built azad)
- Classic snap confinement chosen for VPN capabilities (network control, proxy, firewall) -- requires Snap Store manual review

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required. CI secrets (AUR_KEY, SNAPCRAFT_STORE_CREDENTIALS) are documented inline via YAML comments and will be configured during CI setup.

## Next Phase Readiness
- Distribution phase complete: GoReleaser configured for all target platforms
- macOS: Homebrew tap (06-03), universal binary via tar.gz archives (06-01)
- Linux: .deb, .rpm, AUR PKGBUILD, snap (this plan)
- CI/CD: GitHub Actions workflow (06-01), install script (06-03)
- Ready for release workflow testing

## Self-Check: PASSED

All files exist and all commits verified.

---
*Phase: 06-distribution*
*Completed: 2026-02-26*

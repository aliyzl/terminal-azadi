# Phase 6: Distribution - Research

**Researched:** 2026-02-26
**Domain:** Cross-platform build automation, package distribution, asset management
**Confidence:** HIGH

## Summary

Phase 6 transforms azad from a locally-built Go binary into a distributable product available through multiple channels: GitHub Releases, Homebrew, curl-pipe installer, and Linux package managers. The standard toolchain for this in the Go ecosystem is GoReleaser (build/release automation) + nFPM (Linux packages) + Syft (SBOM generation), all configured through a single `.goreleaser.yaml` file.

The project is well-positioned for distribution: `cmd/azad/main.go` already has `var version = "dev"` ready for ldflags injection, CGO is not required (xray-core compiles with CGO_ENABLED=0), and cobra provides built-in shell completion generation. The primary complexity lies in four areas: (1) configuring GoReleaser for 4 platform targets with checksums and SBOM, (2) implementing geo asset auto-download for first-run experience, (3) creating a robust curl-pipe installer and Homebrew tap, and (4) producing .deb/.rpm/AUR/snap packages via nFPM.

A critical cross-cutting concern: the sysproxy, killswitch, and lifecycle packages currently contain macOS-only code (networksetup, pfctl, osascript). For Linux builds to function, these packages need platform-gating via build tags or runtime GOOS checks so that macOS-specific operations are skipped or replaced on Linux. The DIST-03 requirement (recovery commands on both platforms) directly depends on this.

**Primary recommendation:** Use GoReleaser v2 with a single `.goreleaser.yaml` controlling all 4 plans -- builds, archives, checksums, SBOM, nFPM packages, Homebrew formula, and AUR PKGBUILD -- triggered by a GitHub Actions workflow on version tags.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| DIST-01 | Single binary for macOS (amd64, arm64) and Linux (amd64, arm64) via GoReleaser with size optimization | GoReleaser builds section with `goos: [darwin, linux]`, `goarch: [amd64, arm64]`, `ldflags: "-s -w"`, `CGO_ENABLED=0`. Confirmed xray-core compiles without CGO. |
| DIST-02 | Auto-download geoip.dat and geosite.dat on first run with progress and integrity check | Loyalsoldier/v2ray-rules-dat GitHub releases provide both files with SHA256 checksums. Engine.Start already sets XRAY_LOCATION_ASSET to DataDir. New geoasset package handles download/verify/cache. |
| DIST-03 | Recovery commands (--cleanup, --reset-terminal) work on macOS and Linux | Current cleanup.go calls macOS-only networksetup/pfctl. Need platform-gating: build tags or runtime.GOOS checks. Linux cleanup skips sysproxy (SOCKS proxy is app-level), kill switch needs iptables equivalent or skip. |
| DIST-04 | curl-pipe install script detects OS/arch, downloads correct binary, places in PATH | Shell script using `uname -s` / `uname -m` for detection, downloads from GitHub Releases, installs to `/usr/local/bin` (or `~/.local/bin` without sudo), verifies SHA256 checksum. |
| DIST-05 | Homebrew tap formula with proper metadata and completions | GoReleaser `brews` section auto-generates formula, pushed to separate `homebrew-tap` repo. Cobra completion generation via `before.hooks` script. Shell completions included in archives. |
| DIST-06 | GitHub Releases with platform binaries, SHA256 checksums, and SBOM | GoReleaser `checksum` (sha256) and `sboms` (syft, spdx-json) sections. GitHub Actions workflow with goreleaser-action@v6 on tag push. |
| DIST-07 | Linux packages: .deb, .rpm, AUR PKGBUILD, snap | GoReleaser `nfpms` section for .deb/.rpm, `aurs` section for AUR PKGBUILD (auto-generates `-bin` package), `snapcrafts` section for snap. |
</phase_requirements>

## Standard Stack

### Core
| Tool | Version | Purpose | Why Standard |
|------|---------|---------|--------------|
| GoReleaser | v2 (latest) | Build, archive, release automation | De facto standard for Go project releases; single YAML drives builds, packages, Homebrew, AUR, checksums, SBOM, GitHub Releases |
| nFPM | (bundled in GoReleaser) | .deb/.rpm/.archlinux package generation | GoReleaser's integrated Linux packager; no runtime deps, supports all major formats |
| Syft | (external, called by GoReleaser) | SBOM generation | Default SBOM tool in GoReleaser; understands Go module dependencies natively |
| goreleaser-action | v6 | GitHub Actions integration | Official action; handles Go setup, caching, tag-triggered releases |

### Supporting
| Tool | Version | Purpose | When to Use |
|------|---------|---------|-------------|
| Cobra completions | (built into spf13/cobra) | Shell completion generation | Before.hooks script runs `go run ./cmd/azad completion {bash,zsh,fish}` to generate completion files for packaging |
| GitHub CLI (gh) | System | Verify releases, test installs | Manual testing and verification of published releases |
| Snapcraft | System | Build snap packages | Required for snap builds; cannot build inside Docker |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| GoReleaser | Makefile + manual scripts | GoReleaser handles 90% of the work declaratively; manual scripts are error-prone for 4-platform matrix |
| nFPM | FPM (Ruby) | nFPM is Go-native, no Ruby dependency, integrated into GoReleaser |
| Syft | Trivy | Syft is GoReleaser's default; Trivy would require custom configuration |
| curl-pipe install | godownloader (deprecated) | godownloader was the GoReleaser companion but is now deprecated; write a targeted install script |

**Installation:**
```bash
# GoReleaser (development machine)
brew install goreleaser/tap/goreleaser

# Syft (for SBOM, CI will install via goreleaser)
brew install syft

# Snapcraft (for snap builds, Linux CI only)
sudo snap install snapcraft --classic
```

## Architecture Patterns

### Recommended Project Structure (new files)
```
.goreleaser.yaml              # All GoReleaser configuration
.github/workflows/release.yml # GitHub Actions workflow for tag-triggered releases
scripts/completions.sh        # Generates bash/zsh/fish completions via cobra
scripts/install.sh            # curl-pipe installer script
internal/geoasset/            # New package for geo file download/verify
  geoasset.go                 # Download, checksum verify, progress display
  geoasset_test.go            # Unit tests with HTTP mock
```

### Pattern 1: GoReleaser Single-File Configuration
**What:** All build, package, and release configuration lives in `.goreleaser.yaml`
**When to use:** Always -- this is the standard Go release pattern
**Example:**
```yaml
# Source: Context7 /goreleaser/goreleaser
version: 2

env:
  - CGO_ENABLED=0

before:
  hooks:
    - go mod tidy
    - ./scripts/completions.sh

builds:
  - id: azad
    main: ./cmd/azad
    binary: azad
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
    goos:
      - darwin
      - linux
    goarch:
      - amd64
      - arm64
    flags:
      - -trimpath

archives:
  - id: default
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    format: tar.gz
    files:
      - LICENSE*
      - README*
      - completions/*

checksum:
  name_template: "checksums.txt"
  algorithm: sha256

sboms:
  - artifacts: archive

release:
  github:
    owner: leejooy96
    name: azad
```

### Pattern 2: Geo Asset Auto-Download with Integrity Verification
**What:** On first run, detect missing geoip.dat/geosite.dat, download from GitHub releases, verify SHA256, cache locally
**When to use:** In engine.Start() before Xray initialization, or as a new pre-flight check
**Example:**
```go
// internal/geoasset/geoasset.go
package geoasset

const (
    geoIPURL    = "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geoip.dat"
    geoSiteURL  = "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geosite.dat"
    geoIPSHA    = "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geoip.dat.sha256sum"
    geoSiteSHA  = "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geosite.dat.sha256sum"
)

// EnsureAssets checks for geoip.dat and geosite.dat in dataDir.
// If missing, downloads them with progress display and SHA256 verification.
func EnsureAssets(dataDir string, showProgress bool) error {
    // 1. Check if files exist
    // 2. If missing, download with io.TeeReader for progress
    // 3. Download .sha256sum file, verify hash
    // 4. Atomic write: download to .tmp, verify, rename
}
```

### Pattern 3: Platform-Gated Recovery Commands
**What:** Use runtime.GOOS checks (or build tags) to skip macOS-specific operations on Linux
**When to use:** In lifecycle/cleanup.go and any code calling networksetup or pfctl
**Example:**
```go
// In lifecycle/cleanup.go
import "runtime"

func RunCleanup(configDir string) error {
    // ... read state ...
    if state.ProxySet && runtime.GOOS == "darwin" {
        // macOS: unset system proxy via networksetup
        if err := sysproxy.UnsetSystemProxy(state.NetworkService); err != nil {
            fmt.Printf("Warning: failed to unset system proxy: %v\n", err)
        }
    } else if state.ProxySet && runtime.GOOS == "linux" {
        // Linux: SOCKS/HTTP proxy is application-level, no system-wide proxy to unset
        fmt.Println("Linux: proxy was application-level, no system proxy to clean up.")
    }

    if state.KillSwitchActive {
        if runtime.GOOS == "darwin" {
            // macOS: pfctl cleanup (existing code)
        } else if runtime.GOOS == "linux" {
            // Linux: iptables cleanup (future, or skip with message)
            fmt.Println("Linux: kill switch cleanup not yet supported. Manual: sudo iptables -F")
        }
    }
    // ...
}
```

### Pattern 4: Curl-Pipe Installer Script
**What:** A POSIX-compatible shell script that detects OS/arch, downloads the correct binary from GitHub Releases, verifies the checksum, and installs to PATH
**When to use:** Hosted at a stable URL for `curl -sSL <url> | bash`
**Example structure:**
```bash
#!/bin/sh
set -e

REPO="leejooy96/azad"
INSTALL_DIR="/usr/local/bin"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
  *)       echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Get latest version from GitHub API
VERSION=$(curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/')

# Download, verify, install
FILENAME="azad_${VERSION}_${OS}_${ARCH}.tar.gz"
curl -sSL "https://github.com/${REPO}/releases/download/v${VERSION}/${FILENAME}" -o "/tmp/${FILENAME}"
# ... verify checksum, extract, install ...
```

### Anti-Patterns to Avoid
- **Hardcoding geo asset URLs to specific releases:** Use the `/latest/download/` redirect URL so downloads always get current data
- **Building without `-trimpath`:** Without trimpath, local filesystem paths leak into the binary (privacy and reproducibility issue)
- **Using `pfctl -d` in Linux cleanup:** pfctl does not exist on Linux; all macOS-specific commands must be gated
- **Requiring sudo in the install script without fallback:** Offer `~/.local/bin` as an alternative when user lacks sudo
- **Embedding geo files in the binary:** geoip.dat (~7MB) and geosite.dat (~10MB) would bloat the already large xray-core binary (~44MB with ldflags)

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Cross-platform build matrix | Custom Makefile with GOOS/GOARCH loops | GoReleaser `builds` section | GoReleaser handles CGO, ldflags, ignore lists, naming conventions, and parallel builds |
| .deb/.rpm generation | dpkg-deb/rpmbuild scripts | GoReleaser `nfpms` section (nFPM) | nFPM handles metadata, dependencies, file permissions, systemd units, pre/post scripts |
| Homebrew formula | Hand-written Ruby formula | GoReleaser `brews` section | Auto-generates formula with correct SHA256, pushes to tap repo on release |
| AUR PKGBUILD | Manual PKGBUILD + makepkg | GoReleaser `aurs` section | Auto-generates PKGBUILD, commits to AUR git repo with correct checksums |
| Checksum file | `sha256sum` in Makefile | GoReleaser `checksum` section | Generates checksums for all artifacts consistently |
| SBOM | Custom dependency scanner | GoReleaser `sboms` section (Syft) | Syft understands Go modules, produces SPDX/CycloneDX format automatically |
| Shell completions | Manual completion scripts | Cobra's built-in `completion` command | Cobra generates accurate completions based on actual command structure; `go run ./cmd/azad completion bash/zsh/fish` |
| Install script OS detection | Complex platform detection library | `uname -s` / `uname -m` in POSIX shell | Two commands cover all macOS/Linux variants; well-established pattern |

**Key insight:** GoReleaser's declarative YAML replaces hundreds of lines of release scripts. The entire build/package/release pipeline (4 OS/arch targets, 5 package formats, Homebrew tap, checksums, SBOM, GitHub Release) is driven by a single configuration file.

## Common Pitfalls

### Pitfall 1: Xray-Core Binary Size
**What goes wrong:** The binary including xray-core is ~44MB even with ldflags optimization. Without `-s -w`, it exceeds 50MB.
**Why it happens:** Xray-core links many protocols, transports, and crypto libraries. The `_ "github.com/xtls/xray-core/main/distro/all"` import registers everything.
**How to avoid:** Always use `ldflags: -s -w` and `-trimpath`. Accept the ~44MB size as the cost of embedding xray-core. Do NOT try to UPX-compress -- it breaks Go binaries on some platforms and increases startup time.
**Warning signs:** Binary exceeding 60MB suggests ldflags were not applied.

### Pitfall 2: macOS-Only Code in Linux Builds
**What goes wrong:** The binary compiles for Linux but crashes at runtime when attempting to call networksetup, pfctl, or osascript.
**Why it happens:** These are macOS-only binaries. The Go code compiles fine (os/exec.Command accepts any string), but fails at runtime.
**How to avoid:** Add `runtime.GOOS` checks before all platform-specific operations. Alternatively, use Go build tags (`//go:build darwin`) to create platform-specific files. The runtime check approach is simpler for this codebase since the macOS paths are small.
**Warning signs:** CI passes on macOS but Linux users report "exec: networksetup: not found" errors.

### Pitfall 3: Geo File Download Race on First Run
**What goes wrong:** Xray-core panics or returns cryptic errors when geoip.dat/geosite.dat are missing because routing rules reference `geoip:private`.
**Why it happens:** The `geoip:private` routing rule in BuildConfig requires geoip.dat. If the file doesn't exist when core.New() is called, xray-core fails.
**How to avoid:** Run geo asset check BEFORE Engine.Start(). Make it a blocking pre-flight check. Display progress to user so they know what's happening on first run.
**Warning signs:** "failed to load geoip.dat" error on fresh install.

### Pitfall 4: GITHUB_TOKEN Scope for Homebrew Tap
**What goes wrong:** GoReleaser fails to push the Homebrew formula to the tap repository.
**Why it happens:** The default `GITHUB_TOKEN` in GitHub Actions only has permission for the current repository, not the separate `homebrew-tap` repository.
**How to avoid:** Create a Personal Access Token (PAT) with `repo` scope, or use a GitHub App token. Store as `GH_PAT` secret and pass to GoReleaser: `env: GITHUB_TOKEN: ${{ secrets.GH_PAT }}`.
**Warning signs:** "403 Forbidden" or "Resource not accessible by integration" errors in CI release step.

### Pitfall 5: Snap Confinement Restrictions
**What goes wrong:** Snap package cannot access network interfaces, system proxy settings, or firewall rules.
**Why it happens:** Snap's `strict` confinement sandboxes the app. VPN-related operations (setting system proxy, managing firewall rules) require privileged access.
**How to avoid:** Use `classic` confinement for the snap, which gives full system access. This requires manual review by the Snap store team (takes days/weeks). Alternatively, declare specific interfaces: `network`, `network-bind`, `network-control`, `firewall-control`.
**Warning signs:** "Permission denied" errors only in snap-installed version.

### Pitfall 6: Install Script PATH Issues
**What goes wrong:** User installs the binary but can't run `azad` because the install directory isn't in their PATH.
**Why it happens:** `/usr/local/bin` requires sudo. `~/.local/bin` is not always in PATH on fresh systems (especially older Linux distros).
**How to avoid:** The install script should: (1) try `/usr/local/bin` with sudo, (2) fall back to `~/.local/bin`, (3) check if the chosen directory is in PATH, (4) print a warning with `export PATH` instruction if not.
**Warning signs:** "command not found: azad" after successful install.

## Code Examples

### GoReleaser Build Configuration for Azad
```yaml
# Source: Context7 /goreleaser/goreleaser + project-specific adaptation
version: 2

env:
  - CGO_ENABLED=0

before:
  hooks:
    - go mod tidy
    - ./scripts/completions.sh

builds:
  - id: azad
    main: ./cmd/azad
    binary: azad
    flags:
      - -trimpath
    ldflags:
      - -s -w -X main.version={{.Version}}
    goos:
      - darwin
      - linux
    goarch:
      - amd64
      - arm64

archives:
  - id: default
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    format: tar.gz
    files:
      - LICENSE*
      - README*
      - completions/*

checksum:
  name_template: "checksums.txt"
  algorithm: sha256

sboms:
  - artifacts: archive

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^test:"
      - "^ci:"

release:
  github:
    owner: leejooy96
    name: azad
  name_template: "Azad v{{.Version}}"
```

### nFPM Linux Package Configuration
```yaml
# Source: Context7 /goreleaser/goreleaser nfpm customization
nfpms:
  - id: azad-linux
    package_name: azad
    file_name_template: "{{ .ConventionalFileName }}"
    vendor: leejooy96
    homepage: https://github.com/leejooy96/azad
    maintainer: leejooy96
    description: Beautiful terminal VPN client - one command to connect to the fastest server
    license: MIT
    formats:
      - deb
      - rpm
    bindir: /usr/bin
    contents:
      - src: completions/azad.bash
        dst: /usr/share/bash-completion/completions/azad
      - src: completions/azad.zsh
        dst: /usr/share/zsh/vendor-completions/_azad
      - src: completions/azad.fish
        dst: /usr/share/fish/vendor_completions.d/azad.fish
```

### Homebrew Tap Configuration
```yaml
# Source: Context7 /goreleaser/goreleaser homebrew formulas
brews:
  - repository:
      owner: leejooy96
      name: homebrew-tap
    directory: Formula
    homepage: https://github.com/leejooy96/azad
    description: Beautiful terminal VPN client
    license: MIT
    test: |
      system "#{bin}/azad", "--version"
    extra_install: |-
      bash_completion.install "completions/azad.bash" => "azad"
      zsh_completion.install "completions/azad.zsh" => "_azad"
      fish_completion.install "completions/azad.fish"
```

### AUR PKGBUILD Configuration
```yaml
# Source: GoReleaser AUR documentation
aurs:
  - name: azad-bin
    homepage: https://github.com/leejooy96/azad
    description: Beautiful terminal VPN client
    maintainers:
      - "leejooy96"
    license: MIT
    private_key: "{{ .Env.AUR_KEY }}"
    git_url: "ssh://[email protected]/azad-bin.git"
    package: |-
      install -Dm755 "./azad" "${pkgdir}/usr/bin/azad"
      install -Dm644 "./completions/azad.bash" "${pkgdir}/usr/share/bash-completion/completions/azad"
      install -Dm644 "./completions/azad.zsh" "${pkgdir}/usr/share/zsh/site-functions/_azad"
      install -Dm644 "./completions/azad.fish" "${pkgdir}/usr/share/fish/vendor_completions.d/azad.fish"
```

### Snap Configuration
```yaml
# Source: GoReleaser snapcraft documentation
snapcrafts:
  - name: azad
    summary: Beautiful terminal VPN client
    description: One command to connect to the fastest VPN server through a stunning terminal interface
    grade: stable
    confinement: classic
    license: MIT
    publish: true
```

### Shell Completions Generation Script
```bash
#!/bin/sh
# scripts/completions.sh
# Source: carlosbecker.com/posts/golang-completions-cobra
set -e
rm -rf completions
mkdir completions
for sh in bash zsh fish; do
  go run ./cmd/azad completion "$sh" > "completions/azad.$sh"
done
```

### GitHub Actions Release Workflow
```yaml
# .github/workflows/release.yml
name: Release
on:
  push:
    tags:
      - "v*"

permissions:
  contents: write
  id-token: write
  attestations: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - uses: goreleaser/goreleaser-action@v6
        with:
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GH_PAT }}
```

### Geo Asset Download Function
```go
// internal/geoasset/geoasset.go
package geoasset

import (
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "strings"
)

const (
    baseURL = "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/"
)

type Asset struct {
    Name    string
    URL     string
    SHA256URL string
}

var Assets = []Asset{
    {Name: "geoip.dat", URL: baseURL + "geoip.dat", SHA256URL: baseURL + "geoip.dat.sha256sum"},
    {Name: "geosite.dat", URL: baseURL + "geosite.dat", SHA256URL: baseURL + "geosite.dat.sha256sum"},
}

// EnsureAssets verifies geo data files exist in dataDir.
// Missing files are downloaded with SHA256 verification.
func EnsureAssets(dataDir string) error {
    for _, asset := range Assets {
        path := filepath.Join(dataDir, asset.Name)
        if _, err := os.Stat(path); err == nil {
            continue // file exists
        }
        fmt.Printf("Downloading %s...\n", asset.Name)
        if err := downloadAndVerify(asset, path); err != nil {
            return fmt.Errorf("downloading %s: %w", asset.Name, err)
        }
    }
    return nil
}

func downloadAndVerify(asset Asset, destPath string) error {
    // Download to temp file
    tmpPath := destPath + ".tmp"
    // ... HTTP GET with progress ...
    // ... Download SHA256 checksum file ...
    // ... Verify hash matches ...
    // ... Atomic rename tmp -> final ...
    return os.Rename(tmpPath, destPath)
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| GoReleaser v1 YAML | GoReleaser v2 YAML (`version: 2`) | 2024 | New config format; `folder` renamed to `directory`, `tap` renamed to `repository` in brews |
| `brews` (deprecated) | `homebrew_casks` (for GUI apps) / `brews` (for CLI) | GoReleaser v2.10 (2025) | CLI tools still use `brews`; `homebrew_casks` is for .app bundles only |
| godownloader (generate install.sh) | Manual install.sh or third-party (deprecated godownloader) | 2022 | godownloader is unmaintained; write a targeted POSIX install script |
| Manual PKGBUILD maintenance | GoReleaser `aurs` section | GoReleaser v1.14 (2022) | Auto-generates and pushes PKGBUILD to AUR git repo on release |
| FPM (Ruby) for .deb/.rpm | nFPM (Go, integrated in GoReleaser) | 2019 | No Ruby dependency; declarative YAML; integrated into GoReleaser pipeline |

**Deprecated/outdated:**
- godownloader: was the companion install script generator for GoReleaser, now archived and unmaintained
- GoReleaser v1 config format: still works but new projects should use `version: 2`
- `folder` key in brews: renamed to `directory` in v2
- `tap` key in brews: renamed to `repository` in v2

## Open Questions

1. **Homebrew tap repository naming**
   - What we know: Convention is `homebrew-tap` or `homebrew-azad` repo under the same GitHub user/org
   - What's unclear: Whether `leejooy96/homebrew-tap` already exists or needs creation
   - Recommendation: Create `leejooy96/homebrew-tap` if it doesn't exist; GoReleaser will auto-push the formula

2. **AUR SSH key setup**
   - What we know: GoReleaser needs an SSH private key to push to AUR git
   - What's unclear: Whether the maintainer has an AUR account and SSH key pair
   - Recommendation: Document the AUR key setup steps; use `skip_upload: "{{ .IsSnapshot }}"` during development

3. **Linux kill switch implementation scope**
   - What we know: DIST-03 requires recovery commands on Linux. Current kill switch uses macOS pfctl.
   - What's unclear: Whether Linux kill switch (iptables/nftables) should be implemented in this phase or deferred
   - Recommendation: For DIST-03, gate the macOS-specific cleanup code behind runtime.GOOS checks. On Linux, cleanup should handle the proxy state file and print a message about kill switch not being supported on Linux yet. Full Linux kill switch is a v2 feature.

4. **Snap classic confinement review timeline**
   - What we know: Classic confinement requires Snap store review, which can take days to weeks
   - What's unclear: Exact timeline and whether the app qualifies for classic confinement
   - Recommendation: Start with the snap manifest in the GoReleaser config. Submit for review early. If review is delayed, the other distribution channels (GitHub Releases, Homebrew, .deb/.rpm, AUR) provide coverage.

5. **Geo asset update mechanism**
   - What we know: Files are downloaded on first run if missing
   - What's unclear: Whether there should be an update/refresh mechanism for geo files
   - Recommendation: For v1, only download if files are missing (simple and reliable). A future `azad update-geo` command or periodic check can be added later. The Loyalsoldier releases update daily, but users rarely need the latest geo data.

## Sources

### Primary (HIGH confidence)
- Context7 `/goreleaser/goreleaser` - GoReleaser builds, archives, checksums, SBOM, nFPM, brews, AUR configuration
- GoReleaser official docs (goreleaser.com/customization/sbom, /nfpm, /homebrew_formulas, /aur, /snapcraft) - All configuration sections verified
- Loyalsoldier/v2ray-rules-dat GitHub releases - Geo asset download URLs, SHA256 checksum files confirmed (release 202602252226)

### Secondary (MEDIUM confidence)
- carlosbecker.com/posts/golang-completions-cobra - Cobra + GoReleaser shell completions pattern (verified with Context7)
- goreleaser/goreleaser-action GitHub - GitHub Actions workflow patterns for tag-triggered releases
- XTLS/Xray-core GitHub issues (#4603) - CGO_ENABLED=0 build confirmed, binary size ~44MB with ldflags

### Tertiary (LOW confidence)
- Snap classic confinement review timeline - based on community reports, not official documentation
- godownloader deprecation status - archived repo confirms, but no official announcement found

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - GoReleaser is the definitive Go release tool, verified via Context7 with full configuration examples
- Architecture: HIGH - Patterns follow established GoReleaser conventions; geo asset download uses standard Go HTTP patterns
- Pitfalls: HIGH - Binary size confirmed from STATE.md; macOS-only code verified by codebase inspection; GITHUB_TOKEN scope is well-documented
- Linux compatibility: MEDIUM - Platform-gating approach is sound but the exact Linux kill switch cleanup behavior needs validation during implementation

**Research date:** 2026-02-26
**Valid until:** 2026-03-26 (GoReleaser is stable; geo asset URLs may change if Loyalsoldier repo is reorganized)

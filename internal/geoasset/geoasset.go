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
	"time"
)

const baseURL = "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/"

// Asset describes a geo data file to download and verify.
type Asset struct {
	Name      string
	URL       string
	SHA256URL string
}

// Assets is the list of geo data files required by Xray routing rules.
// Tests may override this slice to point at a local test server.
var Assets = []Asset{
	{Name: "geoip.dat", URL: baseURL + "geoip.dat", SHA256URL: baseURL + "geoip.dat.sha256sum"},
	{Name: "geosite.dat", URL: baseURL + "geosite.dat", SHA256URL: baseURL + "geosite.dat.sha256sum"},
}

// httpClient is the HTTP client used for downloads. Package-level var for testability.
var httpClient = &http.Client{Timeout: 5 * time.Minute}

// EnsureAssets checks that all required geo data files exist in dataDir.
// Missing files are downloaded and verified against their SHA256 checksums.
func EnsureAssets(dataDir string) error {
	for _, asset := range Assets {
		destPath := filepath.Join(dataDir, asset.Name)
		if _, err := os.Stat(destPath); err == nil {
			// File exists, skip.
			continue
		}
		fmt.Printf("Downloading %s...\n", asset.Name)
		if err := downloadAndVerify(asset, destPath); err != nil {
			return fmt.Errorf("%s: %w", asset.Name, err)
		}
	}
	return nil
}

// downloadAndVerify downloads a geo asset file to destPath, verifies its SHA256
// checksum, and atomically renames the temp file into place.
func downloadAndVerify(asset Asset, destPath string) error {
	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(destPath), 0700); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	tmpPath := destPath + ".tmp"

	// Download the file.
	resp, err := httpClient.Get(asset.URL)
	if err != nil {
		return fmt.Errorf("downloading: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("downloading: HTTP %d", resp.StatusCode)
	}

	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("writing temp file: %w", err)
	}
	tmpFile.Close()

	// Download the SHA256 checksum.
	checksumResp, err := httpClient.Get(asset.SHA256URL)
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("downloading checksum: %w", err)
	}
	defer checksumResp.Body.Close()

	if checksumResp.StatusCode != http.StatusOK {
		os.Remove(tmpPath)
		return fmt.Errorf("downloading checksum: HTTP %d", checksumResp.StatusCode)
	}

	checksumBody, err := io.ReadAll(checksumResp.Body)
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("reading checksum: %w", err)
	}

	// Parse the expected hash (format: "<hash>  <filename>\n").
	fields := strings.Fields(string(checksumBody))
	if len(fields) < 1 {
		os.Remove(tmpPath)
		return fmt.Errorf("invalid checksum format")
	}
	expectedHash := fields[0]

	// Compute SHA256 of the downloaded file.
	f, err := os.Open(tmpPath)
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("opening temp file for checksum: %w", err)
	}

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("computing checksum: %w", err)
	}
	f.Close()

	computedHash := hex.EncodeToString(h.Sum(nil))

	// Compare hashes (case-insensitive).
	if !strings.EqualFold(computedHash, expectedHash) {
		os.Remove(tmpPath)
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, computedHash)
	}

	// Atomic rename.
	if err := os.Rename(tmpPath, destPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("renaming temp file: %w", err)
	}

	fmt.Printf("Downloaded %s (verified)\n", asset.Name)
	return nil
}

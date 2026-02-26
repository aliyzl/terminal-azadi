package geoasset

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestEnsureAssets_DownloadsWhenMissing(t *testing.T) {
	fakeContent := []byte("fake geoip data for testing")
	hash := sha256.Sum256(fakeContent)
	hashStr := hex.EncodeToString(hash[:])
	checksumBody := fmt.Sprintf("%s  geoip.dat\n", hashStr)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/geoip.dat":
			w.Write(fakeContent)
		case "/geoip.dat.sha256sum":
			w.Write([]byte(checksumBody))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	// Save and restore original Assets.
	origAssets := Assets
	defer func() { Assets = origAssets }()

	Assets = []Asset{
		{
			Name:      "geoip.dat",
			URL:       srv.URL + "/geoip.dat",
			SHA256URL: srv.URL + "/geoip.dat.sha256sum",
		},
	}

	tmpDir := t.TempDir()
	if err := EnsureAssets(tmpDir); err != nil {
		t.Fatalf("EnsureAssets failed: %v", err)
	}

	// Verify file exists and content matches.
	got, err := os.ReadFile(filepath.Join(tmpDir, "geoip.dat"))
	if err != nil {
		t.Fatalf("reading downloaded file: %v", err)
	}
	if string(got) != string(fakeContent) {
		t.Errorf("content mismatch: got %q, want %q", got, fakeContent)
	}
}

func TestEnsureAssets_SkipsExisting(t *testing.T) {
	var requestCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.Write([]byte("should not be called"))
	}))
	defer srv.Close()

	origAssets := Assets
	defer func() { Assets = origAssets }()

	Assets = []Asset{
		{
			Name:      "geoip.dat",
			URL:       srv.URL + "/geoip.dat",
			SHA256URL: srv.URL + "/geoip.dat.sha256sum",
		},
	}

	tmpDir := t.TempDir()

	// Pre-create the file so EnsureAssets should skip it.
	if err := os.WriteFile(filepath.Join(tmpDir, "geoip.dat"), []byte("existing"), 0644); err != nil {
		t.Fatalf("writing existing file: %v", err)
	}

	if err := EnsureAssets(tmpDir); err != nil {
		t.Fatalf("EnsureAssets failed: %v", err)
	}

	if count := requestCount.Load(); count != 0 {
		t.Errorf("expected 0 HTTP requests for existing file, got %d", count)
	}
}

func TestEnsureAssets_FailsOnChecksumMismatch(t *testing.T) {
	fakeContent := []byte("real file content")
	wrongChecksum := "0000000000000000000000000000000000000000000000000000000000000000"
	checksumBody := fmt.Sprintf("%s  geoip.dat\n", wrongChecksum)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/geoip.dat":
			w.Write(fakeContent)
		case "/geoip.dat.sha256sum":
			w.Write([]byte(checksumBody))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	origAssets := Assets
	defer func() { Assets = origAssets }()

	Assets = []Asset{
		{
			Name:      "geoip.dat",
			URL:       srv.URL + "/geoip.dat",
			SHA256URL: srv.URL + "/geoip.dat.sha256sum",
		},
	}

	tmpDir := t.TempDir()
	err := EnsureAssets(tmpDir)
	if err == nil {
		t.Fatal("expected error for checksum mismatch, got nil")
	}

	// Error should mention checksum.
	if got := err.Error(); !contains(got, "checksum") {
		t.Errorf("error should mention checksum, got: %s", got)
	}

	// Temp file should be cleaned up -- no file at destination.
	if _, err := os.Stat(filepath.Join(tmpDir, "geoip.dat")); !os.IsNotExist(err) {
		t.Error("expected no file at destination after checksum mismatch")
	}

	// Temp file should also not remain.
	if _, err := os.Stat(filepath.Join(tmpDir, "geoip.dat.tmp")); !os.IsNotExist(err) {
		t.Error("expected temp file to be cleaned up after checksum mismatch")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

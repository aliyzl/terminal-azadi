package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.yaml")

	cfg := Defaults()
	cfg.Proxy.SOCKSPort = 2080
	cfg.Server.LastUsed = "my-server"

	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Save using a fresh koanf instance
	k := func() error {
		return Save(cfg, path)
	}
	if err := k(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	cfg2, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg2.Proxy.SOCKSPort != 2080 {
		t.Errorf("expected socks_port 2080, got %d", cfg2.Proxy.SOCKSPort)
	}
	if cfg2.Proxy.HTTPPort != 8080 {
		t.Errorf("expected http_port 8080, got %d", cfg2.Proxy.HTTPPort)
	}
	if cfg2.Server.LastUsed != "my-server" {
		t.Errorf("expected last_used my-server, got %s", cfg2.Server.LastUsed)
	}

	// Verify file permissions
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("expected perm 0600, got %04o", perm)
	}
}

func TestLoadDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "nonexistent.yaml")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Proxy.SOCKSPort != 1080 {
		t.Errorf("expected socks_port 1080, got %d", cfg.Proxy.SOCKSPort)
	}
	if cfg.Proxy.HTTPPort != 8080 {
		t.Errorf("expected http_port 8080, got %d", cfg.Proxy.HTTPPort)
	}
}

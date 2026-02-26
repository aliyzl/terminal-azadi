package config

import (
	"fmt"
	"os"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
)

// Config is the top-level application configuration.
type Config struct {
	Proxy       ProxyConfig       `koanf:"proxy"`
	Server      ServerConfig      `koanf:"server"`
	SplitTunnel SplitTunnelConfig `koanf:"split_tunnel"`
}

// ProxyConfig holds proxy port settings.
type ProxyConfig struct {
	SOCKSPort int `koanf:"socks_port"`
	HTTPPort  int `koanf:"http_port"`
}

// ServerConfig holds server-related settings.
type ServerConfig struct {
	LastUsed string `koanf:"last_used"`
}

// SplitTunnelConfig holds split tunneling settings.
type SplitTunnelConfig struct {
	Enabled bool              `koanf:"enabled"`
	Mode    string            `koanf:"mode"`
	Rules   []SplitTunnelRule `koanf:"rules"`
}

// SplitTunnelRule represents a single split tunnel rule for persistence.
type SplitTunnelRule struct {
	Value string `koanf:"value"`
	Type  string `koanf:"type"`
}

// Defaults returns a Config with default values.
func Defaults() *Config {
	return &Config{
		Proxy: ProxyConfig{
			SOCKSPort: 1080,
			HTTPPort:  8080,
		},
	}
}

// Load reads configuration from the given path, applying defaults first.
// If the file does not exist, only defaults are returned.
func Load(path string) (*Config, error) {
	k := koanf.New(".")

	// Load defaults via confmap with dot-notation keys.
	if err := k.Load(confmap.Provider(map[string]interface{}{
		"proxy.socks_port": 1080,
		"proxy.http_port":  8080,
	}, "."), nil); err != nil {
		return nil, fmt.Errorf("loading defaults: %w", err)
	}

	// Load from file if it exists (merges on top of defaults).
	if _, err := os.Stat(path); err == nil {
		if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("loading config file: %w", err)
		}
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}

// Save writes the configuration to the given path as YAML.
// It creates the config directory if needed.
// Uses a fresh koanf instance for writes to avoid race conditions
// (no package-level koanf instance for writes per RESEARCH anti-patterns).
func Save(cfg *Config, path string) error {
	if err := EnsureDir(); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	// Create a fresh koanf instance for serialization.
	k := koanf.New(".")
	if err := k.Load(structs.Provider(cfg, "koanf"), nil); err != nil {
		return fmt.Errorf("loading struct into koanf: %w", err)
	}

	b, err := k.Marshal(yaml.Parser())
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	return os.WriteFile(path, b, 0600)
}

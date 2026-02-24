package config

import (
	"os"
	"path/filepath"
)

const appName = "azad"
const configFileName = "config.yaml"
const stateFileName = ".state.json"

// Dir returns the config directory path (~/.config/azad or XDG equivalent).
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, appName), nil
}

// FilePath returns the full config file path.
func FilePath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFileName), nil
}

// EnsureDir creates the config directory if it doesn't exist.
func EnsureDir() error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0700)
}

// DataDir returns the data directory path.
// In Phase 1, data lives alongside config. Can split to XDG_DATA_HOME later.
func DataDir() (string, error) {
	return Dir()
}

// StateFilePath returns the path to the proxy state file used by --cleanup.
func StateFilePath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, stateFileName), nil
}

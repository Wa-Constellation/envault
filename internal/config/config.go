package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const (
	DefaultConfigDir  = ".config/envault"
	DefaultConfigName = "config.toml"
	DefaultVaultName  = "vault.enc"
)

// Config holds envault configuration.
type Config struct {
	Backend    string `toml:"backend"`     // "keyring" or "file"
	VaultFile  string `toml:"vault_file"`  // path to encrypted vault file
	DefaultTTL string `toml:"default_ttl"` // default TTL for new profiles
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Backend:    "file",
		VaultFile:  DefaultVaultPath(),
		DefaultTTL: "0",
	}
}

// DefaultConfigPath returns the default config file path.
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, DefaultConfigDir, DefaultConfigName)
}

// DefaultVaultPath returns the default vault file path.
func DefaultVaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, DefaultConfigDir, DefaultVaultName)
}

// Load reads config from the given path, falling back to defaults.
// If the config file doesn't exist, returns defaults (no error).
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	if path == "" {
		path = DefaultConfigPath()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}

	// Expand ~ in vault file path
	if cfg.VaultFile == "" {
		cfg.VaultFile = DefaultVaultPath()
	}

	return cfg, nil
}

// Save writes the config to the given path, creating directories as needed.
func (c *Config) Save(path string) error {
	if path == "" {
		path = DefaultConfigPath()
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("creating config file: %w", err)
	}
	defer f.Close()

	enc := toml.NewEncoder(f)
	if err := enc.Encode(c); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

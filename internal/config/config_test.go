package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Backend != "file" {
		t.Errorf("default backend: got %q, want %q", cfg.Backend, "file")
	}
	if cfg.DefaultTTL != "0" {
		t.Errorf("default TTL: got %q, want %q", cfg.DefaultTTL, "0")
	}
}

func TestLoadMissing(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.toml")
	if err != nil {
		t.Fatalf("Load missing: %v", err)
	}
	if cfg.Backend != "file" {
		t.Errorf("backend: got %q, want %q", cfg.Backend, "file")
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	cfg := &Config{
		Backend:    "keyring",
		VaultFile:  "/tmp/test-vault.enc",
		DefaultTTL: "8h",
	}

	if err := cfg.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify file permissions
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("file permissions: got %o, want 0600", perm)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Backend != cfg.Backend {
		t.Errorf("backend: got %q, want %q", loaded.Backend, cfg.Backend)
	}
	if loaded.VaultFile != cfg.VaultFile {
		t.Errorf("vault_file: got %q, want %q", loaded.VaultFile, cfg.VaultFile)
	}
	if loaded.DefaultTTL != cfg.DefaultTTL {
		t.Errorf("default_ttl: got %q, want %q", loaded.DefaultTTL, cfg.DefaultTTL)
	}
}

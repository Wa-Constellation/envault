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

func TestDefaultPathsUseHomeDir(t *testing.T) {
	// Assert the default paths are actually rooted at the user's home dir
	// when UserHomeDir succeeds (i.e., the err-handling branch picks the
	// right side). Without this, a flipped `if err != nil` would silently
	// fall back to "." in the happy path.
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir on this system")
	}

	vault := DefaultVaultPath()
	wantVault := filepath.Join(home, DefaultConfigDir, DefaultVaultName)
	if vault != wantVault {
		t.Errorf("DefaultVaultPath: got %q, want %q", vault, wantVault)
	}

	cfgPath := DefaultConfigPath()
	wantCfg := filepath.Join(home, DefaultConfigDir, DefaultConfigName)
	if cfgPath != wantCfg {
		t.Errorf("DefaultConfigPath: got %q, want %q", cfgPath, wantCfg)
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

func TestLoadExpandsTildeInVaultFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	if err := os.WriteFile(path, []byte(`backend = "file"
vault_file = "~/secrets/vault.enc"
default_ttl = "0"
`), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir on this system")
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	want := filepath.Join(home, "secrets/vault.enc")
	if loaded.VaultFile != want {
		t.Errorf("vault_file: got %q, want %q", loaded.VaultFile, want)
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

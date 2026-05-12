package profile

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/envault/envault/internal/backend"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "vault.enc")
	os.Setenv("ENVAULT_PASSPHRASE", "test-passphrase")
	t.Cleanup(func() { os.Unsetenv("ENVAULT_PASSPHRASE") })
	fb := backend.NewFileBackend(path)
	return NewStore(fb)
}

func TestStorePutGet(t *testing.T) {
	store := newTestStore(t)

	p := &Profile{
		Name:      "dev",
		Vars:      map[string]string{"KEY": "VALUE", "DB_HOST": "localhost"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := store.Put(p); err != nil {
		t.Fatalf("Put: %v", err)
	}

	got, err := store.Get("dev")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got.Name != "dev" {
		t.Errorf("Name: got %q, want %q", got.Name, "dev")
	}
	if got.Vars["KEY"] != "VALUE" {
		t.Errorf("Vars[KEY]: got %q, want %q", got.Vars["KEY"], "VALUE")
	}
	if got.Vars["DB_HOST"] != "localhost" {
		t.Errorf("Vars[DB_HOST]: got %q, want %q", got.Vars["DB_HOST"], "localhost")
	}
}

func TestStoreGetNotFound(t *testing.T) {
	store := newTestStore(t)

	_, err := store.Get("nonexistent")
	if err != backend.ErrProfileNotFound {
		t.Errorf("expected ErrProfileNotFound, got %v", err)
	}
}

func TestStoreList(t *testing.T) {
	store := newTestStore(t)

	for _, name := range []string{"alpha", "beta", "gamma"} {
		p := &Profile{
			Name:      name,
			Vars:      map[string]string{},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := store.Put(p); err != nil {
			t.Fatalf("Put %s: %v", name, err)
		}
	}

	names, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(names) != 3 {
		t.Errorf("List: got %d names, want 3", len(names))
	}
}

func TestStoreDelete(t *testing.T) {
	store := newTestStore(t)

	p := &Profile{
		Name:      "test",
		Vars:      map[string]string{"KEY": "VALUE"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := store.Put(p); err != nil {
		t.Fatalf("Put: %v", err)
	}

	if err := store.Delete("test"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := store.Get("test")
	if err != backend.ErrProfileNotFound {
		t.Errorf("expected ErrProfileNotFound after delete, got %v", err)
	}
}

func TestProfileIsExpired(t *testing.T) {
	// Not expired: no TTL
	p := &Profile{TTL: 0, UpdatedAt: time.Now().Add(-24 * time.Hour)}
	if p.IsExpired() {
		t.Error("profile with no TTL should not be expired")
	}

	// Not expired: within TTL
	p = &Profile{TTL: Duration(1 * time.Hour), UpdatedAt: time.Now()}
	if p.IsExpired() {
		t.Error("profile within TTL should not be expired")
	}

	// Expired: TTL exceeded
	p = &Profile{TTL: Duration(1 * time.Second), UpdatedAt: time.Now().Add(-2 * time.Second)}
	if !p.IsExpired() {
		t.Error("profile past TTL should be expired")
	}
}

func TestProfileResolveInheritance(t *testing.T) {
	store := newTestStore(t)

	parent := &Profile{
		Name:      "base",
		Vars:      map[string]string{"REGION": "us-east-1", "ENV": "base"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := store.Put(parent); err != nil {
		t.Fatalf("Put parent: %v", err)
	}

	child := &Profile{
		Name:      "dev",
		Vars:      map[string]string{"ENV": "dev", "DEBUG": "true"},
		Inherits:  "base",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := store.Put(child); err != nil {
		t.Fatalf("Put child: %v", err)
	}

	got, err := store.Get("dev")
	if err != nil {
		t.Fatalf("Get child: %v", err)
	}

	vars, err := got.Resolve(store)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	// Child should override parent's ENV
	if vars["ENV"] != "dev" {
		t.Errorf("ENV: got %q, want %q", vars["ENV"], "dev")
	}
	// Parent's REGION should be inherited
	if vars["REGION"] != "us-east-1" {
		t.Errorf("REGION: got %q, want %q", vars["REGION"], "us-east-1")
	}
	// Child's own var
	if vars["DEBUG"] != "true" {
		t.Errorf("DEBUG: got %q, want %q", vars["DEBUG"], "true")
	}
}

func TestProfileResolveNoInheritance(t *testing.T) {
	store := newTestStore(t)

	p := &Profile{
		Name:      "standalone",
		Vars:      map[string]string{"KEY": "VALUE"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := store.Put(p); err != nil {
		t.Fatalf("Put: %v", err)
	}

	got, err := store.Get("standalone")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	vars, err := got.Resolve(store)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if len(vars) != 1 || vars["KEY"] != "VALUE" {
		t.Errorf("Resolve: got %v, want {KEY:VALUE}", vars)
	}
}

func TestValidateProfileName(t *testing.T) {
	valid := []string{"dev", "prod-us", "my.project", "test_123", "a", "_leading-underscore"}
	for _, name := range valid {
		if err := ValidateProfileName(name); err != nil {
			t.Errorf("ValidateProfileName(%q): unexpected error: %v", name, err)
		}
	}

	invalid := []string{"", "__reserved", "__envault_index__", "-starts-with-dash", ".starts-with-dot"}
	for _, name := range invalid {
		if err := ValidateProfileName(name); err == nil {
			t.Errorf("ValidateProfileName(%q): expected error", name)
		}
	}
}

func TestValidateEnvKey(t *testing.T) {
	valid := []string{"KEY", "MY_VAR", "_PRIVATE", "a", "TF_VAR_db_host"}
	for _, key := range valid {
		if err := ValidateEnvKey(key); err != nil {
			t.Errorf("ValidateEnvKey(%q): unexpected error: %v", key, err)
		}
	}

	invalid := []string{"", "123", "KEY=VALUE", "my-var", "has space"}
	for _, key := range invalid {
		if err := ValidateEnvKey(key); err == nil {
			t.Errorf("ValidateEnvKey(%q): expected error", key)
		}
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
	}{
		{"0", 0},
		{"30m", 30 * time.Minute},
		{"1h", time.Hour},
		{"24h", 24 * time.Hour},
		{"7d", 7 * 24 * time.Hour},
		{"1d", 24 * time.Hour},
	}

	for _, tt := range tests {
		got, err := ParseDuration(tt.input)
		if err != nil {
			t.Errorf("ParseDuration(%q): %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseDuration(%q): got %v, want %v", tt.input, got, tt.want)
		}
	}
}

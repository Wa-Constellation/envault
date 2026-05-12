package backend

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func newTestFileBackend(t *testing.T) *FileBackend {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "vault.enc")

	// Set passphrase via env for testing
	os.Setenv("ENVAULT_PASSPHRASE", "test-passphrase-123")
	t.Cleanup(func() {
		os.Unsetenv("ENVAULT_PASSPHRASE")
	})

	return NewFileBackend(path)
}

func TestFileBackendPutGet(t *testing.T) {
	fb := newTestFileBackend(t)

	data := []byte(`{"name":"test","vars":{"KEY":"VALUE"}}`)
	if err := fb.Put("test", data); err != nil {
		t.Fatalf("Put: %v", err)
	}

	got, err := fb.Get("test")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if string(got) != string(data) {
		t.Errorf("Get: got %q, want %q", got, data)
	}
}

func TestFileBackendGetNotFound(t *testing.T) {
	fb := newTestFileBackend(t)

	_, err := fb.Get("nonexistent")
	if err != ErrProfileNotFound {
		t.Errorf("expected ErrProfileNotFound, got %v", err)
	}
}

func TestFileBackendList(t *testing.T) {
	fb := newTestFileBackend(t)

	data1 := []byte(`{"name":"alpha"}`)
	data2 := []byte(`{"name":"beta"}`)

	if err := fb.Put("alpha", data1); err != nil {
		t.Fatalf("Put alpha: %v", err)
	}
	if err := fb.Put("beta", data2); err != nil {
		t.Fatalf("Put beta: %v", err)
	}

	names, err := fb.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	sort.Strings(names)
	if len(names) != 2 || names[0] != "alpha" || names[1] != "beta" {
		t.Errorf("List: got %v, want [alpha beta]", names)
	}
}

func TestFileBackendDelete(t *testing.T) {
	fb := newTestFileBackend(t)

	data := []byte(`{"name":"test"}`)
	if err := fb.Put("test", data); err != nil {
		t.Fatalf("Put: %v", err)
	}

	if err := fb.Delete("test"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := fb.Get("test")
	if err != ErrProfileNotFound {
		t.Errorf("expected ErrProfileNotFound after delete, got %v", err)
	}
}

func TestFileBackendDeleteNotFound(t *testing.T) {
	fb := newTestFileBackend(t)

	err := fb.Delete("nonexistent")
	if err != ErrProfileNotFound {
		t.Errorf("expected ErrProfileNotFound, got %v", err)
	}
}

func TestFileBackendOverwrite(t *testing.T) {
	fb := newTestFileBackend(t)

	data1 := []byte(`{"name":"test","vars":{"KEY":"V1"}}`)
	data2 := []byte(`{"name":"test","vars":{"KEY":"V2"}}`)

	if err := fb.Put("test", data1); err != nil {
		t.Fatalf("Put 1: %v", err)
	}
	if err := fb.Put("test", data2); err != nil {
		t.Fatalf("Put 2: %v", err)
	}

	got, err := fb.Get("test")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if string(got) != string(data2) {
		t.Errorf("overwritten Get: got %q, want %q", got, data2)
	}
}

func TestFileBackendMultipleProfiles(t *testing.T) {
	fb := newTestFileBackend(t)

	profiles := map[string][]byte{
		"dev":     []byte(`{"name":"dev","vars":{"ENV":"dev"}}`),
		"staging": []byte(`{"name":"staging","vars":{"ENV":"staging"}}`),
		"prod":    []byte(`{"name":"prod","vars":{"ENV":"prod"}}`),
	}

	for name, data := range profiles {
		if err := fb.Put(name, data); err != nil {
			t.Fatalf("Put %s: %v", name, err)
		}
	}

	for name, want := range profiles {
		got, err := fb.Get(name)
		if err != nil {
			t.Fatalf("Get %s: %v", name, err)
		}
		if string(got) != string(want) {
			t.Errorf("Get %s: got %q, want %q", name, got, want)
		}
	}

	// Delete one and verify others still work
	if err := fb.Delete("staging"); err != nil {
		t.Fatalf("Delete staging: %v", err)
	}

	names, err := fb.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(names) != 2 {
		t.Errorf("List after delete: got %d names, want 2", len(names))
	}
}

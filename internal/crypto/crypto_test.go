package crypto

import (
	"bytes"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	passphrase := []byte("test-passphrase-123")
	salt, err := GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt: %v", err)
	}

	key := DeriveKey(passphrase, salt)

	tests := []struct {
		name      string
		plaintext []byte
	}{
		{"empty", []byte{}},
		{"small", []byte("hello world")},
		{"json", []byte(`{"key": "value", "nested": {"a": 1}}`)},
		{"large", bytes.Repeat([]byte("A"), 100000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nonce, ciphertext, err := Encrypt(tt.plaintext, key)
			if err != nil {
				t.Fatalf("Encrypt: %v", err)
			}

			decrypted, err := Decrypt(ciphertext, key, nonce)
			if err != nil {
				t.Fatalf("Decrypt: %v", err)
			}

			if !bytes.Equal(decrypted, tt.plaintext) {
				t.Errorf("round-trip failed: got %q, want %q", decrypted, tt.plaintext)
			}
		})
	}
}

func TestWrongPassphrase(t *testing.T) {
	salt, err := GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt: %v", err)
	}

	key1 := DeriveKey([]byte("correct-passphrase"), salt)
	key2 := DeriveKey([]byte("wrong-passphrase"), salt)

	plaintext := []byte("secret data")
	nonce, ciphertext, err := Encrypt(plaintext, key1)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	_, err = Decrypt(ciphertext, key2, nonce)
	if err == nil {
		t.Fatal("expected error when decrypting with wrong key")
	}
}

func TestDeriveKeyDeterministic(t *testing.T) {
	passphrase := []byte("my-passphrase")
	salt := []byte("0123456789abcdef0123456789abcdef") // 32 bytes

	key1 := DeriveKey(passphrase, salt)
	key2 := DeriveKey(passphrase, salt)

	if !bytes.Equal(key1, key2) {
		t.Error("DeriveKey should be deterministic for same inputs")
	}
}

func TestDeriveKeyDifferentSalts(t *testing.T) {
	passphrase := []byte("my-passphrase")
	salt1 := []byte("0123456789abcdef0123456789abcdef")
	salt2 := []byte("fedcba9876543210fedcba9876543210")

	key1 := DeriveKey(passphrase, salt1)
	key2 := DeriveKey(passphrase, salt2)

	if bytes.Equal(key1, key2) {
		t.Error("DeriveKey should produce different keys for different salts")
	}
}

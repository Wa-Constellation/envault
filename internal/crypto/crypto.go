package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/argon2"
)

// Argon2id parameters (OWASP recommended minimum)
const (
	argonTime    = 3
	argonMemory  = 64 * 1024 // 64 MB
	argonThreads = 4
	argonKeyLen  = 32 // 256 bits for AES-256
)

// DeriveKey derives a 256-bit key from a passphrase and salt using Argon2id.
func DeriveKey(passphrase, salt []byte) []byte {
	return argon2.IDKey(passphrase, salt, argonTime, argonMemory, argonThreads, argonKeyLen)
}

// Encrypt encrypts plaintext using AES-256-GCM with the given key.
// Returns the nonce and ciphertext. The nonce is randomly generated.
func Encrypt(plaintext, key []byte) (nonce, ciphertext []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("creating GCM: %w", err)
	}

	nonce = make([]byte, gcm.NonceSize()) // 12 bytes
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("generating nonce: %w", err)
	}

	ciphertext = gcm.Seal(nil, nonce, plaintext, nil)
	return nonce, ciphertext, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM with the given key and nonce.
func Decrypt(ciphertext, key, nonce []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed (wrong passphrase?): %w", err)
	}

	return plaintext, nil
}

// GenerateSalt generates a random 32-byte salt.
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generating salt: %w", err)
	}
	return salt, nil
}

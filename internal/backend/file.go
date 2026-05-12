package backend

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/envault/envault/internal/crypto"
	"golang.org/x/term"
)

// File format constants
var magicBytes = [4]byte{'E', 'N', 'V', 'T'}

const (
	fileVersion = 1
	saltSize    = 32
	nonceSize   = 12
	headerSize  = 4 + 2 + saltSize + nonceSize // magic(4) + version(2) + salt(32) + nonce(12) = 50
)

// FileBackend stores all profiles in a single encrypted file.
// Default path: ~/.config/envault/vault.enc
type FileBackend struct {
	Path       string
	passphrase []byte
	mu         sync.Mutex
}

// NewFileBackend creates a new file backend with the given vault path.
func NewFileBackend(path string) *FileBackend {
	return &FileBackend{Path: path}
}

// getPassphrase returns the cached passphrase or prompts for one.
func (fb *FileBackend) getPassphrase() ([]byte, error) {
	if fb.passphrase != nil {
		return fb.passphrase, nil
	}

	// Check env var first (for CI/automation). An explicitly empty value is
	// rejected to match the interactive path — an empty passphrase silently
	// deriving a key would be a footgun.
	if p, ok := os.LookupEnv("ENVAULT_PASSPHRASE"); ok {
		if p == "" {
			return nil, fmt.Errorf("ENVAULT_PASSPHRASE is set but empty")
		}
		fb.passphrase = []byte(p)
		return fb.passphrase, nil
	}

	// Prompt user
	fmt.Fprint(os.Stderr, "Enter vault passphrase: ")
	pass, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Fprintln(os.Stderr) // newline after password input
	if err != nil {
		return nil, fmt.Errorf("reading passphrase: %w", err)
	}

	if len(pass) == 0 {
		return nil, fmt.Errorf("passphrase cannot be empty")
	}

	fb.passphrase = pass
	return fb.passphrase, nil
}

// readVault reads and decrypts the vault file, returning all profiles.
func (fb *FileBackend) readVault() (map[string]json.RawMessage, []byte, error) {
	data, err := os.ReadFile(fb.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]json.RawMessage), nil, nil
		}
		return nil, nil, fmt.Errorf("reading vault file: %w", err)
	}

	if len(data) < headerSize {
		return nil, nil, fmt.Errorf("vault file too small (corrupt?)")
	}

	// Validate magic bytes
	if data[0] != magicBytes[0] || data[1] != magicBytes[1] ||
		data[2] != magicBytes[2] || data[3] != magicBytes[3] {
		return nil, nil, fmt.Errorf("invalid vault file (bad magic bytes)")
	}

	// Validate version
	version := binary.BigEndian.Uint16(data[4:6])
	if version != fileVersion {
		return nil, nil, fmt.Errorf("unsupported vault version %d (expected %d)", version, fileVersion)
	}

	salt := data[6:38]
	nonce := data[38:50]
	ciphertext := data[50:]

	passphrase, err := fb.getPassphrase()
	if err != nil {
		return nil, nil, err
	}

	key := crypto.DeriveKey(passphrase, salt)

	plaintext, err := crypto.Decrypt(ciphertext, key, nonce)
	if err != nil {
		// Reset passphrase on failure so user can retry
		fb.passphrase = nil
		return nil, nil, fmt.Errorf("decrypting vault: %w", err)
	}

	var profiles map[string]json.RawMessage
	if err := json.Unmarshal(plaintext, &profiles); err != nil {
		return nil, nil, fmt.Errorf("parsing vault data: %w", err)
	}

	return profiles, salt, nil
}

// writeVault encrypts and writes all profiles to the vault file.
func (fb *FileBackend) writeVault(profiles map[string]json.RawMessage, existingSalt []byte) error {
	passphrase, err := fb.getPassphrase()
	if err != nil {
		return err
	}

	// Use existing salt or generate new one
	salt := existingSalt
	if salt == nil {
		salt, err = crypto.GenerateSalt()
		if err != nil {
			return err
		}
	}

	plaintext, err := json.Marshal(profiles)
	if err != nil {
		return fmt.Errorf("marshaling vault data: %w", err)
	}

	key := crypto.DeriveKey(passphrase, salt)

	nonce, ciphertext, err := crypto.Encrypt(plaintext, key)
	if err != nil {
		return fmt.Errorf("encrypting vault: %w", err)
	}

	// Build the file content
	buf := make([]byte, 0, headerSize+len(ciphertext))
	buf = append(buf, magicBytes[:]...)
	buf = binary.BigEndian.AppendUint16(buf, fileVersion)
	buf = append(buf, salt...)
	buf = append(buf, nonce...)
	buf = append(buf, ciphertext...)

	// Ensure directory exists with secure permissions
	dir := filepath.Dir(fb.Path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating vault directory: %w", err)
	}

	// Atomic write: write to temp file then rename
	tmpFile, err := os.CreateTemp(dir, ".vault-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Set permissions before writing data
	if err := tmpFile.Chmod(0600); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("setting file permissions: %w", err)
	}

	if _, err := tmpFile.Write(buf); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("writing vault data: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("closing temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, fb.Path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("renaming vault file: %w", err)
	}

	return nil
}

func (fb *FileBackend) Get(profileName string) ([]byte, error) {
	fb.mu.Lock()
	defer fb.mu.Unlock()

	profiles, _, err := fb.readVault()
	if err != nil {
		return nil, err
	}

	data, ok := profiles[profileName]
	if !ok {
		return nil, ErrProfileNotFound
	}

	return data, nil
}

func (fb *FileBackend) Put(profileName string, data []byte) error {
	fb.mu.Lock()
	defer fb.mu.Unlock()

	profiles, salt, err := fb.readVault()
	if err != nil {
		return err
	}

	profiles[profileName] = json.RawMessage(data)
	return fb.writeVault(profiles, salt)
}

func (fb *FileBackend) Delete(profileName string) error {
	fb.mu.Lock()
	defer fb.mu.Unlock()

	profiles, salt, err := fb.readVault()
	if err != nil {
		return err
	}

	if _, ok := profiles[profileName]; !ok {
		return ErrProfileNotFound
	}

	delete(profiles, profileName)
	return fb.writeVault(profiles, salt)
}

func (fb *FileBackend) List() ([]string, error) {
	fb.mu.Lock()
	defer fb.mu.Unlock()

	profiles, _, err := fb.readVault()
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(profiles))
	for name := range profiles {
		names = append(names, name)
	}
	return names, nil
}

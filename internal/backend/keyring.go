package backend

import (
	"encoding/json"
	"errors"
	"sort"

	"github.com/zalando/go-keyring"
)

const (
	keyringService = "envault"
	keyringIndex   = "__envault_index__"
)

// KeyringBackend stores profiles using the OS keyring.
type KeyringBackend struct{}

// NewKeyringBackend creates a new keyring backend.
func NewKeyringBackend() *KeyringBackend {
	return &KeyringBackend{}
}

func (kb *KeyringBackend) getIndex() ([]string, error) {
	data, err := keyring.Get(keyringService, keyringIndex)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return []string{}, nil
		}
		return nil, err
	}

	var names []string
	if err := json.Unmarshal([]byte(data), &names); err != nil {
		return nil, err
	}
	return names, nil
}

func (kb *KeyringBackend) setIndex(names []string) error {
	sort.Strings(names)
	data, err := json.Marshal(names)
	if err != nil {
		return err
	}
	return keyring.Set(keyringService, keyringIndex, string(data))
}

func (kb *KeyringBackend) Get(profileName string) ([]byte, error) {
	data, err := keyring.Get(keyringService, profileName)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil, ErrProfileNotFound
		}
		return nil, err
	}
	return []byte(data), nil
}

func (kb *KeyringBackend) Put(profileName string, data []byte) error {
	// Update the index FIRST when adding a new entry. If the data write later
	// fails, we leave a "phantom" index entry — visible to List, returns
	// ErrProfileNotFound on Get — which self-heals on a retry. The reverse
	// order would orphan the data: written to the keyring but invisible.
	names, err := kb.getIndex()
	if err != nil {
		return err
	}

	found := false
	for _, n := range names {
		if n == profileName {
			found = true
			break
		}
	}
	if !found {
		names = append(names, profileName)
		if err := kb.setIndex(names); err != nil {
			return err
		}
	}

	return keyring.Set(keyringService, profileName, string(data))
}

func (kb *KeyringBackend) Delete(profileName string) error {
	err := keyring.Delete(keyringService, profileName)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return ErrProfileNotFound
		}
		return err
	}

	// Update index
	names, err := kb.getIndex()
	if err != nil {
		return err
	}

	filtered := make([]string, 0, len(names))
	for _, n := range names {
		if n != profileName {
			filtered = append(filtered, n)
		}
	}
	return kb.setIndex(filtered)
}

func (kb *KeyringBackend) List() ([]string, error) {
	return kb.getIndex()
}

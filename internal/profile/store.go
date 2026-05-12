package profile

import (
	"encoding/json"
	"fmt"

	"github.com/Wa-Constellation/envault/internal/backend"
)

// Store provides high-level CRUD for profiles using a Backend.
type Store struct {
	backend backend.Backend
}

// NewStore creates a new profile store backed by the given Backend.
func NewStore(b backend.Backend) *Store {
	return &Store{backend: b}
}

// Get retrieves a profile by name.
func (s *Store) Get(name string) (*Profile, error) {
	data, err := s.backend.Get(name)
	if err != nil {
		return nil, err
	}
	var p Profile
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("corrupt profile %q: %w", name, err)
	}
	return &p, nil
}

// Put saves a profile.
func (s *Store) Put(p *Profile) error {
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return s.backend.Put(p.Name, data)
}

// Delete removes a profile by name.
func (s *Store) Delete(name string) error {
	return s.backend.Delete(name)
}

// List returns the names of all stored profiles.
func (s *Store) List() ([]string, error) {
	return s.backend.List()
}

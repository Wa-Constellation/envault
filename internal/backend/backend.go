package backend

import "errors"

// ErrProfileNotFound is returned when a profile does not exist.
var ErrProfileNotFound = errors.New("profile not found")

// Backend stores and retrieves encrypted profile data.
// Each profile is stored as a single blob (JSON bytes).
type Backend interface {
	// Get retrieves the raw JSON bytes for a profile.
	// Returns ErrProfileNotFound if the profile doesn't exist.
	Get(profileName string) ([]byte, error)

	// Put stores raw JSON bytes for a profile, overwriting any existing data.
	Put(profileName string, data []byte) error

	// Delete removes a profile entirely.
	// Returns ErrProfileNotFound if the profile doesn't exist.
	Delete(profileName string) error

	// List returns the names of all stored profiles.
	List() ([]string, error)
}

package profile

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Duration wraps time.Duration for proper JSON serialization.
type Duration time.Duration

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		// Try as number (nanoseconds) for backwards compat
		var ns int64
		if err2 := json.Unmarshal(b, &ns); err2 != nil {
			return err
		}
		*d = Duration(time.Duration(ns))
		return nil
	}
	if s == "" || s == "0" || s == "0s" {
		*d = 0
		return nil
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(parsed)
	return nil
}

func (d Duration) String() string {
	return time.Duration(d).String()
}

// Profile represents a named collection of environment variables.
type Profile struct {
	Name      string            `json:"name"`
	Vars      map[string]string `json:"vars"`
	Inherits  string            `json:"inherits,omitempty"`
	TTL       Duration          `json:"ttl,omitempty"`
	UpdatedAt time.Time         `json:"updated_at"`
	CreatedAt time.Time         `json:"created_at"`
}

// IsExpired returns true if the profile has a TTL and it has been exceeded
// since the last update.
func (p *Profile) IsExpired() bool {
	if p.TTL == 0 {
		return false
	}
	return time.Since(p.UpdatedAt) > time.Duration(p.TTL)
}

// Resolve merges parent vars (if any) with this profile's vars.
// Child vars override parent vars on key collision.
func (p *Profile) Resolve(store *Store) (map[string]string, error) {
	merged := make(map[string]string)

	if p.Inherits != "" {
		parent, err := store.Get(p.Inherits)
		if err != nil {
			return nil, fmt.Errorf("parent profile %q: %w", p.Inherits, err)
		}
		// Single-level inheritance only: use parent's own vars, not recursively resolved
		for k, v := range parent.Vars {
			merged[k] = v
		}
	}

	// Child overrides parent
	for k, v := range p.Vars {
		merged[k] = v
	}
	return merged, nil
}

// Profile name validation. Leading single underscore is allowed; the double-
// underscore prefix is reserved for internal sentinels (see keyring backend).
var validProfileName = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9._-]{0,127}$`)

// ValidateProfileName checks that a profile name is valid.
func ValidateProfileName(name string) error {
	if !validProfileName.MatchString(name) {
		return fmt.Errorf("invalid profile name %q: must be 1-128 chars, alphanumeric/hyphens/underscores/dots, must start with alphanumeric or '_'", name)
	}
	if strings.HasPrefix(name, "__") {
		return fmt.Errorf("invalid profile name %q: names starting with '__' are reserved", name)
	}
	return nil
}

// Env var key validation
var validEnvKey = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// ValidateEnvKey checks that an environment variable key is valid.
func ValidateEnvKey(key string) error {
	if !validEnvKey.MatchString(key) {
		return fmt.Errorf("invalid env var key %q: must match [A-Za-z_][A-Za-z0-9_]*", key)
	}
	return nil
}

// ParseDuration extends time.ParseDuration to support day-based durations (e.g., "7d").
func ParseDuration(s string) (time.Duration, error) {
	if s == "0" {
		return 0, nil
	}

	// Check for day suffix
	if len(s) > 1 && s[len(s)-1] == 'd' {
		// Parse the number part
		var days float64
		if _, err := fmt.Sscanf(s[:len(s)-1], "%f", &days); err != nil {
			return 0, fmt.Errorf("invalid duration %q", s)
		}
		return time.Duration(days * float64(24*time.Hour)), nil
	}

	return time.ParseDuration(s)
}

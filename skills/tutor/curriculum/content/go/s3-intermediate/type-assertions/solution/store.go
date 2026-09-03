package inspect

import (
	"errors"
	"fmt"
)

// ErrNotFound reports a key that has no value in the store.
var ErrNotFound = errors.New("not found")

// A ValidationError describes a request that was malformed before the
// store was even consulted.
type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("invalid %s: %s", e.Field, e.Reason)
}

var store = map[string]string{
	"greeting": "hello",
	"farewell": "goodbye",
}

// Get returns the value stored under key.
//
// An empty key returns a wrapped *ValidationError; a key with no value
// returns a wrapped ErrNotFound. Both are wrapped with %w so callers
// can inspect the chain with errors.Is and errors.As.
func Get(key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("get: %w",
			&ValidationError{Field: "key", Reason: "must not be empty"})
	}
	v, ok := store[key]
	if !ok {
		return "", fmt.Errorf("get %q: %w", key, ErrNotFound)
	}
	return v, nil
}

// IsNotFound reports whether err — or anything it wraps — is
// ErrNotFound.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// InvalidField returns the offending field name when err — or anything
// it wraps — is a *ValidationError; ok reports whether one was found.
func InvalidField(err error) (string, bool) {
	var ve *ValidationError
	if errors.As(err, &ve) {
		return ve.Field, true
	}
	return "", false
}

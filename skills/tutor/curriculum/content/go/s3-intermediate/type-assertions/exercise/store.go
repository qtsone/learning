package inspect

import "errors"

// ErrNotFound reports a key that has no value in the store.
var ErrNotFound = errors.New("not found")

// A ValidationError describes a request that was malformed before the
// store was even consulted.
type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	// TODO: format as `invalid <Field>: <Reason>`,
	// e.g. `invalid key: must not be empty`.
	return ""
}

var store = map[string]string{
	"greeting": "hello",
	"farewell": "goodbye",
}

// Get returns the value stored under key.
//
// An empty key returns a *ValidationError (field "key", reason
// "must not be empty") wrapped with context; a key with no value
// returns ErrNotFound wrapped with context. See LESSON.md acceptance
// criterion 4 for the exact messages. Wrap with %w so callers can
// still inspect the chain.
func Get(key string) (string, error) {
	// TODO
	return "", nil
}

// IsNotFound reports whether err — or anything it wraps — is
// ErrNotFound.
func IsNotFound(err error) bool {
	// TODO
	return false
}

// InvalidField returns the offending field name when err — or anything
// it wraps — is a *ValidationError; ok reports whether one was found.
func InvalidField(err error) (field string, ok bool) {
	// TODO
	return "", false
}

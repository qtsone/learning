package vault

import (
	"errors"
	"os"
	"path/filepath"
)

// ErrBadName is returned when a note name tries to escape the notes directory.
var ErrBadName = errors.New("invalid note name")

// ReadNote returns the contents of the note called name inside dir.
//
// TODO: seeded vulnerability — the caller controls name. What paths can
// they reach with it?
func ReadNote(dir, name string) ([]byte, error) {
	return os.ReadFile(filepath.Join(dir, name))
}

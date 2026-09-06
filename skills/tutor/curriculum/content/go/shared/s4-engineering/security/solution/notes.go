package vault

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrBadName is returned when a note name tries to escape the notes directory.
var ErrBadName = errors.New("invalid note name")

// ReadNote returns the contents of the note called name inside dir.
// filepath.IsLocal rejects absolute paths and any relative path that could
// escape dir (`..`), so the read below cannot leave the notes directory.
func ReadNote(dir, name string) ([]byte, error) {
	if !filepath.IsLocal(name) {
		return nil, fmt.Errorf("%w: %q", ErrBadName, name)
	}
	return os.ReadFile(filepath.Join(dir, name))
}

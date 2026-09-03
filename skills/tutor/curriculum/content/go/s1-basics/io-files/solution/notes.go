// Package notes stores a plain-text notebook: one note per line of a file.
package notes

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

// Save writes notes to the file at path, one note per line, replacing
// whatever the file held before. Saving an empty notebook writes an empty
// file.
func Save(path string, notes []string) error {
	var b strings.Builder
	for _, n := range notes {
		b.WriteString(n)
		b.WriteByte('\n')
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("save notebook %s: %w", path, err)
	}
	return nil
}

// Load reads the notebook at path and returns its notes in file order.
// A missing file is not an error — it is an empty notebook.
func Load(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load notebook %s: %w", path, err)
	}
	if len(data) == 0 {
		return nil, nil
	}
	return strings.Split(strings.TrimSuffix(string(data), "\n"), "\n"), nil
}

// Append adds one note to the end of the notebook, creating the file if it
// does not exist yet. Existing notes are kept.
func Append(path, note string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("append to notebook %s: %w", path, err)
	}
	if _, err := f.WriteString(note + "\n"); err != nil {
		f.Close()
		return fmt.Errorf("append to notebook %s: %w", path, err)
	}
	// This is a write path: Close can surface a real failure, so its error
	// is checked instead of deferred away.
	if err := f.Close(); err != nil {
		return fmt.Errorf("append to notebook %s: %w", path, err)
	}
	return nil
}

// Search streams the notebook line by line and returns the notes that
// contain keyword, in file order. Unlike Load, a missing file IS an error:
// the caller asked to search something that does not exist.
func Search(path, keyword string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("search notebook %s: %w", path, err)
	}
	defer f.Close()

	var matches []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), keyword) {
			matches = append(matches, scanner.Text())
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("search notebook %s: %w", path, err)
	}
	return matches, nil
}

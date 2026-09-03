// Package notes stores a plain-text notebook: one note per line of a file.
package notes

// Save writes notes to the file at path, one note per line, replacing
// whatever the file held before. Saving an empty notebook writes an empty
// file.
func Save(path string, notes []string) error {
	// TODO: build the content (each note followed by "\n") and write it
	// in one call with os.WriteFile. Wrap any error with context.
	return nil
}

// Load reads the notebook at path and returns its notes in file order.
// A missing file is not an error — it is an empty notebook.
func Load(path string) ([]string, error) {
	// TODO: read the whole file with os.ReadFile.
	// Missing file (errors.Is + os.ErrNotExist) means no notes, no error.
	// Otherwise split the content into lines — mind the final "\n".
	return nil, nil
}

// Append adds one note to the end of the notebook, creating the file if it
// does not exist yet. Existing notes are kept.
func Append(path, note string) error {
	// TODO: os.OpenFile with O_APPEND|O_CREATE|O_WRONLY, write note + "\n",
	// then close the file — checking the Close error, because this is a write.
	return nil
}

// Search streams the notebook line by line and returns the notes that
// contain keyword, in file order. Unlike Load, a missing file IS an error:
// the caller asked to search something that does not exist.
func Search(path, keyword string) ([]string, error) {
	// TODO: os.Open + defer Close, then loop with bufio.Scanner. Collect
	// lines that strings.Contains the keyword, and check scanner.Err()
	// after the loop.
	return nil, nil
}

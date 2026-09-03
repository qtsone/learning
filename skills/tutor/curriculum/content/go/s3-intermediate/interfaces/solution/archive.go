package report

import "fmt"

// Saver is the single behavior Archive needs from a storage backend.
// It lives here, next to its consumer — not in the packages that implement
// storage. Any type with a matching Save method satisfies it implicitly.
type Saver interface {
	Save(name string, data []byte) error
}

// Event is one entry in a report.
type Event struct {
	Name string
	Size int64
}

// Archive stores one document per event through s: the document is named
// after the event and its body is "<name> <size>" (e.g. "build.log 2048").
// It stops at the first failure and returns an error that wraps the cause
// and names the event that could not be saved.
func Archive(s Saver, events []Event) error {
	for _, e := range events {
		data := []byte(fmt.Sprintf("%s %d", e.Name, e.Size))
		if err := s.Save(e.Name, data); err != nil {
			return fmt.Errorf("archive %s: %w", e.Name, err)
		}
	}
	return nil
}

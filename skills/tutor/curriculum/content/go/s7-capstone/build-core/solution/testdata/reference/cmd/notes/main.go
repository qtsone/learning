// Command notes prints the seeded note list. It is a composition root: it
// wires the store to the domain and writes to stdout, and holds no rules.
package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"tutor.local/capstone-reference/internal/note"
	"tutor.local/capstone-reference/internal/store"
)

func main() {
	if err := run(os.Stdout, time.Now()); err != nil {
		fmt.Fprintln(os.Stderr, "notes:", err)
		os.Exit(1)
	}
}

func run(w io.Writer, now time.Time) error {
	notes := store.NewMemory()
	seed := []struct {
		id, title string
		tags      []string
	}{
		{"n1", "read the milestone plan", []string{"work"}},
		{"n2", "walking skeleton first", []string{"work", "design"}},
	}
	for _, s := range seed {
		n, err := note.New(s.id, s.title, s.tags, now)
		if err != nil {
			return fmt.Errorf("seed %s: %w", s.id, err)
		}
		if err := notes.Add(n); err != nil {
			return err
		}
	}
	for _, n := range notes.List("") {
		fmt.Fprintln(w, n)
	}
	return nil
}

package vault

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// notesDir builds base/notes/todo.txt plus base/secret.txt OUTSIDE the notes
// directory — the file a traversal attack would steal.
func notesDir(t *testing.T) (notes, secretPath string) {
	t.Helper()
	base := t.TempDir()
	notes = filepath.Join(base, "notes")
	if err := os.Mkdir(notes, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(notes, "todo.txt"), []byte("buy milk"), 0o644); err != nil {
		t.Fatal(err)
	}
	secretPath = filepath.Join(base, "secret.txt")
	if err := os.WriteFile(secretPath, []byte("s3cr3t"), 0o600); err != nil {
		t.Fatal(err)
	}
	return notes, secretPath
}

func TestReadNoteLegitimate(t *testing.T) {
	notes, _ := notesDir(t)
	got, err := ReadNote(notes, "todo.txt")
	if err != nil {
		t.Fatalf("ReadNote(todo.txt) error: %v", err)
	}
	if string(got) != "buy milk" {
		t.Errorf("ReadNote(todo.txt) = %q, want %q", got, "buy milk")
	}
}

func TestReadNoteBlocksTraversal(t *testing.T) {
	notes, secretPath := notesDir(t)
	hostile := []string{
		"../secret.txt",
		"sub/../../secret.txt",
		secretPath, // absolute path straight to the target
	}
	for _, name := range hostile {
		t.Run(name, func(t *testing.T) {
			data, err := ReadNote(notes, name)
			if !errors.Is(err, ErrBadName) {
				t.Errorf("ReadNote(%q) = %q, %v — want ErrBadName: the name escapes the notes directory", name, data, err)
			}
		})
	}
}

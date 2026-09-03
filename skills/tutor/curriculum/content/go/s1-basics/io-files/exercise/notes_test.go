package notes

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// The tests never touch your real files: t.TempDir() hands each test a fresh
// throwaway directory that Go deletes afterwards, and filepath.Join builds a
// path inside it that works on any operating system.

func TestSaveWritesOneNotePerLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.txt")
	if err := Save(path, []string{"feed the cat", "water the plants"}); err != nil {
		t.Fatalf("Save returned an error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Save did not create %s: %v", path, err)
	}
	want := "feed the cat\nwater the plants\n"
	if string(data) != want {
		t.Errorf("file content = %q, want %q (each note followed by \\n)", data, want)
	}
}

func TestSaveNothingWritesEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.txt")
	if err := Save(path, nil); err != nil {
		t.Fatalf("Save returned an error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Save did not create %s: %v", path, err)
	}
	if len(data) != 0 {
		t.Errorf("file content = %q, want an empty file for an empty notebook", data)
	}
}

func TestLoadRoundTrip(t *testing.T) {
	cases := []struct {
		name  string
		notes []string
	}{
		{"one note", []string{"feed the cat"}},
		{"several notes", []string{"feed the cat", "water the plants", "go run ."}},
		{"keeps blank notes", []string{"first", "", "third"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "notes.txt")
			if err := Save(path, c.notes); err != nil {
				t.Fatalf("Save returned an error: %v", err)
			}
			got, err := Load(path)
			if err != nil {
				t.Fatalf("Load returned an error: %v", err)
			}
			if !slices.Equal(got, c.notes) {
				t.Errorf("Load after Save = %q, want %q", got, c.notes)
			}
		})
	}
}

func TestLoadMissingFileIsEmptyNotebook(t *testing.T) {
	path := filepath.Join(t.TempDir(), "never-created.txt")
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load on a missing file should mean \"no notes yet\", not an error; got %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Load on a missing file = %q, want no notes", got)
	}
}

func TestLoadEmptyFileHasNoNotes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned an error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Load of an empty file = %q, want no notes (not one blank note)", got)
	}
}

func TestAppendCreatesThenGrows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.txt")
	if err := Append(path, "first"); err != nil {
		t.Fatalf("Append on a missing file should create it; got error: %v", err)
	}
	if err := Append(path, "second"); err != nil {
		t.Fatalf("Append returned an error: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned an error: %v", err)
	}
	want := []string{"first", "second"}
	if !slices.Equal(got, want) {
		t.Errorf("notebook after two Appends = %q, want %q (append must keep existing notes)", got, want)
	}
}

func TestAppendKeepsSavedNotes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.txt")
	if err := Save(path, []string{"feed the cat"}); err != nil {
		t.Fatalf("Save returned an error: %v", err)
	}
	if err := Append(path, "water the plants"); err != nil {
		t.Fatalf("Append returned an error: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned an error: %v", err)
	}
	want := []string{"feed the cat", "water the plants"}
	if !slices.Equal(got, want) {
		t.Errorf("notebook after Save then Append = %q, want %q", got, want)
	}
}

func TestSearchReturnsMatchingLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.txt")
	content := "feed the cat\nwater the plants\nbrush the cat\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name    string
		keyword string
		want    []string
	}{
		{"two matches", "cat", []string{"feed the cat", "brush the cat"}},
		{"one match", "plants", []string{"water the plants"}},
		{"no match", "dog", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Search(path, c.keyword)
			if err != nil {
				t.Fatalf("Search returned an error: %v", err)
			}
			if !slices.Equal(got, c.want) {
				t.Errorf("Search(%q) = %q, want %q (whole lines, in file order)", c.keyword, got, c.want)
			}
		})
	}
}

func TestSearchMissingFileReportsNotExist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "never-created.txt")
	_, err := Search(path, "cat")
	if err == nil {
		t.Fatal("Search on a missing file must return an error — the caller asked to search a notebook that is not there")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("error %v should wrap os.ErrNotExist — return the os error or wrap it with %%w, don't invent a new one", err)
	}
}

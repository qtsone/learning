package tracker

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveLoadRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tracker.txt")
	tr := New()
	mustAdd(t, tr, "write tests")
	piped := mustAdd(t, tr, "read | write") // titles may contain the separator
	if err := tr.Complete(piped.ID); err != nil {
		t.Fatalf("Complete(%d): %v", piped.ID, err)
	}
	if err := tr.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load after Save: %v", err)
	}
	got, want := loaded.List(), tr.List()
	if len(got) != len(want) {
		t.Fatalf("loaded %d tasks, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("loaded task %d = %+v, want %+v — every field must survive the roundtrip", i, got[i], want[i])
		}
	}
}

func TestLoadMissingFileIsAFreshStart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.txt")
	tr, err := Load(path)
	if err != nil {
		t.Fatalf("Load(missing file) = %v, want nil — the first run has no file yet", err)
	}
	if tr == nil {
		t.Fatal("Load(missing file) returned a nil tracker, want an empty usable one")
	}
	if n := len(tr.List()); n != 0 {
		t.Errorf("fresh tracker has %d tasks, want 0", n)
	}
}

func TestLoadCorruptFile(t *testing.T) {
	cases := []struct {
		name     string
		content  string
		wantLine string
	}{
		{"too few fields", "1|open\n", "line 1"},
		{"id is not a number", "one|open|buy milk\n", "line 1"},
		{"unknown status", "1|maybe|buy milk\n", "line 1"},
		{"good line then bad line", "1|open|fine\n2|open\n", "line 2"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "tracker.txt")
			if err := os.WriteFile(path, []byte(c.content), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := Load(path)
			if err == nil {
				t.Fatalf("Load of %q = nil error, want a corrupt-file error", c.content)
			}
			if !errors.Is(err, ErrCorrupt) {
				t.Errorf("Load error = %v, want errors.Is(err, ErrCorrupt) to hold — wrap the sentinel with %%w", err)
			}
			if !strings.Contains(err.Error(), c.wantLine) {
				t.Errorf("Load error %q does not mention %q — tell the user which line is broken", err, c.wantLine)
			}
		})
	}
}

func TestLoadSkipsBlankLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tracker.txt")
	content := "1|open|first\n\n2|done|second\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	tr, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v — blank lines are not corruption, skip them", err)
	}
	if n := len(tr.List()); n != 2 {
		t.Errorf("loaded %d tasks, want 2", n)
	}
}

func TestAddContinuesIDsAfterLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tracker.txt")
	if err := os.WriteFile(path, []byte("3|open|third\n7|done|seventh\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tr, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	task := mustAdd(t, tr, "new after load")
	if task.ID != 8 {
		t.Errorf("Add after loading ids 3 and 7 gave id %d, want 8 (highest existing id + 1) — ids must never repeat", task.ID)
	}
}

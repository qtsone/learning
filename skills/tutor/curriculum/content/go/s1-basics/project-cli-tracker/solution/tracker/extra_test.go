package tracker

// This file is the reference for acceptance criterion 8: a learner-written
// table-driven test for behavior of their choice. Any focused table of
// three or more cases the provided tests don't already cover is fine.

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMoreCorruptShapes(t *testing.T) {
	cases := []struct {
		name    string
		content string
	}{
		{"empty status field", "1||buy milk\n"},
		{"id is a float", "1.5|open|buy milk\n"},
		{"status with wrong case", "1|Open|buy milk\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "tracker.txt")
			if err := os.WriteFile(path, []byte(c.content), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := Load(path)
			if !errors.Is(err, ErrCorrupt) {
				t.Errorf("Load of %q error = %v, want errors.Is(err, ErrCorrupt) to hold", c.content, err)
			}
		})
	}
}

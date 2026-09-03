package httpapi_test

import (
	"errors"
	"strings"
	"testing"

	"tutor.local/capstone-reference/internal/httpapi"
)

// TestValidateWorkIDRejectsHostileIDs is the regression test for the pass's one
// real finding: the handler bounded the id's length and nothing else, so a
// traversal sequence, an ANSI escape or a NUL byte was echoed straight back.
func TestValidateWorkIDRejectsHostileIDs(t *testing.T) {
	hostile := []string{
		"",
		"../../etc/passwd",
		"..",
		"a/b",
		"a\\b",
		"%2e%2e%2f",
		"id with space",
		"id;rm -rf /",
		"id\x00",
		"\x1b[31mred",
		"idé",
		strings.Repeat("x", 65),
	}
	for _, id := range hostile {
		err := httpapi.ValidateWorkID(id)
		if !errors.Is(err, httpapi.ErrInvalidID) {
			t.Errorf("ValidateWorkID(%q) error = %v, want ErrInvalidID", id, err)
		}
	}

	// The message is sent back to whoever supplied the id, so it reports a
	// length rather than repeating the bytes.
	err := httpapi.ValidateWorkID("../../etc/passwd")
	if err == nil {
		t.Fatal("ValidateWorkID(traversal) error = nil, want ErrInvalidID")
	}
	if strings.Contains(err.Error(), "etc/passwd") {
		t.Errorf("error text echoed the input: %v", err)
	}
}

// FuzzValidateWorkID states the properties that must hold for every input, not
// for the inputs we thought of. The seeds below plus
// testdata/fuzz/FuzzValidateWorkID run on every plain `go test`.
func FuzzValidateWorkID(f *testing.F) {
	for _, seed := range []string{
		"42",
		"job_7-b",
		"",
		"../../etc/passwd",
		"a/b",
		"id\x00",
		"\x1b[31mred",
		strings.Repeat("x", 65),
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, id string) {
		if err := httpapi.ValidateWorkID(id); err != nil {
			if !errors.Is(err, httpapi.ErrInvalidID) {
				t.Fatalf("ValidateWorkID(%q) error = %v, want it to wrap ErrInvalidID", id, err)
			}
			return
		}

		if id == "" {
			t.Fatal("accepted an empty id")
		}
		if len(id) > 64 {
			t.Fatalf("accepted a %d-byte id, limit is 64", len(id))
		}
		for _, r := range id {
			ok := r == '_' || r == '-' ||
				('0' <= r && r <= '9') ||
				('A' <= r && r <= 'Z') ||
				('a' <= r && r <= 'z')
			if !ok {
				t.Fatalf("accepted %q, which contains %q", id, r)
			}
		}
	})
}

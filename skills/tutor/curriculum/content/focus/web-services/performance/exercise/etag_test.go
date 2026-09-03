package apiperf

import "testing"

func TestETagIsQuotedSHA256Hex(t *testing.T) {
	got := ETag([]byte("hello"))
	want := `"2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"`
	if got != want {
		t.Errorf("ETag([]byte(\"hello\")) = %s, want %s (sha256, hex, in double quotes)", got, want)
	}
}

func TestETagChangesWithTheBody(t *testing.T) {
	a := ETag([]byte(`{"items":[]}`))
	b := ETag([]byte(`{"items":[],"next_cursor":""}`))
	if a == b {
		t.Error("two different bodies produced the same ETag")
	}
	if a != ETag([]byte(`{"items":[]}`)) {
		t.Error("the same body must always produce the same ETag; a tag that changes per process breaks every cache")
	}
}

func TestMatchETag(t *testing.T) {
	const tag = `"abc123"`
	cases := []struct {
		name        string
		ifNoneMatch string
		etag        string
		want        bool
	}{
		{"empty header", "", tag, false},
		{"exact", `"abc123"`, tag, true},
		{"different", `"zzz"`, tag, false},
		{"star", "*", tag, true},
		{"star against empty etag", "*", "", false},
		{"weak header, strong tag", `W/"abc123"`, tag, true},
		{"strong header, weak tag", `"abc123"`, `W/"abc123"`, true},
		{"list, first entry", `"abc123", "other"`, tag, true},
		{"list, later entry", `"other" ,  W/"abc123"`, tag, true},
		{"list, no entry", `"other", "another"`, tag, false},
		{"unquoted value", `abc123`, tag, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := MatchETag(tc.ifNoneMatch, tc.etag); got != tc.want {
				t.Errorf("MatchETag(%q, %q) = %v, want %v", tc.ifNoneMatch, tc.etag, got, tc.want)
			}
		})
	}
}

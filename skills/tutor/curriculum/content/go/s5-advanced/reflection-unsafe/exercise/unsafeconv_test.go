package conf

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestUnsafeStringAliases(t *testing.T) {
	b := []byte("hello")
	s := UnsafeString(b)
	if s != "hello" {
		t.Fatalf("UnsafeString(%q) = %q, want %q", b, s, "hello")
	}
	b[0] = 'j'
	if s != "jello" {
		t.Errorf("after b[0] = 'j', the string is %q, want %q — it must share b's backing array. "+
			"A copy is safe and wrong here: sharing is the whole point, and the whole danger.", s, "jello")
	}
}

func TestUnsafeStringEmpty(t *testing.T) {
	if got := UnsafeString(nil); got != "" {
		t.Errorf("UnsafeString(nil) = %q, want %q (and no panic)", got, "")
	}
	if got := UnsafeString([]byte{}); got != "" {
		t.Errorf("UnsafeString([]byte{}) = %q, want %q", got, "")
	}
}

func TestUnsafeBytes(t *testing.T) {
	const s = "config bytes"
	got := UnsafeBytes(s)
	if !bytes.Equal(got, []byte(s)) {
		t.Errorf("UnsafeBytes(%q) = %q, want the same bytes", s, got)
	}
	if len(got) != len(s) {
		t.Errorf("len = %d, want %d", len(got), len(s))
	}
	if got := UnsafeBytes(""); len(got) != 0 {
		t.Errorf("UnsafeBytes(\"\") has len %d, want 0 (and no panic)", len(got))
	}
	// Deliberately not tested: writing to the result. String data may live
	// in read-only memory, so that test would be a crash, not an assertion.
}

func TestUnsafeStringDoesNotCopy(t *testing.T) {
	blobs := make([][]byte, 8)
	for i := range blobs {
		blobs[i] = bytes.Repeat([]byte("x"), 4096)
	}
	var sink string
	allocs := testing.AllocsPerRun(50, func() {
		for _, b := range blobs {
			sink = UnsafeString(b)
		}
	})
	_ = sink
	// Eight 4 KiB conversions: a copying implementation cannot do this in
	// one allocation, an aliasing one does it in zero. The bound is loose
	// on purpose — the claim is "no copies", not a precise count.
	if allocs > 1 {
		t.Errorf("UnsafeString made %v allocations per 8 conversions; want no copying", allocs)
	}
}

func envBlob(lines int) []byte {
	var sb strings.Builder
	sb.WriteString("# generated fixture\n\n")
	for i := range lines {
		fmt.Fprintf(&sb, "KEY_%03d=%s\n", i, strings.Repeat("v", 64))
	}
	sb.WriteString("ADDR = :8080\n")
	sb.WriteString("DSN=postgres://u:p@host/db?a=b\n")
	sb.WriteString("LAST=no trailing newline")
	return []byte(sb.String())
}

func TestLookup(t *testing.T) {
	data := envBlob(4)
	cases := []struct {
		key    string
		want   string
		wantOK bool
	}{
		{"KEY_000", strings.Repeat("v", 64), true},
		{"KEY_003", strings.Repeat("v", 64), true},
		{"ADDR", ":8080", true},
		{"DSN", "postgres://u:p@host/db?a=b", true},
		{"LAST", "no trailing newline", true},
		{"MISSING", "", false},
		{"# generated fixture", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		t.Run(c.key, func(t *testing.T) {
			got, ok := Lookup(data, c.key)
			if ok != c.wantOK || got != c.want {
				t.Errorf("Lookup(data, %q) = %q, %v; want %q, %v", c.key, got, ok, c.want, c.wantOK)
			}
		})
	}
}

func TestLookupResultOutlivesData(t *testing.T) {
	data := []byte("ADDR=:8080\n")
	got, ok := Lookup(data, "ADDR")
	if !ok {
		t.Fatal("Lookup(data, \"ADDR\") = _, false; want the value")
	}
	for i := range data {
		data[i] = 'z'
	}
	if got != ":8080" {
		t.Errorf("the returned value became %q after data was overwritten; want %q. "+
			"The value escapes the call, so it must be a copy — this is exactly where zero-copy stops being clever.", got, ":8080")
	}
}

func TestLookupScansWithoutCopying(t *testing.T) {
	data := envBlob(200) // ~14 KiB
	var sink string
	allocs := testing.AllocsPerRun(50, func() {
		for i := 0; i < 4; i++ {
			sink, _ = Lookup(data, "NOPE")
		}
	})
	_ = sink
	// Four misses over a 14 KiB blob: nothing is returned, so nothing needs
	// to be copied. An implementation that converts the blob (or splits it
	// into lines) allocates once per call at least.
	if allocs > 2 {
		t.Errorf("4 failed lookups made %v allocations; want ~0 — scan the blob in place", allocs)
	}
}

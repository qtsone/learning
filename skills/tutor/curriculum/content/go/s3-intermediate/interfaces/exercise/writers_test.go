package report

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

// Compile-time proof that both types satisfy io.Writer. If one of these
// lines stops compiling, check the method's receiver and signature.
var (
	_ io.Writer = (*CountingWriter)(nil)
	_ io.Writer = (*UpperWriter)(nil)
)

func TestCountingWriterCounts(t *testing.T) {
	var w CountingWriter
	for _, s := range []string{"alpha", " ", "beta"} {
		n, err := w.Write([]byte(s))
		if err != nil {
			t.Fatalf("Write(%q) returned error %v, want nil", s, err)
		}
		if n != len(s) {
			t.Fatalf("Write(%q) reported n=%d, want %d (the io.Writer contract: report every byte consumed)", s, n, len(s))
		}
	}
	if got, want := w.Count(), int64(len("alpha beta")); got != want {
		t.Errorf("Count() = %d, want %d", got, want)
	}
}

func TestCountingWriterComposesWithIoCopy(t *testing.T) {
	const text = "interfaces are satisfied implicitly"
	var w CountingWriter
	n, err := io.Copy(&w, strings.NewReader(text))
	if err != nil {
		t.Fatalf("io.Copy returned error %v — does Write honor the io.Writer contract?", err)
	}
	if n != int64(len(text)) || w.Count() != int64(len(text)) {
		t.Errorf("io.Copy reported %d bytes and Count() = %d, want both %d", n, w.Count(), len(text))
	}
}

func TestUpperWriterUppercases(t *testing.T) {
	var buf bytes.Buffer
	uw := NewUpperWriter(&buf)
	if _, err := io.Copy(uw, strings.NewReader("Hello, interfaces!\n")); err != nil {
		t.Fatalf("io.Copy returned error %v", err)
	}
	if got, want := buf.String(), "HELLO, INTERFACES!\n"; got != want {
		t.Errorf("destination received %q, want %q", got, want)
	}
}

func TestUpperWriterReportsConsumedBytes(t *testing.T) {
	uw := NewUpperWriter(io.Discard)
	p := []byte("abc")
	n, err := uw.Write(p)
	if err != nil || n != len(p) {
		t.Errorf("Write(%q) = (%d, %v), want (%d, nil)", p, n, err, len(p))
	}
}

func TestUpperWriterLeavesInputAlone(t *testing.T) {
	var buf bytes.Buffer
	p := []byte("quiet")
	if _, err := NewUpperWriter(&buf).Write(p); err != nil {
		t.Fatalf("Write returned error %v, want nil", err)
	}
	if string(p) != "quiet" {
		t.Errorf("Write changed the caller's slice to %q — the io.Writer contract forbids modifying p", p)
	}
}

// failingWriter has a value receiver, so failingWriter values (not just
// pointers) satisfy io.Writer — compare with CountingWriter above.
type failingWriter struct{ err error }

func (f failingWriter) Write(p []byte) (int, error) { return 0, f.err }

func TestUpperWriterPropagatesErrors(t *testing.T) {
	errDisk := errors.New("disk full")
	uw := NewUpperWriter(failingWriter{err: errDisk})
	if _, err := uw.Write([]byte("x")); !errors.Is(err, errDisk) {
		t.Errorf("Write returned %v, want the destination's %q error", err, errDisk)
	}
}

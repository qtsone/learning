package streams

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"
)

func TestUpperReader(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"ascii letters", "go rocks", "GO ROCKS"},
		{"mixed case, digits, punctuation", "Go 1.22!", "GO 1.22!"},
		{"empty", "", ""},
		{"non-ascii bytes untouched", "héllo, wörld", "HéLLO, WöRLD"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := io.ReadAll(NewUpperReader(strings.NewReader(c.in)))
			if err != nil {
				t.Fatalf("ReadAll: unexpected error: %v", err)
			}
			if string(got) != c.want {
				t.Errorf("read %q, want %q", got, c.want)
			}
		})
	}
}

func TestUpperReaderPartialReads(t *testing.T) {
	// OneByteReader delivers one byte per Read call — legal per the contract.
	r := NewUpperReader(iotest.OneByteReader(strings.NewReader("drip feed")))
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll: unexpected error: %v", err)
	}
	if string(got) != "DRIP FEED" {
		t.Errorf("read %q, want %q (transform only p[:n] — readers may return fewer bytes than asked)",
			got, "DRIP FEED")
	}
}

func TestUpperReaderDataWithEOF(t *testing.T) {
	// DataErrReader delivers the final bytes TOGETHER with io.EOF — the
	// n > 0 && err == io.EOF case the io.Reader contract allows.
	r := NewUpperReader(iotest.DataErrReader(strings.NewReader("last gasp")))
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll: unexpected error: %v", err)
	}
	if string(got) != "LAST GASP" {
		t.Errorf("read %q, want %q (bytes delivered alongside io.EOF must not be dropped)",
			got, "LAST GASP")
	}
}

func TestUpperReaderPropagatesError(t *testing.T) {
	boom := errors.New("boom")
	_, err := io.ReadAll(NewUpperReader(iotest.ErrReader(boom)))
	if !errors.Is(err, boom) {
		t.Errorf("got error %v, want the wrapped reader's error %v", err, boom)
	}
}

func TestCountingWriter(t *testing.T) {
	var buf bytes.Buffer
	cw := NewCountingWriter(&buf)
	for _, chunk := range []string{"alpha ", "beta ", "gamma"} {
		n, err := cw.Write([]byte(chunk))
		if err != nil {
			t.Fatalf("Write(%q): unexpected error: %v", chunk, err)
		}
		if n != len(chunk) {
			t.Fatalf("Write(%q) = %d, want %d", chunk, n, len(chunk))
		}
	}
	if got, want := buf.String(), "alpha beta gamma"; got != want {
		t.Errorf("wrapped writer received %q, want %q (Write must forward the bytes)", got, want)
	}
	if got, want := cw.Count(), int64(len("alpha beta gamma")); got != want {
		t.Errorf("Count() = %d, want %d", got, want)
	}
}

// chokedWriter accepts at most limit bytes in total, then rejects the rest —
// a legal short write per the io.Writer contract.
type chokedWriter struct {
	accepted int
	limit    int
}

var errChoked = errors.New("choked")

func (w *chokedWriter) Write(p []byte) (int, error) {
	room := w.limit - w.accepted
	if room >= len(p) {
		w.accepted += len(p)
		return len(p), nil
	}
	w.accepted = w.limit
	return room, errChoked
}

func TestCountingWriterShortWrite(t *testing.T) {
	cw := NewCountingWriter(&chokedWriter{limit: 4})
	n, err := cw.Write([]byte("abcdefgh"))
	if !errors.Is(err, errChoked) {
		t.Fatalf("Write returned error %v, want the wrapped writer's error %v", err, errChoked)
	}
	if n != 4 {
		t.Errorf("Write returned n = %d, want 4 (return what the wrapped writer reports)", n)
	}
	if got := cw.Count(); got != 4 {
		t.Errorf("Count() = %d, want 4 (count bytes actually written, not len(p))", got)
	}
}

func TestLineCount(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int
	}{
		{"empty", "", 0},
		{"one line with newline", "one\n", 1},
		{"one line without newline", "one", 1},
		{"multiple lines", "a\nb\nc\n", 3},
		{"no trailing newline", "a\nb\nc", 3},
		{"blank lines count", "\n\n", 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := LineCount(strings.NewReader(c.in))
			if err != nil {
				t.Fatalf("LineCount: unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("LineCount(%q) = %d, want %d", c.in, got, c.want)
			}
		})
	}
}

func TestLineCountStreams(t *testing.T) {
	// One byte per Read: LineCount must not assume whole lines per call.
	got, err := LineCount(iotest.OneByteReader(strings.NewReader("x\ny\nz\n")))
	if err != nil {
		t.Fatalf("LineCount: unexpected error: %v", err)
	}
	if got != 3 {
		t.Errorf("LineCount = %d, want 3", got)
	}
}

func TestLineCountPropagatesError(t *testing.T) {
	boom := errors.New("disk on fire")
	src := io.MultiReader(strings.NewReader("a\nb\n"), iotest.ErrReader(boom))
	if _, err := LineCount(src); !errors.Is(err, boom) {
		t.Errorf("got error %v, want the reader's error %v", err, boom)
	}
}

func TestShout(t *testing.T) {
	var out bytes.Buffer
	n, err := Shout(&out, strings.NewReader("hello, gopher\n"))
	if err != nil {
		t.Fatalf("Shout: unexpected error: %v", err)
	}
	if got, want := out.String(), "HELLO, GOPHER\n"; got != want {
		t.Errorf("dst received %q, want %q", got, want)
	}
	if want := int64(len("hello, gopher\n")); n != want {
		t.Errorf("Shout returned n = %d, want %d", n, want)
	}
}

func TestShoutChunkedSource(t *testing.T) {
	var out bytes.Buffer
	src := iotest.OneByteReader(strings.NewReader("many small reads"))
	if _, err := Shout(&out, src); err != nil {
		t.Fatalf("Shout: unexpected error: %v", err)
	}
	if got, want := out.String(), "MANY SMALL READS"; got != want {
		t.Errorf("dst received %q, want %q", got, want)
	}
}

func TestShoutPropagatesReadError(t *testing.T) {
	boom := errors.New("source failed")
	if _, err := Shout(io.Discard, iotest.ErrReader(boom)); !errors.Is(err, boom) {
		t.Errorf("got error %v, want the source's error %v", err, boom)
	}
}

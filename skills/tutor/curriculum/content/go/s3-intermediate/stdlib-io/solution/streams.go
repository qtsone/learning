package streams

import (
	"bufio"
	"io"
)

// UpperReader is an io.Reader adapter that uppercases ASCII letters
// ('a'..'z') as data flows through it. Every other byte passes unchanged.
type UpperReader struct {
	r io.Reader
}

// NewUpperReader wraps r in an UpperReader.
func NewUpperReader(r io.Reader) *UpperReader {
	return &UpperReader{r: r}
}

// Read implements io.Reader. It transforms p[:n] before inspecting err so
// that bytes delivered together with io.EOF are never dropped, and returns
// the inner reader's n and err unchanged.
func (u *UpperReader) Read(p []byte) (int, error) {
	n, err := u.r.Read(p)
	for i := range n {
		if p[i] >= 'a' && p[i] <= 'z' {
			p[i] -= 'a' - 'A'
		}
	}
	return n, err
}

// CountingWriter is an io.Writer adapter that forwards writes to the wrapped
// writer and tracks the total number of bytes actually written.
type CountingWriter struct {
	w io.Writer
	n int64
}

// NewCountingWriter wraps w in a CountingWriter.
func NewCountingWriter(w io.Writer) *CountingWriter {
	return &CountingWriter{w: w}
}

// Write implements io.Writer. It counts the bytes the wrapped writer
// actually accepted — n, not len(p) — so short writes are counted honestly.
func (c *CountingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	return n, err
}

// Count reports the total number of bytes successfully written so far.
func (c *CountingWriter) Count() int64 {
	return c.n
}

// LineCount reports the number of lines in r, streaming — memory use is
// bounded by the scanner's buffer, not the input size. A final line without
// a trailing newline still counts.
func LineCount(r io.Reader) (int, error) {
	sc := bufio.NewScanner(r)
	n := 0
	for sc.Scan() {
		n++
	}
	return n, sc.Err()
}

// Shout streams src into dst, uppercasing ASCII letters on the way through,
// and returns the number of bytes copied.
func Shout(dst io.Writer, src io.Reader) (int64, error) {
	return io.Copy(dst, NewUpperReader(src))
}

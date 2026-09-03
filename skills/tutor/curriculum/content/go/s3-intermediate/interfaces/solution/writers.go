package report

import (
	"bytes"
	"io"
)

// CountingWriter is an io.Writer that discards the data it receives and
// remembers how many bytes it was given. Useful for measuring output
// without storing it.
type CountingWriter struct {
	n int64
}

// Write implements io.Writer: it consumes all of p.
func (w *CountingWriter) Write(p []byte) (int, error) {
	w.n += int64(len(p))
	return len(p), nil
}

// Count reports the total number of bytes written so far.
func (w *CountingWriter) Count() int64 {
	return w.n
}

// UpperWriter wraps another io.Writer and uppercases everything written
// through it before passing it on.
type UpperWriter struct {
	dst io.Writer
}

// NewUpperWriter returns an UpperWriter that forwards to dst.
func NewUpperWriter(dst io.Writer) *UpperWriter {
	return &UpperWriter{dst: dst}
}

// Write implements io.Writer. bytes.ToUpper allocates a fresh slice, so the
// caller's p is never touched. On success n is len(p) — bytes consumed from
// p, not bytes the destination reported for the transformed copy.
func (w *UpperWriter) Write(p []byte) (int, error) {
	if _, err := w.dst.Write(bytes.ToUpper(p)); err != nil {
		return 0, err
	}
	return len(p), nil
}

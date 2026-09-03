package report

import "io"

// CountingWriter is an io.Writer that discards the data it receives and
// remembers how many bytes it was given. Useful for measuring output
// without storing it.
type CountingWriter struct {
	// TODO: add the field you need to remember the running byte count.
}

// Write implements io.Writer: it consumes all of p.
func (w *CountingWriter) Write(p []byte) (int, error) {
	// TODO: count len(p) and report, per the io.Writer contract, that the
	// whole slice was consumed.
	return 0, nil
}

// Count reports the total number of bytes written so far.
func (w *CountingWriter) Count() int64 {
	// TODO
	return -1
}

// UpperWriter wraps another io.Writer and uppercases everything written
// through it before passing it on.
type UpperWriter struct {
	// TODO: keep the destination writer.
}

// NewUpperWriter returns an UpperWriter that forwards to dst.
func NewUpperWriter(dst io.Writer) *UpperWriter {
	// TODO: store dst so Write can forward to it.
	return &UpperWriter{}
}

// Write implements io.Writer.
func (w *UpperWriter) Write(p []byte) (int, error) {
	// TODO: forward an uppercased copy (bytes.ToUpper) to the destination.
	// On success report len(p) consumed; on failure return the
	// destination's error. Never modify p itself.
	return 0, nil
}

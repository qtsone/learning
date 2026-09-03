package streams

import "io"

// UpperReader is an io.Reader adapter that uppercases ASCII letters
// ('a'..'z') as data flows through it. Every other byte passes unchanged.
type UpperReader struct {
	r io.Reader
}

// NewUpperReader wraps r in an UpperReader.
func NewUpperReader(r io.Reader) *UpperReader {
	return &UpperReader{r: r}
}

// Read implements io.Reader.
//
// TODO: read from the wrapped reader, uppercase the ASCII letters in the
// bytes you actually received, and return the inner reader's n and err
// unchanged. Remember the contract: a reader may return n > 0 together with
// io.EOF — those bytes must be transformed, not dropped.
func (u *UpperReader) Read(p []byte) (int, error) {
	return 0, io.EOF
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

// Write implements io.Writer.
//
// TODO: forward p to the wrapped writer, add the bytes it actually accepted
// to the running count, and return the wrapped writer's n and err. Beware
// short writes: the wrapped writer may accept fewer than len(p) bytes.
func (c *CountingWriter) Write(p []byte) (int, error) {
	return 0, nil
}

// Count reports the total number of bytes successfully written so far.
func (c *CountingWriter) Count() int64 {
	return c.n
}

// LineCount reports the number of lines in r, streaming — it must not load
// the whole input into memory. A final line without a trailing newline still
// counts.
//
// TODO: implement with bufio.Scanner, and report the scanner's error if
// reading fails.
func LineCount(r io.Reader) (int, error) {
	return 0, nil
}

// Shout streams src into dst, uppercasing ASCII letters on the way through,
// and returns the number of bytes copied.
//
// TODO: compose NewUpperReader with io.Copy. No io.ReadAll, no whole-payload
// buffer — memory use must stay constant regardless of stream size.
func Shout(dst io.Writer, src io.Reader) (int64, error) {
	return 0, nil
}

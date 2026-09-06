package netlab

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"
	"testing/iotest"
)

// frameBytes builds the wire form of a frame without using WriteFrame, so the
// reader tests fail for reader reasons only.
func frameBytes(payload []byte) []byte {
	out := make([]byte, 4+len(payload))
	binary.BigEndian.PutUint32(out[:4], uint32(len(payload)))
	copy(out[4:], payload)
	return out
}

func TestWriteFrameWireFormat(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
		want    []byte
	}{
		{"empty payload", []byte{}, []byte{0, 0, 0, 0}},
		{"short payload", []byte("hello"), []byte{0, 0, 0, 5, 'h', 'e', 'l', 'l', 'o'}},
		{"binary payload", []byte{0x00, 0xff, 0x0a, 0x0d}, []byte{0, 0, 0, 4, 0x00, 0xff, 0x0a, 0x0d}},
		{"length is big-endian", bytes.Repeat([]byte("x"), 256),
			append([]byte{0, 0, 1, 0}, bytes.Repeat([]byte("x"), 256)...)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := WriteFrame(&buf, tt.payload); err != nil {
				t.Fatalf("WriteFrame(%d-byte payload) = %v; want nil", len(tt.payload), err)
			}
			if got := buf.Bytes(); !bytes.Equal(got, tt.want) {
				t.Fatalf("wire bytes = % x; want % x (4-byte big-endian length, then the payload)", got, tt.want)
			}
		})
	}
}

func TestWriteFrameRejectsOversizePayload(t *testing.T) {
	var buf bytes.Buffer
	err := WriteFrame(&buf, make([]byte, MaxFrameSize+1))
	if !errors.Is(err, ErrFrameTooLarge) {
		t.Fatalf("WriteFrame(MaxFrameSize+1 bytes) = %v; want an error matching ErrFrameTooLarge", err)
	}
	if buf.Len() != 0 {
		t.Fatalf("WriteFrame wrote %d bytes for a rejected payload; want nothing on the wire (a half-written frame desynchronizes every frame after it)", buf.Len())
	}
}

func TestFrameRoundTrip(t *testing.T) {
	payloads := [][]byte{
		{},
		[]byte("hello"),
		{0x00, 0x01, 0x02, 0xff},
		bytes.Repeat([]byte("payload "), 5000),
		make([]byte, MaxFrameSize), // the boundary is allowed, not rejected
	}
	var buf bytes.Buffer
	for _, p := range payloads {
		if err := WriteFrame(&buf, p); err != nil {
			t.Fatalf("WriteFrame(%d-byte payload) = %v; want nil", len(p), err)
		}
	}
	// One buffer holding five frames: ReadFrame must return them one at a
	// time, in order, and stop at each boundary.
	for i, want := range payloads {
		got, err := ReadFrame(&buf)
		if err != nil {
			t.Fatalf("ReadFrame #%d = %v; want the %d-byte payload", i, err, len(want))
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("ReadFrame #%d returned %d bytes; want %d", i, len(got), len(want))
		}
	}
	if _, err := ReadFrame(&buf); !errors.Is(err, io.EOF) {
		t.Fatalf("ReadFrame after the last frame = %v; want io.EOF (the stream ended cleanly, between frames)", err)
	}
}

func TestReadFrameReassemblesFragmentedStream(t *testing.T) {
	payloads := [][]byte{[]byte("first"), {}, []byte("a longer third payload")}
	var wire bytes.Buffer
	for _, p := range payloads {
		wire.Write(frameBytes(p))
	}
	// iotest.OneByteReader hands over a single byte per Read — the worst
	// case of what TCP is allowed to do to your writes, and a case that must
	// still work.
	r := iotest.OneByteReader(&wire)
	for i, want := range payloads {
		got, err := ReadFrame(r)
		if err != nil {
			t.Fatalf("ReadFrame #%d over a one-byte-at-a-time stream = %v; want %q (a short Read is not an error — keep reading)", i, err, want)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("ReadFrame #%d = %q; want %q", i, got, want)
		}
	}
}

func TestReadFrameCleanEOF(t *testing.T) {
	if _, err := ReadFrame(bytes.NewReader(nil)); !errors.Is(err, io.EOF) {
		t.Fatalf("ReadFrame on an already-empty stream = %v; want io.EOF", err)
	}
}

func TestReadFrameTruncated(t *testing.T) {
	full := frameBytes([]byte("hello"))
	tests := []struct {
		name string
		wire []byte
	}{
		{"half a length header", full[:2]},
		{"header only, body missing entirely", full[:4]},
		{"header plus half the payload", full[:7]},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ReadFrame(bytes.NewReader(tt.wire))
			if !errors.Is(err, io.ErrUnexpectedEOF) {
				t.Fatalf("ReadFrame on a truncated frame = %v; want io.ErrUnexpectedEOF (io.EOF here would report a clean shutdown that never happened)", err)
			}
		})
	}
}

// headerOnlyReader serves the length header and then reports that ReadFrame
// tried to read a body it should have rejected on the header alone.
type headerOnlyReader struct {
	header   []byte
	n        int
	readBody bool
}

var errReadPastHeader = errors.New("test: ReadFrame read past the header")

func (r *headerOnlyReader) Read(p []byte) (int, error) {
	if r.n < len(r.header) {
		n := copy(p, r.header[r.n:])
		r.n += n
		return n, nil
	}
	r.readBody = true
	return 0, errReadPastHeader
}

func TestReadFrameRejectsOversizeHeader(t *testing.T) {
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], MaxFrameSize+1)
	r := &headerOnlyReader{header: header[:]}

	_, err := ReadFrame(r)
	if !errors.Is(err, ErrFrameTooLarge) {
		t.Fatalf("ReadFrame on a header claiming MaxFrameSize+1 = %v; want an error matching ErrFrameTooLarge", err)
	}
	if r.readBody {
		t.Fatal("ReadFrame started reading the body of an oversize frame; reject it from the header alone — the length field is whatever the peer says it is")
	}
}

func TestReadFrameOverFragmentedConn(t *testing.T) {
	client, server := netPipe(t)
	payloads := [][]byte{[]byte("first"), {}, []byte("a longer third payload")}

	var wire bytes.Buffer
	for _, p := range payloads {
		wire.Write(frameBytes(p))
	}

	writeErr := make(chan error, 1)
	go func() {
		defer client.Close()
		b := wire.Bytes()
		// Three bytes at a time, straddling every boundary: a real TCP peer
		// is free to split and merge writes exactly like this.
		for i := 0; i < len(b); i += 3 {
			end := min(i+3, len(b))
			if _, err := client.Write(b[i:end]); err != nil {
				writeErr <- err
				return
			}
		}
		writeErr <- nil
	}()

	for i, want := range payloads {
		got, err := ReadFrame(server)
		if err != nil {
			t.Fatalf("ReadFrame #%d over a fragmented conn = %v; want %q", i, err, want)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("ReadFrame #%d = %q; want %q", i, got, want)
		}
	}
	if _, err := ReadFrame(server); !errors.Is(err, io.EOF) {
		t.Fatalf("ReadFrame after the peer closed = %v; want io.EOF", err)
	}
	if err := <-writeErr; err != nil {
		t.Fatalf("writing the stream failed: %v", err)
	}
}

func TestServeEchoRoundTripsFrames(t *testing.T) {
	client, server := netPipe(t)
	done := make(chan error, 1)
	go func() { done <- ServeEcho(server) }()

	for _, payload := range [][]byte{[]byte("ping"), {}, bytes.Repeat([]byte("z"), 9000)} {
		if err := WriteFrame(client, payload); err != nil {
			t.Fatalf("WriteFrame(%d bytes) = %v; want nil", len(payload), err)
		}
		got, err := ReadFrame(client)
		if err != nil {
			t.Fatalf("ReadFrame of the echo = %v; want the %d-byte payload back", err, len(payload))
		}
		if !bytes.Equal(got, payload) {
			t.Fatalf("echo returned %d bytes; want the same %d bytes back", len(got), len(payload))
		}
	}

	client.Close()
	if err := <-done; err != nil {
		t.Fatalf("ServeEcho after a clean client close = %v; want nil (a peer hanging up between frames is not a server error)", err)
	}
}

func TestServeEchoReportsTruncatedFrame(t *testing.T) {
	client, server := netPipe(t)
	done := make(chan error, 1)
	go func() { done <- ServeEcho(server) }()

	// Announce five bytes, send two, disappear.
	writeErr := make(chan error, 1)
	go func() {
		_, err := client.Write(frameBytes([]byte("hello"))[:6])
		client.Close()
		writeErr <- err
	}()

	if err := <-done; !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("ServeEcho after a truncated frame = %v; want io.ErrUnexpectedEOF", err)
	}
	if err := <-writeErr; err != nil {
		t.Fatalf("writing the partial frame failed: %v", err)
	}
}

// Package netlab implements the two layers that sit under every HTTP
// request: a length-prefixed framing protocol over a raw byte stream, and the
// TLS configuration that decides who you are actually talking to.
package netlab

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// MaxFrameSize is the largest payload a single frame may carry.
//
// The length field is attacker-controlled input: a peer that claims 4 GiB
// must not talk you into allocating 4 GiB. Every framed protocol has this
// ceiling somewhere.
const MaxFrameSize = 1 << 20 // 1 MiB

// headerSize is the width of the big-endian length prefix.
const headerSize = 4

// ErrFrameTooLarge reports a payload — or a claimed length — above
// MaxFrameSize. Callers match it with errors.Is, so wrapping it with more
// detail is fine.
var ErrFrameTooLarge = errors.New("netlab: frame too large")

// WriteFrame writes payload to w as one frame: a 4-byte big-endian length,
// then the payload bytes.
//
// A payload larger than MaxFrameSize is rejected with ErrFrameTooLarge and
// nothing is written — a half-written frame would desynchronize the stream
// for every frame after it.
func WriteFrame(w io.Writer, payload []byte) error {
	if len(payload) > MaxFrameSize {
		return fmt.Errorf("%w: payload is %d bytes, limit %d", ErrFrameTooLarge, len(payload), MaxFrameSize)
	}
	// One buffer, one Write: the header and its payload can never be
	// separated by another writer or by a partial failure.
	frame := make([]byte, headerSize+len(payload))
	binary.BigEndian.PutUint32(frame[:headerSize], uint32(len(payload)))
	copy(frame[headerSize:], payload)
	if _, err := w.Write(frame); err != nil {
		return fmt.Errorf("netlab: write frame: %w", err)
	}
	return nil
}

// ReadFrame reads exactly one frame from r and returns its payload.
//
// TCP preserves byte order but not write boundaries, so a frame arrives in
// any number of chunks: keep reading until you have what the header promised.
// A zero-length payload is a valid frame.
//
// The errors are part of the contract, because callers act on them:
//
//   - io.EOF — the stream ended cleanly, between frames. A normal shutdown.
//   - io.ErrUnexpectedEOF — the stream ended mid-frame. A truncated frame,
//     which is a different event and often a different log line.
//   - ErrFrameTooLarge — the header claims more than MaxFrameSize. Report it
//     from the header alone, without allocating or reading the body.
func ReadFrame(r io.Reader) ([]byte, error) {
	var header [headerSize]byte
	// io.ReadFull already draws the distinction we need: io.EOF when nothing
	// arrived, io.ErrUnexpectedEOF when the header itself was cut short.
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, err
	}

	size := binary.BigEndian.Uint32(header[:])
	if size > MaxFrameSize {
		return nil, fmt.Errorf("%w: header claims %d bytes, limit %d", ErrFrameTooLarge, size, MaxFrameSize)
	}

	payload := make([]byte, size)
	if _, err := io.ReadFull(r, payload); err != nil {
		// A header promised these bytes, so their absence is truncation even
		// when ReadFull saw a plain io.EOF (zero bytes of the body arrived).
		if errors.Is(err, io.EOF) {
			return nil, io.ErrUnexpectedEOF
		}
		return nil, err
	}
	return payload, nil
}

// ServeEcho reads frames from conn and writes each payload straight back,
// until the peer stops sending.
//
// It returns nil when the peer closed cleanly between frames, and the error
// that stopped it otherwise. "The client went away" is not a server error;
// "the client vanished mid-frame" is worth reporting.
func ServeEcho(conn io.ReadWriter) error {
	for {
		payload, err := ReadFrame(conn)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := WriteFrame(conn, payload); err != nil {
			return err
		}
	}
}

// Package netlab implements the two layers that sit under every HTTP
// request: a length-prefixed framing protocol over a raw byte stream, and the
// TLS configuration that decides who you are actually talking to.
package netlab

import (
	"errors"
	"io"
)

// MaxFrameSize is the largest payload a single frame may carry.
//
// The length field is attacker-controlled input: a peer that claims 4 GiB
// must not talk you into allocating 4 GiB. Every framed protocol has this
// ceiling somewhere.
const MaxFrameSize = 1 << 20 // 1 MiB

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
	// TODO: reject oversize payloads before touching w, then encode the
	// length header (encoding/binary carries a big-endian byte order) and
	// write header + payload.
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
	// TODO: read exactly 4 header bytes, decode the length, check it against
	// MaxFrameSize, then read exactly that many payload bytes. io.ReadFull
	// runs the "exactly N bytes" loop for you — read its docs on which EOF
	// it returns when, and translate where the contract above differs.
	return nil, nil
}

// ServeEcho reads frames from conn and writes each payload straight back,
// until the peer stops sending.
//
// It returns nil when the peer closed cleanly between frames, and the error
// that stopped it otherwise. "The client went away" is not a server error;
// "the client vanished mid-frame" is worth reporting.
func ServeEcho(conn io.ReadWriter) error {
	// TODO: loop over ReadFrame/WriteFrame, and map a clean end-of-stream to
	// a nil return.
	return nil
}

package realtime

import (
	"errors"
	"net/http"
)

// acceptGUID is the constant RFC 6455 §1.3 mixes into the client's key. It is
// not a secret and not security: it exists so that a cache or a proxy that
// mistakes the handshake for an ordinary request cannot be tricked into
// completing it.
const acceptGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// Handshake failures, one per thing a client can get wrong.
var (
	ErrBadMethod  = errors.New("realtime: websocket upgrade must be GET")
	ErrNotUpgrade = errors.New("realtime: not a websocket upgrade request")
	ErrBadVersion = errors.New("realtime: unsupported websocket version")
	ErrBadKey     = errors.New("realtime: invalid Sec-WebSocket-Key")
	ErrBadOrigin  = errors.New("realtime: origin not allowed")
)

// AcceptKey computes the Sec-WebSocket-Accept value for a client's
// Sec-WebSocket-Key: base64(SHA-1(key + acceptGUID)).
//
// SHA-1 here is not a security decision you get to revisit — the protocol
// names it, and nothing about this construction depends on collision
// resistance.
func AcceptKey(clientKey string) string {
	// TODO: concatenate, SHA-1, standard base64.
	return ""
}

// CheckUpgrade reports whether r is a valid RFC 6455 handshake:
//
//	GET, with
//	Connection: a comma-separated list containing "upgrade" (case-insensitive)
//	Upgrade: websocket (case-insensitive)
//	Sec-WebSocket-Version: 13
//	Sec-WebSocket-Key: base64 of exactly 16 bytes
func CheckUpgrade(r *http.Request) error {
	// TODO: return the matching sentinel error, or nil. Check the method
	// first, then the two upgrade headers, then the version, then the key.
	return nil
}

// Upgrader validates handshakes for the origins it trusts.
//
// A browser attaches the user's cookies to a websocket handshake, and CORS
// does not apply to it: there is no preflight and no
// Access-Control-Allow-Origin to withhold. Checking Origin yourself is the
// only thing standing between your socket and cross-site websocket hijacking,
// where any page the user visits opens an authenticated socket to you.
type Upgrader struct {
	// Origins that may open a socket. Empty means no browser may: deny by
	// default, exactly as in the authorization lesson.
	Origins []string
}

// Handshake validates r and, when it passes, writes the 101 Switching
// Protocols response. On failure it writes the response itself and returns the
// error:
//
//	ErrBadMethod  → 405, with Allow: GET
//	ErrBadVersion → 426, with Sec-WebSocket-Version: 13 so the client can
//	                retry with a version you speak
//	ErrBadOrigin  → 403
//	anything else → 400
//
// On success the response carries Upgrade: websocket, Connection: Upgrade and
// Sec-WebSocket-Accept. In a real server the connection would then be
// hijacked and handed to a websocket library; this exercise stops at the
// handshake, which is the part that is yours to get right.
func (u Upgrader) Handshake(w http.ResponseWriter, r *http.Request) error {
	// TODO: CheckUpgrade first, then the origin rule — a request with no
	// Origin header is not a browser and passes, a request with one must
	// match u.Origins exactly. Write the status and headers described above.
	return nil
}

package realtime

import (
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"net/http"
	"slices"
	"strings"
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
	sum := sha1.Sum([]byte(clientKey + acceptGUID))
	return base64.StdEncoding.EncodeToString(sum[:])
}

// CheckUpgrade reports whether r is a valid RFC 6455 handshake:
//
//	GET, with
//	Connection: a comma-separated list containing "upgrade" (case-insensitive)
//	Upgrade: websocket (case-insensitive)
//	Sec-WebSocket-Version: 13
//	Sec-WebSocket-Key: base64 of exactly 16 bytes
func CheckUpgrade(r *http.Request) error {
	if r.Method != http.MethodGet {
		return ErrBadMethod
	}
	if !headerHasToken(r, "Connection", "upgrade") {
		return ErrNotUpgrade
	}
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return ErrNotUpgrade
	}
	if r.Header.Get("Sec-WebSocket-Version") != "13" {
		return ErrBadVersion
	}
	key, err := base64.StdEncoding.DecodeString(r.Header.Get("Sec-WebSocket-Key"))
	if err != nil || len(key) != 16 {
		return ErrBadKey
	}
	return nil
}

// headerHasToken reports whether a comma-separated header field contains a
// token, ignoring case and surrounding space. Connection: "keep-alive, Upgrade"
// is one header with two tokens, and either order is legal.
func headerHasToken(r *http.Request, header, token string) bool {
	for _, value := range r.Header.Values(header) {
		for _, part := range strings.Split(value, ",") {
			if strings.EqualFold(strings.TrimSpace(part), token) {
				return true
			}
		}
	}
	return false
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
	err := CheckUpgrade(r)
	if err == nil {
		// An absent Origin means the caller is not a browser, so there is no
		// ambient cookie to abuse; a present one is compared exactly, because
		// https://app.example.com.evil.test ends with the trusted name.
		if origin := r.Header.Get("Origin"); origin != "" && !slices.Contains(u.Origins, origin) {
			err = ErrBadOrigin
		}
	}
	if err != nil {
		switch {
		case errors.Is(err, ErrBadMethod):
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "websocket upgrade must be GET", http.StatusMethodNotAllowed)
		case errors.Is(err, ErrBadVersion):
			w.Header().Set("Sec-WebSocket-Version", "13")
			http.Error(w, "unsupported websocket version", http.StatusUpgradeRequired)
		case errors.Is(err, ErrBadOrigin):
			http.Error(w, "origin not allowed", http.StatusForbidden)
		default:
			http.Error(w, "not a websocket upgrade request", http.StatusBadRequest)
		}
		return err
	}

	header := w.Header()
	header.Set("Upgrade", "websocket")
	header.Set("Connection", "Upgrade")
	header.Set("Sec-WebSocket-Accept", AcceptKey(r.Header.Get("Sec-WebSocket-Key")))
	w.WriteHeader(http.StatusSwitchingProtocols)
	return nil
}

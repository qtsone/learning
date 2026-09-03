package auth

import (
	"encoding/base64"
	"errors"
	"time"
)

var (
	ErrMalformedToken = errors.New("auth: malformed token")
	ErrBadAlgorithm   = errors.New("auth: unexpected token algorithm")
	ErrBadSignature   = errors.New("auth: bad token signature")
	ErrTokenExpired   = errors.New("auth: token expired")
)

// jose is the JWT header. Only alg matters to a verifier, and it matters
// enormously: it is attacker-controlled input that must be checked against
// what *we* accept, never used to pick an algorithm.
type jose struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

// Claims is the payload. Registered claim names are three letters by
// convention (sub, iat, exp); anything else is yours.
type Claims struct {
	Subject   string `json:"sub"`
	Role      string `json:"role,omitempty"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

// b64/unb64 are base64url without padding, as the JWT spec requires: standard
// base64 would put '+' and '/' in a value that travels in headers and URLs.
func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func unb64(s string) ([]byte, error) { return base64.RawURLEncoding.DecodeString(s) }

// SignHS256 produces a compact JWS: base64url(header).base64url(payload).
// base64url(HMAC-SHA256 over the first two parts joined by a dot). The header
// is {"alg":"HS256","typ":"JWT"}. An empty key is a programming error.
func SignHS256(c Claims, key []byte) (string, error) {
	// TODO: marshal header and claims, build the signing input, sign it.
	return "", nil
}

// ParseHS256 verifies a token and returns its claims. The order of the checks
// is the lesson — get it wrong and the function is decorative:
//
//  1. exactly three dot-separated parts, or ErrMalformedToken
//  2. the header's alg is exactly "HS256", or ErrBadAlgorithm
//  3. the signature matches, or ErrBadSignature — compared in constant time
//  4. the payload parses, or ErrMalformedToken
//  5. exp is present (a token that never expires is malformed here) and lies
//     strictly after now, or ErrTokenExpired
//
// No claim may influence anything before step 3 succeeds.
func ParseHS256(token string, key []byte, now time.Time) (Claims, error) {
	// TODO: implement the five steps above.
	return Claims{}, nil
}

package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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

// sign returns the HMAC-SHA256 tag over the signing input.
func sign(signingInput string, key []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(signingInput))
	return mac.Sum(nil)
}

// SignHS256 produces header.payload.signature with the header fixed to HS256.
func SignHS256(c Claims, key []byte) (string, error) {
	if len(key) == 0 {
		return "", errors.New("auth: empty signing key")
	}
	header, err := json.Marshal(jose{Alg: "HS256", Typ: "JWT"})
	if err != nil {
		return "", fmt.Errorf("marshal header: %w", err)
	}
	payload, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("marshal claims: %w", err)
	}
	signingInput := b64(header) + "." + b64(payload)
	return signingInput + "." + b64(sign(signingInput, key)), nil
}

// ParseHS256 verifies a token and returns its claims. The order below is the
// lesson: reject the algorithm we did not choose, verify the signature, and
// only then let any claim influence a decision.
func ParseHS256(token string, key []byte, now time.Time) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, ErrMalformedToken
	}

	rawHeader, err := unb64(parts[0])
	if err != nil {
		return Claims{}, ErrMalformedToken
	}
	var header jose
	if err := json.Unmarshal(rawHeader, &header); err != nil {
		return Claims{}, ErrMalformedToken
	}
	// Pinned, not read: "none" and "RS256" are both rejected here.
	if header.Alg != "HS256" {
		return Claims{}, ErrBadAlgorithm
	}

	signature, err := unb64(parts[2])
	if err != nil {
		return Claims{}, ErrMalformedToken
	}
	// hmac.Equal compares in constant time: a byte-by-byte == would leak how
	// many leading bytes an attacker guessed right.
	if !hmac.Equal(signature, sign(parts[0]+"."+parts[1], key)) {
		return Claims{}, ErrBadSignature
	}

	rawClaims, err := unb64(parts[1])
	if err != nil {
		return Claims{}, ErrMalformedToken
	}
	var claims Claims
	if err := json.Unmarshal(rawClaims, &claims); err != nil {
		return Claims{}, ErrMalformedToken
	}
	if claims.ExpiresAt == 0 {
		// A token with no exp never expires. Refuse to mint meaning into it.
		return Claims{}, ErrMalformedToken
	}
	if !now.Before(time.Unix(claims.ExpiresAt, 0)) {
		return Claims{}, ErrTokenExpired
	}
	return claims, nil
}

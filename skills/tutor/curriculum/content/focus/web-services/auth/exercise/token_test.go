package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

var (
	testKey  = []byte("a 32-byte-ish HMAC key for tests")
	otherKey = []byte("a different key of similar size!")
)

func testTokenClaims(now time.Time) Claims {
	return Claims{
		Subject:   "u1",
		Role:      RoleUser,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(15 * time.Minute).Unix(),
	}
}

func mustSign(t *testing.T, c Claims, key []byte) string {
	t.Helper()
	token, err := SignHS256(c, key)
	if err != nil {
		t.Fatalf("SignHS256(…) = error %v, want nil", err)
	}
	if len(strings.Split(token, ".")) != 3 {
		t.Fatalf("SignHS256(…) = %q, want three dot-separated parts", token)
	}
	return token
}

// craft builds a token the way an attacker would: arbitrary header, arbitrary
// payload, and a signature only if we have a key worth signing with.
func craft(t *testing.T, header map[string]string, payload []byte, key []byte) string {
	t.Helper()
	rawHeader, err := json.Marshal(header)
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}
	signingInput := base64.RawURLEncoding.EncodeToString(rawHeader) + "." +
		base64.RawURLEncoding.EncodeToString(payload)
	if key == nil {
		// An alg=none token carries an empty signature — three parts, still.
		return signingInput + "."
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(signingInput))
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func claimsJSON(t *testing.T, c Claims) []byte {
	t.Helper()
	raw, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}
	return raw
}

func TestSignHS256AndRoundTrip(t *testing.T) {
	want := testTokenClaims(testStart)
	token := mustSign(t, want, testKey)

	if strings.Contains(token, "=") {
		t.Errorf("token %q contains padding; JWT uses base64url without padding", token)
	}

	parts := strings.Split(token, ".")
	rawHeader, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatalf("header %q is not unpadded base64url: %v", parts[0], err)
	}
	var header struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(rawHeader, &header); err != nil {
		t.Fatalf("header %q is not JSON: %v", rawHeader, err)
	}
	if header.Alg != "HS256" || header.Typ != "JWT" {
		t.Errorf("header = %+v, want alg HS256 and typ JWT", header)
	}

	got, err := ParseHS256(token, testKey, testStart)
	if err != nil {
		t.Fatalf("ParseHS256(fresh token) = error %v, want nil", err)
	}
	if got != want {
		t.Errorf("round trip = %+v, want %+v", got, want)
	}
}

func TestParseHS256Rejections(t *testing.T) {
	now := testStart
	claims := testTokenClaims(now)
	valid := mustSign(t, claims, testKey)
	exp := time.Unix(claims.ExpiresAt, 0)

	other := claims
	other.Subject = "u2"
	other.Role = RoleAdmin
	parts := strings.Split(valid, ".")
	tamperedPayload := parts[0] + "." +
		base64.RawURLEncoding.EncodeToString(claimsJSON(t, other)) + "." + parts[2]

	noExp := claims
	noExp.ExpiresAt = 0

	cases := []struct {
		name    string
		token   string
		key     []byte
		now     time.Time
		wantErr error
	}{
		{"two segments", "header.payload", testKey, now, ErrMalformedToken},
		{"header is not base64url", "n0t-b@se64.x.y", testKey, now, ErrMalformedToken},
		{
			"alg none with an empty signature",
			craft(t, map[string]string{"alg": "none", "typ": "JWT"}, claimsJSON(t, claims), nil),
			testKey, now, ErrBadAlgorithm,
		},
		{
			"alg none that is signed anyway",
			craft(t, map[string]string{"alg": "none", "typ": "JWT"}, claimsJSON(t, claims), testKey),
			testKey, now, ErrBadAlgorithm,
		},
		{
			"algorithm confusion: RS256 header, HMAC signature",
			craft(t, map[string]string{"alg": "RS256", "typ": "JWT"}, claimsJSON(t, claims), testKey),
			testKey, now, ErrBadAlgorithm,
		},
		{
			"alg is case sensitive",
			craft(t, map[string]string{"alg": "hs256", "typ": "JWT"}, claimsJSON(t, claims), testKey),
			testKey, now, ErrBadAlgorithm,
		},
		{"payload swapped, signature kept", tamperedPayload, testKey, now, ErrBadSignature},
		{"signed with another key", valid, otherKey, now, ErrBadSignature},
		{"signature is not base64url", parts[0] + "." + parts[1] + ".@@@", testKey, now, ErrMalformedToken},
		{
			"payload is not JSON",
			craft(t, map[string]string{"alg": "HS256", "typ": "JWT"}, []byte("not json"), testKey),
			testKey, now, ErrMalformedToken,
		},
		{
			"no exp claim",
			craft(t, map[string]string{"alg": "HS256", "typ": "JWT"}, claimsJSON(t, noExp), testKey),
			testKey, now, ErrMalformedToken,
		},
		{"expired", valid, testKey, exp, ErrTokenExpired},
		// The signature is checked before any claim is trusted, so a tampered
		// expired token reports the tampering, not the expiry.
		{"expired and tampered", tamperedPayload, testKey, exp.Add(time.Hour), ErrBadSignature},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ParseHS256(c.token, c.key, c.now)
			if !errors.Is(err, c.wantErr) {
				t.Fatalf("ParseHS256(…) = (%+v, %v), want error %v", got, err, c.wantErr)
			}
			if got != (Claims{}) {
				t.Errorf("ParseHS256(…) returned claims %+v on failure, want the zero value", got)
			}
		})
	}
}

func TestParseHS256ExpiryBoundary(t *testing.T) {
	claims := testTokenClaims(testStart)
	token := mustSign(t, claims, testKey)
	exp := time.Unix(claims.ExpiresAt, 0)

	if _, err := ParseHS256(token, testKey, exp.Add(-time.Second)); err != nil {
		t.Errorf("ParseHS256() one second before exp = %v, want nil", err)
	}
	if _, err := ParseHS256(token, testKey, exp); !errors.Is(err, ErrTokenExpired) {
		t.Errorf("ParseHS256() exactly at exp = %v, want ErrTokenExpired", err)
	}
}

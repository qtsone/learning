package apiperf

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// ETag returns a strong entity tag for body: SHA-256, hex, in double quotes.
//
// Hashing what you are about to send is the option that cannot be wrong. A
// cheaper tag — a version column, a last-modified stamp — avoids the hash but
// has to account for every input that changes the output, including the ones
// added next quarter.
func ETag(body []byte) string {
	sum := sha256.Sum256(body)
	return `"` + hex.EncodeToString(sum[:]) + `"`
}

// MatchETag reports whether an If-None-Match header matches etag, using RFC
// 9110's weak comparison: `W/"abc"` and `"abc"` are the same entity for the
// purposes of a conditional GET.
//
// The list is split on commas, which is right for every tag this service
// issues (hex digests) but not for the letter of RFC 9110: an entity tag may
// itself contain a comma. Parsing quoted strings properly is the fix if you
// ever accept tags you did not generate.
func MatchETag(ifNoneMatch, etag string) bool {
	if ifNoneMatch == "" || etag == "" {
		return false
	}
	want := strings.TrimPrefix(etag, "W/")
	for _, candidate := range strings.Split(ifNoneMatch, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" {
			return true
		}
		if strings.TrimPrefix(candidate, "W/") == want {
			return true
		}
	}
	return false
}

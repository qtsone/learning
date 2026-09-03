package apiperf

// ETag returns a strong entity tag for body: the SHA-256 of the bytes, hex
// encoded, wrapped in double quotes — `"a1b2…"`, 66 characters.
//
// The quotes are part of the value, not decoration; a tag without them is not
// a valid ETag and caches are entitled to ignore it. Hashing the body is the
// blunt option: it is always correct because it cannot disagree with what you
// send, and it costs a pass over bytes you have already rendered. A cheaper
// tag derived from a version column or a last-modified timestamp avoids the
// hash, at the price of having to be right about every input that changes the
// output.
func ETag(body []byte) string {
	// TODO: crypto/sha256 + encoding/hex, quoted.
	return ""
}

// MatchETag reports whether an If-None-Match header matches etag, per RFC 9110.
//
// The rules, all of which the tests check:
//
//   - an empty header never matches;
//   - "*" matches any non-empty etag;
//   - the header is a comma-separated list, each entry optionally surrounded by
//     spaces, and a match against any entry is a match;
//   - comparison is *weak*: `W/"abc"` and `"abc"` match each other. Strong
//     comparison exists for range requests, which this endpoint does not serve.
//
// Getting this wrong is quiet in both directions: too strict and every
// conditional request re-downloads the body, too loose and a client caches
// somebody else's page forever.
func MatchETag(ifNoneMatch, etag string) bool {
	// TODO: strings.Split on ",", strings.TrimSpace, strip a "W/" prefix from
	// both sides, compare.
	return false
}

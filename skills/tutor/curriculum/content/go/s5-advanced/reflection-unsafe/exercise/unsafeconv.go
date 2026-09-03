package conf

// UnsafeString returns a string that shares b's backing array — no copy, no
// allocation, whatever the length of b.
//
// The caller owns one invariant: b must not be modified for as long as the
// returned string is reachable. Go strings are immutable by contract, and
// everything from the compiler to map hashing assumes it. Break the
// invariant and you get silently wrong results, not a crash.
//
// Use it for read-only scanning of a buffer you control. Never hand the
// result to something that keeps it.
func UnsafeString(b []byte) string {
	// TODO: return unsafe.String(unsafe.SliceData(b), len(b)) — and read
	// the docs for both helpers before you do.
	return string(b)
}

// UnsafeBytes returns a byte slice that shares s's backing array.
//
// Writing to the result is undefined behaviour: string data may live in
// read-only memory, so a write can fault, and shared string literals mean a
// successful write can corrupt unrelated code. Read-only, always.
func UnsafeBytes(s string) []byte {
	// TODO: return unsafe.Slice(unsafe.StringData(s), len(s)).
	return []byte(s)
}

// Lookup finds key in an env-style blob of "KEY=VALUE" lines and returns a
// copy of its value. Blank lines and lines starting with '#' are ignored;
// surrounding spaces are trimmed; the first '=' separates key from value, so
// values may contain '='.
//
// Scanning must not copy the blob — that is what UnsafeString is for. The
// returned value must be a copy (see strings.Clone): it outlives the call,
// and the caller may reuse or overwrite data afterwards.
func Lookup(data []byte, key string) (string, bool) {
	// TODO: implement. Walk the blob as one aliased string and use
	// strings.Cut twice: once to peel off a line, once to split it at '='.
	return "", false
}

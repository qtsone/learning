package conf

import (
	"strings"
	"unsafe"
)

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
	// unsafe.SliceData is nil for an empty slice, and unsafe.String accepts
	// a nil pointer when the length is zero — so no special case is needed.
	return unsafe.String(unsafe.SliceData(b), len(b))
}

// UnsafeBytes returns a byte slice that shares s's backing array.
//
// Writing to the result is undefined behaviour: string data may live in
// read-only memory, so a write can fault, and shared string literals mean a
// successful write can corrupt unrelated code. Read-only, always.
func UnsafeBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
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
	// One aliased view of the whole blob. rest, line, k and v are all
	// substrings of it: no copies until the very last line of this function.
	rest := UnsafeString(data)
	for rest != "" {
		var line string
		line, rest, _ = strings.Cut(rest, "\n")
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(k) != key {
			continue
		}
		// The one place a copy is mandatory: the result outlives data, and
		// nothing stops the caller from reusing that buffer.
		return strings.Clone(strings.TrimSpace(v)), true
	}
	return "", false
}

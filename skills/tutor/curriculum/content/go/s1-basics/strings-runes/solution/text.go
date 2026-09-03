// Package text solves small text-processing tasks with strings, bytes, and runes.
package text

import "strings"

// CountRunes returns the number of characters (runes) in s — which for
// non-ASCII text is not the same as len(s), the number of bytes.
func CountRunes(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}

// Reverse returns s with its characters in reverse order. Multi-byte
// characters must survive intact: Reverse("héllo") is "olléh".
func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// CleanFields splits a comma-separated list, trims the spaces around each
// entry, and drops entries that are empty after trimming.
func CleanFields(csv string) []string {
	var fields []string
	for _, f := range strings.Split(csv, ",") {
		f = strings.TrimSpace(f)
		if f != "" {
			fields = append(fields, f)
		}
	}
	return fields
}

// Slug turns a title into a lower-case, hyphen-separated identifier:
// Slug("  Go Is Fun ") is "go-is-fun".
func Slug(title string) string {
	return strings.Join(strings.Fields(strings.ToLower(title)), "-")
}

// Initials returns the upper-cased first character of each word in name,
// each followed by a period: Initials("ada lovelace") is "A.L.".
func Initials(name string) string {
	var b strings.Builder
	for _, word := range strings.Fields(name) {
		first := []rune(word)[0]
		b.WriteString(strings.ToUpper(string(first)))
		b.WriteRune('.')
	}
	return b.String()
}

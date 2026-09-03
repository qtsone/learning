// Package text solves small text-processing tasks with strings, bytes, and runes.
package text

// CountRunes returns the number of characters (runes) in s — which for
// non-ASCII text is not the same as len(s), the number of bytes.
func CountRunes(s string) int {
	// TODO: range over s — each step of range decodes one rune.
	return 0
}

// Reverse returns s with its characters in reverse order. Multi-byte
// characters must survive intact: Reverse("héllo") is "olléh".
func Reverse(s string) string {
	// TODO: convert to []rune, swap ends toward the middle, convert back.
	return ""
}

// CleanFields splits a comma-separated list, trims the spaces around each
// entry, and drops entries that are empty after trimming.
func CleanFields(csv string) []string {
	// TODO: strings.Split on ",", strings.TrimSpace each piece, keep non-empty ones.
	return nil
}

// Slug turns a title into a lower-case, hyphen-separated identifier:
// Slug("  Go Is Fun ") is "go-is-fun".
func Slug(title string) string {
	// TODO: lower-case, split into words with strings.Fields, join with "-".
	return ""
}

// Initials returns the upper-cased first character of each word in name,
// each followed by a period: Initials("ada lovelace") is "A.L.".
func Initials(name string) string {
	// TODO: build the result with a strings.Builder — and mind multi-byte
	// first letters: word[0] is a byte, not a character.
	return ""
}

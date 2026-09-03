// Package pantry tracks kitchen stock with Go's built-in map type.
package pantry

// Count returns how many times each word appears in words.
func Count(words []string) map[string]int {
	// TODO: build the counts in a map — reading a missing key yields 0.
	return nil
}

// Describe reports the stock level of item: "not stocked" when the pantry
// has no entry for it, "out of stock" when the entry exists with count 0,
// and "<n> in stock" otherwise.
func Describe(pantry map[string]int, item string) string {
	// TODO: use the comma-ok idiom to tell a missing entry from a zero one.
	return ""
}

// Take removes n units of item from the pantry and reports whether it
// succeeded. Taking the last unit deletes the entry entirely; when the item
// is missing or stock is insufficient, the pantry is left untouched.
func Take(pantry map[string]int, item string, n int) bool {
	// TODO: check the stock first, then subtract — delete the entry at zero.
	return false
}

// SortedItems returns the pantry's item names in alphabetical order.
func SortedItems(pantry map[string]int) []string {
	// TODO: collect the keys into a slice, then sort it with slices.Sort.
	return nil
}

// NewSet returns a set containing each distinct item.
func NewSet(items []string) map[string]struct{} {
	// TODO: map each item to struct{}{} — the value carries no information.
	return nil
}

// Has reports whether item is a member of set.
func Has(set map[string]struct{}, item string) bool {
	// TODO: membership is one comma-ok lookup.
	return false
}

// Package pantry tracks kitchen stock with Go's built-in map type.
package pantry

import (
	"fmt"
	"slices"
)

// Count returns how many times each word appears in words.
func Count(words []string) map[string]int {
	counts := make(map[string]int)
	for _, w := range words {
		counts[w]++
	}
	return counts
}

// Describe reports the stock level of item: "not stocked" when the pantry
// has no entry for it, "out of stock" when the entry exists with count 0,
// and "<n> in stock" otherwise.
func Describe(pantry map[string]int, item string) string {
	n, ok := pantry[item]
	switch {
	case !ok:
		return "not stocked"
	case n == 0:
		return "out of stock"
	default:
		return fmt.Sprintf("%d in stock", n)
	}
}

// Take removes n units of item from the pantry and reports whether it
// succeeded. Taking the last unit deletes the entry entirely; when the item
// is missing or stock is insufficient, the pantry is left untouched.
func Take(pantry map[string]int, item string, n int) bool {
	have, ok := pantry[item]
	if !ok || have < n {
		return false
	}
	if have == n {
		delete(pantry, item)
	} else {
		pantry[item] = have - n
	}
	return true
}

// SortedItems returns the pantry's item names in alphabetical order.
func SortedItems(pantry map[string]int) []string {
	items := make([]string, 0, len(pantry))
	for item := range pantry {
		items = append(items, item)
	}
	slices.Sort(items)
	return items
}

// NewSet returns a set containing each distinct item.
func NewSet(items []string) map[string]struct{} {
	set := make(map[string]struct{})
	for _, item := range items {
		set[item] = struct{}{}
	}
	return set
}

// Has reports whether item is a member of set.
func Has(set map[string]struct{}, item string) bool {
	_, ok := set[item]
	return ok
}

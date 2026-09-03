// Package sliceops provides generic helpers over slices.
//
// The signatures and the Number constraint are final — implement the bodies.
// Production code would use the standard library's slices package; here you
// build the essentials yourself to learn how type parameters work.
package sliceops

import "fmt"

// Number is satisfied by any type whose underlying type is int, int64, or
// float64 — the ~ is what admits defined types such as the tests' Celsius.
type Number interface {
	~int | ~int64 | ~float64
}

// Map returns a new slice containing f applied to each element of s, in order.
func Map[T, U any](s []T, f func(T) U) []U {
	out := make([]U, 0, len(s))
	for _, v := range s {
		out = append(out, f(v))
	}
	return out
}

// Filter returns the elements of s for which keep returns true, in order.
func Filter[T any](s []T, keep func(T) bool) []T {
	var out []T
	for _, v := range s {
		if keep(v) {
			out = append(out, v)
		}
	}
	return out
}

// IndexOf returns the index of the first occurrence of target in s, or -1 if
// target is not present.
func IndexOf[T comparable](s []T, target T) int {
	for i, v := range s {
		if v == target {
			return i
		}
	}
	return -1
}

// Sum returns the total of all elements of s.
func Sum[T Number](s []T) T {
	var total T
	for _, v := range s {
		total += v
	}
	return total
}

// DescribeAll returns each element's String() result, in order.
func DescribeAll[T fmt.Stringer](s []T) []string {
	return Map(s, func(v T) string { return v.String() })
}

package inspect

// Describe returns a short human-readable description of v, chosen by
// its dynamic type. See LESSON.md acceptance criterion 1 for the exact
// formats.
func Describe(v any) string {
	// TODO: one type switch — nil, string, int, bool, []string,
	// and a default using %T.
	return ""
}

// Stringify returns a string form of v when v is a plain string or a
// fmt.Stringer; ok reports whether it could. It never panics.
func Stringify(v any) (s string, ok bool) {
	// TODO: two comma-ok assertions — no type switch needed.
	return "", false
}

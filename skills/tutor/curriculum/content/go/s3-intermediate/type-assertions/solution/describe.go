package inspect

import "fmt"

// Describe returns a short human-readable description of v, chosen by
// its dynamic type.
func Describe(v any) string {
	switch v := v.(type) {
	case nil:
		return "nothing"
	case string:
		return fmt.Sprintf("text %q", v)
	case int:
		return fmt.Sprintf("number %d", v)
	case bool:
		return fmt.Sprintf("boolean %t", v)
	case []string:
		return fmt.Sprintf("list of %d items", len(v))
	default:
		return fmt.Sprintf("unexpected type %T", v)
	}
}

// Stringify returns a string form of v when v is a plain string or a
// fmt.Stringer; ok reports whether it could. It never panics.
func Stringify(v any) (string, bool) {
	if s, ok := v.(string); ok {
		return s, true
	}
	if s, ok := v.(fmt.Stringer); ok {
		return s.String(), true
	}
	return "", false
}

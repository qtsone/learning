package vault

import "fmt"

// ValidateUsername decides whether name is acceptable at the trust boundary.
// Allowlist: 3-32 bytes, first byte a lowercase ASCII letter, every byte a
// lowercase letter, digit, or underscore.
func ValidateUsername(name string) error {
	if len(name) < 3 || len(name) > 32 {
		return fmt.Errorf("username must be 3-32 bytes, got %d", len(name))
	}
	if name[0] < 'a' || name[0] > 'z' {
		return fmt.Errorf("username must start with a lowercase letter, got %q", name[0])
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c >= 'a' && c <= 'z':
		case c >= '0' && c <= '9':
		case c == '_':
		default:
			return fmt.Errorf("username contains forbidden byte %q at index %d", c, i)
		}
	}
	return nil
}

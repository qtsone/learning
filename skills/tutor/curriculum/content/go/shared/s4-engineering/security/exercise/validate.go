package vault

// ValidateUsername decides whether name is acceptable at the trust boundary.
// The rules are in LESSON.md acceptance criteria: 3-32 bytes, first byte a
// lowercase ASCII letter, every byte a lowercase letter, digit, or underscore.
func ValidateUsername(name string) error {
	// TODO: implement the allowlist rules. Right now everything gets in.
	return nil
}

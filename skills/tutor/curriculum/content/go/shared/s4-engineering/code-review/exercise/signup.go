// Package signup validates and stores new user accounts.
package signup

import (
	"database/sql"
	"errors"
	"fmt"
)

// ErrInvalidUsername reports a username that fails the rules in
// ValidUsername.
var ErrInvalidUsername = errors.New("signup: invalid username")

// ValidUsername reports whether name is 3-20 characters long and
// contains only lowercase letters, digits, or hyphens, starting with
// a letter.
func ValidUsername(name string) bool {
	if len(name) < 3 || len(name) > 20 {
		return false
	}
	if name[0] < 'a' || name[0] > 'z' {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '-':
		default:
			return false
		}
	}
	return true
}

// Register stores a new account for username. It returns
// ErrInvalidUsername if the username fails validation, and wraps any
// database error otherwise.
func Register(db *sql.DB, username string) error {
	if !ValidUsername(username) {
		return fmt.Errorf("%w: %q", ErrInvalidUsername, username)
	}
	if _, err := db.Exec("INSERT INTO users (username) VALUES (?)", username); err != nil {
		return fmt.Errorf("store user %q: %w", username, err)
	}
	return nil
}

package vault

import (
	"database/sql"
	"errors"
)

// ErrNotFound is returned when no user matches the requested name.
var ErrNotFound = errors.New("user not found")

// User is a row in the users table.
type User struct {
	ID   int64
	Name string
}

// LookupUser returns the user with the given name.
//
// TODO: seeded vulnerability — read this function as an attacker would.
// What can a hostile name make the query do?
func LookupUser(db *sql.DB, name string) (User, error) {
	row := db.QueryRow("SELECT id, name FROM users WHERE name = '" + name + "'")
	var u User
	if err := row.Scan(&u.ID, &u.Name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, err
	}
	return u, nil
}

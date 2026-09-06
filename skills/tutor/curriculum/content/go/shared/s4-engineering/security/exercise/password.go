package vault

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashPassword turns a password into a string safe to store in the database.
//
// TODO: seeded vulnerability — SHA-256 is the wrong tool here. It is
// unsalted (equal passwords produce equal hashes) and far too fast to slow
// down offline guessing. Replace it with bcrypt
// (golang.org/x/crypto/bcrypt — already in go.mod).
func HashPassword(password string) (string, error) {
	sum := sha256.Sum256([]byte(password))
	return hex.EncodeToString(sum[:]), nil
}

// VerifyPassword reports whether password matches the stored hash.
//
// TODO: update alongside HashPassword — bcrypt hashes carry their own salt
// and cost, so verification goes through the bcrypt package too.
func VerifyPassword(hash, password string) bool {
	sum := sha256.Sum256([]byte(password))
	return hash == hex.EncodeToString(sum[:])
}

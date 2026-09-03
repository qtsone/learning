package vault

import "golang.org/x/crypto/bcrypt"

// HashPassword turns a password into a string safe to store in the database.
// bcrypt generates a random salt and embeds salt + cost in the hash itself,
// and its deliberate slowness is the defense against offline guessing.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword reports whether password matches the stored hash.
func VerifyPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

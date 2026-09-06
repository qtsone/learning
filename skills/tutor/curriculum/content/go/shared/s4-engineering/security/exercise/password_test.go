package vault

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestPasswordRoundTrip(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	if !VerifyPassword(hash, "correct horse battery staple") {
		t.Errorf("VerifyPassword rejects the password its own hash was made from")
	}
	if VerifyPassword(hash, "wrong password") {
		t.Errorf("VerifyPassword accepts a wrong password")
	}
}

func TestPasswordHashesAreSalted(t *testing.T) {
	h1, err := HashPassword("hunter2")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	h2, err := HashPassword("hunter2")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	if h1 == h2 {
		t.Errorf("hashing the same password twice produced identical hashes %q — unsalted: everyone with password hunter2 shares one hash, and a rainbow table cracks them all at once", h1)
	}
}

func TestPasswordHashIsNotFastHash(t *testing.T) {
	hash, err := HashPassword("hunter2")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	sum := sha256.Sum256([]byte("hunter2"))
	if hash == hex.EncodeToString(sum[:]) {
		t.Errorf("HashPassword returns plain SHA-256 — a GPU guesses billions of those per second; use a password KDF (bcrypt)")
	}
}

func TestVerifyPasswordToleratesGarbageHash(t *testing.T) {
	if VerifyPassword("not-a-real-hash", "anything") {
		t.Errorf("VerifyPassword(garbage hash) = true, want false")
	}
}

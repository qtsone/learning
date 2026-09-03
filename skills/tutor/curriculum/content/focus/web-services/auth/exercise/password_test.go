package auth

import (
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashIsBcryptAndSalted(t *testing.T) {
	h := Hasher{Cost: testCost}

	first, err := h.Hash(testPassword)
	if err != nil {
		t.Fatalf("Hash(…) = error %v, want nil", err)
	}
	if strings.Contains(first, testPassword) {
		t.Fatalf("Hash(…) = %q, which contains the password itself", first)
	}
	if !strings.HasPrefix(first, "$2") {
		t.Errorf("Hash(…) = %q, want a bcrypt hash starting with $2", first)
	}
	if cost, err := bcrypt.Cost([]byte(first)); err != nil {
		t.Errorf("bcrypt.Cost(%q) = error %v, want a readable cost", first, err)
	} else if cost != testCost {
		t.Errorf("hash cost = %d, want %d (Hasher.Cost must be honored)", cost, testCost)
	}

	second, err := h.Hash(testPassword)
	if err != nil {
		t.Fatalf("Hash(…) = error %v, want nil", err)
	}
	if first == second {
		t.Errorf("hashing the same password twice produced identical output %q; a per-password salt must make them differ", first)
	}
	if err := h.Verify(second, testPassword); err != nil {
		t.Errorf("Verify(second hash, password) = %v, want nil", err)
	}
}

func TestHashRejectsOverlongPassword(t *testing.T) {
	h := Hasher{Cost: testCost}
	if _, err := h.Hash(strings.Repeat("x", MaxPasswordBytes+1)); !errors.Is(err, ErrPasswordTooLong) {
		t.Errorf("Hash(73 bytes) = error %v, want ErrPasswordTooLong (bcrypt would silently ignore the tail)", err)
	}
	if _, err := h.Hash(strings.Repeat("x", MaxPasswordBytes)); err != nil {
		t.Errorf("Hash(72 bytes) = error %v, want nil", err)
	}
}

func TestVerify(t *testing.T) {
	h := Hasher{Cost: testCost}
	encoded, err := h.Hash(testPassword)
	if err != nil {
		t.Fatalf("Hash(…) = error %v, want nil", err)
	}

	if err := h.Verify(encoded, testPassword); err != nil {
		t.Errorf("Verify(hash, correct password) = %v, want nil", err)
	}
	if err := h.Verify(encoded, "not the password"); !errors.Is(err, ErrBadCredentials) {
		t.Errorf("Verify(hash, wrong password) = %v, want ErrBadCredentials", err)
	}

	// A corrupt stored hash is a server bug, not a user mistake: it must be
	// distinguishable so it can be logged and fixed.
	err = h.Verify("definitely-not-a-bcrypt-hash", testPassword)
	if err == nil {
		t.Errorf("Verify(garbage hash, …) = nil, want an error")
	} else if errors.Is(err, ErrBadCredentials) {
		t.Errorf("Verify(garbage hash, …) = ErrBadCredentials, want an error reporting the broken hash instead")
	}
}

func TestNeedsRehash(t *testing.T) {
	encoded, err := Hasher{Cost: testCost}.Hash(testPassword)
	if err != nil {
		t.Fatalf("Hash(…) = error %v, want nil", err)
	}

	cases := []struct {
		name    string
		hasher  Hasher
		encoded string
		want    bool
	}{
		{"same cost", Hasher{Cost: testCost}, encoded, false},
		{"cost raised since the hash was made", Hasher{Cost: testCost + 2}, encoded, true},
		{"unreadable hash", Hasher{Cost: testCost}, "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.hasher.NeedsRehash(c.encoded); got != c.want {
				t.Errorf("NeedsRehash(%q) with cost %d = %v, want %v", c.encoded, c.hasher.Cost, got, c.want)
			}
		})
	}
}

package vault

import (
	"strings"
	"testing"
)

func TestValidateUsernameAccepts(t *testing.T) {
	valid := []string{
		"gopher",
		"abc",
		"a1_b2_c3",
		"z" + strings.Repeat("a", 31), // exactly 32 bytes
	}
	for _, name := range valid {
		if err := ValidateUsername(name); err != nil {
			t.Errorf("ValidateUsername(%q) = %v, want nil (valid name rejected)", name, err)
		}
	}
}

func TestValidateUsernameRejects(t *testing.T) {
	cases := []struct {
		reason string
		in     string
	}{
		{"empty", ""},
		{"too short", "ab"},
		{"too long", strings.Repeat("a", 33)},
		{"uppercase", "Gopher"},
		{"space", "go pher"},
		{"starts with digit", "1abc"},
		{"starts with underscore", "_abc"},
		{"non-ASCII", "héllo"},
		{"control byte", "abc\x00def"},
		{"path metacharacters", "../etc"},
		{"quote for injection", "bob'; drop table users --"},
	}
	for _, c := range cases {
		t.Run(c.reason, func(t *testing.T) {
			if err := ValidateUsername(c.in); err == nil {
				t.Errorf("ValidateUsername(%q) = nil, want error (%s)", c.in, c.reason)
			}
		})
	}
}

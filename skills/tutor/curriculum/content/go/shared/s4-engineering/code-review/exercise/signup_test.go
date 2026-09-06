package signup

import "testing"

func TestValidUsername(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"simple", "gopher", true},
		{"digits and hyphen", "go-4th", true},
		{"minimum length", "abc", true},
		{"too short", "ab", false},
		{"too long", "abcdefghijklmnopqrstu", false},
		{"starts with digit", "4gopher", false},
		{"starts with hyphen", "-gopher", false},
		{"uppercase", "Gopher", false},
		{"illegal character", "go_pher", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidUsername(tt.in); got != tt.want {
				t.Errorf("ValidUsername(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

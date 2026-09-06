package sq

import "testing"

func TestBalanced(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"empty", "", true},
		{"single pair", "()", true},
		{"all three kinds", "([]{})", true},
		{"nested", "{[()]}", true},
		{"non-brackets ignored", "a(b)c", true},
		{"interleaved", "([)]", false},
		{"unclosed openers", "(((", false},
		{"extra closer", "())", false},
		{"closer first", ")(", false},
		{"lone closer among text", "fn(x]", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Balanced(c.in); got != c.want {
				t.Errorf("Balanced(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

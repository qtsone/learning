package text

import (
	"slices"
	"testing"
)

func TestCountRunes(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int
	}{
		{"empty", "", 0},
		{"ascii", "go", 2},
		{"accented", "héllo", 5},
		{"japanese", "日本語", 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := CountRunes(c.in); got != c.want {
				t.Errorf("CountRunes(%q) = %d, want %d (len is %d — bytes are not characters)",
					c.in, got, c.want, len(c.in))
			}
		})
	}
}

func TestReverse(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"ascii", "go", "og"},
		{"accented", "héllo", "olléh"},
		{"japanese", "日本語", "語本日"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Reverse(c.in); got != c.want {
				t.Errorf("Reverse(%q) = %q, want %q (mangled output means bytes were reversed, not runes)",
					c.in, got, c.want)
			}
		})
	}
}

func TestCleanFields(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"plain", "a,b,c", []string{"a", "b", "c"}},
		{"padded", " a , b , c ", []string{"a", "b", "c"}},
		{"empties dropped", " a ,, b , ", []string{"a", "b"}},
		{"only separators", " , , ", nil},
		{"empty input", "", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := CleanFields(c.in); !slices.Equal(got, c.want) {
				t.Errorf("CleanFields(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestSlug(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"simple", "Go Is Fun", "go-is-fun"},
		{"extra spaces", "  Go   Is  Fun ", "go-is-fun"},
		{"already clean", "hello", "hello"},
		{"accented", "Héllo World", "héllo-world"},
		{"empty", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Slug(c.in); got != c.want {
				t.Errorf("Slug(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestInitials(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"two words", "ada lovelace", "A.L."},
		{"one word", "grace", "G."},
		{"extra spaces", "  alan   turing ", "A.T."},
		{"accented first letter", "émile zola", "É.Z."},
		{"empty", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Initials(c.in); got != c.want {
				t.Errorf("Initials(%q) = %q, want %q (a broken É means word[0] grabbed one byte of a two-byte rune)",
					c.in, got, c.want)
			}
		})
	}
}

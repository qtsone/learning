package main

import "testing"

func TestGreeting(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"named", "Gopher", "Hello, Gopher!"},
		{"another name", "Ada", "Hello, Ada!"},
		{"empty falls back to world", "", "Hello, world!"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Greeting(c.in); got != c.want {
				t.Errorf("Greeting(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

package inspect

import (
	"fmt"
	"testing"
)

type celsius int

func (c celsius) String() string { return fmt.Sprintf("%d°C", int(c)) }

func TestDescribe(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want string
	}{
		{"nil", nil, "nothing"},
		{"string", "hello", `text "hello"`},
		{"empty string", "", `text ""`},
		{"int", 42, "number 42"},
		{"negative int", -7, "number -7"},
		{"bool true", true, "boolean true"},
		{"bool false", false, "boolean false"},
		{"string slice", []string{"a", "b", "c"}, "list of 3 items"},
		{"empty slice", []string{}, "list of 0 items"},
		{"float is not handled", 3.14, "unexpected type float64"},
		{"int64 is not int", int64(1), "unexpected type int64"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Describe(c.in); got != c.want {
				t.Errorf("Describe(%#v) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestStringify(t *testing.T) {
	cases := []struct {
		name   string
		in     any
		want   string
		wantOK bool
	}{
		{"plain string", "hi", "hi", true},
		{"Stringer uses String()", celsius(21), "21°C", true},
		{"int has no string form", 42, "", false},
		{"nil must not panic", nil, "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := Stringify(c.in)
			if got != c.want || ok != c.wantOK {
				t.Errorf("Stringify(%#v) = (%q, %t), want (%q, %t)",
					c.in, got, ok, c.want, c.wantOK)
			}
		})
	}
}

// This file is in package temperature_test, not package temperature.
// It sits outside the package it tests, so it can only reach exported
// names — the same view any importer of the package gets.
package temperature_test

import (
	"math"
	"testing"

	"tutor.local/packages-modules/temperature"
)

const epsilon = 1e-6

func TestCToF(t *testing.T) {
	cases := []struct {
		name string
		in   float64
		want float64
	}{
		{"freezing point", 0, 32},
		{"boiling point", 100, 212},
		{"room-ish", 25, 77},
		{"scales cross", -40, -40},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := temperature.CToF(c.in); math.Abs(got-c.want) > epsilon {
				t.Errorf("CToF(%v) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

func TestFToC(t *testing.T) {
	cases := []struct {
		name string
		in   float64
		want float64
	}{
		{"freezing point", 32, 0},
		{"boiling point", 212, 100},
		{"body temperature", 98.6, 37},
		{"scales cross", -40, -40},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := temperature.FToC(c.in); math.Abs(got-c.want) > epsilon {
				t.Errorf("FToC(%v) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

func TestDescribe(t *testing.T) {
	cases := []struct {
		name string
		in   float64
		want string
	}{
		{"well below zero", -10, "freezing"},
		{"zero is cold", 0, "cold"},
		{"just under mild", 14.9, "cold"},
		{"mild boundary", 15, "mild"},
		{"just under hot", 24.9, "mild"},
		{"hot boundary", 25, "hot"},
		{"heat wave", 33, "hot"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := temperature.Describe(c.in); got != c.want {
				t.Errorf("Describe(%v) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

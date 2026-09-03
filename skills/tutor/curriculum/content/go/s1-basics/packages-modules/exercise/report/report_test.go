// External test package: sees only what report exports.
package report_test

import (
	"testing"

	"tutor.local/packages-modules/report"
)

func TestLine(t *testing.T) {
	cases := []struct {
		name    string
		city    string
		celsius float64
		want    string
	}{
		{"hot", "Berlin", 25, "Berlin: 25.0°C / 77.0°F (hot)"},
		{"freezing with decimals", "Oslo", -8.5, "Oslo: -8.5°C / 16.7°F (freezing)"},
		{"mild", "Lisbon", 17, "Lisbon: 17.0°C / 62.6°F (mild)"},
		{"cold", "Reykjavik", 3, "Reykjavik: 3.0°C / 37.4°F (cold)"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := report.Line(c.city, c.celsius); got != c.want {
				t.Errorf("Line(%q, %v) = %q, want %q", c.city, c.celsius, got, c.want)
			}
		})
	}
}

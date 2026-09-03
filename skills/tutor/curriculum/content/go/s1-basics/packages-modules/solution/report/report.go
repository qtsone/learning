// Package report formats city temperature readings for display.
package report

import (
	"fmt"

	"tutor.local/packages-modules/temperature"
)

// Line renders one city's reading, for example:
//
//	Line("Berlin", 25) == "Berlin: 25.0°C / 77.0°F (hot)"
//
// The Fahrenheit value and the word in parentheses come from the
// temperature package.
func Line(city string, celsius float64) string {
	fahrenheit := temperature.CToF(celsius)
	band := temperature.Describe(celsius)
	return fmt.Sprintf("%s: %.1f°C / %.1f°F (%s)", city, celsius, fahrenheit, band)
}

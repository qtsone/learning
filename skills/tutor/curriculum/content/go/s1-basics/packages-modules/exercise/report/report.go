// Package report formats city temperature readings for display.
package report

// Line renders one city's reading, for example:
//
//	Line("Berlin", 25) == "Berlin: 25.0°C / 77.0°F (hot)"
//
// The Fahrenheit value and the word in parentheses must come from the
// temperature package.
func Line(city string, celsius float64) string {
	// TODO: convert with temperature.CToF, classify with temperature.Describe,
	// and assemble the string with fmt.Sprintf (%.1f keeps one decimal).
	// You will need to import both packages.
	return ""
}

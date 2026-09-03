// Package temperature converts between Celsius and Fahrenheit and
// classifies temperatures in plain words.
package temperature

// CToF converts degrees Celsius to degrees Fahrenheit.
func CToF(celsius float64) float64 {
	return celsius*9/5 + 32
}

// FToC converts degrees Fahrenheit to degrees Celsius.
func FToC(fahrenheit float64) float64 {
	return (fahrenheit - 32) * 5 / 9
}

// Describe classifies a Celsius temperature as "freezing" (below 0),
// "cold" (0 up to but not including 15), "mild" (15 up to but not
// including 25), or "hot" (25 and above).
func Describe(celsius float64) string {
	switch {
	case celsius < 0:
		return "freezing"
	case celsius < 15:
		return "cold"
	case celsius < 25:
		return "mild"
	default:
		return "hot"
	}
}

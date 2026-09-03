package main

// Greeting returns the text to print for the given name.
// An empty name falls back to "world".
func Greeting(name string) string {
	if name == "" {
		name = "world"
	}
	return "Hello, " + name + "!"
}

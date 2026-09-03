package main

import "fmt"

// EnvLookup mirrors os.LookupEnv: it reports a variable's value and whether it
// was set at all. Taking it as a parameter keeps the color policy testable
// without mutating the process environment.
type EnvLookup func(key string) (string, bool)

// ColorMode is the user's stated preference, usually from a --color flag.
type ColorMode int

const (
	// ColorAuto lets the environment and the stream decide.
	ColorAuto ColorMode = iota
	// ColorAlways forces styling on, even into a pipe.
	ColorAlways
	// ColorNever forces styling off, even on a terminal.
	ColorNever
)

// ParseColorMode maps a --color flag value onto a ColorMode. Accepted values
// are "auto", "always" and "never".
func ParseColorMode(s string) (ColorMode, error) {
	switch s {
	case "auto":
		return ColorAuto, nil
	case "always":
		return ColorAlways, nil
	case "never":
		return ColorNever, nil
	}
	return ColorAuto, fmt.Errorf("invalid color mode %q: want auto, always or never", s)
}

// ResolveColor decides whether ANSI styling may be emitted on a stream.
//
// The precedence chain, highest priority first:
//
//  1. ColorNever  — the user said no.
//  2. ColorAlways — the user said yes (this beats NO_COLOR: an explicit flag
//     on this invocation is more specific than an environment default).
//  3. NO_COLOR set to a non-empty value — the cross-tool convention.
//  4. TERM is "dumb" — the terminal cannot render escape sequences.
//  5. otherwise — style only when the stream is a terminal.
func ResolveColor(mode ColorMode, isTerminal bool, env EnvLookup) bool {
	switch mode {
	case ColorNever:
		return false
	case ColorAlways:
		return true
	}
	if v, ok := env("NO_COLOR"); ok && v != "" {
		return false
	}
	if term, _ := env("TERM"); term == "dumb" {
		return false
	}
	return isTerminal
}

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
//
// TODO: return the matching mode, or a useful error naming the bad value and
// listing what is accepted.
func ParseColorMode(s string) (ColorMode, error) {
	return ColorAuto, fmt.Errorf("ParseColorMode not implemented")
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
//
// TODO: implement exactly that chain. env is never nil.
func ResolveColor(mode ColorMode, isTerminal bool, env EnvLookup) bool {
	return false
}

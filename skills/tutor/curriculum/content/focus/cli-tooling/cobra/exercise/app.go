package main

import (
	"github.com/spf13/cobra"
)

// App holds everything the commands need from the outside world. Handlers reach
// into these fields instead of package-level globals — that single choice is
// what lets a test build its own App and never touch os.Getenv or real storage.
type App struct {
	Store  Store
	Getenv func(string) string
}

// env reads an environment variable through the injected lookup, tolerating a
// zero App in tests that do not care about the environment. Getenv, not
// LookupEnv: none of this tool's settings has a meaningful empty value, so empty
// and unset are the same answer here — unlike the config loader in the flags
// lesson, where an empty endpoint was a real (and invalid) setting.
func (a *App) env(key string) string {
	if a.Getenv == nil {
		return ""
	}
	return a.Getenv(key)
}

// Settings is the resolved configuration for a single command run. Every source
// collapses into this one struct before a handler does any work, so no handler
// ever asks "where would this value come from?".
type Settings struct {
	Format string // "text" or "json"
	Limit  int    // 0 means "no limit"
}

// Resolve collapses defaults, environment and flags into Settings for the
// command that is about to run.
//
// Precedence, lowest to highest: default < environment < flag.
func (a *App) Resolve(cmd *cobra.Command) (Settings, error) {
	// TODO: implement per the acceptance criteria in LESSON.md.
	// Hints: cmd.Flags() is the merged set (local + inherited persistent) for
	// this command; Lookup returns nil for a flag this command cannot see, and
	// a flag's Changed field tells you whether the user actually set it.
	return Settings{}, nil
}

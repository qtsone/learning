package main

import (
	"fmt"
	"strconv"

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
	s := Settings{Format: "text"}

	if v := a.env("NOTES_FORMAT"); v != "" {
		s.Format = v
	}
	// Changed is the whole trick: it distinguishes "the user typed --format"
	// from "the flag is sitting at its default", so a default can never
	// overwrite a value that came from the environment.
	if f := cmd.Flags().Lookup("format"); f != nil && f.Changed {
		s.Format = f.Value.String()
	}
	if s.Format != "text" && s.Format != "json" {
		return Settings{}, fmt.Errorf(`invalid format %q: want "text" or "json"`, s.Format)
	}

	if v := a.env("NOTES_LIMIT"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return Settings{}, fmt.Errorf("invalid NOTES_LIMIT %q: want a non-negative integer", v)
		}
		s.Limit = n
	}
	// --limit is local to `list`, so Lookup returns nil on every other command.
	if f := cmd.Flags().Lookup("limit"); f != nil && f.Changed {
		n, err := cmd.Flags().GetInt("limit")
		if err != nil {
			return Settings{}, fmt.Errorf("invalid --limit: %w", err)
		}
		s.Limit = n
	}

	return s, nil
}

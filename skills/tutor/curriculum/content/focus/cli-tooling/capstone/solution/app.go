package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// Defaults live next to the code that applies them, and the flag declarations
// reuse these constants so help text can never drift from behaviour.
const (
	DefaultTop   = 10
	DefaultLimit = 10
)

// DefaultIgnore is the "obviously not your source code" list. Resolve copies it
// rather than handing the slice out, so no run can mutate the default.
var DefaultIgnore = []string{".git", "node_modules", "vendor"}

// ErrUsage marks "you invoked me wrong": an unknown flag, an unreadable config
// file, a value that is not a number. Scripts act on the difference — exit 2
// means fix the command line, exit 1 means the work itself failed — so the
// distinction has to survive as far as exitCode.
var ErrUsage = errors.New("usage")

// ColorMode is the *policy* the user asked for, not the answer. ResolveColor
// turns it into the single bool everything below the edge sees.
type ColorMode int

const (
	ColorAuto ColorMode = iota
	ColorAlways
	ColorNever
)

func ParseColorMode(s string) (ColorMode, error) {
	switch s {
	case "auto":
		return ColorAuto, nil
	case "always":
		return ColorAlways, nil
	case "never":
		return ColorNever, nil
	}
	return ColorAuto, fmt.Errorf(`%w: invalid color %q: want "auto", "always" or "never"`, ErrUsage, s)
}

// App is everything the commands need from outside the program. main fills it
// from the real world; a test fills it from a map and a t.TempDir().
type App struct {
	Getenv           func(string) string
	ConfigDir        string   // os.UserConfigDir(); "" when the machine has none
	Git              []string // the VCS command, e.g. []string{"git"}
	StdoutIsTerminal bool
}

// env reads an environment variable through the injected lookup. Getenv, not
// LookupEnv: none of this tool's settings has a meaningful empty value, so empty
// and unset are the same answer here — unlike the config loader in the flags
// lesson, where an empty endpoint was a real (and invalid) setting.
func (a *App) env(key string) string {
	if a.Getenv == nil {
		return ""
	}
	return a.Getenv(key)
}

// Settings is the resolved configuration for one command run: every source has
// already been collapsed, so no handler ever asks where a value came from.
type Settings struct {
	Color  bool // the answer, not the policy
	JSON   bool
	Top    int
	Limit  int
	Ignore []string
}

// FileConfig mirrors the config file. Every field is a pointer because the file
// has to distinguish "absent" from "present and zero": a non-pointer `json:false`
// is indistinguishable from a key nobody wrote, and would silently overwrite
// whatever a lower layer set.
type FileConfig struct {
	Color  *string   `json:"color"`
	JSON   *bool     `json:"json"`
	Top    *int      `json:"top"`
	Limit  *int      `json:"limit"`
	Ignore *[]string `json:"ignore"`
}

// LoadConfigFile reads one config file. os.Open's error is returned unchanged
// so the caller can tell "no file there" (fs.ErrNotExist, fine at the default
// location) from "unreadable" — the caller knows whether the user named this
// path, and this function does not.
func LoadConfigFile(path string) (FileConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return FileConfig{}, err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	// A typo'd key that is silently ignored is a support ticket six months
	// later: "I set it and nothing happened."
	dec.DisallowUnknownFields()
	var fc FileConfig
	if err := dec.Decode(&fc); err != nil {
		return FileConfig{}, fmt.Errorf("%w: %s: %v", ErrUsage, path, err)
	}
	return fc, nil
}

// configPath answers *which file* to read — itself a precedence decision, and a
// different one from what the file contains. explicit reports whether the user
// named the path, which decides whether a missing file is an error.
func (a *App) configPath(cmd *cobra.Command) (path string, explicit bool) {
	if f := cmd.Flags().Lookup("config"); f != nil && f.Changed {
		return f.Value.String(), true
	}
	if v := a.env("SCOUT_CONFIG"); v != "" {
		return v, true
	}
	if a.ConfigDir == "" {
		return "", false
	}
	return filepath.Join(a.ConfigDir, "scout", "config.json"), false
}

// Resolve collapses every source into Settings for the command about to run.
//
// Precedence, lowest to highest: defaults < config file < environment < flags.
// Each layer only writes what it actually has, which is why the file uses
// pointers and the flags are gated on Changed.
func (a *App) Resolve(cmd *cobra.Command) (Settings, error) {
	s := Settings{Top: DefaultTop, Limit: DefaultLimit, Ignore: slices.Clone(DefaultIgnore)}
	mode := ColorAuto

	// A non-negative count, wherever it came from, named by its source so the
	// error tells the user which thing to edit.
	setCount := func(n int, source string, dst *int) error {
		if n < 0 {
			return fmt.Errorf("%w: invalid %s %d: want a non-negative integer", ErrUsage, source, n)
		}
		*dst = n
		return nil
	}

	path, explicit := a.configPath(cmd)
	if path != "" {
		fc, err := LoadConfigFile(path)
		switch {
		case err == nil:
			if fc.Top != nil {
				if err := setCount(*fc.Top, "top in "+path, &s.Top); err != nil {
					return Settings{}, err
				}
			}
			if fc.Limit != nil {
				if err := setCount(*fc.Limit, "limit in "+path, &s.Limit); err != nil {
					return Settings{}, err
				}
			}
			if fc.JSON != nil {
				s.JSON = *fc.JSON
			}
			if fc.Ignore != nil {
				s.Ignore = slices.Clone(*fc.Ignore)
			}
			if fc.Color != nil {
				if mode, err = ParseColorMode(*fc.Color); err != nil {
					return Settings{}, err
				}
			}
		case errors.Is(err, fs.ErrNotExist) && !explicit:
			// Nobody asked for this file; its absence is the normal case.
		default:
			// Two facts about one error: it is a usage problem *and* it is
			// whatever os.Open said. fmt.Errorf takes more than one %w.
			return Settings{}, fmt.Errorf("%w: %w", ErrUsage, err)
		}
	}

	if v := a.env("SCOUT_TOP"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return Settings{}, fmt.Errorf("%w: invalid SCOUT_TOP %q: want a non-negative integer", ErrUsage, v)
		}
		if err := setCount(n, "SCOUT_TOP", &s.Top); err != nil {
			return Settings{}, err
		}
	}
	if v := a.env("SCOUT_LIMIT"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return Settings{}, fmt.Errorf("%w: invalid SCOUT_LIMIT %q: want a non-negative integer", ErrUsage, v)
		}
		if err := setCount(n, "SCOUT_LIMIT", &s.Limit); err != nil {
			return Settings{}, err
		}
	}
	if v := a.env("SCOUT_JSON"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return Settings{}, fmt.Errorf("%w: invalid SCOUT_JSON %q: want true or false", ErrUsage, v)
		}
		s.JSON = b
	}
	if v := a.env("SCOUT_IGNORE"); v != "" {
		s.Ignore = splitList(v)
	}
	if v := a.env("SCOUT_COLOR"); v != "" {
		var err error
		if mode, err = ParseColorMode(v); err != nil {
			return Settings{}, err
		}
	}

	// Flags last, and only when the user actually typed them: a flag sitting at
	// its declared default must not overwrite the environment.
	if f := cmd.Flags().Lookup("top"); f != nil && f.Changed {
		n, err := cmd.Flags().GetInt("top")
		if err != nil {
			return Settings{}, fmt.Errorf("%w: %w", ErrUsage, err)
		}
		if err := setCount(n, "--top", &s.Top); err != nil {
			return Settings{}, err
		}
	}
	if f := cmd.Flags().Lookup("limit"); f != nil && f.Changed {
		n, err := cmd.Flags().GetInt("limit")
		if err != nil {
			return Settings{}, fmt.Errorf("%w: %w", ErrUsage, err)
		}
		if err := setCount(n, "--limit", &s.Limit); err != nil {
			return Settings{}, err
		}
	}
	if f := cmd.Flags().Lookup("json"); f != nil && f.Changed {
		b, err := cmd.Flags().GetBool("json")
		if err != nil {
			return Settings{}, fmt.Errorf("%w: %w", ErrUsage, err)
		}
		s.JSON = b
	}
	if f := cmd.Flags().Lookup("ignore"); f != nil && f.Changed {
		names, err := cmd.Flags().GetStringSlice("ignore")
		if err != nil {
			return Settings{}, fmt.Errorf("%w: %w", ErrUsage, err)
		}
		s.Ignore = names
	}
	if f := cmd.Flags().Lookup("color"); f != nil && f.Changed {
		var err error
		if mode, err = ParseColorMode(f.Value.String()); err != nil {
			return Settings{}, err
		}
	}

	s.Color = ResolveColor(mode, a.StdoutIsTerminal, a.env)
	if s.JSON {
		// Machine-readable output is never styled, whatever the colour policy
		// says: the consumer is a parser, not an eye.
		s.Color = false
	}
	return s, nil
}

func splitList(v string) []string {
	out := []string{}
	for _, part := range strings.Split(v, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

// ResolveColor turns the policy into one bool, at the edge, once.
//
// Applies, highest priority first: never, then always, then a non-empty
// NO_COLOR, then TERM=dumb, then the terminal answer — an explicit flag for this
// run beats a machine-wide environment default, which beats the terminal's own
// report.
func ResolveColor(mode ColorMode, isTerminal bool, getenv func(string) string) bool {
	switch mode {
	case ColorNever:
		return false
	case ColorAlways:
		return true
	}
	// NO_COLOR vetoes only when non-empty, so Getenv's missing "was it set?"
	// answer costs nothing here.
	if getenv("NO_COLOR") != "" {
		return false
	}
	if getenv("TERM") == "dumb" {
		return false
	}
	return isTerminal
}

// exitCode is the tool's contract with shell scripts. Keep it small enough to
// document in --help and stable enough to depend on.
func exitCode(err error) int {
	switch {
	case err == nil:
		return 0
	// 130 is 128 + SIGINT, what a shell reports for a job killed by Ctrl-C. A
	// deadline shares the code because scout owns no clock: an expired context
	// can only have been imposed from outside, which is the same event. 124 —
	// timeout(1)'s code, and runner's, back when a -timeout flag made it the
	// tool's own decision — would claim something scout never decided.
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return 130
	case errors.Is(err, ErrUsage):
		return 2
	default:
		return 1
	}
}

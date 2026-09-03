package main

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

// Defaults live next to the code that applies them, and the flag declarations
// reuse these constants so help text can never drift from behaviour.
const (
	DefaultTop   = 10
	DefaultLimit = 10
)

// DefaultIgnore is the "obviously not your source code" list. Resolve must copy
// it rather than hand the slice out, so no run can mutate the default.
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
// has to distinguish "absent" from "present and zero": a non-pointer field set
// to false is indistinguishable from a key nobody wrote, and would silently
// overwrite whatever a lower layer set.
type FileConfig struct {
	Color  *string   `json:"color"`
	JSON   *bool     `json:"json"`
	Top    *int      `json:"top"`
	Limit  *int      `json:"limit"`
	Ignore *[]string `json:"ignore"`
}

// LoadConfigFile reads one config file.
//
// TODO: decode the JSON document at path into a FileConfig.
//   - Return os.Open's error unchanged so the caller can recognise
//     fs.ErrNotExist: only the caller knows whether the user named this path.
//   - Reject unknown keys (json.Decoder.DisallowUnknownFields): a typo that is
//     silently ignored becomes "I set it and nothing happened" six months later.
//   - A malformed document is an ErrUsage error naming the file.
func LoadConfigFile(path string) (FileConfig, error) {
	return FileConfig{}, nil
}

// configPath answers *which file* to read — itself a precedence decision, and a
// different one from what the file contains. explicit reports whether the user
// named the path, which decides whether a missing file is an error.
//
// TODO: --config, then SCOUT_CONFIG, then <ConfigDir>/scout/config.json.
// Return "" when there is nowhere to look.
func (a *App) configPath(cmd *cobra.Command) (path string, explicit bool) {
	return "", false
}

// Resolve collapses every source into Settings for the command about to run.
//
// Precedence, lowest to highest: defaults < config file < environment < flags.
//
// TODO: implement it. Notes that will save you an hour:
//   - Each layer writes only what it actually has. That is why FileConfig uses
//     pointers, and why a flag is only read when its Changed field is set.
//   - cmd.Flags().Lookup returns nil for a flag this command cannot see, so the
//     same resolver works for scan, authors and version.
//   - Environment keys: SCOUT_TOP, SCOUT_LIMIT, SCOUT_JSON, SCOUT_IGNORE
//     (comma-separated), SCOUT_COLOR.
//   - Bad values are ErrUsage errors that name their source, so the user knows
//     which thing to edit.
//   - Colour is resolved last, and --json switches it off whatever the policy
//     says.
func (a *App) Resolve(cmd *cobra.Command) (Settings, error) {
	return Settings{}, nil
}

// ResolveColor turns the policy into one bool, at the edge, once.
//
// TODO: apply, highest priority first: never, then always, then a non-empty
// NO_COLOR, then TERM=dumb, then the terminal answer. An explicit choice for
// this run beats a machine-wide environment default, which beats the terminal's
// own report. NO_COLOR vetoes only when it is non-empty.
func ResolveColor(mode ColorMode, isTerminal bool, getenv func(string) string) bool {
	return false
}

// exitCode is the tool's contract with shell scripts. Keep it small enough to
// document in --help and stable enough to depend on.
//
// TODO: 0 success, 2 usage (ErrUsage), 130 interrupted, 1 anything else.
//
// 130 is 128 + SIGINT, what a shell reports for a job killed by Ctrl-C, and it
// covers context.DeadlineExceeded as well as context.Canceled: scout owns no
// clock, so an expired context can only have been imposed from outside. That is
// why this is not the 124 runner returned for a deadline last lesson — runner
// had a -timeout flag, and scout does not. The exit-code section argues it.
func exitCode(err error) int {
	return 0
}

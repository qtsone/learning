package main

import (
	"errors"

	"tutor.local/project-cli-tracker/tracker"
)

// dispatch runs one subcommand against t and returns the text main should
// print. args is os.Args without the program name, e.g. ["add", "buy",
// "milk"]. Supported subcommands (exact outputs in LESSON.md):
//
//	add <words...>   words joined with spaces become the title
//	list             one line per task, or "no tasks yet"
//	complete <id>    marks the task done
//	summary          "N open, M done"
//
// No arguments, an unknown subcommand, or bad subcommand input is an
// error. Errors from the tracker are returned as-is or wrapped — never
// swallowed.
func dispatch(t *tracker.Tracker, args []string) (string, error) {
	// TODO: implement per the doc comment and cli_test.go.
	return "", errors.New("TODO: implement dispatch")
}

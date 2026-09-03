package main

import (
	"context"
	"errors"
)

// ErrVCS reports that the version-control command failed — it was missing, or
// it exited non-zero. Callers match the sentinel; the message carries the
// child's own words.
var ErrVCS = errors.New("vcs command failed")

type Author struct {
	Name    string `json:"name"`
	Commits int    `json:"commits"`
}

// Authors ranks commit authors in dir by shelling out to the configured VCS
// command: `git log --format=%an`, one author name per line.
//
// The command lives in App.Git rather than being the literal string "git",
// which is what lets a test point it at a helper process instead.
//
// TODO: implement it with os/exec.
//   - exec.CommandContext, so a cancelled context kills the child — and refuses
//     to start it at all when the context is already cancelled.
//   - cmd.Dir is the directory to inspect. Capture stdout and stderr into
//     *separate* buffers: the child's diagnostics are not your data.
//   - When Run fails, ask why before asking how. A cancelled child dies of a
//     signal, so its ExitError says "killed" — true, and useless; ctx.Err() is
//     the reason, and it must reach the caller intact.
//   - An *exec.ExitError means the child ran and refused: report ErrVCS with
//     its exit code and the first line of its stderr. Anything else means it
//     never started: report ErrVCS too, but say so differently.
//   - Rank busiest first, ties by name, truncate to limit (<= 0 means all), and
//     never return nil — an empty slice encodes as [], a nil one as null.
func (a *App) Authors(ctx context.Context, dir string, limit int) ([]Author, error) {
	return nil, nil
}

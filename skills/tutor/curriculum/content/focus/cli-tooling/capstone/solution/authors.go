package main

import (
	"bufio"
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strings"
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
// command (`git log --format=%an`).
//
// The command itself is a field on App rather than the literal string "git",
// which is what lets a test point it at a helper process instead.
func (a *App) Authors(ctx context.Context, dir string, limit int) ([]Author, error) {
	argv := a.Git
	if len(argv) == 0 {
		return nil, fmt.Errorf("%w: no VCS command configured", ErrVCS)
	}
	argv = append(slices.Clone(argv), "log", "--format=%an")

	// CommandContext kills the child when ctx is done — and refuses to start it
	// at all if ctx is already cancelled.
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir = dir
	// Separate buffers, not CombinedOutput: the child's diagnostics must never
	// be parsed as data, exactly as with your own two streams.
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Ask why before asking how. A cancelled child dies of a signal, so the
		// ExitError says "killed" — true, and useless. ctx.Err() is the reason.
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("%w: %s exited with code %d: %s",
				ErrVCS, argv[0], exitErr.ExitCode(), firstLine(stderr.String()))
		}
		// Not started at all: no such binary, not executable, bad directory.
		return nil, fmt.Errorf("%w: %s: %v", ErrVCS, argv[0], err)
	}

	return rankAuthors(&stdout, limit), nil
}

// rankAuthors counts one author name per line and returns the busiest first,
// ties broken by name. limit <= 0 means "all of them".
func rankAuthors(r *bytes.Buffer, limit int) []Author {
	counts := map[string]int{}
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		name := strings.TrimSpace(sc.Text())
		if name == "" {
			continue
		}
		counts[name]++
	}

	// Never nil: an empty slice encodes as [], a nil one as null, and every
	// consumer that loops over the array breaks on null.
	out := []Author{}
	for name, n := range counts {
		out = append(out, Author{Name: name, Commits: n})
	}
	slices.SortFunc(out, func(a, b Author) int {
		if c := cmp.Compare(b.Commits, a.Commits); c != 0 {
			return c
		}
		return strings.Compare(a.Name, b.Name)
	})
	if limit > 0 && limit < len(out) {
		out = out[:limit]
	}
	return out
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

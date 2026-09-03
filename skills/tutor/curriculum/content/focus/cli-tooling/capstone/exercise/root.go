package main

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// Execute builds the tree, runs it, and returns the process exit code. It is
// the seam between "a program" and "a function": main is a shell around this,
// and tests call it directly to assert on exit codes and stderr.
//
// Provided complete — note that it is the only place that prints an error.
func Execute(ctx context.Context, app *App, args []string, out, errw io.Writer) int {
	if args == nil {
		args = []string{} // SetArgs(nil) makes cobra fall back to os.Args[1:]
	}
	root := NewRootCmd(app)
	root.SetOut(out)
	root.SetErr(errw)
	root.SetArgs(args)

	err := root.ExecuteContext(ctx)
	switch {
	case err == nil:
	case errors.Is(err, context.Canceled):
		fmt.Fprintln(errw, "scout: interrupted")
	default:
		fmt.Fprintln(errw, "scout:", err)
	}
	return exitCode(err)
}

// NewRootCmd builds a fresh command tree bound to app. Fresh matters: a
// cobra.Command stores parsed flag values on itself, so a shared tree leaks the
// previous run's flags into the next one.
//
// The tree:
//
//	scout                       --json --color --config --ignore --version
//	├── scan [dir]              --top
//	├── authors [dir]           --limit
//	└── version
//
// TODO: build it.
//   - Silence cobra's own error and usage printing: Execute is the one place
//     that reports a failure.
//   - --json, --color, --config and --ignore are persistent (the whole tree);
//     --top, --limit and --version are local to one command each.
//   - SetFlagErrorFunc turns every parse failure into an ErrUsage error, so
//     exitCode can return 2 without any handler knowing about exit codes.
//   - Handlers call app.Resolve(cmd), write to cmd.OutOrStdout(), pass
//     cmd.Context() to anything that can block, and return errors instead of
//     calling os.Exit.
//   - `scout` with no arguments prints help and succeeds; `scout --version`
//     prints the same thing `scout version` does, in whichever format the
//     settings say.
func NewRootCmd(app *App) *cobra.Command {
	root := &cobra.Command{
		Use:   "scout",
		Short: "Repository statistics without leaving the terminal",
		Long: "scout walks a directory and reports what is in it: files, lines and\n" +
			"bytes by extension, and who has been committing.\n\n" +
			"Exit codes: 0 success, 1 failure, 2 bad usage, 130 interrupted.",
		Args: cobra.NoArgs,
	}
	return root
}

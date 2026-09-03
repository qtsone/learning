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
func NewRootCmd(app *App) *cobra.Command {
	root := &cobra.Command{
		Use:   "scout",
		Short: "Repository statistics without leaving the terminal",
		Long: "scout walks a directory and reports what is in it: files, lines and\n" +
			"bytes by extension, and who has been committing.\n\n" +
			"Exit codes: 0 success, 1 failure, 2 bad usage, 130 interrupted.",
		Args: cobra.NoArgs,
		// Execute still returns the error; these two stop cobra from printing
		// it — and a wall of usage text — behind Execute's back.
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if v, err := cmd.Flags().GetBool("version"); err == nil && v {
				return renderVersion(app, cmd)
			}
			return cmd.Help()
		},
	}

	pf := root.PersistentFlags()
	pf.Bool("json", false, "write machine-readable JSON to stdout")
	pf.String("color", "auto", `when to colorize: "auto", "always" or "never"`)
	pf.String("config", "", "path to a config file (default: <user config dir>/scout/config.json)")
	pf.StringSlice("ignore", nil, "directory or file name to skip (repeatable)")

	// --version belongs to root alone: `scout scan --version` is a typo, not a
	// request, and inheriting the flag would hide the typo.
	root.Flags().Bool("version", false, "print version information and exit")

	// Every parse failure becomes a usage error, so exitCode can return 2
	// without any of the handlers knowing about exit codes. Children inherit
	// this function from root.
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return fmt.Errorf("%w: %w", ErrUsage, err)
	})

	root.AddCommand(newScanCmd(app), newAuthorsCmd(app), newVersionCmd(app))
	return root
}

func newScanCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan [dir]",
		Short: "Count files, lines and bytes by extension",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			set, err := app.Resolve(cmd)
			if err != nil {
				return err
			}
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			// cmd.Context() is the context Execute passed to ExecuteContext —
			// the one signal.NotifyContext cancels on Ctrl-C.
			res, err := Scan(cmd.Context(), dir, set.Ignore)
			if err != nil {
				return err
			}
			res.Exts = TopExts(res.Exts, set.Top)
			return NewRenderer(cmd.OutOrStdout(), set).RenderScan(res)
		},
	}
	cmd.Flags().Int("top", DefaultTop, "show at most N extensions (0 means all)")
	return cmd
}

func newAuthorsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "authors [dir]",
		Short: "Rank commit authors, busiest first",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			set, err := app.Resolve(cmd)
			if err != nil {
				return err
			}
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			list, err := app.Authors(cmd.Context(), dir, set.Limit)
			if err != nil {
				return err
			}
			return NewRenderer(cmd.OutOrStdout(), set).RenderAuthors(list)
		},
	}
	cmd.Flags().Int("limit", DefaultLimit, "show at most N authors (0 means all)")
	return cmd
}

func newVersionCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return renderVersion(app, cmd)
		},
	}
}

// renderVersion serves both `scout version` and `scout --version`; cobra's own
// Version field would print a fixed template and ignore --json.
func renderVersion(app *App, cmd *cobra.Command) error {
	set, err := app.Resolve(cmd)
	if err != nil {
		return err
	}
	return NewRenderer(cmd.OutOrStdout(), set).RenderVersion(buildInfo())
}

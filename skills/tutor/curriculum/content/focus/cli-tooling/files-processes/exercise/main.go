package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// This file is complete. It is the wiring you write once per tool, and the only
// file here that touches os.Args, the real streams, or the user's directories.
// Everything below it is handed what it needs.

func main() {
	ctx, stop := WithInterrupt(context.Background())
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fset := flag.NewFlagSet("runner", flag.ContinueOnError)
	fset.SetOutput(stderr)
	timeout := fset.Duration("timeout", 30*time.Second, "give up on the child after this long")
	if err := fset.Parse(args); err != nil {
		return 2
	}
	if fset.NArg() == 0 {
		fmt.Fprintln(stderr, "usage: runner [-timeout d] <program> [args...]")
		return 2
	}

	ctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()

	res, err := Run(ctx, Command{Name: fset.Arg(0), Args: fset.Args()[1:], Stdin: stdin})
	if err != nil {
		fmt.Fprintf(stderr, "runner: %v\n", err)
		return ExitCodeFor(err)
	}
	io.WriteString(stdout, res.Stdout)
	io.WriteString(stderr, res.Stderr)
	if err := saveLastRun(fset.Arg(0), res); err != nil {
		// A cache is a nice-to-have: say so and carry on, because failing to
		// record the run does not mean the run failed.
		fmt.Fprintf(stderr, "runner: could not record this run: %v\n", err)
	}
	return res.ExitCode
}

type lastRun struct {
	Command  string `json:"command"`
	ExitCode int    `json:"exit_code"`
}

func saveLastRun(name string, res Result) error {
	dir, err := AppDir(os.UserCacheDir, "runner")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(lastRun{Command: name, ExitCode: res.ExitCode})
	if err != nil {
		return err
	}
	return WriteFileAtomic(filepath.Join(dir, "last-run.json"), data, 0o600)
}

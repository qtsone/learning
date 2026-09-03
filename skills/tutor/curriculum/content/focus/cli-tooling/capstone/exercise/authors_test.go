package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// helperGit points App.Git at this test binary instead of the real git.
//
// The child is the same binary, re-executed with -test.run=TestHelperProcess
// and a sentinel in the environment it inherits. That is the standard Go way to
// test code that shells out: no fixture binary to build, no assumption about
// what is installed, identical behaviour on every platform.
func helperGit(t *testing.T, mode string) []string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	// Setenv on the parent: the child inherits it, and the testing package
	// restores it when this test ends.
	t.Setenv("SCOUT_TEST_HELPER", mode)
	return []string{exe, "-test.run=TestHelperProcess", "--"}
}

func TestAuthorsRanking(t *testing.T) {
	app := newApp(t, nil)
	app.Git = helperGit(t, "log")

	got, err := app.Authors(context.Background(), t.TempDir(), 0)
	if err != nil {
		t.Fatalf("Authors: %v", err)
	}
	want := []Author{
		{Name: "Ada Lovelace", Commits: 3},
		{Name: "Grace Hopper", Commits: 2},
		{Name: "Alan Turing", Commits: 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Authors = %+v, want %+v (busiest first, ties by name; the child's stderr is not data)", got, want)
	}
}

func TestAuthorsLimit(t *testing.T) {
	app := newApp(t, nil)
	app.Git = helperGit(t, "log")

	got, err := app.Authors(context.Background(), t.TempDir(), 2)
	if err != nil {
		t.Fatalf("Authors: %v", err)
	}
	if len(got) != 2 || got[0].Name != "Ada Lovelace" {
		t.Errorf("Authors with limit 2 = %+v, want the top two", got)
	}
}

func TestAuthorsEmptyHistory(t *testing.T) {
	app := newApp(t, nil)
	app.Git = helperGit(t, "empty")

	got, err := app.Authors(context.Background(), t.TempDir(), 0)
	if err != nil {
		t.Fatalf("Authors: %v", err)
	}
	if got == nil {
		t.Fatal("Authors returned nil; want an empty non-nil slice, so JSON encodes [] and not null")
	}
	if len(got) != 0 {
		t.Errorf("Authors = %+v, want empty", got)
	}
}

// A child that fails tells you two things: its exit code, and what it printed
// on stderr. Both belong in the error; neither belongs on stdout.
func TestAuthorsChildFails(t *testing.T) {
	app := newApp(t, nil)
	app.Git = helperGit(t, "fail")

	_, err := app.Authors(context.Background(), t.TempDir(), 0)
	if err == nil {
		t.Fatal("no error from a child that exited 128")
	}
	if !errors.Is(err, ErrVCS) {
		t.Errorf("err = %v, want one matching ErrVCS", err)
	}
	if !strings.Contains(err.Error(), "128") {
		t.Errorf("err = %v, want the child's exit code in the message (exec.ExitError.ExitCode)", err)
	}
	if !strings.Contains(err.Error(), "not a git repository") {
		t.Errorf("err = %v, want the child's stderr in the message", err)
	}
}

func TestAuthorsMissingBinary(t *testing.T) {
	app := newApp(t, nil)
	app.Git = []string{filepath.Join(t.TempDir(), "definitely-not-a-binary")}

	_, err := app.Authors(context.Background(), t.TempDir(), 0)
	if !errors.Is(err, ErrVCS) {
		t.Errorf("err = %v, want one matching ErrVCS when the command cannot start", err)
	}
}

// The child must run in the directory the user asked about, not in the
// program's own working directory.
func TestAuthorsRunsInTheGivenDirectory(t *testing.T) {
	app := newApp(t, nil)
	app.Git = helperGit(t, "cwd")
	dir := t.TempDir()

	if _, err := app.Authors(context.Background(), dir, 0); err != nil {
		t.Fatalf("Authors: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "ran-here.txt")); err != nil {
		t.Errorf("the child did not run in %s: %v (set cmd.Dir)", dir, err)
	}
}

// Cancellation is reported as cancellation. exec.CommandContext refuses to
// start a child under a cancelled context, and a killed child would otherwise
// look like an ordinary non-zero exit.
func TestAuthorsCancelled(t *testing.T) {
	app := newApp(t, nil)
	app.Git = helperGit(t, "log")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := app.Authors(ctx, t.TempDir(), 0)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want one matching context.Canceled via errors.Is", err)
	}
	if errors.Is(err, ErrVCS) {
		t.Error("a cancelled run must not be reported as a VCS failure: check ctx.Err() before classifying")
	}
}

func TestAuthorsCommand(t *testing.T) {
	app := newApp(t, nil)
	app.Git = helperGit(t, "log")
	dir := t.TempDir()

	got := run(t, app, "authors", dir)
	if got.err != nil {
		t.Fatalf("unexpected error: %v", got.err)
	}
	want := fmt.Sprintf("%6d  %s\n%6d  %s\n%6d  %s\n",
		3, "Ada Lovelace", 2, "Grace Hopper", 1, "Alan Turing")
	if got.stdout != want {
		t.Errorf("stdout = %q, want %q", got.stdout, want)
	}
	if got.stderr != "" {
		t.Errorf("stderr = %q, want empty: the child's warnings are not this tool's output", got.stderr)
	}

	jsonRun := run(t, app, "authors", dir, "--json", "--limit", "1")
	var doc struct {
		Authors []Author `json:"authors"`
	}
	if err := json.Unmarshal([]byte(jsonRun.stdout), &doc); err != nil {
		t.Fatalf("authors --json: not a JSON document: %v\ngot %q", err, jsonRun.stdout)
	}
	if len(doc.Authors) != 1 || doc.Authors[0].Name != "Ada Lovelace" || doc.Authors[0].Commits != 3 {
		t.Errorf(`authors --json --limit 1 = %+v, want one entry {"name":"Ada Lovelace","commits":3}`, doc.Authors)
	}
}

func TestAuthorsCommandEmptyJSON(t *testing.T) {
	app := newApp(t, nil)
	app.Git = helperGit(t, "empty")

	got := run(t, app, "authors", t.TempDir(), "--json")
	if got.stdout != "{\"authors\":[]}\n" {
		t.Errorf("stdout = %q, want %q: a nil slice would encode as null and break every consumer",
			got.stdout, "{\"authors\":[]}\n")
	}
}

// TestHelperProcess is not a test. It is the child process: when the sentinel
// is set in the environment, the re-executed test binary plays the part of git
// and exits. Without the sentinel — an ordinary `go test` run — it does
// nothing.
func TestHelperProcess(t *testing.T) {
	mode := os.Getenv("SCOUT_TEST_HELPER")
	if mode == "" {
		return
	}
	args := helperArgs()
	if len(args) == 0 || args[0] != "log" {
		fmt.Fprintf(os.Stderr, "helper: unexpected arguments %q, want `log --format=%%an`\n", args)
		os.Exit(4)
	}

	switch mode {
	case "log":
		// Diagnostics on stderr, data on stdout — the same discipline this
		// tool owes its own callers. A parser that reads both gets this wrong.
		fmt.Fprintln(os.Stderr, "warning: refname 'HEAD' is ambiguous")
		fmt.Fprint(os.Stdout, "Ada Lovelace\nGrace Hopper\nAda Lovelace\n\nAlan Turing\nAda Lovelace\nGrace Hopper\n")
	case "empty":
		// A repository with no commits: success, no output.
	case "fail":
		fmt.Fprintln(os.Stderr, "fatal: not a git repository (or any of the parent directories): .git")
		os.Exit(128)
	case "cwd":
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, "helper:", err)
			os.Exit(5)
		}
		if err := os.WriteFile(filepath.Join(wd, "ran-here.txt"), []byte("x\n"), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "helper:", err)
			os.Exit(5)
		}
		fmt.Fprintln(os.Stdout, "Ada Lovelace")
	default:
		fmt.Fprintf(os.Stderr, "helper: unknown mode %q\n", mode)
		os.Exit(6)
	}
	// os.Exit, not return: otherwise the testing package prints its own "PASS"
	// into the stdout the parent is about to parse.
	os.Exit(0)
}

// helperArgs returns the arguments the parent passed after the "--" separator.
func helperArgs() []string {
	for i, a := range os.Args {
		if a == "--" {
			return os.Args[i+1:]
		}
	}
	return nil
}

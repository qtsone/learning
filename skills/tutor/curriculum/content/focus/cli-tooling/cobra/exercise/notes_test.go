package main

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

type result struct {
	stdout string
	stderr string
	err    error
}

func newApp(env map[string]string) *App {
	return &App{
		Store:  NewMemStore(),
		Getenv: func(k string) string { return env[k] },
	}
}

// run builds a fresh command tree, executes it with args, and captures
// everything the tree wrote. This is the whole testable-command pattern: no
// subprocess, no os.Args, no os.Exit.
func run(t *testing.T, app *App, args ...string) result {
	t.Helper()
	if args == nil {
		// SetArgs(nil) makes cobra fall back to os.Args[1:], which under
		// `go test` is full of -test.* flags. Always pass a non-nil slice.
		args = []string{}
	}
	var out, errOut bytes.Buffer
	root := NewRootCmd(app)
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetArgs(args)
	err := root.Execute()
	return result{stdout: out.String(), stderr: errOut.String(), err: err}
}

func wantOK(t *testing.T, got result, wantStdout string) {
	t.Helper()
	if got.err != nil {
		t.Fatalf("unexpected error: %v", got.err)
	}
	if got.stdout != wantStdout {
		t.Errorf("stdout = %q, want %q", got.stdout, wantStdout)
	}
	if got.stderr != "" {
		t.Errorf("stderr = %q, want empty: cobra must not print diagnostics for you", got.stderr)
	}
}

func findCmd(t *testing.T, parent *cobra.Command, name string) *cobra.Command {
	t.Helper()
	for _, c := range parent.Commands() {
		if c.Name() == name {
			return c
		}
	}
	t.Fatalf("no %q subcommand under %q", name, parent.Name())
	return nil
}

func TestAddThenList(t *testing.T) {
	app := newApp(nil)

	wantOK(t, run(t, app, "add", "buy milk"), "added n1\n")
	wantOK(t, run(t, app, "add", "call ana", "--tag", "home", "--tag", "urgent"), "added n2\n")
	wantOK(t, run(t, app, "list"), "n1 buy milk\nn2 call ana #home #urgent\n")
}

func TestListEmpty(t *testing.T) {
	wantOK(t, run(t, newApp(nil), "list"), "no notes\n")
}

func TestListLimitFlag(t *testing.T) {
	app := newApp(nil)
	for i, text := range []string{"one", "two", "three"} {
		wantOK(t, run(t, app, "add", text), fmt.Sprintf("added n%d\n", i+1))
	}
	wantOK(t, run(t, app, "list", "--limit", "2"), "n1 one\nn2 two\n")
	wantOK(t, run(t, app, "list", "--limit", "0"), "n1 one\nn2 two\nn3 three\n")
}

func TestJSONOutput(t *testing.T) {
	app := newApp(nil)

	// A persistent flag is accepted before the subcommand …
	wantOK(t, run(t, app, "--format", "json", "add", "buy milk"),
		`{"id":"n1","text":"buy milk","tags":[]}`+"\n")
	// … and after it, because the child inherits the same flag.
	wantOK(t, run(t, app, "add", "call ana", "--tag", "home", "--format", "json"),
		`{"id":"n2","text":"call ana","tags":["home"]}`+"\n")
	wantOK(t, run(t, app, "list", "--format", "json"),
		`{"notes":[{"id":"n1","text":"buy milk","tags":[]},{"id":"n2","text":"call ana","tags":["home"]}]}`+"\n")
}

func TestFormatPrecedence(t *testing.T) {
	cases := []struct {
		name       string
		env        map[string]string
		args       []string
		wantStdout string
		wantErr    string
	}{
		{
			name:       "default is text",
			args:       []string{"add", "x"},
			wantStdout: "added n1\n",
		},
		{
			name:       "environment beats the default",
			env:        map[string]string{"NOTES_FORMAT": "json"},
			args:       []string{"add", "x"},
			wantStdout: `{"id":"n1","text":"x","tags":[]}` + "\n",
		},
		{
			name:       "flag beats the environment",
			env:        map[string]string{"NOTES_FORMAT": "json"},
			args:       []string{"--format", "text", "add", "x"},
			wantStdout: "added n1\n",
		},
		{
			name:    "invalid value from the environment",
			env:     map[string]string{"NOTES_FORMAT": "yaml"},
			args:    []string{"add", "x"},
			wantErr: `invalid format "yaml"`,
		},
		{
			name:    "invalid value from the flag",
			args:    []string{"--format", "yaml", "add", "x"},
			wantErr: `invalid format "yaml"`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := run(t, newApp(c.env), c.args...)
			if c.wantErr != "" {
				if got.err == nil || !strings.Contains(got.err.Error(), c.wantErr) {
					t.Fatalf("error = %v, want one containing %q", got.err, c.wantErr)
				}
				if got.stdout != "" {
					t.Errorf("stdout = %q, want empty on a failed run", got.stdout)
				}
				return
			}
			wantOK(t, got, c.wantStdout)
		})
	}
}

func TestLimitPrecedence(t *testing.T) {
	cases := []struct {
		name       string
		env        map[string]string
		args       []string
		wantStdout string
		wantErr    string
	}{
		{
			name:       "default shows everything",
			args:       []string{"list"},
			wantStdout: "n1 one\nn2 two\nn3 three\n",
		},
		{
			name:       "environment beats the default",
			env:        map[string]string{"NOTES_LIMIT": "1"},
			args:       []string{"list"},
			wantStdout: "n1 one\n",
		},
		{
			name:       "flag beats the environment",
			env:        map[string]string{"NOTES_LIMIT": "1"},
			args:       []string{"list", "--limit", "2"},
			wantStdout: "n1 one\nn2 two\n",
		},
		{
			name:    "unparseable environment value is an error",
			env:     map[string]string{"NOTES_LIMIT": "abc"},
			args:    []string{"list"},
			wantErr: `invalid NOTES_LIMIT "abc"`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			app := newApp(c.env)
			for _, text := range []string{"one", "two", "three"} {
				app.Store.Add(text, nil)
			}
			got := run(t, app, c.args...)
			if c.wantErr != "" {
				if got.err == nil || !strings.Contains(got.err.Error(), c.wantErr) {
					t.Fatalf("error = %v, want one containing %q", got.err, c.wantErr)
				}
				return
			}
			wantOK(t, got, c.wantStdout)
		})
	}
}

func TestTagSubtree(t *testing.T) {
	app := newApp(nil)

	wantOK(t, run(t, app, "add", "buy milk"), "added n1\n")
	wantOK(t, run(t, app, "tag", "add", "n1", "home", "shopping"), "tagged n1\n")
	wantOK(t, run(t, app, "list"), "n1 buy milk #home #shopping\n")
	wantOK(t, run(t, app, "tag", "list"), "home\nshopping\n")
	// The persistent flag declared on root reaches a grandchild command.
	wantOK(t, run(t, app, "tag", "list", "--format", "json"), `{"tags":["home","shopping"]}`+"\n")
}

func TestTagListEmpty(t *testing.T) {
	wantOK(t, run(t, newApp(nil), "tag", "list"), "no tags\n")
}

func TestTagAddUnknownNote(t *testing.T) {
	got := run(t, newApp(nil), "tag", "add", "n9", "home")
	if !errors.Is(got.err, ErrNoteNotFound) {
		t.Fatalf("error = %v, want one matching ErrNoteNotFound via errors.Is", got.err)
	}
	if got.stdout != "" {
		t.Errorf("stdout = %q, want empty on a failed run", got.stdout)
	}
}

func TestCommandErrors(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"unknown top-level command", []string{"frobnicate"}, `unknown command "frobnicate"`},
		{"unknown child of a group command", []string{"tag", "frobnicate"}, `unknown command "frobnicate"`},
		{"add needs exactly one argument", []string{"add"}, "accepts 1 arg"},
		{"add rejects a second argument", []string{"add", "a", "b"}, "accepts 1 arg"},
		{"list takes no positional arguments", []string{"list", "extra"}, `unknown command "extra"`},
		{"tag add needs a note and a tag", []string{"tag", "add", "n1"}, "requires at least 2 arg"},
		{"unknown flag", []string{"add", "x", "--nope"}, "unknown flag"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := run(t, newApp(nil), c.args...)
			if got.err == nil || !strings.Contains(got.err.Error(), c.want) {
				t.Fatalf("error = %v, want one containing %q", got.err, c.want)
			}
			if got.stdout != "" {
				t.Errorf("stdout = %q, want empty on a failed run", got.stdout)
			}
			if got.stderr != "" {
				t.Errorf("stderr = %q, want empty: main decides how errors are printed", got.stderr)
			}
		})
	}
}

func TestFlagScopes(t *testing.T) {
	root := NewRootCmd(newApp(nil))
	add := findCmd(t, root, "add")
	list := findCmd(t, root, "list")
	tag := findCmd(t, root, "tag")
	tagAdd := findCmd(t, tag, "add")

	if root.LocalFlags().Lookup("format") == nil {
		t.Error("--format must be declared on root as a persistent flag")
	}
	if add.InheritedFlags().Lookup("format") == nil {
		t.Error("add must inherit --format from root")
	}
	if add.LocalFlags().Lookup("format") != nil {
		t.Error("--format must be inherited by add, not redeclared on it")
	}
	if add.LocalFlags().Lookup("tag") == nil {
		t.Error("--tag must be a local flag on add")
	}
	if list.LocalFlags().Lookup("tag") != nil {
		t.Error("add's local --tag must not be visible on list")
	}
	if add.Flags().Lookup("limit") != nil {
		t.Error("list's local --limit must not be visible on add")
	}
	if list.LocalFlags().Lookup("limit") == nil {
		t.Error("--limit must be a local flag on list")
	}
	if tagAdd.InheritedFlags().Lookup("format") == nil {
		t.Error("a persistent flag must reach grandchildren such as `tag add`")
	}
}

// A command that only groups other commands must still answer for itself: with
// no arguments it prints its help and succeeds.
func TestGroupCommandsPrintHelp(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want []string
	}{
		{"root", []string{}, []string{"add", "list", "tag"}},
		{"tag group", []string{"tag"}, []string{"add", "list"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := run(t, newApp(nil), c.args...)
			if got.err != nil {
				t.Fatalf("unexpected error: %v", got.err)
			}
			for _, want := range c.want {
				if !strings.Contains(got.stdout, want) {
					t.Errorf("help output does not mention %q; got:\n%s", want, got.stdout)
				}
			}
		})
	}
}

func TestHelpGoesToTheInjectedWriter(t *testing.T) {
	got := run(t, newApp(nil), "--help")
	if got.err != nil {
		t.Fatalf("--help returned an error: %v", got.err)
	}
	for _, want := range []string{"add", "list", "tag", "--format"} {
		if !strings.Contains(got.stdout, want) {
			t.Errorf("help output does not mention %q; got:\n%s", want, got.stdout)
		}
	}
	if got.stderr != "" {
		t.Errorf("stderr = %q, want empty: help is not a diagnostic", got.stderr)
	}
}

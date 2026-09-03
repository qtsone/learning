package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

type result struct {
	stdout string
	stderr string
	err    error
}

// newApp builds an App with a private, empty config directory, so no test can
// be steered by the real machine's user config. A test that wants a config file
// at the default location writes one into app.ConfigDir itself.
func newApp(t *testing.T, env map[string]string) *App {
	t.Helper()
	return &App{
		Getenv:    func(k string) string { return env[k] },
		ConfigDir: t.TempDir(),
	}
}

// run executes a fresh command tree in-process and captures both streams.
func run(t *testing.T, app *App, args ...string) result {
	t.Helper()
	return runCtx(t, context.Background(), app, args...)
}

func runCtx(t *testing.T, ctx context.Context, app *App, args ...string) result {
	t.Helper()
	if args == nil {
		args = []string{}
	}
	var out, errOut bytes.Buffer
	root := NewRootCmd(app)
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetArgs(args)
	err := root.ExecuteContext(ctx)
	return result{stdout: out.String(), stderr: errOut.String(), err: err}
}

// execute goes through Execute, so it sees what the shell would: an exit code
// and whatever landed on stderr.
func execute(t *testing.T, ctx context.Context, app *App, args ...string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := Execute(ctx, app, args, &out, &errOut)
	return code, out.String(), errOut.String()
}

// writeTree creates files under a fresh temp dir; keys are slash-separated
// relative paths and parent directories are created as needed.
func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", name, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return dir
}

// sampleTree is 5 files: 2 .go (4 lines, 33 bytes), 2 .md (2 lines, 14 bytes),
// 1 extensionless (1 line, 5 bytes), plus a .git directory that must be pruned.
func sampleTree(t *testing.T) string {
	t.Helper()
	return writeTree(t, map[string]string{
		"a.go":          "package a\n",
		"pkg/b.go":      "package b\n\nfunc B() {}\n",
		"README.md":     "# scout\n",
		"docs/guide.md": "hello\n",
		"Makefile":      "all:\n",
		".git/config":   "[core]\n",
	})
}

// The report format is restated here on purpose: these tests pin the bytes a
// user sees, not whatever constants you happen to declare.
func scanHeader() string {
	return fmt.Sprintf("%-10s %6s %8s %10s\n", "ext", "files", "lines", "bytes")
}

func scanRow(ext string, files, lines int, bytes int64) string {
	return fmt.Sprintf("%-10s %6d %8d %10d\n", ext, files, lines, bytes)
}

func TestScanCommandText(t *testing.T) {
	dir := sampleTree(t)
	got := run(t, newApp(t, nil), "scan", dir)
	if got.err != nil {
		t.Fatalf("unexpected error: %v", got.err)
	}
	want := scanHeader() +
		scanRow(".go", 2, 4, 33) +
		scanRow(".md", 2, 2, 14) +
		scanRow("(none)", 1, 1, 5) +
		scanRow("total", 5, 7, 52)
	if got.stdout != want {
		t.Errorf("stdout =\n%q\nwant\n%q", got.stdout, want)
	}
	if got.stderr != "" {
		t.Errorf("stderr = %q, want empty: the report is data, not diagnostics", got.stderr)
	}
}

func TestScanCommandTopFlag(t *testing.T) {
	dir := sampleTree(t)
	got := run(t, newApp(t, nil), "scan", dir, "--top", "1")
	if got.err != nil {
		t.Fatalf("unexpected error: %v", got.err)
	}
	want := scanHeader() + scanRow(".go", 2, 4, 33) + scanRow("total", 5, 7, 52)
	if got.stdout != want {
		t.Errorf("stdout =\n%q\nwant\n%q\n(--top truncates the table; the totals still cover the whole tree)",
			got.stdout, want)
	}
}

func TestScanCommandJSON(t *testing.T) {
	dir := sampleTree(t)
	// --color always must lose to --json: machine output is never styled.
	got := run(t, newApp(t, nil), "scan", dir, "--json", "--color", "always")
	if got.err != nil {
		t.Fatalf("unexpected error: %v", got.err)
	}
	if strings.Contains(got.stdout, "\x1b") {
		t.Fatalf("JSON output contains an escape sequence: %q", got.stdout)
	}
	var res ScanResult
	if err := json.Unmarshal([]byte(got.stdout), &res); err != nil {
		t.Fatalf("stdout is not one JSON document: %v\ngot: %q", err, got.stdout)
	}
	if res.Root != dir {
		t.Errorf("root = %q, want %q", res.Root, dir)
	}
	if res.Files != 5 || res.Lines != 7 || res.Bytes != 52 {
		t.Errorf("totals = %d files, %d lines, %d bytes; want 5, 7, 52", res.Files, res.Lines, res.Bytes)
	}
	if len(res.Exts) != 3 || res.Exts[0].Ext != ".go" {
		t.Errorf("exts = %+v, want three buckets with .go first", res.Exts)
	}
}

func TestScanCommandEmptyDir(t *testing.T) {
	got := run(t, newApp(t, nil), "scan", t.TempDir())
	if got.err != nil {
		t.Fatalf("unexpected error: %v", got.err)
	}
	if got.stdout != "no files\n" {
		t.Errorf("stdout = %q, want %q", got.stdout, "no files\n")
	}
}

func TestScanCommandColor(t *testing.T) {
	dir := sampleTree(t)
	app := newApp(t, nil) // not a terminal …
	got := run(t, app, "scan", dir, "--color", "always")
	if got.err != nil {
		t.Fatalf("unexpected error: %v", got.err)
	}
	header := strings.TrimSuffix(scanHeader(), "\n")
	wantFirst := Bold + header + Reset
	if first, _, _ := strings.Cut(got.stdout, "\n"); first != wantFirst {
		t.Errorf("first line = %q, want %q (pad first, then paint: the reset goes before the newline)",
			first, wantFirst)
	}
	// … and with the default policy the same run must be plain.
	plain := run(t, app, "scan", dir)
	if strings.Contains(plain.stdout, "\x1b") {
		t.Errorf("auto colour wrote escapes into a non-terminal stream: %q", plain.stdout)
	}
}

func TestVersion(t *testing.T) {
	wantText := fmt.Sprintf("scout %s (commit %s, built %s, %s, %s)\n",
		"dev", "none", "unknown", runtime.Version(), runtime.GOOS+"/"+runtime.GOARCH)

	for _, args := range [][]string{{"version"}, {"--version"}} {
		got := run(t, newApp(t, nil), args...)
		if got.err != nil {
			t.Fatalf("%q: unexpected error: %v", args, got.err)
		}
		if got.stdout != wantText {
			t.Errorf("%q: stdout = %q, want %q", args, got.stdout, wantText)
		}
	}

	got := run(t, newApp(t, nil), "version", "--json")
	var info BuildInfo
	if err := json.Unmarshal([]byte(got.stdout), &info); err != nil {
		t.Fatalf("version --json: not a JSON document: %v\ngot: %q", err, got.stdout)
	}
	if info.Version != "dev" || info.Commit != "none" || info.Date != "unknown" {
		t.Errorf("version info = %+v, want the unpatched defaults dev/none/unknown", info)
	}
	if info.Platform != runtime.GOOS+"/"+runtime.GOARCH {
		t.Errorf("platform = %q, want %q", info.Platform, runtime.GOOS+"/"+runtime.GOARCH)
	}
}

// --version belongs to root alone: on a subcommand it is a typo, and a typo
// must not be silently accepted.
func TestVersionFlagIsRootOnly(t *testing.T) {
	got := run(t, newApp(t, nil), "scan", t.TempDir(), "--version")
	if got.err == nil {
		t.Fatal("scan --version returned no error, want an unknown-flag failure")
	}
}

func TestExitCodesEndToEnd(t *testing.T) {
	dir := sampleTree(t)
	missing := filepath.Join(dir, "nope")

	cases := []struct {
		name     string
		args     []string
		env      map[string]string
		wantCode int
		wantErr  string // substring required on stderr, "" means stderr must be empty
	}{
		{name: "success", args: []string{"scan", dir}, wantCode: 0},
		{name: "unknown flag", args: []string{"scan", dir, "--nope"}, wantCode: 2, wantErr: "unknown flag"},
		{name: "invalid colour value", args: []string{"scan", dir, "--color", "purple"}, wantCode: 2, wantErr: `invalid color "purple"`},
		{name: "bad environment value", args: []string{"scan", dir}, env: map[string]string{"SCOUT_TOP": "abc"}, wantCode: 2, wantErr: `invalid SCOUT_TOP "abc"`},
		{name: "missing directory", args: []string{"scan", missing}, wantCode: 1, wantErr: "nope"},
		{name: "unknown command", args: []string{"frobnicate"}, wantCode: 1, wantErr: `unknown command "frobnicate"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, stdout, stderr := execute(t, context.Background(), newApp(t, c.env), c.args...)
			if code != c.wantCode {
				t.Errorf("exit code = %d, want %d (stderr: %s)", code, c.wantCode, stderr)
			}
			if c.wantErr == "" {
				if stderr != "" {
					t.Errorf("stderr = %q, want empty on success", stderr)
				}
				return
			}
			if !strings.Contains(stderr, c.wantErr) {
				t.Errorf("stderr = %q, want it to contain %q", stderr, c.wantErr)
			}
			if !strings.HasPrefix(stderr, "scout: ") {
				t.Errorf("stderr = %q, want the message prefixed with the tool name", stderr)
			}
			if strings.Count(strings.TrimSuffix(stderr, "\n"), "\n") != 0 {
				t.Errorf("stderr = %q, want exactly one line: the error is reported once", stderr)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty on a failed run", stdout)
			}
		})
	}
}

// Ctrl-C is a cancelled context by the time it reaches the tool: the run stops,
// the exit code says why, and nothing half-written lands on stdout.
func TestInterruptedRun(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	code, stdout, stderr := execute(t, ctx, newApp(t, nil), "scan", sampleTree(t))
	if code != 130 {
		t.Errorf("exit code = %d, want 130 (128 + SIGINT)", code)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty: a cancelled scan has no report to print", stdout)
	}
	if stderr != "scout: interrupted\n" {
		t.Errorf("stderr = %q, want %q", stderr, "scout: interrupted\n")
	}
}

func TestRootHelp(t *testing.T) {
	got := run(t, newApp(t, nil))
	if got.err != nil {
		t.Fatalf("bare invocation returned an error: %v", got.err)
	}
	for _, want := range []string{"scan", "authors", "version", "--json", "--color", "--config"} {
		if !strings.Contains(got.stdout, want) {
			t.Errorf("help does not mention %q; got:\n%s", want, got.stdout)
		}
	}
}

// Every run must get its own tree: a cobra.Command stores parsed flag values on
// itself, so a package-level tree carries one run's flags into the next.
func TestTreeIsFreshPerRun(t *testing.T) {
	app := newApp(t, nil)
	dir := sampleTree(t)

	first := run(t, app, "scan", dir, "--top", "1")
	second := run(t, app, "scan", dir)
	if first.err != nil || second.err != nil {
		t.Fatalf("unexpected errors: %v / %v", first.err, second.err)
	}
	if strings.Count(second.stdout, "\n") != 5 {
		t.Errorf("the second run shows %q\nwant the full table: --top from the first run leaked into it",
			second.stdout)
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

func TestFlagScopes(t *testing.T) {
	root := NewRootCmd(newApp(t, nil))
	scan := findCmd(t, root, "scan")
	authors := findCmd(t, root, "authors")

	for _, name := range []string{"json", "color", "config", "ignore"} {
		if root.LocalFlags().Lookup(name) == nil {
			t.Errorf("--%s must be declared on root as a persistent flag", name)
		}
		if scan.InheritedFlags().Lookup(name) == nil {
			t.Errorf("scan must inherit --%s from root", name)
		}
	}
	if scan.LocalFlags().Lookup("top") == nil {
		t.Error("--top must be a local flag on scan")
	}
	if authors.Flags().Lookup("top") != nil {
		t.Error("scan's local --top must not be visible on authors")
	}
	if authors.LocalFlags().Lookup("limit") == nil {
		t.Error("--limit must be a local flag on authors")
	}
	if scan.Flags().Lookup("version") != nil {
		t.Error("--version must be local to root, not inherited by subcommands")
	}
}

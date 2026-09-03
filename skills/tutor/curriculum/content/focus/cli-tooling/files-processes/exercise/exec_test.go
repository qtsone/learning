package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// --- the helper-process idiom ---------------------------------------------
//
// These tests need a child process, and "which programs exist" is exactly what
// you cannot assume: sleep is not on Windows, echo is a shell builtin in some
// places, and a test that depends on either passes on your laptop and fails on
// someone else's. So the child is *this test binary*.
//
// TestMain looks for a sentinel environment variable. When it is set, the
// binary behaves like the small program the test needs and exits; when it is
// not, it runs the tests as usual. The child is therefore always present,
// always the same build, and always deterministic.

const helperEnv = "TUTOR_HELPER_MODE"

func TestMain(m *testing.M) {
	if mode := os.Getenv(helperEnv); mode != "" {
		os.Exit(helperProcess(mode, os.Args[1:]))
	}
	os.Exit(m.Run())
}

// helperProcess is the child. It never runs during a normal test run.
func helperProcess(mode string, args []string) int {
	switch mode {
	case "echo": // one line per argument, exactly as received
		for _, a := range args {
			fmt.Println(a)
		}
	case "streams":
		fmt.Fprint(os.Stdout, "data\n")
		fmt.Fprint(os.Stderr, "diagnostic\n")
	case "exit":
		code, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 99
		}
		fmt.Fprintf(os.Stderr, "exiting with %d\n", code)
		return code
	case "env":
		for _, name := range args {
			fmt.Printf("%s=%s\n", name, os.Getenv(name))
		}
	case "cat":
		if _, err := io.Copy(os.Stdout, os.Stdin); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	case "pwd":
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println(wd)
	case "block":
		// Long enough that only cancellation ends it. Note that select{} would
		// not do: the runtime spots the deadlock and kills the process.
		time.Sleep(time.Minute)
	default:
		fmt.Fprintf(os.Stderr, "unknown helper mode %q\n", mode)
		return 2
	}
	return 0
}

// helperCmd builds a Command that re-executes this test binary in the given
// helper mode. Callers may set Dir, Stdin, or append to Env.
func helperCmd(t *testing.T, mode string, args ...string) Command {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	return Command{Name: exe, Args: args, Env: []string{helperEnv + "=" + mode}}
}

// --- Run -------------------------------------------------------------------

func TestRunCapturesStreamsSeparately(t *testing.T) {
	res, err := Run(context.Background(), helperCmd(t, "streams"))
	if err != nil {
		t.Fatalf("Run returned err = %v, want nil", err)
	}
	if res.Stdout != "data\n" {
		t.Errorf("Stdout = %q, want %q", res.Stdout, "data\n")
	}
	if res.Stderr != "diagnostic\n" {
		t.Errorf("Stderr = %q, want %q", res.Stderr, "diagnostic\n")
	}
	if res.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", res.ExitCode)
	}
}

func TestRunArgumentsAreNotShellWords(t *testing.T) {
	dangerous := []string{"hello; rm -rf /", "$HOME && whoami", "*"}
	res, err := Run(context.Background(), helperCmd(t, "echo", dangerous...))
	if err != nil {
		t.Fatalf("Run returned err = %v, want nil", err)
	}
	want := strings.Join(dangerous, "\n") + "\n"
	if res.Stdout != want {
		t.Errorf("Stdout = %q, want %q — arguments must reach the child verbatim, "+
			"with no shell to expand, split or chain them", res.Stdout, want)
	}
	if res.Stderr != "" {
		t.Errorf("Stderr = %q, want empty", res.Stderr)
	}
}

func TestRunReportsExitCode(t *testing.T) {
	for _, code := range []int{0, 1, 3, 42} {
		t.Run(strconv.Itoa(code), func(t *testing.T) {
			res, err := Run(context.Background(), helperCmd(t, "exit", strconv.Itoa(code)))
			if err != nil {
				t.Fatalf("Run returned err = %v, want nil — a non-zero exit is a "+
					"result, not a Go error", err)
			}
			if res.ExitCode != code {
				t.Errorf("ExitCode = %d, want %d", res.ExitCode, code)
			}
			if want := fmt.Sprintf("exiting with %d\n", code); res.Stderr != want {
				t.Errorf("Stderr = %q, want %q", res.Stderr, want)
			}
		})
	}
}

func TestRunSendsStdin(t *testing.T) {
	c := helperCmd(t, "cat")
	c.Stdin = strings.NewReader("one\ntwo\n")
	res, err := Run(context.Background(), c)
	if err != nil {
		t.Fatalf("Run returned err = %v, want nil", err)
	}
	if res.Stdout != "one\ntwo\n" {
		t.Errorf("Stdout = %q, want %q", res.Stdout, "one\ntwo\n")
	}
}

func TestRunEnvironment(t *testing.T) {
	t.Setenv("TUTOR_INHERITED", "from-parent")
	t.Setenv("TUTOR_OVERRIDDEN", "from-parent")

	c := helperCmd(t, "env", "TUTOR_INHERITED", "TUTOR_OVERRIDDEN", "TUTOR_EXTRA")
	c.Env = append(c.Env, "TUTOR_OVERRIDDEN=from-child", "TUTOR_EXTRA=only-in-child")

	res, err := Run(context.Background(), c)
	if err != nil {
		t.Fatalf("Run returned err = %v, want nil", err)
	}
	want := "TUTOR_INHERITED=from-parent\n" +
		"TUTOR_OVERRIDDEN=from-child\n" +
		"TUTOR_EXTRA=only-in-child\n"
	if res.Stdout != want {
		t.Errorf("child environment =\n%q\nwant\n%q\n"+
			"Command.Env adds to the inherited environment and overrides it; it "+
			"does not replace it", res.Stdout, want)
	}
}

func TestRunUsesDir(t *testing.T) {
	dir := t.TempDir()
	c := helperCmd(t, "pwd")
	c.Dir = dir

	res, err := Run(context.Background(), c)
	if err != nil {
		t.Fatalf("Run returned err = %v, want nil", err)
	}
	// The kernel reports the resolved path, and t.TempDir() goes through a
	// symlink on macOS — so compare what both sides resolve to.
	got, err := filepath.EvalSymlinks(strings.TrimSpace(res.Stdout))
	if err != nil {
		t.Fatalf("resolving child's working directory: %v", err)
	}
	want, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("resolving temp dir: %v", err)
	}
	if got != want {
		t.Errorf("child working directory = %q, want %q", got, want)
	}
}

func TestRunMissingExecutable(t *testing.T) {
	res, err := Run(context.Background(), Command{Name: "tutor-no-such-program"})
	if !errors.Is(err, exec.ErrNotFound) {
		t.Fatalf("Run returned err = %v, want one wrapping exec.ErrNotFound", err)
	}
	if res.ExitCode != -1 {
		t.Errorf("ExitCode = %d, want -1 for a command that never ran", res.ExitCode)
	}
}

func TestRunEmptyName(t *testing.T) {
	res, err := Run(context.Background(), Command{})
	if !errors.Is(err, ErrEmptyCommand) {
		t.Fatalf("Run returned err = %v, want one wrapping ErrEmptyCommand", err)
	}
	if res.ExitCode != -1 {
		t.Errorf("ExitCode = %d, want -1", res.ExitCode)
	}
}

func TestRunCancelledBeforeStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	res, err := Run(ctx, helperCmd(t, "echo", "should not appear"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run returned err = %v, want one wrapping context.Canceled", err)
	}
	if res.Stdout != "" {
		t.Errorf("Stdout = %q, want empty — the child must never start", res.Stdout)
	}
	if res.ExitCode != -1 {
		t.Errorf("ExitCode = %d, want -1", res.ExitCode)
	}
}

func TestRunDeadline(t *testing.T) {
	// The child sleeps for a minute; the deadline is short. Nothing here
	// asserts how long anything took — only which error came back, which is the
	// same whether the deadline beat the child to the start line or not.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	res, err := Run(ctx, helperCmd(t, "block"))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Run returned err = %v, want one wrapping context.DeadlineExceeded — "+
			"a killed child looks like any other failure until you check ctx.Err()", err)
	}
	if res.ExitCode != -1 {
		t.Errorf("ExitCode = %d, want -1", res.ExitCode)
	}
}

// --- RunSteps --------------------------------------------------------------

func TestRunStepsAllSucceed(t *testing.T) {
	cmds := []Command{
		helperCmd(t, "echo", "one"),
		helperCmd(t, "echo", "two"),
		helperCmd(t, "echo", "three"),
	}
	results, err := RunSteps(context.Background(), cmds)
	if err != nil {
		t.Fatalf("RunSteps returned err = %v, want nil", err)
	}
	if len(results) != 3 {
		t.Fatalf("got %d results, want 3", len(results))
	}
	for i, want := range []string{"one\n", "two\n", "three\n"} {
		if results[i].Stdout != want {
			t.Errorf("results[%d].Stdout = %q, want %q", i, results[i].Stdout, want)
		}
	}
}

func TestRunStepsStopsAtFirstFailure(t *testing.T) {
	cmds := []Command{
		helperCmd(t, "echo", "one"),
		helperCmd(t, "exit", "7"),
		helperCmd(t, "echo", "three"),
	}
	results, err := RunSteps(context.Background(), cmds)

	var se *StepError
	if !errors.As(err, &se) {
		t.Fatalf("RunSteps returned err = %v (%T), want a *StepError", err, err)
	}
	if se.Index != 1 {
		t.Errorf("StepError.Index = %d, want 1", se.Index)
	}
	if se.Err != nil {
		t.Errorf("StepError.Err = %v, want nil — the step ran, it just exited non-zero", se.Err)
	}
	if se.Result.ExitCode != 7 {
		t.Errorf("StepError.Result.ExitCode = %d, want 7", se.Result.ExitCode)
	}
	if !strings.Contains(se.Error(), "step 1") || !strings.Contains(se.Error(), "exit status 7") {
		t.Errorf("StepError.Error() = %q, want it to mention %q and %q",
			se.Error(), "step 1", "exit status 7")
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2 — the third step must not run", len(results))
	}
	if results[0].Stdout != "one\n" {
		t.Errorf("results[0].Stdout = %q, want %q", results[0].Stdout, "one\n")
	}
	if results[1].ExitCode != 7 {
		t.Errorf("results[1].ExitCode = %d, want 7 — the failing step's result is kept",
			results[1].ExitCode)
	}
}

func TestRunStepsStepCannotStart(t *testing.T) {
	cmds := []Command{
		helperCmd(t, "echo", "one"),
		{Name: "tutor-no-such-program"},
	}
	results, err := RunSteps(context.Background(), cmds)

	var se *StepError
	if !errors.As(err, &se) {
		t.Fatalf("RunSteps returned err = %v (%T), want a *StepError", err, err)
	}
	if se.Index != 1 {
		t.Errorf("StepError.Index = %d, want 1", se.Index)
	}
	if !errors.Is(err, exec.ErrNotFound) {
		t.Errorf("errors.Is(err, exec.ErrNotFound) = false; StepError must unwrap to its cause")
	}
	if len(results) != 1 {
		t.Errorf("got %d results, want 1 — a step that never ran has no result", len(results))
	}
}

func TestRunStepsStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cmds := []Command{helperCmd(t, "echo", "one"), helperCmd(t, "echo", "two")}
	results, err := RunSteps(ctx, cmds)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RunSteps returned err = %v, want one wrapping context.Canceled", err)
	}
	var se *StepError
	if !errors.As(err, &se) {
		t.Fatalf("RunSteps returned err = %v (%T), want a *StepError", err, err)
	}
	if se.Index != 0 {
		t.Errorf("StepError.Index = %d, want 0 — nothing should have started", se.Index)
	}
	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}

package logkit

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// runCLI drives Run the way a testscript-style test drives a binary: fixed
// arguments and stdin, captured stdout, stderr and exit code.
func runCLI(t *testing.T, args []string, stdin string) (stdout, stderr string, code int) {
	t.Helper()
	var out, errOut bytes.Buffer
	code = Run(args, strings.NewReader(stdin), &out, &errOut)
	return out.String(), errOut.String(), code
}

func TestRunGoldenReport(t *testing.T) {
	input, err := os.ReadFile("testdata/service.log")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	stdout, stderr, code := runCLI(t, nil, string(input))
	if code != 0 {
		t.Errorf("exit code = %d, want 0 (every line in the fixture is valid)", code)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}
	assertGolden(t, "service-report.txt", []byte(stdout))
}

func TestRunMinLevel(t *testing.T) {
	input, err := os.ReadFile("testdata/service.log")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	stdout, _, code := runCLI(t, []string{"-min-level=warn"}, string(input))
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if strings.Contains(stdout, "debug") || strings.Contains(stdout, "info ") {
		t.Errorf("-min-level=warn kept levels below warn:\n%s", stdout)
	}
	if !strings.Contains(stdout, "total: 4 events, 0 skipped") {
		t.Errorf("want 4 surviving events, got:\n%s", stdout)
	}
}

func TestRunSkipsMalformedLines(t *testing.T) {
	stdin := "info|api|ok\n\nnot a log line\nerror|db|down\ntrace|api|bad level\n"
	stdout, stderr, code := runCLI(t, nil, stdin)
	if code != 1 {
		t.Errorf("exit code = %d, want 1 (some lines were skipped)", code)
	}
	for _, want := range []string{"logreport: line 3:", "logreport: line 5:"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr missing %q; got:\n%s", want, stderr)
		}
	}
	if strings.Contains(stderr, "line 2:") {
		t.Errorf("blank lines must not be reported as malformed; got:\n%s", stderr)
	}
	if !strings.Contains(stdout, "total: 2 events, 2 skipped") {
		t.Errorf("report should still be written for the good lines; got:\n%s", stdout)
	}
}

func TestRunUsageErrors(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"unknown flag", []string{"-nope"}},
		{"unknown min-level", []string{"-min-level=trace"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stdout, stderr, code := runCLI(t, c.args, "info|api|ok\n")
			if code != 2 {
				t.Errorf("exit code = %d, want 2 for a usage error", code)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty on a usage error", stdout)
			}
			if stderr == "" {
				t.Error("stderr is empty; a usage error must explain itself")
			}
		})
	}
}

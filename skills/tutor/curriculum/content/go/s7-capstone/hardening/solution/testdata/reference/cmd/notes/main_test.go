package main

import (
	"context"
	"strings"
	"testing"
	"time"
)

var runAt = time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)

func TestRunPrintsTheListing(t *testing.T) {
	t.Setenv(webhookEnv, "") // never publish from a test

	var out, problems strings.Builder
	in := strings.NewReader("n1|first|Work\n\nn2|second|work,home\n")
	if err := run(context.Background(), in, &out, &problems, runAt); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	want := "n1  first  [work]\nn2  second  [home work]\n"
	if out.String() != want {
		t.Errorf("stdout = %q, want %q", out.String(), want)
	}
	if problems.String() != "" {
		t.Errorf("stderr = %q, want empty", problems.String())
	}
}

// Malformed and duplicate lines are reported and skipped: one bad line in an
// import must not cost the caller the other 999.
func TestRunReportsMalformedLinesAndContinues(t *testing.T) {
	t.Setenv(webhookEnv, "")

	var out, problems strings.Builder
	in := strings.NewReader(strings.Join([]string{
		"n1|first",
		"../../etc/passwd|traversal",
		"no separator here",
		"n1|duplicate id",
		"n2|second",
	}, "\n"))
	if err := run(context.Background(), in, &out, &problems, runAt); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if want := "n1  first\nn2  second\n"; out.String() != want {
		t.Errorf("stdout = %q, want %q", out.String(), want)
	}
	report := problems.String()
	for _, want := range []string{"line 2 rejected", "line 3 rejected", "line 4 rejected", "3 line(s) rejected"} {
		if !strings.Contains(report, want) {
			t.Errorf("stderr = %q, want it to contain %q", report, want)
		}
	}
}

// A line past the buffer limit must fail loudly rather than growing the
// buffer until the input decides how much memory we use.
func TestRunRejectsOversizedLine(t *testing.T) {
	t.Setenv(webhookEnv, "")

	var out, problems strings.Builder
	in := strings.NewReader("n1|" + strings.Repeat("x", 64<<10) + "\n")
	err := run(context.Background(), in, &out, &problems, runAt)
	if err == nil {
		t.Fatal("run(oversized line) error = nil, want an error")
	}
	if !strings.Contains(err.Error(), "read input") {
		t.Errorf("run() error = %v, want it to mention reading input", err)
	}
}

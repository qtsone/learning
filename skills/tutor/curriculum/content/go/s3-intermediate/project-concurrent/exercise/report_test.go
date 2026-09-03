package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"
)

func TestParseURLs(t *testing.T) {
	in := strings.Join([]string{
		"https://example.com/one",
		"",
		"# nightly link audit",
		"  https://example.com/two  ",
		"\thttps://example.com/three",
		"",
	}, "\n")
	want := []string{
		"https://example.com/one",
		"https://example.com/two",
		"https://example.com/three",
	}

	got, err := ParseURLs(strings.NewReader(in))
	if err != nil {
		t.Fatalf("ParseURLs returned unexpected error: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("ParseURLs returned %d URLs %q, want %d %q", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ParseURLs()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestParseURLsEmptyInput(t *testing.T) {
	got, err := ParseURLs(strings.NewReader(""))
	if err != nil {
		t.Fatalf("ParseURLs(empty) returned unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("ParseURLs(empty) = %q, want no URLs", got)
	}
}

func TestParseURLsPropagatesReadError(t *testing.T) {
	sentinel := errors.New("disk failure")
	_, err := ParseURLs(iotest.ErrReader(sentinel))
	if !errors.Is(err, sentinel) {
		t.Errorf("ParseURLs with a failing reader: err = %v, want it to wrap %v", err, sentinel)
	}
}

func TestSummarize(t *testing.T) {
	results := []Result{
		{URL: "a", Status: 200},
		{URL: "b", Status: 301},
		{URL: "c", Status: 404},
		{URL: "d", Err: errors.New("connection refused")},
		{URL: "e", Status: 500},
	}
	got := Summarize(results)
	want := Summary{Checked: 5, OK: 2, Failed: 3}
	if got != want {
		t.Errorf("Summarize() = %+v, want %+v (OK is Err == nil && Status < 400)", got, want)
	}
}

func TestRunReportsFailures(t *testing.T) {
	srv := statusServer(t)
	in := strings.Join([]string{
		srv.URL + "/ok",
		"# checked nightly",
		srv.URL + "/gone",
		"",
		srv.URL + "/ok",
	}, "\n") + "\n"

	var out bytes.Buffer
	err := run(context.Background(), []string{"-c", "2", "-t", "2s"}, strings.NewReader(in), &out)
	if err == nil {
		t.Fatal("run returned nil with a broken link in the list, want an error")
	}

	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("run printed %d lines, want 4 (one per URL plus a summary):\n%s", len(lines), out.String())
	}
	wantOrder := []string{srv.URL + "/ok", srv.URL + "/gone", srv.URL + "/ok"}
	for i, u := range wantOrder {
		if !strings.Contains(lines[i], u) {
			t.Errorf("line %d = %q, want it to mention %s (report in input order)", i, lines[i], u)
		}
	}
	if !strings.HasPrefix(lines[0], "ok ") {
		t.Errorf("line 0 = %q, want prefix %q", lines[0], "ok ")
	}
	if !strings.HasPrefix(lines[1], "fail ") {
		t.Errorf("line 1 = %q, want prefix %q", lines[1], "fail ")
	}
	if !strings.HasPrefix(lines[2], "ok ") {
		t.Errorf("line 2 = %q, want prefix %q", lines[2], "ok ")
	}
	if want := "checked 3: 2 ok, 1 failed"; lines[3] != want {
		t.Errorf("summary line = %q, want %q", lines[3], want)
	}
}

func TestRunAllOKReturnsNil(t *testing.T) {
	srv := statusServer(t)
	in := srv.URL + "/ok\n" + srv.URL + "/empty\n"

	var out bytes.Buffer
	if err := run(context.Background(), nil, strings.NewReader(in), &out); err != nil {
		t.Fatalf("run returned %v with only healthy links, want nil", err)
	}
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if want := "checked 2: 2 ok, 0 failed"; lines[len(lines)-1] != want {
		t.Errorf("summary line = %q, want %q", lines[len(lines)-1], want)
	}
}

func TestRunRejectsBadFlags(t *testing.T) {
	err := run(context.Background(), []string{"-c", "banana"}, strings.NewReader(""), io.Discard)
	if err == nil {
		t.Error("run with a non-numeric -c returned nil, want a flag parse error")
	}
}

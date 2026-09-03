package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// newTestRenderer returns a Renderer writing into two buffers you can assert
// on — the same injection the real program performs with os.Stdout/os.Stderr.
func newTestRenderer(color bool, level Level) (*Renderer, *bytes.Buffer, *bytes.Buffer) {
	var out, errw bytes.Buffer
	return NewRenderer(&out, &errw, color, level), &out, &errw
}

func TestPaint(t *testing.T) {
	cases := []struct {
		name  string
		color bool
		in    string
		codes []string
		want  string
	}{
		{"one code", true, "ok", []string{Green}, "\x1b[32mok\x1b[0m"},
		{"two codes in order", true, "ok", []string{Green, Bold}, "\x1b[32m\x1b[1mok\x1b[0m"},
		{"no codes stays bare", true, "ok", nil, "ok"},
		{"color off drops the codes", false, "ok", []string{Green, Bold}, "ok"},
		{"color off with no codes", false, "ok", nil, "ok"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, _, _ := newTestRenderer(c.color, LevelNormal)
			if got := r.Paint(c.in, c.codes...); got != c.want {
				t.Errorf("Paint(%q, %q) = %q, want %q", c.in, c.codes, got, c.want)
			}
		})
	}
}

func TestRendererStreamsAndLevels(t *testing.T) {
	cases := []struct {
		name    string
		level   Level
		color   bool
		call    func(r *Renderer)
		wantOut string
		wantErr string
	}{
		{
			name:    "data goes to stdout even when quiet",
			level:   LevelQuiet,
			call:    func(r *Renderer) { r.Out("hello %s", "world") },
			wantOut: "hello world\n",
		},
		{
			name:    "info goes to stderr at normal level",
			level:   LevelNormal,
			call:    func(r *Renderer) { r.Info("scanned %d files", 3) },
			wantErr: "scanned 3 files\n",
		},
		{
			name:  "info is silent when quiet",
			level: LevelQuiet,
			call:  func(r *Renderer) { r.Info("scanned %d files", 3) },
		},
		{
			name:    "info still prints when verbose",
			level:   LevelVerbose,
			call:    func(r *Renderer) { r.Info("scanned %d files", 3) },
			wantErr: "scanned 3 files\n",
		},
		{
			name:  "debug is silent at normal level",
			level: LevelNormal,
			call:  func(r *Renderer) { r.Debug("opening %s", "go.mod") },
		},
		{
			name:    "debug prints when verbose",
			level:   LevelVerbose,
			call:    func(r *Renderer) { r.Debug("opening %s", "go.mod") },
			wantErr: "opening go.mod\n",
		},
		{
			name:    "errors survive quiet mode",
			level:   LevelQuiet,
			call:    func(r *Renderer) { r.Errorf("boom %d", 2) },
			wantErr: "error: boom 2\n",
		},
		{
			name:    "the error prefix is styled when color is on",
			level:   LevelNormal,
			color:   true,
			call:    func(r *Renderer) { r.Errorf("boom %d", 2) },
			wantErr: "\x1b[31m\x1b[1merror:\x1b[0m boom 2\n",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, out, errw := newTestRenderer(c.color, c.level)
			c.call(r)
			if out.String() != c.wantOut {
				t.Errorf("stdout = %q, want %q", out.String(), c.wantOut)
			}
			if errw.String() != c.wantErr {
				t.Errorf("stderr = %q, want %q", errw.String(), c.wantErr)
			}
		})
	}
}

var sampleResults = []Result{
	{Name: "go.mod", Status: "ok"},
	{Name: "README", Status: "fail", Detail: "missing header"},
	{Name: "notes", Status: "skip"},
}

func TestResultsHumanPlain(t *testing.T) {
	r, out, errw := newTestRenderer(false, LevelNormal)
	if err := r.Results(sampleResults, false); err != nil {
		t.Fatalf("Results returned %v, want nil", err)
	}
	want := "ok    go.mod\n" +
		"fail  README: missing header\n" +
		"skip  notes\n"
	if out.String() != want {
		t.Errorf("stdout =\n%q\nwant\n%q", out.String(), want)
	}
	if errw.Len() != 0 {
		t.Errorf("stderr = %q, want results to stay on stdout", errw.String())
	}
}

func TestResultsHumanColored(t *testing.T) {
	r, out, _ := newTestRenderer(true, LevelNormal)
	if err := r.Results(sampleResults, false); err != nil {
		t.Fatalf("Results returned %v, want nil", err)
	}
	// The visible columns must line up exactly as in the plain case: escape
	// sequences take no columns, so the padding is computed from the status
	// word, not from the painted string.
	want := "\x1b[32mok\x1b[0m    go.mod\n" +
		"\x1b[31mfail\x1b[0m  README: missing header\n" +
		"skip  notes\n"
	if out.String() != want {
		t.Errorf("stdout =\n%q\nwant\n%q", out.String(), want)
	}
}

func TestResultsJSON(t *testing.T) {
	// Color is deliberately on: JSON output must never carry escape sequences.
	r, out, errw := newTestRenderer(true, LevelNormal)
	if err := r.Results(sampleResults, true); err != nil {
		t.Fatalf("Results returned %v, want nil", err)
	}
	want := `{"results":[` +
		`{"name":"go.mod","status":"ok"},` +
		`{"name":"README","status":"fail","detail":"missing header"},` +
		`{"name":"notes","status":"skip"}` +
		"]}\n"
	if out.String() != want {
		t.Errorf("stdout =\n%q\nwant\n%q", out.String(), want)
	}
	if strings.Contains(out.String(), "\x1b") {
		t.Error("JSON output contains an ANSI escape sequence")
	}
	if errw.Len() != 0 {
		t.Errorf("stderr = %q, want the data stream to carry the JSON alone", errw.String())
	}
}

func TestResultsJSONEmptyIsAnArray(t *testing.T) {
	for _, c := range []struct {
		name string
		res  []Result
	}{
		{"nil slice", nil},
		{"empty slice", []Result{}},
	} {
		t.Run(c.name, func(t *testing.T) {
			r, out, _ := newTestRenderer(false, LevelNormal)
			if err := r.Results(c.res, true); err != nil {
				t.Fatalf("Results returned %v, want nil", err)
			}
			want := "{\"results\":[]}\n"
			if out.String() != want {
				t.Errorf("stdout = %q, want %q — a consumer that loops over the "+
					"array should not have to special-case null", out.String(), want)
			}
		})
	}
}

// failingWriter accepts nothing, so Results has to report the write error
// instead of losing it.
type failingWriter struct{ err error }

func (f failingWriter) Write(p []byte) (int, error) { return 0, f.err }

func TestResultsReportsWriteErrors(t *testing.T) {
	sentinel := errors.New("disk full")
	for _, asJSON := range []bool{false, true} {
		var errw bytes.Buffer
		r := NewRenderer(failingWriter{err: sentinel}, &errw, false, LevelNormal)
		if err := r.Results(sampleResults, asJSON); !errors.Is(err, sentinel) {
			t.Errorf("Results(asJSON=%v) error = %v, want it to carry %v",
				asJSON, err, sentinel)
		}
	}
}

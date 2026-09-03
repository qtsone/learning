package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

// ANSI SGR (Select Graphic Rendition) sequences. Each one turns an attribute
// on; Reset turns every attribute back off.
const (
	Reset  = "\x1b[0m"
	Bold   = "\x1b[1m"
	Dim    = "\x1b[2m"
	Red    = "\x1b[31m"
	Green  = "\x1b[32m"
	Yellow = "\x1b[33m"
)

// statusColumn is the visible width reserved for the status word.
const statusColumn = 4

// Level is how chatty the tool should be on its diagnostic stream.
type Level int

const (
	// LevelQuiet prints results and errors only.
	LevelQuiet Level = iota
	// LevelNormal adds progress and informational messages.
	LevelNormal
	// LevelVerbose adds debug detail.
	LevelVerbose
)

// Result is one line of the tool's output: the thing that was checked and how
// it went.
type Result struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// Report is the machine-readable document written in JSON mode. It exists so
// the JSON output is a single self-describing object with a stable shape,
// rather than a bare array that cannot grow new fields.
type Report struct {
	Results []Result `json:"results"`
}

// Renderer writes a program's output to the two streams it was handed. It
// never touches os.Stdout or os.Stderr itself, which is what makes every
// method below assertable byte for byte in a test.
type Renderer struct {
	out   io.Writer
	errw  io.Writer
	color bool
	level Level
}

// NewRenderer builds a Renderer over the data stream out and the diagnostic
// stream errw. color is the already-resolved answer from ResolveColor — the
// renderer applies policy, it does not decide it.
func NewRenderer(out, errw io.Writer, color bool, level Level) *Renderer {
	return &Renderer{out: out, errw: errw, color: color, level: level}
}

// Paint wraps s in the given SGR codes followed by a single Reset when color
// is enabled, and returns s untouched when it is not.
func (r *Renderer) Paint(s string, codes ...string) string {
	if !r.color || len(codes) == 0 {
		return s
	}
	return strings.Join(codes, "") + s + Reset
}

// Out writes one data line to the data stream. Data is the program's product,
// so quiet mode does not suppress it.
func (r *Renderer) Out(format string, a ...any) {
	fmt.Fprintf(r.out, format+"\n", a...)
}

// Info writes one diagnostic line to the diagnostic stream at LevelNormal and
// above.
func (r *Renderer) Info(format string, a ...any) {
	if r.level >= LevelNormal {
		fmt.Fprintf(r.errw, format+"\n", a...)
	}
}

// Debug writes one diagnostic line to the diagnostic stream at LevelVerbose
// only.
func (r *Renderer) Debug(format string, a ...any) {
	if r.level >= LevelVerbose {
		fmt.Fprintf(r.errw, format+"\n", a...)
	}
}

// Errorf writes one error line to the diagnostic stream at every level,
// including LevelQuiet: silencing progress must never silence failure.
func (r *Renderer) Errorf(format string, a ...any) {
	fmt.Fprintf(r.errw, "%s %s\n", r.Paint("error:", Red, Bold), fmt.Sprintf(format, a...))
}

// Results writes res to the data stream and reports any write error.
func (r *Renderer) Results(res []Result, asJSON bool) error {
	if asJSON {
		if res == nil {
			// A nil slice marshals to null; consumers expect an array they can
			// loop over, so hand the encoder an empty one instead.
			res = []Result{}
		}
		return json.NewEncoder(r.out).Encode(Report{Results: res})
	}

	var b strings.Builder
	for _, item := range res {
		// Pad by the visible width of the status word: the escape sequences
		// Paint adds occupy no columns, so padding the painted string with
		// %-4s would push every colored line out of alignment.
		b.WriteString(r.Paint(item.Status, statusCodes(item.Status)...))
		// Count runes, not bytes: a non-ASCII status word would otherwise
		// measure longer than the columns it occupies and under-pad.
		b.WriteString(strings.Repeat(" ", max(statusColumn-utf8.RuneCountInString(item.Status), 0)))
		b.WriteString("  ")
		b.WriteString(item.Name)
		if item.Detail != "" {
			b.WriteString(": ")
			b.WriteString(item.Detail)
		}
		b.WriteByte('\n')
	}
	_, err := io.WriteString(r.out, b.String())
	return err
}

func statusCodes(status string) []string {
	switch status {
	case "ok":
		return []string{Green}
	case "fail":
		return []string{Red}
	}
	return nil
}

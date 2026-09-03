package main

import "io"

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
// is enabled, and returns s untouched when it is not. Painting with no codes
// returns s unchanged either way — never emit a lone Reset.
//
// TODO: implement.
func (r *Renderer) Paint(s string, codes ...string) string {
	return ""
}

// Out writes one data line to the data stream. Data is the program's product,
// so quiet mode does not suppress it.
//
// TODO: format like fmt.Printf and append a newline.
func (r *Renderer) Out(format string, a ...any) {
}

// Info writes one diagnostic line to the diagnostic stream at LevelNormal and
// above.
//
// TODO: implement.
func (r *Renderer) Info(format string, a ...any) {
}

// Debug writes one diagnostic line to the diagnostic stream at LevelVerbose
// only.
//
// TODO: implement.
func (r *Renderer) Debug(format string, a ...any) {
}

// Errorf writes one error line to the diagnostic stream at every level,
// including LevelQuiet: silencing progress must never silence failure. The
// line starts with the word "error:" painted Red and Bold, then a space, then
// the formatted message, then a newline.
//
// TODO: implement.
func (r *Renderer) Errorf(format string, a ...any) {
}

// Results writes res to the data stream and reports any write error.
//
// In JSON mode it encodes a single Report document — compact, one line,
// newline-terminated (encoding/json's Encoder does this for you) — and emits
// no ANSI even when color is enabled. An empty result set must encode as
// {"results":[]}, never {"results":null}.
//
// In human mode it writes one line per result:
//
//	<status><padding to 4 columns><two spaces><name>[": " <detail>]
//
// The status word is painted Green for "ok", Red for "fail", and left
// unpainted otherwise. Pad by the *visible* width of the status word, not the
// length of the painted string: escape sequences occupy no columns, so
// fmt.Sprintf("%-4s", painted) misaligns every colored line.
//
// TODO: implement both modes. Build the human output in one strings.Builder
// and write it once.
func (r *Renderer) Results(res []Result, asJSON bool) error {
	return nil
}

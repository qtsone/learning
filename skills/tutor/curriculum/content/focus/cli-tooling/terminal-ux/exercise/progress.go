package main

import "io"

// EraseLine clears from the cursor to the end of the line. It is cursor
// control rather than color, so NO_COLOR has no say over it — but a stream
// that is not a terminal must never see it, because nothing there interprets
// escape sequences.
const EraseLine = "\x1b[K"

// Progress reports how far a long-running operation has got. It is driven by
// explicit Update calls from the work loop, not by a timer, so the same code
// produces the same bytes on every run.
type Progress struct {
	w          io.Writer
	isTerminal bool
	label      string
	total      int
	n          int
}

// NewProgress builds a Progress over w — in a real program the diagnostic
// stream, so that progress never contaminates piped data. A total of 0 or less
// means "unknown length": report a running count instead of a percentage.
//
// Pass io.Discard as w to switch progress off in quiet mode.
func NewProgress(w io.Writer, isTerminal bool, label string, total int) *Progress {
	return &Progress{w: w, isTerminal: isTerminal, label: label, total: total}
}

// Update records that n items are done and redraws the status line.
//
// On a terminal it writes a carriage return, the status text, then EraseLine
// so that leftovers from a longer previous line disappear:
//
//	"\rlabel: 3/10 (30%)\x1b[K"      // known total, percentage floored
//	"\rlabel: 3\x1b[K"               // unknown total
//
// When the stream is not a terminal it writes nothing at all: a redraw needs a
// cursor, and a log file has none.
//
// TODO: implement.
func (p *Progress) Update(n int) {
}

// Done finishes the operation with a summary line that stays on screen:
//
//	"\rlabel: done (10/10)\x1b[K\n"  // terminal
//	"label: done (10/10)\n"          // not a terminal — no cursor tricks
//
// With an unknown total the summary is "label: done (10)".
//
// TODO: implement.
func (p *Progress) Done() {
}

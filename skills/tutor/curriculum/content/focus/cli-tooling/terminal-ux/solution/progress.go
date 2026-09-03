package main

import (
	"fmt"
	"io"
)

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

// Update records that n items are done and redraws the status line. Off a
// terminal it writes nothing: a redraw needs a cursor, and a log file has none.
func (p *Progress) Update(n int) {
	p.n = n
	if !p.isTerminal {
		return
	}
	// Carriage return first, erase-to-end-of-line last, so a shorter line can
	// never leave the tail of a longer one on screen.
	fmt.Fprintf(p.w, "\r%s%s", p.status(), EraseLine)
}

// Done finishes the operation with a summary line that stays on screen.
func (p *Progress) Done() {
	summary := fmt.Sprintf("%s: done (%d)", p.label, p.n)
	if p.total > 0 {
		summary = fmt.Sprintf("%s: done (%d/%d)", p.label, p.n, p.total)
	}
	if p.isTerminal {
		fmt.Fprintf(p.w, "\r%s%s\n", summary, EraseLine)
		return
	}
	fmt.Fprintln(p.w, summary)
}

func (p *Progress) status() string {
	if p.total <= 0 {
		return fmt.Sprintf("%s: %d", p.label, p.n)
	}
	return fmt.Sprintf("%s: %d/%d (%d%%)", p.label, p.n, p.total, p.n*100/p.total)
}

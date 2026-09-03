package logkit

import (
	"bufio"
	"flag"
	"fmt"
	"io"
)

// Exit codes returned by Run.
const (
	exitOK      = 0
	exitSkipped = 1
	exitUsage   = 2
)

// Run executes the logreport command with everything it touches passed in:
// arguments, input, output, diagnostics. It never reads os.Args, never writes
// to os.Stdout and never calls os.Exit — that is what makes a CLI testable.
//
// It reads wire lines from stdin, drops events below -min-level, writes the
// report from Report to stdout and one diagnostic per bad line to stderr, and
// returns the exit code the process should use:
//
//	0  every non-blank line parsed
//	1  the report was written, but at least one line was skipped
//	2  usage error: bad flags, unknown -min-level, or unreadable input
//
// Diagnostics use the form "logreport: line <n>: <err>", where <n> counts
// every line read, including blank ones.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("logreport", flag.ContinueOnError)
	fs.SetOutput(stderr)
	minLevel := fs.String("min-level", "debug", "drop events below this level")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	minRank, ok := LevelRank(*minLevel)
	if !ok {
		fmt.Fprintf(stderr, "logreport: unknown -min-level %q\n", *minLevel)
		return exitUsage
	}

	var events []Event
	skipped := 0
	scanner := bufio.NewScanner(stdin)
	for line := 1; scanner.Scan(); line++ {
		text := scanner.Text()
		if text == "" {
			continue
		}
		ev, err := ParseLine(text)
		if err != nil {
			fmt.Fprintf(stderr, "logreport: line %d: %v\n", line, err)
			skipped++
			continue
		}
		if rank, _ := LevelRank(ev.Level); rank < minRank {
			continue
		}
		events = append(events, ev)
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(stderr, "logreport: read input: %v\n", err)
		return exitUsage
	}

	fmt.Fprint(stdout, Report(events, skipped))
	if skipped > 0 {
		return exitSkipped
	}
	return exitOK
}

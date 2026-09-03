package logkit

import "io"

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
	// TODO: parse flags into a flag.FlagSet of your own (never
	// flag.CommandLine) with output directed at stderr, scan stdin, and
	// render the report.
	return 2
}

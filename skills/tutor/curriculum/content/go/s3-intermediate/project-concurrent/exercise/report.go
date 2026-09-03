package main

import (
	"context"
	"io"
)

// ParseURLs reads one URL per line from r, trimming surrounding whitespace
// and skipping blank lines and lines starting with "#". It returns the
// reader's error if reading fails.
func ParseURLs(r io.Reader) ([]string, error) {
	// TODO: implement with a bufio.Scanner.
	return nil, nil
}

// Summary aggregates a run: Checked counts every result, OK counts results
// with no error and a status below 400, Failed counts everything else.
type Summary struct {
	Checked int
	OK      int
	Failed  int
}

// Summarize folds results into a Summary.
func Summarize(results []Result) Summary {
	// TODO: implement.
	return Summary{}
}

// run is the whole tool behind a testable signature: parse flags from args
// (-c concurrency, default 4; -t per-request timeout, default 10s), read
// URLs from in, check them, and print the report to out — one line per URL
// in input order ("ok <status> <url>" or "fail <url>: ...") followed by
// "checked <n>: <ok> ok, <failed> failed". It returns a non-nil error if
// setup fails or any link failed.
func run(ctx context.Context, args []string, in io.Reader, out io.Writer) error {
	// TODO: implement with a flag.NewFlagSet (the global flag package can
	// only parse once per process — tests call run many times).
	return nil
}

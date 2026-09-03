package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"
)

// ParseURLs reads one URL per line from r, trimming surrounding whitespace
// and skipping blank lines and lines starting with "#". It returns the
// reader's error if reading fails.
func ParseURLs(r io.Reader) ([]string, error) {
	var urls []string
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		urls = append(urls, line)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("reading URL list: %w", err)
	}
	return urls, nil
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
	s := Summary{Checked: len(results)}
	for _, r := range results {
		if r.Err == nil && r.Status < 400 {
			s.OK++
		} else {
			s.Failed++
		}
	}
	return s
}

// run is the whole tool behind a testable signature: parse flags from args
// (-c concurrency, default 4; -t per-request timeout, default 10s), read
// URLs from in, check them, and print the report to out — one line per URL
// in input order ("ok <status> <url>" or "fail <url>: ...") followed by
// "checked <n>: <ok> ok, <failed> failed". It returns a non-nil error if
// setup fails or any link failed.
func run(ctx context.Context, args []string, in io.Reader, out io.Writer) error {
	// A fresh FlagSet per call: the global flag package can only parse
	// once per process, which would break repeated calls from tests.
	fs := flag.NewFlagSet("linkcheck", flag.ContinueOnError)
	fs.SetOutput(out)
	concurrency := fs.Int("c", 4, "maximum requests in flight at once")
	timeout := fs.Duration("t", 10*time.Second, "per-request timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}

	urls, err := ParseURLs(in)
	if err != nil {
		return err
	}

	c := &Checker{Concurrency: *concurrency, Timeout: *timeout}
	results := c.Check(ctx, urls)

	for _, r := range results {
		switch {
		case r.Err != nil:
			fmt.Fprintf(out, "fail %s: %v\n", r.URL, r.Err)
		case r.Status >= 400:
			fmt.Fprintf(out, "fail %s: status %d\n", r.URL, r.Status)
		default:
			fmt.Fprintf(out, "ok %d %s\n", r.Status, r.URL)
		}
	}

	s := Summarize(results)
	fmt.Fprintf(out, "checked %d: %d ok, %d failed\n", s.Checked, s.OK, s.Failed)
	if s.Failed > 0 {
		return fmt.Errorf("%d of %d links failed", s.Failed, s.Checked)
	}
	return nil
}

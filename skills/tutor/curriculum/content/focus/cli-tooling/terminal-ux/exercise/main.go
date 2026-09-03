package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

// streams is everything this program is allowed to know about the outside
// world: three byte streams, and three answers to "is that a terminal?".
// Everything below main takes this struct, so a test can build one out of
// buffers and booleans.
type streams struct {
	in                    io.Reader
	out, errw             io.Writer
	inTTY, outTTY, errTTY bool
}

func main() {
	os.Exit(run(os.Args[1:], streams{
		in:     os.Stdin,
		out:    os.Stdout,
		errw:   os.Stderr,
		inTTY:  isTerminal(os.Stdin),
		outTTY: isTerminal(os.Stdout),
		errTTY: isTerminal(os.Stderr),
	}, os.LookupEnv))
}

// isTerminal reports whether f is attached to a character device. This is the
// only place in the program that asks the operating system about its streams;
// every layer below is handed the answer as a plain bool.
func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func run(args []string, s streams, env EnvLookup) int {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	fs.SetOutput(s.errw)
	colorFlag := fs.String("color", "auto", `when to colorize: "auto", "always" or "never"`)
	noColor := fs.Bool("no-color", false, "shorthand for -color=never")
	asJSON := fs.Bool("json", false, "write machine-readable JSON to stdout")
	quiet := fs.Bool("q", false, "print results and errors only")
	verbose := fs.Bool("v", false, "print debug detail")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	mode, err := ParseColorMode(*colorFlag)
	if err != nil {
		fmt.Fprintf(s.errw, "error: %v\n", err)
		return 2
	}
	if *noColor {
		mode = ColorNever
	}

	level := LevelNormal
	switch {
	case *quiet:
		level = LevelQuiet
	case *verbose:
		level = LevelVerbose
	}

	// JSON is a contract with a parser: never style it, whatever the flags say.
	color := !*asJSON && ResolveColor(mode, s.outTTY, env)
	r := NewRenderer(s.out, s.errw, color, level)

	names := fs.Args()
	if len(names) == 0 {
		names = []string{"go.mod", "main.go", "README"}
	}

	progressTo := s.errw
	if level == LevelQuiet || *asJSON {
		progressTo = io.Discard
	}
	p := NewProgress(progressTo, s.errTTY, "checking", len(names))

	results := make([]Result, 0, len(names))
	failed := 0
	for i, name := range names {
		r.Debug("checking %s", name)
		res := check(name)
		if res.Status != "ok" {
			failed++
		}
		results = append(results, res)
		p.Update(i + 1)
	}
	p.Done()
	r.Info("%d checked, %d failed", len(results), failed)

	// Quiet and JSON both mean "nobody is watching": do not stop to ask.
	if failed > 0 && !*asJSON && level > LevelQuiet {
		answer, err := NewPrompter(s.in, s.errw, s.inTTY).
			Ask(r.Paint("List the failures too?", Bold)+" (yes/no)", "yes", yesNo)
		if err != nil {
			r.Errorf("%v", err)
			return 2
		}
		if answer == "no" {
			results = onlyOK(results)
		}
	}

	if err := r.Results(results, *asJSON); err != nil {
		r.Errorf("writing results: %v", err)
		return 1
	}
	if failed > 0 {
		return 1
	}
	return 0
}

// check stands in for whatever real work the tool would do.
func check(name string) Result {
	if strings.HasSuffix(name, ".go") || name == "go.mod" {
		return Result{Name: name, Status: "ok"}
	}
	return Result{Name: name, Status: "fail", Detail: "not a Go source file"}
}

func yesNo(answer string) error {
	switch answer {
	case "yes", "no":
		return nil
	}
	return fmt.Errorf("%q is not an answer, type yes or no", answer)
}

func onlyOK(res []Result) []Result {
	kept := make([]Result, 0, len(res))
	for _, r := range res {
		if r.Status == "ok" {
			kept = append(kept, r)
		}
	}
	return kept
}

package capstone

import (
	"fmt"
	"strings"
	"testing"
)

// Criterion 1: the benchmark that measured the bottleneck is committed next to
// the code it measures, and at least one benchmark reports allocations.
func TestBenchmarksAreCommitted(t *testing.T) {
	dir := project(t)
	found, err := benchmarks(dir)
	if err != nil {
		t.Fatalf("scanning %s for benchmarks: %v", dir, err)
	}
	if len(found) == 0 {
		t.Fatalf(`no Benchmark functions found in %s

A benchmark you deleted is a measurement nobody can repeat. Commit the one you
used for the baseline, in a _test.go file beside the code it measures:

    func BenchmarkThing(b *testing.B) {
        input := buildInput()   // setup, outside the loop
        b.ReportAllocs()
        b.ResetTimer()
        for range b.N {
            sink = Thing(input)
        }
    }`, dir)
	}

	var names []string
	allocs := false
	for _, b := range found {
		names = append(names, fmt.Sprintf("  %s  (%s)", b.Name, b.File))
		if b.Allocs {
			allocs = true
		}
	}
	t.Logf("%d benchmark(s):\n%s", len(found), strings.Join(names, "\n"))

	if !allocs {
		t.Errorf(`no benchmark calls b.ReportAllocs()

ns/op depends on the machine that ran it; allocs/op and B/op do not. Add
b.ReportAllocs() to at least the benchmark covering your bottleneck, so your
before/after numbers carry a figure a reviewer can reproduce on other hardware.`)
	}
}

// Criterion 2: the benchmarks still run. -benchtime=1x makes every reported
// number meaningless, which is the point — this asserts they execute, never
// how fast. Wall-clock assertions belong in no automated grader.
func TestBenchmarksRun(t *testing.T) {
	dir := project(t)
	out, err := runGo(dir, "test", "-bench=.", "-benchtime=1x", "-run=^$", "./...")
	if err != nil {
		t.Fatalf("go test -bench=. -benchtime=1x -run=^$ ./... failed in %s: %v\n\n%s\n\n"+
			"A benchmark that no longer compiles or panics is a measurement you "+
			"cannot repeat, so it does not count as evidence.", dir, err, tail(out, 4000))
	}
	results := benchResultRe.FindAllStringSubmatch(out, -1)
	if len(results) == 0 {
		t.Fatalf("`go test -bench=.` produced no benchmark results in %s:\n\n%s\n\n"+
			"The functions exist but nothing ran — check they are named Benchmark…, "+
			"take *testing.B, and live in a package `go test ./...` reaches.",
			dir, tail(out, 3000))
	}
	var ran []string
	for _, m := range results {
		ran = append(ran, m[1])
	}
	t.Logf("%d benchmark result(s) executed: %s", len(ran), strings.Join(ran, " "))
}

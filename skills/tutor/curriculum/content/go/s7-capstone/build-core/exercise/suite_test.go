package capstone

import (
	"strings"
	"testing"
)

// coverageFloor is a floor, not a goal: below it the suite demonstrably does
// not exercise the module, above it the number says nothing about quality.
const coverageFloor = 60.0

// Criterion 3: your own test suite passes, with the race detector on.
func TestProjectTestsPassUnderRace(t *testing.T) {
	run := projectSuite(t)
	if run.err == nil {
		return
	}
	what := "go test -race ./... failed"
	if strings.Contains(run.output, "DATA RACE") {
		what = "go test -race ./... detected a data race"
	}
	t.Fatalf("%s: %v\n\n%s", what, run.err, tail(run.output, 6000))
}

// Criterion 4: statement coverage across the module clears the floor.
func TestCoverageMeetsFloor(t *testing.T) {
	dir := project(t)
	run := projectSuite(t)
	if run.err != nil {
		t.Fatalf("coverage cannot be measured until the suite passes — " +
			"fix TestProjectTestsPassUnderRace first")
	}
	out, err := runGo(dir, "tool", "cover", "-func="+run.profile)
	if err != nil {
		t.Fatalf("go tool cover -func failed: %v\n\n%s", err, tail(out, 2000))
	}
	total, err := coverageTotal(out)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if total < coverageFloor {
		t.Fatalf("total statement coverage is %.1f%%, floor is %.1f%%\n\n"+
			"untested functions (up to 15):\n%s\n"+
			"run `go tool cover -func` yourself for the full list, and cover "+
			"behaviour that matters rather than padding the number",
			total, coverageFloor, uncovered(out, 15))
	}
	t.Logf("total statement coverage: %.1f%%", total)
}

// uncovered lists the functions `go tool cover -func` reports at zero. The
// percentage is the last column, and it has to be compared as a whole field:
// a suffix match on "0.0%" also matches 100.0%, 50.0% and 20.0%, which would
// hand you a list of well-tested functions and tell you to go test them.
func uncovered(coverFunc string, limit int) string {
	var lines []string
	for _, line := range strings.Split(coverFunc, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "total:") {
			continue
		}
		fields := strings.Fields(trimmed)
		if fields[len(fields)-1] == "0.0%" {
			lines = append(lines, "  "+trimmed)
			if len(lines) == limit {
				break
			}
		}
	}
	if len(lines) == 0 {
		return "  (none — the shortfall is spread across partly covered functions)"
	}
	return strings.Join(lines, "\n")
}

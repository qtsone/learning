package capstone

import (
	"strings"
	"testing"
)

const maxNamedTests = 25

// Criterion 6: the optimized path is pinned by tests you can name, and those
// tests pass. A faster wrong answer is not an optimization, so the Correctness
// section of PERF.md has to name the tests that prove behaviour is unchanged —
// and the harness runs exactly those.
func TestNamedCorrectnessTestsPass(t *testing.T) {
	dir := project(t)
	s := requireSection(t, "correctness")

	names := testNames(s.Body)
	if len(names) == 0 {
		t.Fatalf(`%s:%d: `+"`%s`"+` names no test functions

Name the tests that would fail if your optimization changed behaviour, by their
Go identifiers, e.g.

    Covered by `+"`TestListFiltersAndKeepsOrder`"+` and
    `+"`TestListByTagMatchesLinearScan`"+`, which cross-checks the fast path
    against the straightforward one over generated input.`,
			perfFileName, s.Line, s.Heading)
	}
	if len(names) > maxNamedTests {
		names = names[:maxNamedTests]
	}

	pattern := "^(" + strings.Join(names, "|") + ")$"
	out, err := runGo(dir, "test", "-run", pattern, "-count=1", "-v", "./...")
	if err != nil {
		t.Fatalf("the tests named in `%s` do not pass: %v\n\n%s",
			s.Heading, err, tail(out, 6000))
	}

	wanted := map[string]bool{}
	for _, n := range names {
		wanted[n] = true
	}
	var passed []string
	seen := map[string]bool{}
	for _, m := range testPassRe.FindAllStringSubmatch(out, -1) {
		name := strings.SplitN(m[1], "/", 2)[0]
		if wanted[name] && !seen[name] {
			seen[name] = true
			passed = append(passed, name)
		}
	}
	if len(passed) == 0 {
		t.Fatalf("none of the tests named in `%s` exist in %s: %s\n\n%s\n\n"+
			"Check the spelling against the function names in your _test.go files.",
			s.Heading, dir, strings.Join(names, ", "), tail(out, 3000))
	}
	if missing := len(names) - len(passed); missing > 0 {
		t.Errorf("%d of the %d tests named in `%s` did not run: %s",
			missing, len(names), s.Heading, strings.Join(notIn(names, seen), ", "))
	}
	t.Logf("correctness tests passing: %s", strings.Join(passed, " "))
}

func notIn(names []string, seen map[string]bool) []string {
	var out []string
	for _, n := range names {
		if !seen[n] {
			out = append(out, n)
		}
	}
	return out
}

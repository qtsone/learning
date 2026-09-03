package capstone

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

var (
	projectOnce sync.Once
	reportOnce  sync.Once
	projectPath string
	projectErr  error
)

// project resolves the capstone directory once for the whole run. Every test
// starts here, so a missing project fails fast with instructions instead of
// nine walls of identical text — only the first failure spells them out.
func project(t *testing.T) string {
	t.Helper()
	projectOnce.Do(func() { projectPath, projectErr = findProject() })
	if projectErr != nil {
		first := false
		reportOnce.Do(func() { first = true })
		if first {
			t.Fatalf("%v", projectErr)
		}
		t.Fatalf("no capstone project found — see the first failure for how to point the harness at yours")
	}
	return projectPath
}

// suiteRun is the single `go test -race` invocation the harness makes inside
// your project. Running it once keeps grading tolerable.
type suiteRun struct {
	output string
	err    error
}

var (
	suiteOnce sync.Once
	suite     suiteRun
)

func projectSuite(t *testing.T) suiteRun {
	t.Helper()
	dir := project(t)
	suiteOnce.Do(func() {
		out, err := runGo(dir, "test", "-race", "./...")
		suite = suiteRun{output: out, err: err}
	})
	return suite
}

// tail keeps the end of a tool's output, which is where the real message is.
func tail(s string, n int) string {
	s = strings.TrimRight(s, "\n")
	if len(s) <= n {
		return s
	}
	return "…\n" + s[len(s)-n:]
}

// bullets renders a list for a failure message, capped so one noisy file does
// not bury the instruction that follows it.
func bullets(items []string, limit int) string {
	var b strings.Builder
	for i, item := range items {
		if i == limit {
			fmt.Fprintf(&b, "  … and %d more\n", len(items)-limit)
			break
		}
		fmt.Fprintf(&b, "  %s\n", item)
	}
	return strings.TrimRight(b.String(), "\n")
}

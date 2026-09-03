package capstone

import (
	"os"
	"path/filepath"
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
// eight walls of identical text — only the first failure spells them out.
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

// suiteRun is the single `go test` invocation the harness makes inside your
// project. Three criteria read it: the suite passes, it is race-clean, and it
// covers enough of the module — running it once keeps grading tolerable.
//
// The run carries -coverpkg=./... on purpose. Without it the go tool credits
// statements only to the package whose own tests ran them, so an integration
// test in cmd/ that drives your whole application would leave every package it
// exercised reported at 0%. Coverage here is a property of the module, not a
// scorecard per directory.
type suiteRun struct {
	output  string
	err     error
	profile string
}

var (
	suiteOnce sync.Once
	suite     suiteRun
	suiteTemp string
)

func projectSuite(t *testing.T) suiteRun {
	t.Helper()
	dir := project(t)
	suiteOnce.Do(func() {
		tmp, err := os.MkdirTemp("", "capstone-cover")
		if err != nil {
			suite.err = err
			return
		}
		suiteTemp = tmp
		profile := filepath.Join(tmp, "cover.out")
		out, err := runGo(dir, "test", "-race", "-covermode=atomic",
			"-coverpkg=./...", "-coverprofile="+profile, "./...")
		suite = suiteRun{output: out, err: err, profile: profile}
	})
	return suite
}

func TestMain(m *testing.M) {
	code := m.Run()
	if suiteTemp != "" {
		os.RemoveAll(suiteTemp)
	}
	os.Exit(code)
}

// tail keeps the end of a tool's output, which is where the real message is.
func tail(s string, n int) string {
	s = strings.TrimRight(s, "\n")
	if len(s) <= n {
		return s
	}
	return "…\n" + s[len(s)-n:]
}

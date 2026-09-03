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
// six walls of identical text — only the first failure spells them out.
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

// perfFile is PERF.md, read and split into sections once. Four criteria read
// it, and they should all agree about what it says.
type perfFile struct {
	path     string
	raw      string
	sections []section
	err      error
}

var (
	perfOnce sync.Once
	perfDoc  perfFile
)

func perf(t *testing.T) perfFile {
	t.Helper()
	dir := project(t)
	perfOnce.Do(func() {
		path := filepath.Join(dir, perfFileName)
		raw, err := os.ReadFile(path)
		perfDoc = perfFile{path: path, raw: string(raw), err: err}
		if err == nil {
			perfDoc.sections = sections(perfDoc.raw)
		}
	})
	if perfDoc.err != nil {
		t.Fatalf("no %s at the project root (%s): %v\n\n"+
			"Copy exercise/PERF-template.md into your project as %s and fill it in. "+
			"It is the deliverable of this lesson: the performance claim a reviewer "+
			"can check without trusting you.", perfFileName, dir, perfDoc.err, perfFileName)
	}
	return perfDoc
}

// requireSection returns the named section of PERF.md, failing with the
// template's heading when it is missing.
func requireSection(t *testing.T, key string) section {
	t.Helper()
	doc := perf(t)
	for _, req := range requiredSections {
		if req.Key != key {
			continue
		}
		s, ok := findSection(doc.sections, req.Match)
		if !ok {
			t.Fatalf("%s has no `%s` section — it should hold %s",
				perfFileName, req.Heading, req.Wants)
		}
		return s
	}
	t.Fatalf("harness bug: unknown section %q", key)
	return section{}
}

// tail keeps the end of a tool's output, which is where the real message is.
func tail(s string, n int) string {
	s = strings.TrimRight(s, "\n")
	if len(s) <= n {
		return s
	}
	return "…\n" + s[len(s)-n:]
}

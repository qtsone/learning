package capstone

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

// Criterion 5: the profile that named the bottleneck is committed, and the
// Evidence section points at it. Either paste pprof output into the section
// itself, or commit an artifact and name it there — a claim whose evidence
// only ever existed in your terminal is not checkable.
func TestProfileEvidenceIsCommitted(t *testing.T) {
	dir := project(t)
	s := requireSection(t, "evidence")

	if pprofTextRe.MatchString(s.Body) {
		t.Logf("`%s` embeds pprof output directly", s.Heading)
		return
	}

	found, err := profileArtifacts(dir, map[string]bool{perfFileName: true})
	if err != nil {
		t.Fatalf("scanning %s for profile artifacts: %v", dir, err)
	}
	if len(found) == 0 {
		t.Fatalf(`no profile evidence found in %s

Commit what the profiler told you, one of:
  - the profile itself: cpu.out, mem.out, *.pprof, *.pb.gz
  - a summary a reader can open without pprof, e.g.
        go tool pprof -top cpu.out    > docs/perf/cpu-top.txt
        go tool pprof -list Hot cpu.out > docs/perf/cpu-list.txt

…then name that file in the `+"`%s`"+` section of %s. Pasting the pprof output
into that section instead also counts.`, dir, s.Heading, perfFileName)
	}

	var named []string
	for _, a := range found {
		if strings.Contains(s.Body, filepath.Base(a.Path)) || strings.Contains(s.Body, a.Path) {
			named = append(named, a.Path)
		}
	}
	if len(named) == 0 {
		var list []string
		for _, a := range found {
			list = append(list, fmt.Sprintf("  %s (%s)", a.Path, a.Kind))
		}
		t.Fatalf("%s:%d: `%s` names none of the profile artifacts committed in the project:\n%s\n\n"+
			"Point at the evidence by path, so a reviewer can open the same file you read.",
			perfFileName, s.Line, s.Heading, strings.Join(list, "\n"))
	}
	t.Logf("evidence: %s", strings.Join(named, " "))
}

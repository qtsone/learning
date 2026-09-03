package capstone

import (
	"strings"
	"testing"
)

// Criterion 1: the suite still passes with the race detector on. It passed in
// the last lesson; it has to keep passing while you add tests that push on the
// ugly paths, which is where races usually surface.
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

// Criterion 2: go vet is clean. Vet is the cheapest static analysis you own
// and several of its checks — lost cancel, unsafe printf, copied locks — are
// security findings wearing a style-checker's coat.
func TestProjectVetsClean(t *testing.T) {
	dir := project(t)
	out, err := runGo(dir, "vet", "./...")
	if err != nil {
		t.Fatalf("go vet ./... reported problems in %s: %v\n\n%s", dir, err, tail(out, 4000))
	}
}

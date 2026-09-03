package capstone

import "testing"

// Criterion 1: the whole module compiles.
func TestProjectBuilds(t *testing.T) {
	dir := project(t)
	out, err := runGo(dir, "build", "./...")
	if err != nil {
		t.Fatalf("go build ./... failed in %s: %v\n\n%s", dir, err, tail(out, 4000))
	}
}

// Criterion 2: go vet is clean. Vet findings are not style opinions; each one
// is a construct that compiles and is still very likely wrong.
func TestProjectVetsClean(t *testing.T) {
	dir := project(t)
	out, err := runGo(dir, "vet", "./...")
	if err != nil {
		t.Fatalf("go vet ./... reported problems in %s: %v\n\n%s", dir, err, tail(out, 4000))
	}
}

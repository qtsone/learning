package capstone

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	minReadmeBytes = 200
	minMilestones  = 3
)

// Criterion 7: a README a stranger can start from.
func TestReadmeExplainsTheProject(t *testing.T) {
	dir := project(t)
	path := filepath.Join(dir, "README.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("no README.md at the project root (%s): %v\n"+
			"Write what the project is, who it is for, and how to run it.", dir, err)
	}
	if len(raw) < minReadmeBytes {
		t.Errorf("README.md is %d bytes, want at least %d — say what it is and how to run it",
			len(raw), minReadmeBytes)
	}
	if !strings.Contains(string(raw), "```") {
		t.Errorf("README.md has no fenced code block — show the exact commands to build and run it")
	}
}

// Criterion 8: MILESTONES.md from the planning lesson, every box ticked.
func TestMilestonesAllTicked(t *testing.T) {
	dir := project(t)
	path := filepath.Join(dir, "MILESTONES.md")
	ticked, unticked, err := checklist(path)
	if err != nil {
		t.Fatalf("no MILESTONES.md at the project root (%s): %v\n"+
			"Copy the plan you wrote in the planning lesson into the project.", dir, err)
	}
	if total := len(ticked) + len(unticked); total < minMilestones {
		t.Errorf("MILESTONES.md has %d checklist item(s) (`- [ ]` / `- [x]`), want at least %d",
			total, minMilestones)
	}
	for _, item := range unticked {
		t.Errorf("milestone not delivered: %s", item)
	}
}

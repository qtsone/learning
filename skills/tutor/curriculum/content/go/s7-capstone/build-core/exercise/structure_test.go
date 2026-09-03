package capstone

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

const (
	minNonMainPackages = 2
	minNonMainShare    = 0.60
)

// Criterion 5: the module has real package structure. main is a composition
// root, not the program.
func TestPackageStructure(t *testing.T) {
	dir := project(t)
	l, err := readLayout(dir)
	if err != nil {
		t.Fatalf("reading package layout of %s: %v", dir, err)
	}
	if l.TotalLines == 0 {
		t.Fatalf("no non-test Go files found in %s", dir)
	}
	if len(l.OtherDirs) < minNonMainPackages {
		t.Errorf("found %d package(s) outside main, want at least %d\n%s",
			len(l.OtherDirs), minNonMainPackages, describe(l))
	}
	if l.OtherShare < minNonMainShare {
		t.Errorf("packages outside main hold %.0f%% of non-test Go lines, want at least %.0f%%\n%s",
			l.OtherShare*100, minNonMainShare*100, describe(l))
	}
}

func describe(l layout) string {
	dirs := make([]string, 0, len(l.PackageName))
	for d := range l.PackageName {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	var b strings.Builder
	b.WriteString("packages found:\n")
	for _, d := range dirs {
		b.WriteString(fmt.Sprintf("  %-40s package %s\n", d, l.PackageName[d]))
	}
	b.WriteString(fmt.Sprintf("  non-test lines: %d total, %d in package main\n",
		l.TotalLines, l.MainLines))
	return b.String()
}

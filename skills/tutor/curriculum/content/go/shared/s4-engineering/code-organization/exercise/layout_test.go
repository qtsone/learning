package main

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const modulePath = "tutor.local/code-organization"

// allowedInternal encodes the project's dependency direction: each package
// directory maps to the module-internal packages it may import. Anything
// else is an arrow pointing the wrong way.
var allowedInternal = map[string][]string{
	"expense": {},
	"ledger":  {"expense"},
	"report":  {},
}

func internalImports(t *testing.T, dir string) map[string][]string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatalf("globbing %s: %v", dir, err)
	}
	byFile := map[string][]string{}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), f, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parsing %s: %v", f, err)
		}
		for _, imp := range parsed.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if strings.HasPrefix(path, modulePath+"/") {
				byFile[f] = append(byFile[f], strings.TrimPrefix(path, modulePath+"/"))
			}
		}
	}
	return byFile
}

func TestDependencyDirection(t *testing.T) {
	for dir, allowed := range allowedInternal {
		for file, imports := range internalImports(t, dir) {
			for _, imp := range imports {
				if !slices.Contains(allowed, imp) {
					t.Errorf("%s imports %s/%s: this dependency points the wrong way — "+
						"package %s may import only %v from this module "+
						"(see LESSON.md on dependency direction)",
						file, modulePath, imp, dir, allowed)
				}
			}
		}
	}
}

func TestMainWiring(t *testing.T) {
	found := map[string]bool{}
	for _, imports := range internalImports(t, ".") {
		for _, imp := range imports {
			found[imp] = true
		}
	}
	for _, want := range []string{"expense", "ledger", "report"} {
		if !found[want] {
			t.Errorf("no file in package main imports %s/%s — after the refactor, "+
				"main should be thin wiring: build expenses with expense.New, "+
				"Add the valid ones to a ledger.Ledger, print report.Summary",
				modulePath, want)
		}
	}
}

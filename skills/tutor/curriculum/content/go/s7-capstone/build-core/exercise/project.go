// Package capstone is the conformance harness for your capstone project.
//
// There is nothing here for you to implement. The code you are graded on
// lives in your own project; these tests find it, run its toolchain, and
// check the engineering properties listed in LESSON.md.
//
// The project directory is resolved in this order:
//
//  1. the TUTOR_CAPSTONE_DIR environment variable,
//  2. a single path on the first line of capstone.path next to this file
//     (relative paths resolve against this directory),
//  3. projects/capstone at your workspace root, found by walking up from this
//     directory — ../../../../projects/capstone in a scaffolded workspace.
package capstone

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	// envVar overrides every other way of locating the project.
	envVar = "TUTOR_CAPSTONE_DIR"
	// pathFile holds one path, used when envVar is unset.
	pathFile = "capstone.path"
	// defaultRel is the fallback as it looks from a scaffolded exercise
	// directory. It is only ever printed: how deep an exercise sits below the
	// workspace root is decided by lesson_workspace_dir() in tutor.py, so
	// findDefaultDir searches for the project instead of counting on it.
	defaultRel = "../../../../projects/capstone"
	// defaultMaxUp bounds that upward search.
	defaultMaxUp = 6

	// commandTimeout bounds every command run inside your project. It is
	// generous on purpose: a cold build of a real project is not fast.
	commandTimeout = 5 * time.Minute
)

// findDefaultDir returns the projects/capstone directory of the workspace this
// exercise was scaffolded into: the nearest ancestor holding
// projects/capstone/go.mod, which is the layout the planning lesson sets up.
func findDefaultDir() (string, error) {
	start, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := start
	for up := 0; up <= defaultMaxUp; up++ {
		if abs, err := checkProject(filepath.Join(dir, "projects", "capstone")); err == nil {
			return abs, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("no projects/capstone containing a go.mod above %s", start)
}

// findProject returns the absolute path of the capstone project, or an error
// that tells the learner exactly how to point the harness at it.
func findProject() (string, error) {
	var tried []string

	if raw := strings.TrimSpace(os.Getenv(envVar)); raw != "" {
		dir, err := checkProject(raw)
		if err == nil {
			return dir, nil
		}
		tried = append(tried, fmt.Sprintf("%s=%s — %v", envVar, raw, err))
	} else {
		tried = append(tried, envVar+" — not set")
	}

	if raw, err := os.ReadFile(pathFile); err == nil {
		line := firstLine(string(raw))
		if line == "" {
			tried = append(tried, pathFile+" — present but empty")
		} else {
			dir, err := checkProject(line)
			if err == nil {
				return dir, nil
			}
			tried = append(tried, fmt.Sprintf("%s → %s — %v", pathFile, line, err))
		}
	} else {
		tried = append(tried, pathFile+" — not present")
	}

	dir, err := findDefaultDir()
	if err == nil {
		return dir, nil
	}
	tried = append(tried, fmt.Sprintf("%s — %v", defaultRel, err))

	return "", fmt.Errorf(`no capstone project found. Tried:
  %s

Point the harness at your project in one of these ways:
  1. put its path on the first line of exercise/capstone.path, e.g.
       echo ../../../../projects/capstone > capstone.path
     then check it: ls "$(cat capstone.path)/go.mod"
  2. export TUTOR_CAPSTONE_DIR=/absolute/path/to/your/project
  3. create the project where the planning lesson put it: projects/capstone at
     your workspace root — the directory holding lessons/ — and go mod init
     there. Do not create one under lessons/: that decoy grades instead of
     your capstone.

A capstone project is a directory containing a go.mod`, strings.Join(tried, "\n  "))
}

func checkProject(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("%s: %w", abs, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s: not a directory", abs)
	}
	if _, err := os.Stat(filepath.Join(abs, "go.mod")); err != nil {
		return "", fmt.Errorf("%s: no go.mod", abs)
	}
	return abs, nil
}

func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// runGo runs a go command inside dir and returns its combined output.
// The environment is pinned so grading never reaches the network: whatever
// your project depends on must already be in the local module cache.
func runGo(dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOPROXY=off", "GOWORK=off")
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return string(out), fmt.Errorf("go %s timed out after %s: %w",
			strings.Join(args, " "), commandTimeout, ctx.Err())
	}
	return string(out), err
}

// sourceFiles lists the .go files of a module, skipping directories the go
// tool itself ignores plus anything hidden.
func sourceFiles(root string, includeTests bool) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if path == root {
				return nil
			}
			switch {
			case name == "testdata", name == "vendor", name == "node_modules":
				return filepath.SkipDir
			case strings.HasPrefix(name, "."), strings.HasPrefix(name, "_"):
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".go") {
			return nil
		}
		if !includeTests && strings.HasSuffix(name, "_test.go") {
			return nil
		}
		files = append(files, path)
		return nil
	})
	sort.Strings(files)
	return files, err
}

// layout summarises the package structure of a module.
type layout struct {
	MainDirs    []string // directories holding package main
	OtherDirs   []string // directories holding any other package
	MainLines   int      // non-test lines in package main
	TotalLines  int      // non-test lines in the whole module
	OtherShare  float64  // share of non-test lines outside package main
	PackageName map[string]string
}

func readLayout(root string) (layout, error) {
	l := layout{PackageName: map[string]string{}}
	files, err := sourceFiles(root, false)
	if err != nil {
		return l, err
	}
	mainDirs := map[string]bool{}
	otherDirs := map[string]bool{}
	fset := token.NewFileSet()
	for _, file := range files {
		f, err := parser.ParseFile(fset, file, nil, parser.PackageClauseOnly)
		if err != nil {
			return l, fmt.Errorf("%s: %w", rel(root, file), err)
		}
		src, err := os.ReadFile(file)
		if err != nil {
			return l, err
		}
		lines := len(strings.Split(strings.TrimRight(string(src), "\n"), "\n"))
		dir := rel(root, filepath.Dir(file))
		l.PackageName[dir] = f.Name.Name
		l.TotalLines += lines
		if f.Name.Name == "main" {
			mainDirs[dir] = true
			l.MainLines += lines
		} else {
			otherDirs[dir] = true
		}
	}
	l.MainDirs = keys(mainDirs)
	l.OtherDirs = keys(otherDirs)
	if l.TotalLines > 0 {
		l.OtherShare = float64(l.TotalLines-l.MainLines) / float64(l.TotalLines)
	}
	return l, nil
}

func keys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func rel(root, path string) string {
	if r, err := filepath.Rel(root, path); err == nil {
		return r
	}
	return path
}

// finding is one hygiene problem, reported with enough location to fix it.
type finding struct {
	File string
	Line int
	Rule string
	Text string
}

func (f finding) String() string {
	return fmt.Sprintf("%s:%d: %s: %s", f.File, f.Line, f.Rule, f.Text)
}

var (
	todoRe       = regexp.MustCompile(`\b(TODO|FIXME)\b`)
	unfinishedRe = regexp.MustCompile(`(?i)not implemented|unimplemented|implement me|todo`)
	directiveRe  = regexp.MustCompile(`^//([a-z][a-z0-9]*:|\s*\+build\b)`)
	markerRe     = regexp.MustCompile(`:=|\{|\}|;|\+\+|--|==|!=|<-`)
	structureRe  = regexp.MustCompile(`:=|[{}]|;|\+\+|--|<-`)
	labelProseRe = regexp.MustCompile(`^[a-z][a-z0-9]*:\s+(\S.*)$`)
	keywordRe    = regexp.MustCompile(`^(if|for|func|return|switch|select|go|defer|var|const|type|case|else|break|continue|range)\b`)
)

// scanHygiene reports leftovers in non-test code: TODO/FIXME markers,
// panics that stand in for missing implementations, and commented-out code.
func scanHygiene(root string) ([]finding, error) {
	files, err := sourceFiles(root, false)
	if err != nil {
		return nil, err
	}
	var out []finding
	fset := token.NewFileSet()
	for _, file := range files {
		f, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", rel(root, file), err)
		}
		name := rel(root, file)
		docs := docComments(f)

		for _, group := range f.Comments {
			for _, c := range group.List {
				if m := todoRe.FindString(c.Text); m != "" {
					out = append(out, finding{name, fset.Position(c.Pos()).Line,
						"leftover marker", m + " in " + oneLine(c.Text)})
				}
			}
			if docs[group] {
				continue
			}
			body := commentBody(group)
			if looksLikeCode(body) {
				out = append(out, finding{name, fset.Position(group.Pos()).Line,
					"commented-out code", oneLine(group.List[0].Text)})
			}
		}

		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			id, ok := call.Fun.(*ast.Ident)
			if !ok || id.Name != "panic" || len(call.Args) == 0 {
				return true
			}
			lit, ok := call.Args[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			text, err := strconv.Unquote(lit.Value)
			if err != nil {
				return true
			}
			if unfinishedRe.MatchString(text) {
				out = append(out, finding{name, fset.Position(call.Pos()).Line,
					"stub panic", fmt.Sprintf("panic(%q)", text)})
			}
			return true
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	return out, nil
}

// docComments collects the comment groups attached to a declaration. Doc
// comments are exempt from the commented-out-code rule: a documented example
// is meant to look like code.
func docComments(f *ast.File) map[*ast.CommentGroup]bool {
	set := map[*ast.CommentGroup]bool{}
	add := func(groups ...*ast.CommentGroup) {
		for _, g := range groups {
			if g != nil {
				set[g] = true
			}
		}
	}
	add(f.Doc)
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.GenDecl:
			add(x.Doc)
		case *ast.FuncDecl:
			add(x.Doc)
		case *ast.TypeSpec:
			add(x.Doc, x.Comment)
		case *ast.ValueSpec:
			add(x.Doc, x.Comment)
		case *ast.ImportSpec:
			add(x.Doc, x.Comment)
		case *ast.Field:
			add(x.Doc, x.Comment)
		}
		return true
	})
	return set
}

func commentBody(group *ast.CommentGroup) string {
	var b strings.Builder
	for _, c := range group.List {
		text := c.Text
		switch {
		case strings.HasPrefix(text, "//"):
			if directiveRe.MatchString(text) {
				return ""
			}
			text = text[2:]
		case strings.HasPrefix(text, "/*"):
			text = strings.TrimSuffix(strings.TrimPrefix(text, "/*"), "*/")
		}
		b.WriteString(strings.TrimPrefix(text, " "))
		b.WriteString("\n")
	}
	return b.String()
}

// looksLikeCode reports whether a comment body is commented-out Go rather than
// prose. Parsing alone is far too weak a test: Go accepts a bare expression as
// a statement and any `word:` prefix as a label, so "invariant: len(x) == len(y)"
// — exactly the kind of why-comment this stage asks you to write — parses
// perfectly. A false positive here tells you to delete a sentence that explains
// something, so the bar is three gates: the body must not open with a prose
// label, it must carry a code marker, and either it spans more than one line or
// its single line has real structure (a leading keyword, `:=`, a brace, a
// semicolon, `++`/`--`, `<-`). Only then does it have to parse.
func looksLikeCode(body string) bool {
	var lines []string
	for _, line := range strings.Split(body, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) == 0 || labelledProse(lines[0]) {
		return false
	}
	hinted := false
	for _, line := range lines {
		if markerRe.MatchString(line) || keywordRe.MatchString(line) {
			hinted = true
			break
		}
	}
	if !hinted {
		return false
	}
	if len(lines) == 1 && !structureRe.MatchString(lines[0]) && !keywordRe.MatchString(lines[0]) {
		return false
	}
	fset := token.NewFileSet()
	if f, err := parser.ParseFile(fset, "c.go", "package p\nfunc _() {\n"+body+"\n}\n", 0); err == nil {
		if len(f.Decls) == 1 {
			if fn, ok := f.Decls[0].(*ast.FuncDecl); ok && fn.Body != nil && len(fn.Body.List) > 0 {
				return true
			}
		}
	}
	if f, err := parser.ParseFile(fset, "c.go", "package p\n"+body+"\n", 0); err == nil && len(f.Decls) > 0 {
		return true
	}
	return false
}

// labelledProse reports whether a line is a sentence introduced by a label,
// like "invariant: len(x) == len(y)". Go writes a real label on a line of its
// own, so a label sharing its line with anything that is not the start of a
// statement is prose, whatever the parser makes of it.
func labelledProse(line string) bool {
	m := labelProseRe.FindStringSubmatch(line)
	if m == nil {
		return false
	}
	rest := m[1]
	return !structureRe.MatchString(rest) && !keywordRe.MatchString(rest)
}

func oneLine(s string) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if len(s) > 72 {
		s = s[:69] + "..."
	}
	return s
}

var (
	checklistRe  = regexp.MustCompile(`^\s*[-*] \[([ xX])\]\s*(.*)$`)
	coverTotalRe = regexp.MustCompile(`total:\s+\(statements\)\s+([0-9.]+)%`)
)

// checklist returns the ticked and unticked items of a markdown checklist.
func checklist(path string) (ticked []string, unticked []string, err error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	for _, line := range strings.Split(string(raw), "\n") {
		m := checklistRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		item := strings.TrimSpace(m[2])
		if m[1] == " " {
			unticked = append(unticked, item)
		} else {
			ticked = append(ticked, item)
		}
	}
	return ticked, unticked, nil
}

func coverageTotal(funcOutput string) (float64, error) {
	m := coverTotalRe.FindStringSubmatch(funcOutput)
	if m == nil {
		return 0, fmt.Errorf("no total line in coverage report:\n%s", funcOutput)
	}
	return strconv.ParseFloat(m[1], 64)
}

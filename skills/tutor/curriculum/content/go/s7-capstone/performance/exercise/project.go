// Package capstone is the conformance harness for your capstone project.
//
// There is nothing here for you to implement. The work you are graded on
// happened in your own project: a bottleneck you found, a change you measured,
// and PERF.md — the claim a reviewer can check without taking your word for it.
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

	// perfFileName is this lesson's deliverable, at the project root.
	perfFileName = "PERF.md"

	// commandTimeout bounds every command run inside your project. It is
	// generous on purpose: a cold build of a real project is not fast.
	commandTimeout = 5 * time.Minute

	// maxTextProfileBytes caps the files read looking for pprof output.
	maxTextProfileBytes = 1 << 20
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

// walkProject visits every regular file under root, skipping the directories
// the go tool and reviewers both ignore.
func walkProject(root string, visit func(path string, d os.DirEntry) error) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
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
		if !d.Type().IsRegular() {
			return nil
		}
		return visit(path, d)
	})
}

func rel(root, path string) string {
	if r, err := filepath.Rel(root, path); err == nil {
		return r
	}
	return path
}

func testFiles(root string) ([]string, error) {
	var files []string
	err := walkProject(root, func(path string, d os.DirEntry) error {
		if strings.HasSuffix(d.Name(), "_test.go") {
			files = append(files, path)
		}
		return nil
	})
	sort.Strings(files)
	return files, err
}

// benchmark is one committed Benchmark function.
type benchmark struct {
	File   string
	Name   string
	Allocs bool // calls b.ReportAllocs()
}

// benchmarks finds every func BenchmarkXxx(b *testing.B) in the module.
func benchmarks(root string) ([]benchmark, error) {
	files, err := testFiles(root)
	if err != nil {
		return nil, err
	}
	var out []benchmark
	fset := token.NewFileSet()
	for _, file := range files {
		f, err := parser.ParseFile(fset, file, nil, parser.SkipObjectResolution)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", rel(root, file), err)
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Body == nil {
				continue
			}
			if !strings.HasPrefix(fn.Name.Name, "Benchmark") || !takesTestingB(fn) {
				continue
			}
			out = append(out, benchmark{
				File:   rel(root, file),
				Name:   fn.Name.Name,
				Allocs: callsMethod(fn, "ReportAllocs"),
			})
		}
	}
	return out, nil
}

func takesTestingB(fn *ast.FuncDecl) bool {
	params := fn.Type.Params
	if params == nil || len(params.List) != 1 {
		return false
	}
	star, ok := params.List[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "B"
}

// callsMethod reports whether fn (including any closure inside it) calls a
// method with the given name.
func callsMethod(fn *ast.FuncDecl, name string) bool {
	found := false
	ast.Inspect(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == name {
			found = true
			return false
		}
		return true
	})
	return found
}

// section is one markdown heading and everything under it, up to the next
// heading of the same or a higher level — so `### Before` stays inside
// `## Result`.
type section struct {
	Heading string
	Level   int
	Line    int
	Body    string
}

var (
	headingRe = regexp.MustCompile(`^(#{1,6})\s+(.+?)\s*#*$`)
	fenceRe   = regexp.MustCompile("^\\s*(```|~~~)")
)

func sections(doc string) []section {
	lines := strings.Split(doc, "\n")
	type head struct {
		idx   int
		level int
		text  string
	}
	var heads []head
	inFence := false
	for i, line := range lines {
		if fenceRe.MatchString(line) {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if m := headingRe.FindStringSubmatch(line); m != nil {
			heads = append(heads, head{i, len(m[1]), strings.TrimSpace(m[2])})
		}
	}
	out := make([]section, 0, len(heads))
	for i, h := range heads {
		end := len(lines)
		for j := i + 1; j < len(heads); j++ {
			if heads[j].level <= h.level {
				end = heads[j].idx
				break
			}
		}
		out = append(out, section{
			Heading: h.text,
			Level:   h.level,
			Line:    h.idx + 1,
			Body:    strings.TrimSpace(strings.Join(lines[h.idx+1:end], "\n")),
		})
	}
	return out
}

func findSection(secs []section, match *regexp.Regexp) (section, bool) {
	for _, s := range secs {
		if match.MatchString(strings.ToLower(s.Heading)) {
			return s, true
		}
	}
	return section{}, false
}

// requirement is one section PERF.md must contain. Heading is what the
// template uses; Match is deliberately looser so your own wording survives.
type requirement struct {
	Key     string
	Heading string
	Match   *regexp.Regexp
	Wants   string
}

var requiredSections = []requirement{
	{"question", "## Question",
		regexp.MustCompile(`^(the )?(question|goal|problem)\b`),
		"the operation, the input size, the number today and the number you wanted"},
	{"method", "## Method",
		regexp.MustCompile(`^(measurement |how i )?(method|measured|measurement)\b`),
		"how you measured: benchmark, inputs, -count, machine, and what you kept quiet"},
	{"evidence", "## Evidence",
		regexp.MustCompile(`^(profile )?(evidence|profiles?)\b`),
		"the profile that named the dominant cost, and the artifact a reader can open"},
	{"change", "## Change",
		regexp.MustCompile(`^(the )?(change|optimi[sz]ation|fix)\b`),
		"what you changed, in one paragraph, and why it should have helped"},
	{"result", "## Result",
		regexp.MustCompile(`^(results?|before|numbers)\b`),
		"before and after numbers with units, from repeated runs"},
	{"tradeoff", "## Trade-off",
		regexp.MustCompile(`^(trade[- ]?offs?|costs?)\b`),
		"what the speed cost you, and why that price is acceptable here"},
	{"correctness", "## Correctness",
		regexp.MustCompile(`^correctness\b`),
		"the named tests proving behaviour did not change"},
}

var (
	placeholderRe = regexp.MustCompile(`(?mi)^\s*(TODO|FIXME)\b`)
	numberRe      = regexp.MustCompile(`\d+(?:[.,]\d+)?`)
	unitRe        = regexp.MustCompile(`(?i)(ns|µs|us|ms|s|b|kb|mb|gb|kib|mib|gib|allocs)/op|µs|\b(ns|ms|us|sec|secs|seconds?|bytes?|kb|mb|gb|kib|mib|gib|rps|qps|allocs?)\b|req/s|ops/s|%`)
	testNameRe    = regexp.MustCompile(`\bTest[A-Z][A-Za-z0-9_]*`)
	testPassRe    = regexp.MustCompile(`(?m)^\s*--- PASS: (\w+)`)

	// benchResultRe matches a benchmark result line: name, iterations, then
	// whatever the benchmark reported.
	benchResultRe = regexp.MustCompile(`(?m)^(Benchmark\S*)\s+(\d+)\s+`)
)

// testNames returns the unique TestXxx identifiers named in a block of text.
func testNames(body string) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range testNameRe.FindAllString(body, -1) {
		if !seen[m] {
			seen[m] = true
			out = append(out, m)
		}
	}
	sort.Strings(out)
	return out
}

// profileArtifact is committed profile evidence: a pprof file, or a text file
// holding pprof-derived output someone else can read.
type profileArtifact struct {
	Path string
	Kind string
}

var (
	profileFileRe = regexp.MustCompile(`(?i)\.(pprof|prof)$|\.pb\.gz$|^(cpu|mem|heap|alloc|block|mutex|goroutine|trace)\.out$`)
	pprofTextRe   = regexp.MustCompile(`(?m)^Showing nodes accounting for|^\s*flat\s+flat%\s+sum%\s+cum\s+cum%|^ROUTINE ={4,}|^Type:\s+(cpu|alloc_space|alloc_objects|inuse_space|inuse_objects|goroutine|block|mutex|delay|contentions)\b|^Duration:\s+.*Total samples`)

	textProfileExt = map[string]bool{".txt": true, ".md": true, ".log": true, ".out": true}
)

// profileArtifacts finds the profile evidence committed in the project. skip
// holds project-relative paths to ignore (PERF.md grades itself elsewhere).
func profileArtifacts(root string, skip map[string]bool) ([]profileArtifact, error) {
	var out []profileArtifact
	err := walkProject(root, func(path string, d os.DirEntry) error {
		name := d.Name()
		relPath := rel(root, path)
		if skip[relPath] {
			return nil
		}
		if profileFileRe.MatchString(name) {
			out = append(out, profileArtifact{relPath, "pprof file"})
			return nil
		}
		if !textProfileExt[strings.ToLower(filepath.Ext(name))] {
			return nil
		}
		if info, err := d.Info(); err != nil || info.Size() > maxTextProfileBytes {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if pprofTextRe.Match(raw) {
			out = append(out, profileArtifact{relPath, "profile summary"})
		}
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, err
}

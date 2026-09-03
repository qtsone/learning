// Package capstone is the conformance harness for the hardening pass over
// your capstone project.
//
// There is nothing here for you to implement. The code you are graded on
// lives in your own project; these tests find it, run its toolchain, and
// inspect its source and its documents for the properties LESSON.md describes.
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
	// generous on purpose: a cold race-detector build is not fast.
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

// fileKind selects which half of a module's Go files a scan looks at. The
// distinction matters constantly here: hostile inputs belong in test files,
// and unbounded HTTP clients do not belong in the other half.
type fileKind int

const (
	nonTest fileKind = iota
	testOnly
	anyFile
)

// sourceFiles lists the .go files of a module, skipping directories the go
// tool itself ignores plus anything hidden.
func sourceFiles(root string, kind fileKind) ([]string, error) {
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
		isTest := strings.HasSuffix(name, "_test.go")
		if (kind == nonTest && isTest) || (kind == testOnly && !isTest) {
			return nil
		}
		files = append(files, path)
		return nil
	})
	sort.Strings(files)
	return files, err
}

// parsedFile carries everything a scanner needs about one file: the AST, the
// position information, and a path short enough to print.
type parsedFile struct {
	Name string // path relative to the project root
	Path string // absolute path
	Fset *token.FileSet
	File *ast.File
}

func (p parsedFile) line(n ast.Node) int { return p.Fset.Position(n.Pos()).Line }

func parseFiles(root string, kind fileKind) ([]parsedFile, error) {
	paths, err := sourceFiles(root, kind)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	out := make([]parsedFile, 0, len(paths))
	for _, path := range paths {
		f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", rel(root, path), err)
		}
		out = append(out, parsedFile{rel(root, path), path, fset, f})
	}
	return out, nil
}

func rel(root, path string) string {
	if r, err := filepath.Rel(root, path); err == nil {
		return r
	}
	return path
}

// finding is one problem, reported with enough location to fix it.
type finding struct {
	File string
	Line int
	Rule string
	Text string
}

func (f finding) String() string {
	return fmt.Sprintf("%s:%d: %s: %s", f.File, f.Line, f.Rule, f.Text)
}

func sortFindings(f []finding) {
	sort.Slice(f, func(i, j int) bool {
		if f[i].File != f[j].File {
			return f[i].File < f[j].File
		}
		return f[i].Line < f[j].Line
	})
}

// importName returns the name a file uses for an imported path, and whether
// the file imports it at all. Blank and dot imports are reported as absent:
// nothing in this harness can say anything useful about them.
func importName(f *ast.File, path string) (string, bool) {
	for _, spec := range f.Imports {
		p, err := strconv.Unquote(spec.Path.Value)
		if err != nil || p != path {
			continue
		}
		if spec.Name != nil {
			if spec.Name.Name == "_" || spec.Name.Name == "." {
				return "", false
			}
			return spec.Name.Name, true
		}
		return path[strings.LastIndex(path, "/")+1:], true
	}
	return "", false
}

// selectorIs reports whether expr is exactly pkg.name.
func selectorIs(expr ast.Expr, pkg, name string) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != name {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	return ok && id.Name == pkg
}

// ---------------------------------------------------------------- HTTP clients

// unboundedHTTPCalls are the package-level helpers that use
// http.DefaultClient, which has no timeout at all.
var unboundedHTTPCalls = []string{"Get", "Post", "PostForm", "Head"}

// scanHTTPClients reports outbound HTTP clients built without a timeout in
// non-test code. A client with no Timeout waits forever for a peer that never
// answers, and forever is the one deadline you never meant to pick.
func scanHTTPClients(root string) ([]finding, error) {
	files, err := parseFiles(root, nonTest)
	if err != nil {
		return nil, err
	}
	var out []finding
	for _, pf := range files {
		httpPkg, ok := importName(pf.File, "net/http")
		if !ok {
			continue
		}
		ast.Inspect(pf.File, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.CompositeLit:
				if !selectorIs(x.Type, httpPkg, "Client") {
					return true
				}
				if !hasField(x, "Timeout") {
					out = append(out, finding{pf.Name, pf.line(x), "client without timeout",
						"http.Client literal has no Timeout field — one hung peer blocks this caller forever"})
				}
			case *ast.SelectorExpr:
				if selectorIs(x, httpPkg, "DefaultClient") {
					out = append(out, finding{pf.Name, pf.line(x), "default client",
						"http.DefaultClient has no timeout — construct your own client"})
				}
			case *ast.CallExpr:
				for _, name := range unboundedHTTPCalls {
					if selectorIs(x.Fun, httpPkg, name) {
						out = append(out, finding{pf.Name, pf.line(x), "default client",
							fmt.Sprintf("http.%s uses http.DefaultClient, which has no timeout", name)})
					}
				}
				if id, ok := x.Fun.(*ast.Ident); ok && id.Name == "new" && len(x.Args) == 1 &&
					selectorIs(x.Args[0], httpPkg, "Client") {
					out = append(out, finding{pf.Name, pf.line(x), "client without timeout",
						"new(http.Client) has a zero Timeout, which means no timeout"})
				}
			}
			return true
		})
	}
	sortFindings(out)
	return out, nil
}

func hasField(lit *ast.CompositeLit, name string) bool {
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		if id, ok := kv.Key.(*ast.Ident); ok && id.Name == name {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------- context use

// scanContextUse reports two things in non-test code: any context.TODO(),
// which is an undecided decision left in shipped code, and any
// context.Background() inside a function that was handed a context.Context —
// which throws away the caller's deadline and cancellation.
func scanContextUse(root string) ([]finding, error) {
	files, err := parseFiles(root, nonTest)
	if err != nil {
		return nil, err
	}
	var out []finding
	for _, pf := range files {
		ctxPkg, ok := importName(pf.File, "context")
		if !ok {
			continue
		}
		ast.Inspect(pf.File, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if ok && selectorIs(call.Fun, ctxPkg, "TODO") {
				out = append(out, finding{pf.Name, pf.line(call), "undecided context",
					"context.TODO() means \"I have not decided yet\" — decide"})
			}
			return true
		})
		for _, decl := range pf.File.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || !takesContext(fn.Type, ctxPkg) {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if ok && selectorIs(call.Fun, ctxPkg, "Background") {
					out = append(out, finding{pf.Name, pf.line(call), "detached context",
						fmt.Sprintf("%s receives a context.Context but starts a fresh root here — "+
							"the caller's deadline and cancellation are discarded", fn.Name.Name)})
				}
				return true
			})
		}
	}
	sortFindings(out)
	return out, nil
}

func takesContext(ft *ast.FuncType, ctxPkg string) bool {
	if ft.Params == nil {
		return false
	}
	for _, p := range ft.Params.List {
		if selectorIs(p.Type, ctxPkg, "Context") {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------- fuzz targets

// fuzzTarget is one FuzzXxx function and the seed corpus committed for it.
type fuzzTarget struct {
	Name  string
	File  string
	Seeds []string
}

func fuzzTargets(root string) ([]fuzzTarget, error) {
	files, err := parseFiles(root, testOnly)
	if err != nil {
		return nil, err
	}
	var out []fuzzTarget
	for _, pf := range files {
		for _, decl := range pf.File.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || !strings.HasPrefix(fn.Name.Name, "Fuzz") {
				continue
			}
			if !takesTestingF(fn.Type) {
				continue
			}
			dir := filepath.Dir(pf.Path)
			seeds, err := corpusFiles(filepath.Join(dir, "testdata", "fuzz", fn.Name.Name))
			if err != nil {
				return nil, err
			}
			out = append(out, fuzzTarget{fn.Name.Name, pf.Name, seeds})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func takesTestingF(ft *ast.FuncType) bool {
	if ft.Params == nil || len(ft.Params.List) != 1 {
		return false
	}
	star, ok := ft.Params.List[0].Type.(*ast.StarExpr)
	return ok && selectorIs(star.X, "testing", "F")
}

// corpusFiles lists the committed corpus entries of one fuzz target. A
// missing directory is not an error — it is the finding.
func corpusFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		out = append(out, e.Name())
	}
	sort.Strings(out)
	return out, nil
}

// ---------------------------------------------------------------- test suite

var (
	errIdentRe  = regexp.MustCompile(`(?i)err`)
	wantErrRe   = regexp.MustCompile(`(?i)^(want|expect|expected|should)err(or)?s?$`)
	testFailure = map[string]bool{"Error": true, "Errorf": true, "Fatal": true, "Fatalf": true}
	testHelper  = map[string]bool{"t": true, "tb": true, "b": true, "f": true}
	hostileRe   = regexp.MustCompile(`(?i)hostile|malicious|malformed|invalid|adversar|` +
		`injection|traversal|oversiz|toolarge|toolong|truncat|corrupt|garbage|` +
		`untrusted|overflow|denial|abuse|nulbyte|unicode`)
)

// errorAssertions counts, per test file, the places where a test asserts on a
// failure rather than on a success: errors.Is/As calls, `if err == nil` guards
// that fail the test, and table-test fields named wantErr and friends.
func errorAssertions(root string) (map[string]int, error) {
	files, err := parseFiles(root, testOnly)
	if err != nil {
		return nil, err
	}
	counts := map[string]int{}
	for _, pf := range files {
		errPkg, hasErrors := importName(pf.File, "errors")
		ast.Inspect(pf.File, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.CallExpr:
				if hasErrors && (selectorIs(x.Fun, errPkg, "Is") || selectorIs(x.Fun, errPkg, "As")) {
					counts[pf.Name]++
				}
			case *ast.IfStmt:
				if expectsError(x) {
					counts[pf.Name]++
				}
			case *ast.Field:
				for _, name := range x.Names {
					if wantErrRe.MatchString(name.Name) {
						counts[pf.Name]++
					}
				}
			}
			return true
		})
	}
	return counts, nil
}

// expectsError reports whether an if statement is the shape "we asked for a
// failure and did not get one": `if err == nil { t.Fatalf(…) }`.
func expectsError(stmt *ast.IfStmt) bool {
	bin, ok := stmt.Cond.(*ast.BinaryExpr)
	if !ok || bin.Op != token.EQL {
		return false
	}
	rhs, ok := bin.Y.(*ast.Ident)
	if !ok || rhs.Name != "nil" {
		return false
	}
	lhs, ok := bin.X.(*ast.Ident)
	if !ok || !errIdentRe.MatchString(lhs.Name) {
		return false
	}
	return failsTest(stmt.Body)
}

// failsTest reports whether a block calls t.Error/t.Fatal (or the same on a
// testing.B/F receiver). The receiver check keeps err.Error() out of the count.
func failsTest(n ast.Node) bool {
	found := false
	ast.Inspect(n, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !testFailure[sel.Sel.Name] {
			return true
		}
		if id, ok := sel.X.(*ast.Ident); ok && testHelper[id.Name] {
			found = true
			return false
		}
		return true
	})
	return found
}

// testFunctions maps every Test/Fuzz/Example function name in the project to
// the file and line that declares it.
func testFunctions(root string) (map[string]string, error) {
	files, err := parseFiles(root, testOnly)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, pf := range files {
		for _, decl := range pf.File.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil {
				continue
			}
			name := fn.Name.Name
			if strings.HasPrefix(name, "Test") || strings.HasPrefix(name, "Fuzz") ||
				strings.HasPrefix(name, "Example") || strings.HasPrefix(name, "Benchmark") {
				out[name] = fmt.Sprintf("%s:%d", pf.Name, pf.line(fn))
			}
		}
	}
	return out, nil
}

// ------------------------------------------------------- untrusted input

// untrustedSurfaces are standard-library entry points through which bytes you
// did not write reach your program. Import path → selectors, or nil for "any
// use of this package counts".
var untrustedSurfaces = map[string][]string{
	"os":            {"Args", "Stdin", "Open", "OpenFile", "ReadFile"},
	"flag":          nil,
	"net/http":      nil,
	"net":           {"Listen", "ListenPacket", "Dial", "DialTimeout"},
	"database/sql":  nil,
	"bufio":         {"NewScanner", "NewReader"},
	"encoding/json": {"Unmarshal", "NewDecoder"},
	"encoding/xml":  {"Unmarshal", "NewDecoder"},
	"encoding/csv":  {"NewReader"},
	"io":            {"ReadAll"},
}

// findUntrustedSurfaces reports where the project accepts input from outside
// itself. An empty result means the hostile-input criterion does not apply.
func findUntrustedSurfaces(root string) ([]finding, error) {
	files, err := parseFiles(root, nonTest)
	if err != nil {
		return nil, err
	}
	var out []finding
	for _, pf := range files {
		for path, selectors := range untrustedSurfaces {
			pkg, ok := importName(pf.File, path)
			if !ok {
				continue
			}
			ast.Inspect(pf.File, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				id, ok := sel.X.(*ast.Ident)
				if !ok || id.Name != pkg {
					return true
				}
				if selectors != nil && !contains(selectors, sel.Sel.Name) {
					return true
				}
				out = append(out, finding{pf.Name, pf.line(sel), "input surface",
					pkg + "." + sel.Sel.Name})
				return true
			})
		}
	}
	sortFindings(out)
	return out, nil
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------- documents

// mdSection is one markdown heading and everything under it, up to the next
// heading of the same or a higher level — so `### Command line` and its prose
// stay inside `## Inputs and validation` rather than ending it.
type mdSection struct {
	Heading string
	Body    string
	Line    int
}

var headingRe = regexp.MustCompile(`^ {0,3}(#{1,6})\s+(.*?)\s*#*\s*$`)

// markdownSections splits a document into its headed sections. Fenced code
// blocks are skipped so a shell comment never reads as a heading.
func markdownSections(raw string) []mdSection {
	lines := strings.Split(raw, "\n")
	type heading struct {
		idx   int
		level int
		text  string
	}
	var heads []heading
	fenced := false
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			fenced = !fenced
			continue
		}
		if fenced {
			continue
		}
		if m := headingRe.FindStringSubmatch(line); m != nil {
			heads = append(heads, heading{i, len(m[1]), m[2]})
		}
	}
	out := make([]mdSection, 0, len(heads))
	for i, h := range heads {
		end := len(lines)
		for j := i + 1; j < len(heads); j++ {
			if heads[j].level <= h.level {
				end = heads[j].idx
				break
			}
		}
		out = append(out, mdSection{
			Heading: h.text,
			Body:    strings.TrimSpace(strings.Join(lines[h.idx+1:end], "\n")),
			Line:    h.idx + 1,
		})
	}
	return out
}

// findSection returns the first section whose heading matches.
func findSection(sections []mdSection, re *regexp.Regexp) (mdSection, bool) {
	for _, s := range sections {
		if re.MatchString(s.Heading) {
			return s, true
		}
	}
	return mdSection{}, false
}

// docTopic is one section the security document has to carry: the name the
// harness reports it by, the heading it suggests, the pattern that recognises
// a heading of yours, and what belongs underneath.
type docTopic struct {
	Name    string
	Heading string
	Match   *regexp.Regexp
	Wants   string
}

var securityTopics = []docTopic{
	{
		"trust boundaries", "Trust boundaries", regexp.MustCompile(`(?i)trust\s*boundar`),
		"every place data crosses from something you do not control into something you do, and what is checked at the crossing",
	},
	{
		"inputs", "Inputs and validation", regexp.MustCompile(`(?i)\binputs?\b|untrusted`),
		"each input surface — arguments, files, stdin, network, environment — with the validation it gets and the limits it enforces",
	},
	{
		"secrets handling", "Secrets handling", regexp.MustCompile(`(?i)\bsecrets?\b|credential`),
		"what counts as a secret here, where it comes from, and what stops it reaching logs, error strings or the repository",
	},
	{
		"dependency policy", "Dependency policy", regexp.MustCompile(`(?i)\bdependenc|third[- ]party`),
		"the rule you apply before adding a module, and the govulncheck run you read (name the tool and what it said)",
	},
	{
		"findings and fixes", "Findings and fixes", regexp.MustCompile(`(?i)\bfinding|\bfixe?s\b|\bdefect`),
		"what this pass turned up and what you did about it, each with a `Regression test: TestName` line",
	},
	{
		"accepted risks", "Accepted risks", regexp.MustCompile(`(?i)\baccept`),
		"what you deliberately did not harden, the risk you are carrying, and what would change your mind",
	},
}

// securityDocNames are the places the harness looks for the document, in order.
var securityDocNames = []string{
	"SECURITY.md",
	"THREAT-MODEL.md",
	filepath.Join("docs", "SECURITY.md"),
	filepath.Join("docs", "THREAT-MODEL.md"),
}

func findSecurityDoc(root string) (name string, raw string, err error) {
	for _, candidate := range securityDocNames {
		data, err := os.ReadFile(filepath.Join(root, candidate))
		if err == nil {
			return candidate, string(data), nil
		}
	}
	return "", "", fmt.Errorf("no security document found — looked for %s",
		strings.Join(securityDocNames, ", "))
}

var regressionRe = regexp.MustCompile(`(?i)regression tests?\s*:\s*(.+)`)
var identifierRe = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`)

// regressionTestNames returns the test names claimed by "Regression test:"
// lines in the security document.
func regressionTestNames(raw string) []string {
	var out []string
	seen := map[string]bool{}
	for _, m := range regressionRe.FindAllStringSubmatch(raw, -1) {
		for _, name := range identifierRe.FindAllString(m[1], -1) {
			if !strings.HasPrefix(name, "Test") && !strings.HasPrefix(name, "Fuzz") &&
				!strings.HasPrefix(name, "Example") {
				continue
			}
			if !seen[name] {
				seen[name] = true
				out = append(out, name)
			}
		}
	}
	sort.Strings(out)
	return out
}

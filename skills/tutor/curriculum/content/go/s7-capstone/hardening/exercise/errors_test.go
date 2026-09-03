package capstone

import (
	"fmt"
	"sort"
	"testing"
)

const (
	minErrorAssertions = 4
	minErrorFiles      = 2
)

// Criterion 4: the suite tests failure, not only success. A suite that only
// walks happy paths proves the code works when nothing goes wrong, which is
// the case nobody was worried about.
func TestErrorPathsAreTested(t *testing.T) {
	dir := project(t)
	counts, err := errorAssertions(dir)
	if err != nil {
		t.Fatalf("scanning %s for error assertions: %v", dir, err)
	}

	total := 0
	lines := make([]string, 0, len(counts))
	for file, n := range counts {
		total += n
		lines = append(lines, fmt.Sprintf("%-48s %d", file, n))
	}
	sort.Strings(lines)

	if total >= minErrorAssertions && len(counts) >= minErrorFiles {
		t.Logf("error-path assertions:\n%s", bullets(lines, 20))
		return
	}

	found := "none found"
	if len(lines) > 0 {
		found = "found:\n" + bullets(lines, 20)
	}
	t.Fatalf(`the suite asserts on %d error path(s) across %d test file(s); want at least %d across %d.

%s

The harness counts three shapes, all in _test.go files:
  errors.Is(err, ErrSomething) or errors.As(err, &target)
  if err == nil { t.Fatalf(...) }          — we asked for a failure
  a table-test field named wantErr / expectErr

For every function that can fail, there is an input that makes it fail. Write
that input down: the wrong type, the missing field, the closed file, the
cancelled context, the duplicate key. Assert on the sentinel with errors.Is,
not on the message text — messages are prose and prose changes.`,
		total, len(counts), minErrorAssertions, minErrorFiles, found)
}

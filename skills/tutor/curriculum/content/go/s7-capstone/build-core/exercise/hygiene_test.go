package capstone

import "testing"

// The commented-out-code rule has to leave prose alone, because the comments
// this stage asks you to write are exactly the ones a naive parser mistakes
// for code. These cases pin both directions of that judgement.
func TestLooksLikeCode(t *testing.T) {
	cases := []struct {
		name string
		body string
		want bool
	}{
		{"labelled invariant", "invariant: len(x) == len(y)\n", false},
		{"labelled fast path", "fast path: n == 0\n", false},
		{"labelled note", "note: n != 0\n", false},
		{"prose about equality", "we return early when n == 0\n", false},
		{"prose with semicolon", "callers hold mu; see Add\n", false},
		{"plain sentence", "the zero value is ready to use\n", false},
		{"short assignment", "x := compute(y)\n", true},
		{"bare return", "return nil\n", true},
		{"block", "if v > 0 {\nreturn v\n}\n", true},
		{"labelled loop", "retry: for i := 0; i < 3; i++ {\nsend(i)\n}\n", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := looksLikeCode(c.body); got != c.want {
				t.Errorf("looksLikeCode(%q) = %v, want %v", c.body, got, c.want)
			}
		})
	}
}

// Criterion 6: no unfinished work left in non-test code — no TODO/FIXME
// markers, no panics standing in for an implementation, no commented-out
// code. Test files are exempt; so are doc comments, which are allowed to
// contain example code.
func TestNoUnfinishedWork(t *testing.T) {
	dir := project(t)
	findings, err := scanHygiene(dir)
	if err != nil {
		t.Fatalf("scanning %s: %v", dir, err)
	}
	if len(findings) == 0 {
		return
	}
	for _, f := range findings {
		t.Errorf("%s", f)
	}
	t.Logf("%d leftover(s). Finish it, delete it, or move it to the backlog — "+
		"a TODO in shipped code is a decision nobody made.", len(findings))
}

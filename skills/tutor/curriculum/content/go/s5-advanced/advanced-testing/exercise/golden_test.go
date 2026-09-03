package logkit

import "testing"

// TODO: declare the -update flag here, at package level, so that
// `go test -run TestRunGoldenReport -update .` rewrites the golden files
// instead of comparing against them. It belongs in a _test.go file: only the
// test binary should ever grow that flag.

// assertGolden compares got against the bytes stored in
// testdata/golden/<name>.
//
// With -update it writes got to that file (creating the directory if needed),
// logs what it did, and returns without asserting. Without -update it fails
// the test if the file is missing or its contents differ, and the failure
// message must show both sides — a golden test whose diff you cannot read is
// worse than no test.
func assertGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	// TODO: implement per the acceptance criteria in LESSON.md.
	t.Fatalf("assertGolden is not implemented (golden file %q, %d bytes of output)", name, len(got))
}

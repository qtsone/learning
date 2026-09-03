package logkit

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

// updateGolden lives in a _test.go file on purpose: only the test binary
// should ever grow a -update flag.
var updateGolden = flag.Bool("update", false, "rewrite the golden files in testdata/golden")

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
	path := filepath.Join("testdata", "golden", name)

	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create golden directory: %v", err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("update %s: %v", path, err)
		}
		t.Logf("updated %s (%d bytes) — read the diff before committing it", path, len(got))
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v (run `go test -update .` to create it)", path, err)
	}
	if string(got) != string(want) {
		t.Errorf("output does not match %s\n--- want ---\n%s\n--- got ---\n%s", path, want, got)
	}
}

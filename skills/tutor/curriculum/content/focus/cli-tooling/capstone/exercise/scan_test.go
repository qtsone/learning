package main

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"reflect"
	"testing"
)

func TestScanTotalsAndBuckets(t *testing.T) {
	dir := sampleTree(t)

	res, err := Scan(context.Background(), dir, DefaultIgnore)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if res.Root != dir {
		t.Errorf("Root = %q, want %q", res.Root, dir)
	}
	if res.Files != 5 || res.Lines != 7 || res.Bytes != 52 {
		t.Errorf("totals = %d files, %d lines, %d bytes; want 5, 7, 52 (is .git pruned?)",
			res.Files, res.Lines, res.Bytes)
	}
	// Busiest first, ties broken by name — map order is random, and a report
	// that reshuffles itself between identical runs cannot be diffed.
	want := []ExtStat{
		{Ext: ".go", Files: 2, Lines: 4, Bytes: 33},
		{Ext: ".md", Files: 2, Lines: 2, Bytes: 14},
		{Ext: NoExt, Files: 1, Lines: 1, Bytes: 5},
	}
	if !reflect.DeepEqual(res.Exts, want) {
		t.Errorf("Exts = %+v, want %+v", res.Exts, want)
	}
}

func TestScanExtensionBuckets(t *testing.T) {
	dir := writeTree(t, map[string]string{
		"UPPER.GO":   "x\n", // extensions are compared lowercased
		"lower.go":   "x\n",
		".gitignore": "x\n", // filepath.Ext says ".gitignore" — it is not an extension
		"LICENSE":    "x\n",
	})

	res, err := Scan(context.Background(), dir, nil)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	// Equal file counts, so the tie-break is the bucket name — and "(" sorts
	// before "." in byte order.
	want := []ExtStat{
		{Ext: NoExt, Files: 2, Lines: 2, Bytes: 4},
		{Ext: ".go", Files: 2, Lines: 2, Bytes: 4},
	}
	if !reflect.DeepEqual(res.Exts, want) {
		t.Errorf("Exts = %+v, want %+v (dotfiles and extensionless files share the %q bucket)",
			res.Exts, want, NoExt)
	}
}

func TestScanLineCounting(t *testing.T) {
	cases := []struct {
		name      string
		content   string
		wantLines int
		wantBytes int64
	}{
		{"empty file", "", 0, 0},
		{"one terminated line", "one\n", 1, 4},
		{"unterminated last line", "one\ntwo", 2, 7},
		{"blank line counts", "\n\n", 2, 2},
		{"no newline at all", "one", 1, 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := writeTree(t, map[string]string{"f.txt": c.content})
			res, err := Scan(context.Background(), dir, nil)
			if err != nil {
				t.Fatalf("Scan: %v", err)
			}
			if res.Lines != c.wantLines || res.Bytes != c.wantBytes {
				t.Errorf("%q -> %d lines, %d bytes; want %d, %d",
					c.content, res.Lines, res.Bytes, c.wantLines, c.wantBytes)
			}
		})
	}
}

func TestScanIgnore(t *testing.T) {
	dir := writeTree(t, map[string]string{
		"keep.go":                      "x\n",
		"node_modules/dep/index.js":    "x\n",
		"node_modules/dep/nested/a.js": "x\n",
		"build/out.bin":                "x\n",
		"notes.tmp":                    "x\n",
	})

	res, err := Scan(context.Background(), dir, []string{"node_modules", "build", "notes.tmp"})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if res.Files != 1 {
		t.Errorf("Files = %d, want 1: ignored directories are pruned (not walked) and ignored file names are skipped",
			res.Files)
	}
	if len(res.Exts) != 1 || res.Exts[0].Ext != ".go" {
		t.Errorf("Exts = %+v, want only .go", res.Exts)
	}
}

func TestScanCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := Scan(ctx, sampleTree(t), nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want one matching context.Canceled via errors.Is", err)
	}
}

func TestScanMissingRoot(t *testing.T) {
	_, err := Scan(context.Background(), filepath.Join(t.TempDir(), "nope"), nil)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("err = %v, want one matching fs.ErrNotExist (propagate the walk's error, don't swallow it)", err)
	}
}

func TestTopExts(t *testing.T) {
	exts := []ExtStat{{Ext: ".go"}, {Ext: ".md"}, {Ext: ".txt"}}
	cases := []struct {
		n    int
		want int
	}{{0, 3}, {-1, 3}, {1, 1}, {2, 2}, {3, 3}, {99, 3}}
	for _, c := range cases {
		if got := TopExts(exts, c.n); len(got) != c.want {
			t.Errorf("TopExts(3 items, %d) kept %d, want %d", c.n, len(got), c.want)
		}
	}
}

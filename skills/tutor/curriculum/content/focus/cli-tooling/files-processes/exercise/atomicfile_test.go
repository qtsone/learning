package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWriteFileAtomicCreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	if err := WriteFileAtomic(path, []byte(`{"a":1}`), 0o600); err != nil {
		t.Fatalf("WriteFileAtomic returned err = %v, want nil", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading back: %v", err)
	}
	if string(got) != `{"a":1}` {
		t.Errorf("contents = %q, want %q", got, `{"a":1}`)
	}
	assertOnlyEntry(t, dir, "config.json")

	if runtime.GOOS == "windows" {
		t.Skip("Windows does not have Unix permission bits")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("mode = %04o, want 0600 — os.CreateTemp makes the temp file 0600, "+
			"and the caller's perm has to be applied before the rename", perm)
	}
}

func TestWriteFileAtomicReplacesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(path, []byte("old and much longer contents"), 0o644); err != nil {
		t.Fatalf("seeding: %v", err)
	}

	if err := WriteFileAtomic(path, []byte("new"), 0o644); err != nil {
		t.Fatalf("WriteFileAtomic returned err = %v, want nil", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading back: %v", err)
	}
	if string(got) != "new" {
		t.Errorf("contents = %q, want %q — the old bytes must be gone, not overwritten in place", got, "new")
	}
	assertOnlyEntry(t, dir, "notes.txt")
}

func TestWriteFileAtomicReportsFailureWithoutLeavingRubbish(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing-subdir", "config.json")

	if err := WriteFileAtomic(path, []byte("data"), 0o600); err == nil {
		t.Fatal("WriteFileAtomic returned nil error for a path whose directory does not exist")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading dir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("directory contains %d entries after a failed write, want 0", len(entries))
	}
}

// assertOnlyEntry checks that dir holds exactly one file: no temp file survived
// the write, not even a hidden one.
func assertOnlyEntry(t *testing.T, dir, name string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading dir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != name {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Errorf("directory contains %v, want just [%s] — clean the temp file up on every path", names, name)
	}
}

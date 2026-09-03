package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// WriteFileAtomic replaces the contents of path with data, so that a reader
// arriving at any moment sees either the whole old file or the whole new one,
// and a crash mid-write loses neither.
//
// The recipe: create a temp file *in the same directory as path*, write data
// into it, give it perm, flush it to disk, close it, then rename it over path.
// A rename within one filesystem is atomic; a temp file in os.TempDir() is on a
// different filesystem often enough that the rename fails with "invalid
// cross-device link" on your user's machine and never on yours.
//
// Whatever happens, no temp file survives the call.
func WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
	dir, name := filepath.Split(path)
	if dir == "" {
		dir = "."
	}
	f, err := os.CreateTemp(dir, "."+name+".tmp-*")
	if err != nil {
		return fmt.Errorf("creating temp file for %s: %w", path, err)
	}
	tmp := f.Name()
	// Removing a name that a successful rename has already moved is a harmless
	// no-op, so one defer covers every failure path below.
	defer os.Remove(tmp)

	if err := writeAndClose(f, data, perm); err != nil {
		return fmt.Errorf("writing %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("replacing %s: %w", path, err)
	}
	return nil
}

// writeAndClose fills f, fixes its permissions and flushes it to disk. Close is
// deliberately checked: on a network or full filesystem it is where the write
// error finally surfaces.
func writeAndClose(f *os.File, data []byte, perm fs.FileMode) error {
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	// os.CreateTemp always makes the file 0600. Chmod, unlike the mode argument
	// to os.OpenFile, is not filtered through the umask, so this lands exactly.
	if err := f.Chmod(perm); err != nil {
		f.Close()
		return err
	}
	// Without Sync the rename can outlive the data on a crash: the directory
	// entry points at a file whose blocks were never written.
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

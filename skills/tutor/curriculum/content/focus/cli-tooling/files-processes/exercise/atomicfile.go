package main

import "io/fs"

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
//
// TODO: implement.
func WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
	return nil
}

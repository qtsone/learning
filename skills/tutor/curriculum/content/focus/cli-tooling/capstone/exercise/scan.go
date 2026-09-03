package main

import (
	"bytes"
	"context"
	"io"
	"os"
)

// NoExt is the bucket for files with no extension — including dotfiles such as
// .gitignore, whose "extension" filepath.Ext reports is the whole name.
const NoExt = "(none)"

type ExtStat struct {
	Ext   string `json:"ext"`
	Files int    `json:"files"`
	Lines int    `json:"lines"`
	Bytes int64  `json:"bytes"`
}

type ScanResult struct {
	Root  string    `json:"root"`
	Files int       `json:"files"`
	Lines int       `json:"lines"`
	Bytes int64     `json:"bytes"`
	Exts  []ExtStat `json:"exts"`
}

// Scan walks root and totals regular files by extension.
//
// It takes a context and returns a plain result: cancellation is the caller's
// concern, and the walk is the only thing in here that touches the disk.
//
// TODO: implement it with filepath.WalkDir.
//   - Check ctx.Err() once per entry and return it — that is what turns Ctrl-C
//     into a stopped walk.
//   - Return fs.SkipDir for an ignored directory (prune, do not walk), and skip
//     ignored file names and anything that is not a regular file.
//   - Propagate the walk's errors; a missing root must still match
//     fs.ErrNotExist.
//   - Sort Exts busiest first, ties broken by name: map iteration order is
//     random and a report that reshuffles itself cannot be diffed.
func Scan(ctx context.Context, root string, ignore []string) (ScanResult, error) {
	return ScanResult{}, nil
}

// extOf buckets a file name.
//
// TODO: lowercase the extension; return NoExt for a name that has none — and
// for a dotfile, since filepath.Ext(".gitignore") is ".gitignore".
func extOf(name string) string {
	return NoExt
}

// TopExts keeps the n busiest extensions; n <= 0 means all of them.
//
// TODO: implement.
func TopExts(exts []ExtStat, n int) []ExtStat {
	return exts
}

// countFile returns the number of lines and bytes in one file. A line is a
// newline-terminated run of bytes, plus a final unterminated line if the file
// does not end with a newline — so "a\nb" is two lines and "" is none.
//
// Provided complete: this is the boring half of the walk.
func countFile(path string) (lines int, size int64, err error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	buf := make([]byte, 32*1024)
	var last byte
	for {
		n, readErr := f.Read(buf)
		if n > 0 {
			size += int64(n)
			lines += bytes.Count(buf[:n], []byte{'\n'})
			last = buf[n-1]
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return 0, 0, readErr
		}
	}
	if size > 0 && last != '\n' {
		lines++
	}
	return lines, size, nil
}

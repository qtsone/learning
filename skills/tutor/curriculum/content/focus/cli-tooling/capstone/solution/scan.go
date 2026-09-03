package main

import (
	"bytes"
	"cmp"
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
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
// concern, and the tree walk is the only thing in here that touches the disk.
func Scan(ctx context.Context, root string, ignore []string) (ScanResult, error) {
	skip := make(map[string]bool, len(ignore))
	for _, name := range ignore {
		if name != "" {
			skip[name] = true
		}
	}

	res := ScanResult{Root: root, Exts: []ExtStat{}}
	byExt := map[string]*ExtStat{}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Checked once per entry: a walk of a huge tree stops within one file
		// of the Ctrl-C, and nothing else in the program needs to know.
		if err := ctx.Err(); err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if path != root && skip[name] {
				return fs.SkipDir // prune: never descend into an ignored directory
			}
			return nil
		}
		// Symlinks, sockets and devices are not source code, and reading them
		// can block forever.
		if !d.Type().IsRegular() || skip[name] {
			return nil
		}
		lines, size, err := countFile(path)
		if err != nil {
			return err
		}
		ext := extOf(name)
		stat, ok := byExt[ext]
		if !ok {
			stat = &ExtStat{Ext: ext}
			byExt[ext] = stat
		}
		stat.Files++
		stat.Lines += lines
		stat.Bytes += size
		res.Files++
		res.Lines += lines
		res.Bytes += size
		return nil
	})
	if err != nil {
		return ScanResult{}, err
	}

	for _, stat := range byExt {
		res.Exts = append(res.Exts, *stat)
	}
	// Busiest first, ties broken by name: map iteration order is random, and a
	// report that reorders itself between identical runs cannot be diffed.
	slices.SortFunc(res.Exts, func(a, b ExtStat) int {
		if c := cmp.Compare(b.Files, a.Files); c != 0 {
			return c
		}
		return strings.Compare(a.Ext, b.Ext)
	})
	return res, nil
}

// extOf buckets a file name. filepath.Ext(".gitignore") is ".gitignore", so a
// naive implementation invents an extension per dotfile.
func extOf(name string) string {
	ext := filepath.Ext(name)
	if ext == "" || ext == name {
		return NoExt
	}
	return strings.ToLower(ext)
}

// TopExts keeps the n busiest extensions; n <= 0 means all of them.
func TopExts(exts []ExtStat, n int) []ExtStat {
	if n <= 0 || n >= len(exts) {
		return exts
	}
	return exts[:n]
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

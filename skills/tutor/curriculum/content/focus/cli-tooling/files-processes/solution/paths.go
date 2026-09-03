package main

import (
	"errors"
	"fmt"
	"path/filepath"
)

// ErrUnsafePath reports a path that would escape the directory it was supposed
// to stay inside, or a name that is not a single plain path element.
var ErrUnsafePath = errors.New("unsafe path")

// AppDir returns the per-user directory for app inside the directory base
// reports. Pass os.UserConfigDir for settings the user edits and keeps, and
// os.UserCacheDir for anything you could regenerate:
//
//	dir, err := AppDir(os.UserConfigDir, "runner")
//
// base is a parameter rather than a call to os.UserConfigDir inside this
// function for the reason the whole pack keeps repeating: the seam is what lets
// a test answer the question without owning a home directory.
//
// AppDir does not create the directory — that is the caller's job, with
// os.MkdirAll and a mode of 0o700.
//
// An app name that is not a single plain path element — "", ".", "..", "a/b",
// "/etc" — is an error wrapping ErrUnsafePath. An error from base is wrapped,
// not swallowed, so the caller learns which directory could not be found.
func AppDir(base func() (string, error), app string) (string, error) {
	// IsLocal rules out empty, absolute, rooted and escaping names, and Windows
	// device names such as NUL. It accepts ".", which is a fine path and a
	// terrible application name.
	if !filepath.IsLocal(app) || app == "." || app != filepath.Base(app) {
		return "", fmt.Errorf("%w: %q is not an application name", ErrUnsafePath, app)
	}
	dir, err := base()
	if err != nil {
		return "", fmt.Errorf("locating the directory for %q: %w", app, err)
	}
	return filepath.Join(dir, app), nil
}

// SafeJoin joins rel onto root and guarantees the result stays under root, so
// that a path arriving from a config file, an archive entry or a command line
// cannot reach /etc/passwd. rel is cleaned on the way: "a/../b" is "b".
//
// A rel that is absolute, rooted, empty, or escapes with ".." is an error
// wrapping ErrUnsafePath whose message names the offending path.
func SafeJoin(root, rel string) (string, error) {
	if !filepath.IsLocal(rel) {
		return "", fmt.Errorf("%w: %q must stay inside %q", ErrUnsafePath, rel, root)
	}
	return filepath.Join(root, rel), nil
}

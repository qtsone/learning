package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppDirJoinsAppOntoBase(t *testing.T) {
	base := t.TempDir()
	got, err := AppDir(func() (string, error) { return base, nil }, "runner")
	if err != nil {
		t.Fatalf("AppDir returned err = %v, want nil", err)
	}
	if want := filepath.Join(base, "runner"); got != want {
		t.Errorf("AppDir = %q, want %q", got, want)
	}
	if _, err := os.Stat(got); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("AppDir created the directory; it must only compute the path")
	}
}

func TestAppDirWrapsBaseFailure(t *testing.T) {
	sentinel := errors.New("no home directory")
	_, err := AppDir(func() (string, error) { return "", sentinel }, "runner")
	if !errors.Is(err, sentinel) {
		t.Fatalf("AppDir returned err = %v, want one wrapping the base error", err)
	}
	if !strings.Contains(err.Error(), "runner") {
		t.Errorf("error = %q, want it to name the application", err)
	}
}

func TestAppDirRejectsBadNames(t *testing.T) {
	for _, app := range []string{"", "..", "a/b", "/etc", "."} {
		t.Run(app, func(t *testing.T) {
			_, err := AppDir(func() (string, error) { return "/base", nil }, app)
			if !errors.Is(err, ErrUnsafePath) {
				t.Errorf("AppDir(_, %q) returned err = %v, want one wrapping ErrUnsafePath",
					app, err)
			}
		})
	}
}

func TestAppDirWithRealUserDirs(t *testing.T) {
	for name, base := range map[string]func() (string, error){
		"config": os.UserConfigDir,
		"cache":  os.UserCacheDir,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := base(); err != nil {
				t.Skipf("this machine has no %s directory: %v", name, err)
			}
			got, err := AppDir(base, "runner")
			if err != nil {
				t.Fatalf("AppDir returned err = %v, want nil", err)
			}
			if !filepath.IsAbs(got) {
				t.Errorf("AppDir = %q, want an absolute path", got)
			}
			if filepath.Base(got) != "runner" {
				t.Errorf("AppDir = %q, want it to end in the application name", got)
			}
			if strings.Contains(got, "$") || strings.Contains(got, "~") {
				t.Errorf("AppDir = %q — ask the os package, never hand-build a home path", got)
			}
		})
	}
}

func TestSafeJoin(t *testing.T) {
	root := filepath.Join("var", "data")

	ok := []struct {
		rel  string
		want string
	}{
		{"notes.txt", filepath.Join(root, "notes.txt")},
		{"a/b/c.txt", filepath.Join(root, "a", "b", "c.txt")},
		{"./a", filepath.Join(root, "a")},
		{"a/../b", filepath.Join(root, "b")},
	}
	for _, c := range ok {
		got, err := SafeJoin(root, c.rel)
		if err != nil {
			t.Errorf("SafeJoin(%q, %q) returned err = %v, want nil", root, c.rel, err)
			continue
		}
		if got != c.want {
			t.Errorf("SafeJoin(%q, %q) = %q, want %q", root, c.rel, got, c.want)
		}
	}

	bad := []string{"", "..", "../etc/passwd", "a/../../etc", "/etc/passwd", "/"}
	for _, rel := range bad {
		got, err := SafeJoin(root, rel)
		if !errors.Is(err, ErrUnsafePath) {
			t.Errorf("SafeJoin(%q, %q) = %q, err = %v; want an error wrapping ErrUnsafePath",
				root, rel, got, err)
			continue
		}
		if rel != "" && !strings.Contains(err.Error(), rel) {
			t.Errorf("error = %q, want it to name the rejected path %q", err, rel)
		}
	}
}

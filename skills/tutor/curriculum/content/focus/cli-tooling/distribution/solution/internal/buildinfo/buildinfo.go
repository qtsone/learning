// Package buildinfo carries the identity the release pipeline stamps into the
// binary: which version, which commit, built when.
package buildinfo

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

// Overwritten at link time with
// -X tutor.local/digest/internal/buildinfo.<name>=<value>. They must stay
// package-level string vars initialised to constants for -X to reach them.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// String returns the single line printed by `digest --version`.
func String() string {
	v, c, d := version, commit, date
	if v == "dev" {
		v, c, d = fromBuildInfo(v, c, d)
	}
	return fmt.Sprintf("digest %s (commit %s, built %s, %s %s/%s)",
		v, c, d, runtime.Version(), runtime.GOOS, runtime.GOARCH)
}

// fromBuildInfo fills in what the linker did not: `go install tool@v1.4.2`
// records the module version, and a build from a VCS checkout records the
// revision and its timestamp.
func fromBuildInfo(v, c, d string) (string, string, string) {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return v, c, d
	}
	if mv := bi.Main.Version; mv != "" && mv != "(devel)" {
		v = mv
	}
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			if c == "none" {
				c = s.Value
			}
		case "vcs.time":
			if d == "unknown" {
				d = s.Value
			}
		}
	}
	return v, c, d
}

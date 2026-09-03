// Package buildinfo carries the identity the release pipeline stamps into the
// binary: which version, which commit, built when.
package buildinfo

// TODO: declare three package-level string vars — version, commit, date —
// with the placeholder defaults "dev", "none" and "unknown".
//
// They must be package-level *vars* initialised to constant strings: that is
// the only shape `go build -ldflags "-X <import path>.<name>=<value>"` can
// overwrite. The import path of this package is
// tutor.local/digest/internal/buildinfo — not "buildinfo", and not "main".

// String returns the single line printed by `digest --version`.
func String() string {
	// TODO: return, in exactly this shape,
	//
	//   digest <version> (commit <commit>, built <date>, <go version> <goos>/<goarch>)
	//
	// e.g. digest 1.4.2 (commit 9f8e7d6c, built 2024-05-01T10:00:00Z, go1.22.3 linux/amd64)
	//
	// The go version and the target pair come from the runtime package, not
	// from the linker — the binary already knows them.
	//
	// Then handle the `go install` case: when version is still "dev", nobody
	// passed -X, so ask runtime/debug.ReadBuildInfo whether this binary was
	// built from a tagged module (bi.Main.Version) or from a VCS checkout
	// (the "vcs.revision" and "vcs.time" build settings) and prefer those.
	// Careful: "(devel)" and "" are not real versions — a plain local build
	// must still report "dev".
	return "digest"
}

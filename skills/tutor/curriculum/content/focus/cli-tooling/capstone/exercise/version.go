package main

import "runtime"

// These three are patched by the linker at build time:
//
//	go build -ldflags "-X main.version=1.4.0 -X main.commit=$(git rev-parse --short HEAD) -X main.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
//
// -X only works on package-level string variables, which is why they are vars
// and not consts: a const is folded into the code before the linker sees it.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// BuildInfo is the version report as data, so the text and JSON renderers
// describe exactly the same thing.
type BuildInfo struct {
	Version  string `json:"version"`
	Commit   string `json:"commit"`
	Date     string `json:"date"`
	Go       string `json:"go"`
	Platform string `json:"platform"`
}

func buildInfo() BuildInfo {
	return BuildInfo{
		Version:  version,
		Commit:   commit,
		Date:     date,
		Go:       runtime.Version(),
		Platform: runtime.GOOS + "/" + runtime.GOARCH,
	}
}

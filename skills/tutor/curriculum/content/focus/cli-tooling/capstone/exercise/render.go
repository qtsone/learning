package main

import (
	"encoding/json"
	"io"
)

const (
	Reset = "\x1b[0m"
	Bold  = "\x1b[1m"
)

// Renderer owns every byte the tool puts on stdout. It is handed the resolved
// Settings, so no rendering code re-decides whether colour is allowed.
type Renderer struct {
	out   io.Writer
	color bool
	json  bool
}

func NewRenderer(out io.Writer, set Settings) *Renderer {
	return &Renderer{out: out, color: set.Color, json: set.JSON}
}

func (r *Renderer) writeJSON(v any) error {
	return json.NewEncoder(r.out).Encode(v)
}

// Paint wraps s in the codes and one Reset — or returns it untouched when
// colour is off.
//
// TODO: implement. Callers pad *before* painting: an escape sequence has no
// width, so %-10s on a painted string pads by bytes and ruins the columns.
func (r *Renderer) Paint(s string, codes ...string) string {
	return s
}

// RenderScan writes the scan report: a JSON document, or a human table.
//
// TODO: JSON mode encodes the ScanResult itself. Text mode writes
//
//	fmt.Sprintf("%-10s %6s %8s %10s", "ext", "files", "lines", "bytes")
//
// bold as a header, then one "%-10s %6d %8d %10d" row per extension, then the
// same row shape bold for "total". An empty tree prints "no files".
// Report write errors — a full disk is not a success.
func (r *Renderer) RenderScan(res ScanResult) error {
	return nil
}

// RenderAuthors writes the author ranking.
//
// TODO: JSON mode writes the document {"authors":[…]} — not a bare array, which
// could never grow a field. Text mode writes "%6d  %s" per author, or
// "no commits".
func (r *Renderer) RenderAuthors(list []Author) error {
	return nil
}

// RenderVersion writes the build metadata patched in by -ldflags.
//
// TODO: JSON mode writes the BuildInfo document. Text mode writes
// "scout <version> (commit <commit>, built <date>, <go>, <platform>)".
func (r *Renderer) RenderVersion(info BuildInfo) error {
	return nil
}

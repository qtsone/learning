package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const (
	Reset = "\x1b[0m"
	Bold  = "\x1b[1m"
)

// Column widths are shared by the header and the rows so the two cannot drift.
const (
	scanHeaderFmt = "%-10s %6s %8s %10s"
	scanRowFmt    = "%-10s %6d %8d %10d"
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

// Paint wraps s in the codes and one Reset — or returns it untouched when
// colour is off. Callers pad *before* painting: an escape sequence has no
// width, so %-10s on a painted string pads by bytes and ruins the columns.
func (r *Renderer) Paint(s string, codes ...string) string {
	if !r.color || len(codes) == 0 || s == "" {
		return s
	}
	return strings.Join(codes, "") + s + Reset
}

func (r *Renderer) writeJSON(v any) error {
	return json.NewEncoder(r.out).Encode(v)
}

// RenderScan writes the scan report: a JSON document, or a human table.
func (r *Renderer) RenderScan(res ScanResult) error {
	if r.json {
		return r.writeJSON(res)
	}
	if res.Files == 0 {
		_, err := fmt.Fprintln(r.out, "no files")
		return err
	}
	header := fmt.Sprintf(scanHeaderFmt, "ext", "files", "lines", "bytes")
	if _, err := fmt.Fprintln(r.out, r.Paint(header, Bold)); err != nil {
		return err
	}
	for _, e := range res.Exts {
		row := fmt.Sprintf(scanRowFmt, e.Ext, e.Files, e.Lines, e.Bytes)
		if _, err := fmt.Fprintln(r.out, row); err != nil {
			return err
		}
	}
	total := fmt.Sprintf(scanRowFmt, "total", res.Files, res.Lines, res.Bytes)
	_, err := fmt.Fprintln(r.out, r.Paint(total, Bold))
	return err
}

// RenderAuthors writes the author ranking.
func (r *Renderer) RenderAuthors(list []Author) error {
	if r.json {
		// A document, not a bare array: {"authors":[…]} can grow a field later
		// without breaking every consumer.
		return r.writeJSON(struct {
			Authors []Author `json:"authors"`
		}{Authors: list})
	}
	if len(list) == 0 {
		_, err := fmt.Fprintln(r.out, "no commits")
		return err
	}
	for _, a := range list {
		if _, err := fmt.Fprintf(r.out, "%6d  %s\n", a.Commits, a.Name); err != nil {
			return err
		}
	}
	return nil
}

// RenderVersion writes the build metadata patched in by -ldflags.
func (r *Renderer) RenderVersion(info BuildInfo) error {
	if r.json {
		return r.writeJSON(info)
	}
	_, err := fmt.Fprintf(r.out, "scout %s (commit %s, built %s, %s, %s)\n",
		info.Version, info.Commit, info.Date, info.Go, info.Platform)
	return err
}

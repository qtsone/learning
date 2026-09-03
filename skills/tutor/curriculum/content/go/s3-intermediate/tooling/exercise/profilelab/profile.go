// Package profilelab holds two implementations of the same job so you can
// watch pprof tell them apart. Nothing here needs fixing — it needs measuring.
package profilelab

import "strings"

func JoinNaive(lines []string) string {
	out := ""
	for _, line := range lines {
		out += line + "\n"
	}
	return out
}

func JoinBuilder(lines []string) string {
	var b strings.Builder
	for _, line := range lines {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

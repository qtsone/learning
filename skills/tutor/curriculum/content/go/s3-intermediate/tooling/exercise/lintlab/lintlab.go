// Package lintlab is quiet under go vet — every problem here needs
// staticcheck or golangci-lint to surface. Record findings (with check
// codes) in NOTES.md before fixing.
package lintlab

import (
	"os"
	"strings"
)

func NormalizeName(raw string) string {
	name := strings.TrimSpace(raw)
	name = strings.ToLower(raw)
	return name
}

func Headline(s string) string {
	return strings.Title(s)
}

func IsReady(enabled bool, attempts int) bool {
	if enabled == true {
		return attempts > 0
	}
	return false
}

func legacyChecksum(data []byte) int {
	sum := 0
	for _, b := range data {
		sum += int(b)
	}
	return sum
}

func WriteReport(path, body string) {
	f, err := os.Create(path)
	if err != nil {
		return
	}
	f.WriteString(body)
	f.Close()
}

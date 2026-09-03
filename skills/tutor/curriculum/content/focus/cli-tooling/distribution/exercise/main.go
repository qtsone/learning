// Command digest prints the SHA-256 of every file it is given.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"

	"tutor.local/digest/internal/buildinfo"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("digest", flag.ContinueOnError)
	fs.SetOutput(stderr)
	showVersion := fs.Bool("version", false, "print version information and exit")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *showVersion {
		fmt.Fprintln(stdout, buildinfo.String())
		return 0
	}

	if fs.NArg() == 0 {
		fmt.Fprintln(stderr, "usage: digest [--version] <file>...")
		return 2
	}

	status := 0
	for _, name := range fs.Args() {
		sum, err := sumFile(name)
		if err != nil {
			fmt.Fprintf(stderr, "digest: %v\n", err)
			status = 1
			continue
		}
		fmt.Fprintf(stdout, "%s  %s\n", sum, name)
	}
	return status
}

func sumFile(name string) (string, error) {
	f, err := os.Open(name)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("read %s: %w", name, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

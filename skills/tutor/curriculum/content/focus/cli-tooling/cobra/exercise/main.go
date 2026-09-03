package main

import (
	"fmt"
	"os"
)

// main is the only place allowed to know about the process: it builds the real
// dependencies, runs the tree, and turns a returned error into an exit code.
// Everything below it is testable in-process.
func main() {
	app := &App{Store: NewMemStore(), Getenv: os.Getenv}
	if err := NewRootCmd(app).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "notes:", err)
		os.Exit(1)
	}
}

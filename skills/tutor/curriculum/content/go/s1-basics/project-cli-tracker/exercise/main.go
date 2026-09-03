package main

import (
	"fmt"
	"os"
)

// dataFile is where the tracker keeps its tasks — in whatever directory
// you run the program from.
const dataFile = "tracker.txt"

func main() {
	// TODO: load the tracker from dataFile (a missing file is a fresh start).
	// TODO: run dispatch with os.Args[1:] to get the text to print.
	// TODO: save the tracker back to dataFile, then print the text.
	// If any step fails: print the error to os.Stderr and os.Exit(1).
	fmt.Fprintln(os.Stderr, "TODO: wire up main — see the acceptance criteria in LESSON.md")
	os.Exit(1)
}

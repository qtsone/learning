package main

import (
	"fmt"
	"os"

	"tutor.local/project-cli-tracker/tracker"
)

// dataFile is where the tracker keeps its tasks — in whatever directory
// you run the program from.
const dataFile = "tracker.txt"

func main() {
	t, err := tracker.Load(dataFile)
	if err != nil {
		fail(err)
	}
	out, err := dispatch(t, os.Args[1:])
	if err != nil {
		fail(err)
	}
	if err := t.Save(dataFile); err != nil {
		fail(err)
	}
	fmt.Println(out)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}

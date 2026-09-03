package logkit

import (
	"flag"
	"fmt"
	"os"
	"testing"
)

// integrationDir is the scratch directory the integration tests share. It is
// created once for the whole package run and removed when the run ends: that
// is what TestMain is for — setup and teardown too expensive, or too global,
// to repeat per test.
var integrationDir string

func TestMain(m *testing.M) {
	flag.Parse() // testing.Short() only answers correctly after this

	if !testing.Short() {
		dir, err := os.MkdirTemp("", "logkit-integration-")
		if err != nil {
			fmt.Fprintf(os.Stderr, "integration setup: %v\n", err)
			os.Exit(1)
		}
		integrationDir = dir
	}

	code := m.Run()

	// Teardown runs here, not in a defer: os.Exit does not run defers.
	if integrationDir != "" {
		os.RemoveAll(integrationDir)
	}
	os.Exit(code)
}

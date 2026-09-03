// Command logreport summarizes wire-format log lines read from stdin.
//
// main is deliberately three lines: everything worth testing lives in
// logkit.Run, which takes its arguments and streams as parameters.
package main

import (
	"os"

	logkit "tutor.local/adv-testing"
)

func main() {
	os.Exit(logkit.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

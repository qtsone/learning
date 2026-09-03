package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
)

// main is the only place that knows about the real process: the real
// arguments, the real environment, the real stderr. Everything below it is
// injected, which is why the loader is testable at all.
func main() {
	loader := Loader{
		Args:       os.Args[1:],
		LookupEnv:  os.LookupEnv,
		ConfigPath: "toolkit.json",
		Stderr:     os.Stderr,
	}

	res, err := loader.Load()
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return // asking for help is not a failure
		}
		fmt.Fprintln(os.Stderr, "toolkit:", err)
		os.Exit(2)
	}

	// The payoff of tracking origins: "where did this value come from?" is a
	// question the tool can answer itself.
	fmt.Printf("endpoint  %-28q (from %s)\n", res.Config.Endpoint, res.Origins["endpoint"])
	fmt.Printf("timeout   %-28s (from %s)\n", res.Config.Timeout, res.Origins["timeout"])
	fmt.Printf("retries   %-28d (from %s)\n", res.Config.Retries, res.Origins["retries"])
	fmt.Printf("verbose   %-28t (from %s)\n", res.Config.Verbose, res.Origins["verbose"])
	// A width verb applies per element on a slice, so join first.
	fmt.Printf("tags      %-28s (from %s)\n", strings.Join(res.Config.Tags, ","), res.Origins["tags"])
}

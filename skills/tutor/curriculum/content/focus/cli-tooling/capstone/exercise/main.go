package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// main is the only function in this program that knows a process exists. It
// resolves the real world — signals, the environment, the user's config
// directory, whether stdout is a terminal — hands all of it to App, and turns
// the code Execute returns into the process exit status.
//
// Everything below this file is reachable from a test without a subprocess.
func main() {
	ctx, stop := notifyContext(context.Background())
	defer stop()

	configDir, err := os.UserConfigDir()
	if err != nil {
		// No config directory on this machine (some containers, odd
		// environments). Defaults, environment and flags still work.
		configDir = ""
	}

	app := &App{
		Getenv:           os.Getenv,
		ConfigDir:        configDir,
		Git:              []string{"git"},
		StdoutIsTerminal: isTerminal(os.Stdout),
	}

	os.Exit(Execute(ctx, app, os.Args[1:], os.Stdout, os.Stderr))
}

// notifyContext returns a context cancelled by the first Ctrl-C (SIGINT) or by
// SIGTERM — the signal a service manager or `kill` sends to ask a process to
// stop. The returned stop() restores the default handlers, so a second Ctrl-C
// kills the process outright even if shutdown wedges: graceful shutdown must
// never trap the user.
//
// Nothing slow happens here on purpose. A signal handler runs while the program
// is in an unknown state; the only safe thing to do in one is to flip a switch —
// here, cancel a context — and let ordinary code notice.
func notifyContext(parent context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
}

// isTerminal reports whether f is attached to a terminal rather than a pipe or
// a file. The character-device check is the stdlib-only answer from the
// terminal-UX lesson; golang.org/x/term.IsTerminal is the precise one.
func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

package main

import "context"

// Exit codes this tool uses. 0 and 1 are universal; the other two are shell
// conventions worth following, because scripts already read them:
//
//	124 is what timeout(1) returns when it had to give up,
//	130 is 128+2, the status a shell reports for a program killed by SIGINT.
const (
	ExitOK          = 0
	ExitFailure     = 1
	ExitTimeout     = 124
	ExitInterrupted = 130
)

// WithInterrupt returns a copy of parent that is cancelled the first time this
// process receives SIGINT (Ctrl-C) or SIGTERM (the polite "please stop" that
// init systems, Docker and Kubernetes send before they resort to SIGKILL).
//
// Nothing slow happens on the signal itself: the only effect is that a context
// becomes done, and the ordinary code already watching it — your work loop,
// os/exec, anything from S3 — unwinds and cleans up.
//
// The returned stop function releases the registration and restores the default
// behaviour, so a Ctrl-C after it kills the program the usual way. Always defer
// it.
//
// TODO: replace this with a context that also reacts to SIGINT and SIGTERM.
func WithInterrupt(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithCancel(parent)
}

// ExitCodeFor maps the error a run finished with onto this process's exit
// status: nil is ExitOK, a cancelled context is ExitInterrupted, an expired
// deadline is ExitTimeout, and anything else is ExitFailure.
//
// It has to see through wrapping — the error it is handed will usually be a
// *StepError, or something wrapped with %w on the way up.
//
// TODO: implement.
func ExitCodeFor(err error) int {
	return ExitOK
}

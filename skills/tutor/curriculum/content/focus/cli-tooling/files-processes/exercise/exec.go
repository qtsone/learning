package main

import (
	"context"
	"errors"
	"io"
)

// ErrEmptyCommand reports a Command with no executable name. Running one is
// always a bug in the caller, so Run refuses before it touches the operating
// system.
var ErrEmptyCommand = errors.New("empty command name")

// Command describes a child process. There is no shell anywhere in this type:
// Name is an executable, and every element of Args arrives at the child as
// exactly one argument, whatever characters it contains.
type Command struct {
	// Name is the program to run — a bare name looked up on PATH, or a path.
	// Never a command line.
	Name string

	// Args are the arguments after the program name. Args[0] is the first
	// argument, not the program name: os/exec supplies that itself.
	Args []string

	// Dir is the child's working directory. Empty means "inherit ours".
	Dir string

	// Env holds extra KEY=VALUE entries for the child. They are added to the
	// environment this process already has, and an entry here wins over an
	// inherited one with the same key.
	Env []string

	// Stdin, if non-nil, is copied to the child's standard input. A nil Stdin
	// gives the child an empty input rather than ours.
	Stdin io.Reader
}

// Result is everything a finished child process tells us.
type Result struct {
	// Stdout and Stderr are captured separately and never interleaved — one is
	// the child's data, the other its diagnostics, exactly the split the
	// previous lesson drew.
	Stdout string
	Stderr string

	// ExitCode is the status the child exited with, or -1 when it never got
	// that far: it could not be started, or a signal killed it.
	ExitCode int
}

// Run executes c, waits for it to finish, and captures both output streams.
//
// A child that runs and exits non-zero is *not* a Go error: its status lands in
// Result.ExitCode and the returned error is nil. The error is non-nil only when
// the command could not be run to completion — the executable was not found,
// the working directory does not exist, ctx was cancelled — and Result.ExitCode
// is then -1.
//
// When ctx is done the returned error wraps ctx.Err(), so a caller can tell a
// timeout (errors.Is(err, context.DeadlineExceeded)) from a real failure. That
// check has to come before the exit-status check: a child killed by the context
// looks exactly like a child killed by anything else.
//
// TODO: implement.
func Run(ctx context.Context, c Command) (Result, error) {
	return Result{ExitCode: -1}, nil
}

// StepError says which command in a sequence failed, and how.
type StepError struct {
	Index  int    // position in the sequence, counting from 0
	Name   string // the command's Name, so the message reads well
	Result Result // what the step produced; zero if it never ran
	Err    error  // why it could not run at all, or nil for a non-zero exit
}

// Error formats the failure as "step <index> (<name>): <reason>", where the
// reason is Err when there is one and "exit status <code>" otherwise.
//
// TODO: implement.
func (e *StepError) Error() string {
	return ""
}

// Unwrap exposes Err so that errors.Is can see through a StepError to, say,
// context.Canceled.
//
// TODO: implement.
func (e *StepError) Unwrap() error {
	return nil
}

// RunSteps runs cmds in order and stops at the first failure — a step that
// cannot be started, or one that exits non-zero. It returns the results of the
// steps that ran, in order, including the failing one; a step that could not be
// started contributes no Result. The error, when there is one, is a *StepError.
//
// Before each step it checks whether ctx is done, so that a Ctrl-C between
// steps stops the sequence instead of starting more work. That check produces a
// StepError whose Err is ctx.Err().
//
// TODO: implement.
func RunSteps(ctx context.Context, cmds []Command) ([]Result, error) {
	return nil, nil
}

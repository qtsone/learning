package main

import (
	"bufio"
	"errors"
	"io"
)

// MaxPromptAttempts is how many answers Ask will consider before giving up.
const MaxPromptAttempts = 3

// ErrNotInteractive means there is nobody to answer: the input stream is not a
// terminal and the question has no default. Callers should suggest the flag
// that supplies the value instead.
var ErrNotInteractive = errors.New("not interactive: no terminal to prompt on")

// ErrTooManyAttempts means every answer failed validation.
var ErrTooManyAttempts = errors.New("too many invalid answers")

// Prompter asks questions on an injected pair of streams. It owns one
// bufio.Scanner for the whole session: a fresh reader per question would keep
// the bytes it buffered past the newline and lose the next answer.
type Prompter struct {
	sc         *bufio.Scanner
	out        io.Writer
	isTerminal bool
}

// NewPrompter reads answers from in and writes questions to out — in a real
// program the diagnostic stream, so that a prompt never lands in piped data.
func NewPrompter(in io.Reader, out io.Writer, isTerminal bool) *Prompter {
	return &Prompter{sc: bufio.NewScanner(in), out: out, isTerminal: isTerminal}
}

// Ask puts one question to the user and returns the answer.
//
// When the prompter is attached to a terminal it writes the question, reads
// one line, and trims the spaces around it:
//
//	"Region [eu-west-1]: "   // with a default
//	"Region: "               // without one
//
// An empty answer takes the default. When validate is non-nil and rejects the
// answer, Ask writes "error: <message>\n" and asks again, considering at most
// MaxPromptAttempts answers before returning ErrTooManyAttempts.
//
// When the prompter is not attached to a terminal it neither reads nor writes:
// it returns the default, or ErrNotInteractive when there is no default. A
// program that blocks on a prompt inside a pipeline or a CI job is broken.
// Input that ends before an answer arrives lands in the same place: the
// default if there is one, ErrNotInteractive if there is not. The default is
// not re-validated — it comes from the program, not from the user, and
// re-asking a stream that is already at EOF would loop forever.
//
// TODO: implement.
func (p *Prompter) Ask(question, def string, validate func(string) error) (string, error) {
	return "", nil
}

package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
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

// Ask puts one question to the user and returns the answer. See the package
// tests for the exact contract: default handling, validation retries, and the
// non-interactive fallback.
func (p *Prompter) Ask(question, def string, validate func(string) error) (string, error) {
	if !p.isTerminal {
		return fallback(def)
	}
	for range MaxPromptAttempts {
		fmt.Fprint(p.out, promptLine(question, def))
		if !p.sc.Scan() {
			// The stream is exhausted (or broken): asking again would spin
			// forever against an empty reader.
			return fallback(def)
		}
		answer := strings.TrimSpace(p.sc.Text())
		if answer == "" {
			answer = def
		}
		if validate != nil {
			if err := validate(answer); err != nil {
				fmt.Fprintf(p.out, "error: %v\n", err)
				continue
			}
		}
		return answer, nil
	}
	return "", ErrTooManyAttempts
}

func fallback(def string) (string, error) {
	if def == "" {
		return "", ErrNotInteractive
	}
	return def, nil
}

func promptLine(question, def string) string {
	if def == "" {
		return question + ": "
	}
	return fmt.Sprintf("%s [%s]: ", question, def)
}

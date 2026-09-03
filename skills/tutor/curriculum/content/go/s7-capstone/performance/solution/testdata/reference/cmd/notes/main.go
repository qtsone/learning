// Command notes reads note lines from stdin, validates them, and prints the
// listing. It is a composition root: it wires the packages, owns the process
// lifetime, and holds no rules.
package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"tutor.local/capstone-reference/internal/note"
	"tutor.local/capstone-reference/internal/remote"
	"tutor.local/capstone-reference/internal/store"
)

// webhookEnv names the environment variable holding the webhook URL. Secrets
// arrive through the environment, never through a flag (flags are visible in
// the process list) and never through a file in the repository.
const webhookEnv = "NOTES_WEBHOOK"

func main() {
	// The root context belongs here, where the program's lifetime starts.
	// Everything below is handed this one rather than making its own.
	ctx := context.Background()
	if err := run(ctx, os.Stdin, os.Stdout, os.Stderr, time.Now()); err != nil {
		fmt.Fprintln(os.Stderr, "notes:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, in io.Reader, out, problems io.Writer, now time.Time) error {
	notes := store.NewMemory()

	scanner := bufio.NewScanner(in)
	// Bound the line length here as well as in the parser: without a limit
	// bufio grows its buffer until the input decides we are out of memory.
	scanner.Buffer(make([]byte, 0, 4096), note.MaxLineBytes+1)

	rejected := 0
	for line := 1; scanner.Scan(); line++ {
		text := strings.TrimRight(scanner.Text(), "\r")
		if strings.TrimSpace(text) == "" {
			continue
		}
		n, err := note.ParseLine(text, now)
		if err != nil {
			rejected++
			fmt.Fprintf(problems, "line %d rejected: %v\n", line, err)
			continue
		}
		if err := notes.Add(n); err != nil {
			rejected++
			fmt.Fprintf(problems, "line %d rejected: %v\n", line, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	var listing strings.Builder
	for _, n := range notes.List("") {
		fmt.Fprintln(&listing, n)
	}
	if _, err := io.WriteString(out, listing.String()); err != nil {
		return fmt.Errorf("write listing: %w", err)
	}
	if rejected > 0 {
		fmt.Fprintf(problems, "%d line(s) rejected\n", rejected)
	}

	if url := os.Getenv(webhookEnv); url != "" {
		if err := remote.Publish(ctx, remote.NewClient(), url, []byte(listing.String())); err != nil {
			return fmt.Errorf("publish listing: %w", err)
		}
	}
	return nil
}

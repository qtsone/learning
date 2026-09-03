package main

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func mustBeYes(answer string) error {
	if answer != "yes" {
		return fmt.Errorf("%q is not an answer", answer)
	}
	return nil
}

func TestAskReadsAnAnswer(t *testing.T) {
	cases := []struct {
		name     string
		question string
		def      string
		input    string
		wantOut  string
		want     string
	}{
		{
			name:     "typed answer wins",
			question: "Region",
			def:      "eu-west-1",
			input:    "us-east-1\n",
			wantOut:  "Region [eu-west-1]: ",
			want:     "us-east-1",
		},
		{
			name:     "surrounding spaces are trimmed",
			question: "Region",
			def:      "eu-west-1",
			input:    "   us-east-1  \n",
			wantOut:  "Region [eu-west-1]: ",
			want:     "us-east-1",
		},
		{
			name:     "empty answer takes the default",
			question: "Region",
			def:      "eu-west-1",
			input:    "\n",
			wantOut:  "Region [eu-west-1]: ",
			want:     "eu-west-1",
		},
		{
			name:     "end of input counts as an empty answer",
			question: "Region",
			def:      "eu-west-1",
			input:    "",
			wantOut:  "Region [eu-west-1]: ",
			want:     "eu-west-1",
		},
		{
			name:     "no default, no brackets",
			question: "Name",
			def:      "",
			input:    "ada\n",
			wantOut:  "Name: ",
			want:     "ada",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var out bytes.Buffer
			p := NewPrompter(strings.NewReader(c.input), &out, true)
			got, err := p.Ask(c.question, c.def, nil)
			if err != nil {
				t.Fatalf("Ask returned %v, want nil", err)
			}
			if got != c.want {
				t.Errorf("answer = %q, want %q", got, c.want)
			}
			if out.String() != c.wantOut {
				t.Errorf("prompt = %q, want %q", out.String(), c.wantOut)
			}
		})
	}
}

func TestAskRetriesInvalidAnswers(t *testing.T) {
	var out bytes.Buffer
	p := NewPrompter(strings.NewReader("maybe\nnope\nyes\n"), &out, true)
	got, err := p.Ask("Continue? (yes/no)", "", mustBeYes)
	if err != nil {
		t.Fatalf("Ask returned %v, want nil", err)
	}
	if got != "yes" {
		t.Errorf("answer = %q, want %q", got, "yes")
	}
	want := "Continue? (yes/no): " + "error: \"maybe\" is not an answer\n" +
		"Continue? (yes/no): " + "error: \"nope\" is not an answer\n" +
		"Continue? (yes/no): "
	if out.String() != want {
		t.Errorf("prompt =\n%q\nwant\n%q", out.String(), want)
	}
}

func TestAskGivesUpAfterMaxAttempts(t *testing.T) {
	var out bytes.Buffer
	p := NewPrompter(strings.NewReader("a\nb\nc\nyes\n"), &out, true)
	_, err := p.Ask("Continue? (yes/no)", "", mustBeYes)
	if !errors.Is(err, ErrTooManyAttempts) {
		t.Fatalf("Ask error = %v, want ErrTooManyAttempts", err)
	}
	if n := strings.Count(out.String(), "Continue?"); n != MaxPromptAttempts {
		t.Errorf("asked %d times, want %d", n, MaxPromptAttempts)
	}
	if n := strings.Count(out.String(), "error: "); n != MaxPromptAttempts {
		t.Errorf("reported %d validation errors, want %d", n, MaxPromptAttempts)
	}
}

func TestAskWhenNotInteractive(t *testing.T) {
	t.Run("default is used without touching the streams", func(t *testing.T) {
		var out bytes.Buffer
		p := NewPrompter(strings.NewReader("typed\n"), &out, false)
		got, err := p.Ask("Region", "eu-west-1", nil)
		if err != nil {
			t.Fatalf("Ask returned %v, want nil", err)
		}
		if got != "eu-west-1" {
			t.Errorf("answer = %q, want the default %q", got, "eu-west-1")
		}
		if out.Len() != 0 {
			t.Errorf("prompt = %q, want nothing written when there is no terminal", out.String())
		}
	})

	t.Run("no default is a clear error", func(t *testing.T) {
		var out bytes.Buffer
		p := NewPrompter(strings.NewReader("typed\n"), &out, false)
		_, err := p.Ask("Region", "", nil)
		if !errors.Is(err, ErrNotInteractive) {
			t.Fatalf("Ask error = %v, want ErrNotInteractive", err)
		}
		if out.Len() != 0 {
			t.Errorf("prompt = %q, want nothing written when there is no terminal", out.String())
		}
	})

	t.Run("end of input without a default is the same case", func(t *testing.T) {
		var out bytes.Buffer
		p := NewPrompter(strings.NewReader(""), &out, true)
		_, err := p.Ask("Region", "", nil)
		if !errors.Is(err, ErrNotInteractive) {
			t.Fatalf("Ask error = %v, want ErrNotInteractive", err)
		}
	})
}

func TestAskKeepsItsPlaceBetweenQuestions(t *testing.T) {
	var out bytes.Buffer
	p := NewPrompter(strings.NewReader("alice\nbob\n"), &out, true)

	first, err := p.Ask("First", "", nil)
	if err != nil {
		t.Fatalf("first Ask returned %v, want nil", err)
	}
	second, err := p.Ask("Second", "", nil)
	if err != nil {
		t.Fatalf("second Ask returned %v, want nil", err)
	}
	if first != "alice" || second != "bob" {
		t.Errorf("answers = %q, %q; want %q, %q — a reader created per question "+
			"loses the bytes it already buffered", first, second, "alice", "bob")
	}
	if want := "First: Second: "; out.String() != want {
		t.Errorf("prompts = %q, want %q", out.String(), want)
	}
}

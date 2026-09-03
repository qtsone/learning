package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestProgressOnTerminal(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf, true, "checking", 10)
	p.Update(3)
	p.Update(10)
	p.Done()

	want := "\rchecking: 3/10 (30%)\x1b[K" +
		"\rchecking: 10/10 (100%)\x1b[K" +
		"\rchecking: done (10/10)\x1b[K\n"
	if buf.String() != want {
		t.Errorf("progress =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestProgressWhenPiped(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf, false, "checking", 10)
	p.Update(3)
	p.Update(10)
	if buf.Len() != 0 {
		t.Errorf("after two updates piped output = %q, want nothing before Done", buf.String())
	}
	p.Done()

	want := "checking: done (10/10)\n"
	if buf.String() != want {
		t.Errorf("progress = %q, want %q", buf.String(), want)
	}
	if strings.ContainsAny(buf.String(), "\r\x1b") {
		t.Errorf("piped progress = %q, want no carriage returns or escape sequences", buf.String())
	}
}

func TestProgressUnknownTotal(t *testing.T) {
	var tty, piped bytes.Buffer
	pt := NewProgress(&tty, true, "reading", 0)
	pt.Update(7)
	pt.Done()
	if want := "\rreading: 7\x1b[K" + "\rreading: done (7)\x1b[K\n"; tty.String() != want {
		t.Errorf("terminal progress =\n%q\nwant\n%q", tty.String(), want)
	}

	pp := NewProgress(&piped, false, "reading", 0)
	pp.Update(7)
	pp.Done()
	if want := "reading: done (7)\n"; piped.String() != want {
		t.Errorf("piped progress = %q, want %q", piped.String(), want)
	}
}

func TestProgressPercentIsFloored(t *testing.T) {
	cases := []struct {
		n, total int
		want     string
	}{
		{1, 3, "\rwork: 1/3 (33%)\x1b[K"},
		{2, 3, "\rwork: 2/3 (66%)\x1b[K"},
		{0, 4, "\rwork: 0/4 (0%)\x1b[K"},
	}
	for _, c := range cases {
		var buf bytes.Buffer
		p := NewProgress(&buf, true, "work", c.total)
		p.Update(c.n)
		if buf.String() != c.want {
			t.Errorf("Update(%d) of %d = %q, want %q", c.n, c.total, buf.String(), c.want)
		}
	}
}

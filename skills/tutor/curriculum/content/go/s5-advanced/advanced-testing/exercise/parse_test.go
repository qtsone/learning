package logkit

import (
	"errors"
	"fmt"
	"testing"
)

func TestFormat(t *testing.T) {
	cases := []struct {
		name string
		in   Event
		want string
	}{
		{"plain", Event{"info", "api", "started"}, `info|api|started`},
		{"empty fields", Event{"debug", "", ""}, `debug||`},
		{"pipe in message", Event{"error", "db", "a|b"}, `error|db|a\|b`},
		{"backslash in source", Event{"warn", `c:\tmp`, "x"}, `warn|c:\\tmp|x`},
		{"newline in message", Event{"info", "api", "one\ntwo"}, `info|api|one\ntwo`},
		{"carriage return", Event{"info", "api", "one\rtwo"}, `info|api|one\rtwo`},
		{"utf8 survives", Event{"info", "api", "héllo → ok"}, `info|api|héllo → ok`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Format(c.in); got != c.want {
				t.Errorf("Format(%#v) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestParseLine(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want Event
	}{
		{"plain", `info|api|started`, Event{"info", "api", "started"}},
		{"empty fields", `debug||`, Event{"debug", "", ""}},
		{"escaped pipe", `error|db|a\|b`, Event{"error", "db", "a|b"}},
		{"escaped backslash", `warn|c:\\tmp|x`, Event{"warn", `c:\tmp`, "x"}},
		{"escaped newline", `info|api|one\ntwo`, Event{"info", "api", "one\ntwo"}},
		{"escaped cr", `info|api|one\rtwo`, Event{"info", "api", "one\rtwo"}},
		{"escape at field end", `info|a\||b`, Event{"info", "a|", "b"}},
		{"utf8 survives", `info|api|héllo → ok`, Event{"info", "api", "héllo → ok"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ParseLine(c.in)
			if err != nil {
				t.Fatalf("ParseLine(%q) returned error %v, want %#v", c.in, err, c.want)
			}
			if got != c.want {
				t.Errorf("ParseLine(%q) = %#v, want %#v", c.in, got, c.want)
			}
		})
	}
}

func TestParseLineRejects(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"empty input", ``},
		{"too few fields", `info|api`},
		{"too many fields", `info|api|a|b`},
		{"unknown level", `trace|api|x`},
		{"empty level", `|api|x`},
		{"dangling escape", `info|api|x\`},
		{"unknown escape", `info|api|x\tz`},
		{"raw newline", "info|api|x\ny"},
		{"raw carriage return", "info|api|x\ry"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ParseLine(c.in)
			if err == nil {
				t.Fatalf("ParseLine(%q) = %#v, want an error", c.in, got)
			}
			if !errors.Is(err, ErrMalformed) {
				t.Errorf("ParseLine(%q) error %v does not match ErrMalformed", c.in, err)
			}
		})
	}
}

func TestFormatParseRoundTrip(t *testing.T) {
	events := []Event{
		{"info", "api", "started"},
		{"debug", "", ""},
		{"error", "db", `pipes | and \ backslashes`},
		{"warn", `we\|rd`, "multi\nline\r\nmessage"},
		{"info", "api", `\\|\\`},
		{"error", "worker", "héllo → ok"},
	}
	for i, ev := range events {
		t.Run(fmt.Sprintf("event-%d", i), func(t *testing.T) {
			line := Format(ev)
			got, err := ParseLine(line)
			if err != nil {
				t.Fatalf("ParseLine(Format(%#v)) = error %v; the encoding must be decodable", ev, err)
			}
			if got != ev {
				t.Errorf("round trip through %q gave %#v, want %#v", line, got, ev)
			}
		})
	}
}

func TestLevelRank(t *testing.T) {
	if rank, ok := LevelRank("warn"); !ok || rank != 2 {
		t.Errorf(`LevelRank("warn") = %d, %v; want 2, true`, rank, ok)
	}
	if _, ok := LevelRank("trace"); ok {
		t.Error(`LevelRank("trace") reported a known level; want false`)
	}
}

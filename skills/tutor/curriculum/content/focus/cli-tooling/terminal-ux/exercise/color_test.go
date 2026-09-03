package main

import "testing"

// envOf builds an EnvLookup over a fixed map, so a test can describe the
// environment without touching the real one.
func envOf(vars map[string]string) EnvLookup {
	return func(key string) (string, bool) {
		v, ok := vars[key]
		return v, ok
	}
}

func TestParseColorMode(t *testing.T) {
	cases := []struct {
		in      string
		want    ColorMode
		wantErr bool
	}{
		{"auto", ColorAuto, false},
		{"always", ColorAlways, false},
		{"never", ColorNever, false},
		{"", ColorAuto, true},
		{"yes", ColorAuto, true},
		{"Always", ColorAuto, true},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got, err := ParseColorMode(c.in)
			if (err != nil) != c.wantErr {
				t.Fatalf("ParseColorMode(%q) error = %v, want error: %v", c.in, err, c.wantErr)
			}
			if err == nil && got != c.want {
				t.Errorf("ParseColorMode(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

func TestResolveColor(t *testing.T) {
	cases := []struct {
		name string
		mode ColorMode
		tty  bool
		vars map[string]string
		want bool
	}{
		{"auto on a terminal", ColorAuto, true, nil, true},
		{"auto when piped", ColorAuto, false, nil, false},
		{"auto with TERM set normally", ColorAuto, true, map[string]string{"TERM": "xterm-256color"}, true},
		{"NO_COLOR vetoes auto", ColorAuto, true, map[string]string{"NO_COLOR": "1"}, false},
		{"NO_COLOR with any value vetoes", ColorAuto, true, map[string]string{"NO_COLOR": "no"}, false},
		{"NO_COLOR set but empty is not a veto", ColorAuto, true, map[string]string{"NO_COLOR": ""}, true},
		{"dumb terminal cannot render escapes", ColorAuto, true, map[string]string{"TERM": "dumb"}, false},
		{"always beats a pipe", ColorAlways, false, nil, true},
		{"always beats NO_COLOR", ColorAlways, false, map[string]string{"NO_COLOR": "1"}, true},
		{"always beats a dumb terminal", ColorAlways, true, map[string]string{"TERM": "dumb"}, true},
		{"never beats a terminal", ColorNever, true, nil, false},
		{"never beats an empty NO_COLOR", ColorNever, true, map[string]string{"NO_COLOR": ""}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ResolveColor(c.mode, c.tty, envOf(c.vars))
			if got != c.want {
				t.Errorf("ResolveColor(%v, isTerminal=%v, %v) = %v, want %v",
					c.mode, c.tty, c.vars, got, c.want)
			}
		})
	}
}

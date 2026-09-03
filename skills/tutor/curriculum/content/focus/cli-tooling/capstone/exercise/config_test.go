package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// resolve parses args against a fresh tree and resolves the settings for the
// command they select — without running it. Find picks the command, ParseFlags
// fills its flag set (including the persistent flags inherited from root).
func resolve(t *testing.T, app *App, args ...string) (Settings, error) {
	t.Helper()
	root := NewRootCmd(app)
	target, rest, err := root.Find(args)
	if err != nil {
		t.Fatalf("Find(%q): %v", args, err)
	}
	if err := target.ParseFlags(rest); err != nil {
		t.Fatalf("ParseFlags(%q): %v", rest, err)
	}
	return app.Resolve(target)
}

// writeConfig writes a config file at the app's default location.
func writeConfig(t *testing.T, app *App, body string) string {
	t.Helper()
	path := filepath.Join(app.ConfigDir, "scout", "config.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

// writeConfigAt writes a config file anywhere and returns its path.
func writeConfigAt(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "scout.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestResolveDefaults(t *testing.T) {
	app := newApp(t, nil)
	set, err := resolve(t, app, "scan")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if set.Top != DefaultTop || set.Limit != DefaultLimit {
		t.Errorf("Top/Limit = %d/%d, want %d/%d", set.Top, set.Limit, DefaultTop, DefaultLimit)
	}
	if set.JSON {
		t.Error("JSON = true, want false by default")
	}
	if set.Color {
		t.Error("Color = true, want false: stdout is not a terminal in tests")
	}
	if !slices.Equal(set.Ignore, DefaultIgnore) {
		t.Errorf("Ignore = %v, want %v", set.Ignore, DefaultIgnore)
	}
	// A missing file at the *default* location is the normal case, not an
	// error — and it must be this app's location, not a fresh one, or the
	// check proves nothing about the values resolved above.
	if _, err := os.Stat(filepath.Join(app.ConfigDir, "scout", "config.json")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("test setup: the default config path should not exist")
	}
}

func TestResolvePrecedence(t *testing.T) {
	cases := []struct {
		name   string
		config string            // written to the default location when non-empty
		env    map[string]string //
		args   []string
		want   Settings
	}{
		{
			name: "config file beats the defaults",
			config: `{"top": 3, "limit": 4, "json": true,
			          "ignore": ["dist"], "color": "never"}`,
			args: []string{"scan"},
			want: Settings{Top: 3, Limit: 4, JSON: true, Ignore: []string{"dist"}},
		},
		{
			name:   "environment beats the config file",
			config: `{"top": 3, "ignore": ["dist"]}`,
			env:    map[string]string{"SCOUT_TOP": "5", "SCOUT_IGNORE": "out, tmp"},
			args:   []string{"scan"},
			want:   Settings{Top: 5, Limit: DefaultLimit, Ignore: []string{"out", "tmp"}},
		},
		{
			name:   "flag beats the environment",
			config: `{"top": 3}`,
			env:    map[string]string{"SCOUT_TOP": "5"},
			args:   []string{"scan", "--top", "7"},
			want:   Settings{Top: 7, Limit: DefaultLimit, Ignore: DefaultIgnore},
		},
		{
			name: "a flag left alone does not overwrite the environment",
			env:  map[string]string{"SCOUT_TOP": "5"},
			args: []string{"scan"},
			want: Settings{Top: 5, Limit: DefaultLimit, Ignore: DefaultIgnore},
		},
		{
			name: "a flag typed at its default value still wins",
			env:  map[string]string{"SCOUT_TOP": "5"},
			args: []string{"scan", "--top", "10"},
			want: Settings{Top: DefaultTop, Limit: DefaultLimit, Ignore: DefaultIgnore},
		},
		{
			name: "an explicit false flag beats a true environment",
			env:  map[string]string{"SCOUT_JSON": "true"},
			args: []string{"scan", "--json=false"},
			want: Settings{Top: DefaultTop, Limit: DefaultLimit, JSON: false, Ignore: DefaultIgnore},
		},
		{
			name: "environment turns JSON on",
			env:  map[string]string{"SCOUT_JSON": "1"},
			args: []string{"scan"},
			want: Settings{Top: DefaultTop, Limit: DefaultLimit, JSON: true, Ignore: DefaultIgnore},
		},
		{
			name: "authors sees its own --limit and not scan's --top",
			env:  map[string]string{"SCOUT_LIMIT": "2"},
			args: []string{"authors", "--limit", "3"},
			want: Settings{Top: DefaultTop, Limit: 3, Ignore: DefaultIgnore},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			app := newApp(t, c.env)
			if c.config != "" {
				writeConfig(t, app, c.config)
			}
			got, err := resolve(t, app, c.args...)
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if got.Top != c.want.Top || got.Limit != c.want.Limit || got.JSON != c.want.JSON {
				t.Errorf("Top/Limit/JSON = %d/%d/%v, want %d/%d/%v",
					got.Top, got.Limit, got.JSON, c.want.Top, c.want.Limit, c.want.JSON)
			}
			if !slices.Equal(got.Ignore, c.want.Ignore) {
				t.Errorf("Ignore = %v, want %v", got.Ignore, c.want.Ignore)
			}
		})
	}
}

func TestConfigFileLocation(t *testing.T) {
	t.Run("SCOUT_CONFIG points at a file", func(t *testing.T) {
		path := writeConfigAt(t, `{"top": 2}`)
		app := newApp(t, map[string]string{"SCOUT_CONFIG": path})
		writeConfig(t, app, `{"top": 9}`) // the default location must lose
		got, err := resolve(t, app, "scan")
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if got.Top != 2 {
			t.Errorf("Top = %d, want 2 from SCOUT_CONFIG", got.Top)
		}
	})

	t.Run("--config beats SCOUT_CONFIG", func(t *testing.T) {
		flagPath := writeConfigAt(t, `{"top": 1}`)
		envPath := writeConfigAt(t, `{"top": 2}`)
		app := newApp(t, map[string]string{"SCOUT_CONFIG": envPath})
		got, err := resolve(t, app, "scan", "--config", flagPath)
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if got.Top != 1 {
			t.Errorf("Top = %d, want 1 from --config", got.Top)
		}
	})

	t.Run("a named file that does not exist is an error", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "nope.json")
		_, err := resolve(t, newApp(t, nil), "scan", "--config", missing)
		if !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("err = %v, want one matching fs.ErrNotExist", err)
		}
		if !errors.Is(err, ErrUsage) {
			t.Errorf("err = %v, want one matching ErrUsage: the user named a file that is not there", err)
		}
	})

	t.Run("no config directory at all is fine", func(t *testing.T) {
		app := newApp(t, nil)
		app.ConfigDir = ""
		if _, err := resolve(t, app, "scan"); err != nil {
			t.Errorf("Resolve: %v, want defaults when there is nowhere to look", err)
		}
	})
}

func TestLoadConfigFile(t *testing.T) {
	t.Run("missing file returns fs.ErrNotExist unchanged", func(t *testing.T) {
		_, err := LoadConfigFile(filepath.Join(t.TempDir(), "nope.json"))
		if !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("err = %v, want one matching fs.ErrNotExist", err)
		}
	})

	t.Run("unknown key is rejected", func(t *testing.T) {
		path := writeConfigAt(t, `{"tpo": 3}`)
		_, err := LoadConfigFile(path)
		if err == nil {
			t.Fatal("no error for an unknown config key: a typo must not be silently ignored")
		}
		if !errors.Is(err, ErrUsage) || !strings.Contains(err.Error(), "tpo") {
			t.Errorf("err = %v, want a usage error naming the key %q", err, "tpo")
		}
	})

	t.Run("malformed JSON names the file", func(t *testing.T) {
		path := writeConfigAt(t, `{"top": `)
		_, err := LoadConfigFile(path)
		if err == nil || !strings.Contains(err.Error(), path) {
			t.Errorf("err = %v, want an error naming %s", err, path)
		}
	})

	t.Run("absent keys stay absent", func(t *testing.T) {
		fc, err := LoadConfigFile(writeConfigAt(t, `{"json": false}`))
		if err != nil {
			t.Fatalf("LoadConfigFile: %v", err)
		}
		if fc.JSON == nil || *fc.JSON {
			t.Error(`"json": false must decode to a pointer to false, not nil`)
		}
		if fc.Top != nil {
			t.Error("a key that is not in the file must decode to nil, so it cannot overwrite a lower layer")
		}
	})
}

func TestResolveErrors(t *testing.T) {
	cases := []struct {
		name    string
		config  string
		env     map[string]string
		args    []string
		wantMsg string
	}{
		{
			name:    "unparseable SCOUT_TOP",
			env:     map[string]string{"SCOUT_TOP": "abc"},
			args:    []string{"scan"},
			wantMsg: `invalid SCOUT_TOP "abc"`,
		},
		{
			name:    "negative --top",
			args:    []string{"scan", "--top", "-1"},
			wantMsg: "invalid --top -1",
		},
		{
			name:    "negative value in the config file",
			config:  `{"top": -2}`,
			args:    []string{"scan"},
			wantMsg: "invalid top in",
		},
		{
			name:    "unknown colour policy",
			args:    []string{"scan", "--color", "purple"},
			wantMsg: `invalid color "purple"`,
		},
		{
			name:    "unknown colour policy from the environment",
			env:     map[string]string{"SCOUT_COLOR": "beige"},
			args:    []string{"scan"},
			wantMsg: `invalid color "beige"`,
		},
		{
			name:    "unparseable SCOUT_JSON",
			env:     map[string]string{"SCOUT_JSON": "maybe"},
			args:    []string{"scan"},
			wantMsg: `invalid SCOUT_JSON "maybe"`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			app := newApp(t, c.env)
			if c.config != "" {
				writeConfig(t, app, c.config)
			}
			_, err := resolve(t, app, c.args...)
			if err == nil {
				t.Fatalf("no error, want one containing %q", c.wantMsg)
			}
			if !strings.Contains(err.Error(), c.wantMsg) {
				t.Errorf("err = %v, want one containing %q", err, c.wantMsg)
			}
			if !errors.Is(err, ErrUsage) {
				t.Errorf("err = %v, want one matching ErrUsage so exitCode can return 2", err)
			}
		})
	}
}

func TestResolveColorPolicy(t *testing.T) {
	cases := []struct {
		name       string
		mode       ColorMode
		isTerminal bool
		env        map[string]string
		want       bool
	}{
		{"never wins over everything", ColorNever, true, nil, false},
		{"always beats NO_COLOR", ColorAlways, false, map[string]string{"NO_COLOR": "1"}, true},
		{"auto on a terminal", ColorAuto, true, nil, true},
		{"auto off a terminal", ColorAuto, false, nil, false},
		{"NO_COLOR vetoes auto", ColorAuto, true, map[string]string{"NO_COLOR": "1"}, false},
		{"empty NO_COLOR is not a veto", ColorAuto, true, map[string]string{"NO_COLOR": ""}, true},
		{"TERM=dumb vetoes auto", ColorAuto, true, map[string]string{"TERM": "dumb"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			getenv := func(k string) string { return c.env[k] }
			if got := ResolveColor(c.mode, c.isTerminal, getenv); got != c.want {
				t.Errorf("ResolveColor(%v, terminal=%v, %v) = %v, want %v",
					c.mode, c.isTerminal, c.env, got, c.want)
			}
		})
	}
}

func TestJSONForcesColorOff(t *testing.T) {
	app := newApp(t, nil)
	app.StdoutIsTerminal = true

	got, err := resolve(t, app, "scan", "--json", "--color", "always")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Color {
		t.Error("Color = true with --json: machine-readable output is never styled, whatever the policy says")
	}
}

func TestExitCode(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"success", nil, 0},
		{"usage", ErrUsage, 2},
		{"wrapped usage", fmt.Errorf("scan: %w", ErrUsage), 2},
		{"cancelled", context.Canceled, 130},
		{"wrapped cancellation", fmt.Errorf("walk: %w", context.Canceled), 130},
		{"deadline", context.DeadlineExceeded, 130},
		{"wrapped deadline", fmt.Errorf("scan: %w", context.DeadlineExceeded), 130},
		{"vcs failure", ErrVCS, 1},
		{"anything else", errors.New("boom"), 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := exitCode(c.err); got != c.want {
				t.Errorf("exitCode(%v) = %d, want %d", c.err, got, c.want)
			}
		})
	}
}

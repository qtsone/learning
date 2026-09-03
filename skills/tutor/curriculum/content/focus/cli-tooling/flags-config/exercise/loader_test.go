package main

import (
	"bytes"
	"errors"
	"flag"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

const fileJSON = `{
  "endpoint": "https://file.example",
  "timeout": "1s",
  "retries": 1,
  "verbose": true,
  "tags": ["file-a", "file-b"]
}`

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "toolkit.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("writing config file: %v", err)
	}
	return path
}

// envFrom builds a LookupEnv function over a map, so no test ever depends on
// (or disturbs) the real process environment.
func envFrom(vars map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		v, ok := vars[key]
		return v, ok
	}
}

func fullEnv() map[string]string {
	return map[string]string{
		"TOOLKIT_ENDPOINT": "https://env.example",
		"TOOLKIT_TIMEOUT":  "2s",
		"TOOLKIT_RETRIES":  "2",
		"TOOLKIT_VERBOSE":  "false",
		"TOOLKIT_TAGS":     "env-a, env-b",
	}
}

func fullArgs() []string {
	return []string{
		"-endpoint", "https://flag.example",
		"-timeout", "3s",
		"-retries", "4",
		"-verbose=true",
		"-tag", "flag-a",
		"-tag", "flag-b",
	}
}

func TestLoadPrecedence(t *testing.T) {
	cases := []struct {
		name        string
		withFile    bool
		env         map[string]string
		args        []string
		want        Config
		wantOrigins map[string]Source
	}{
		{
			name: "defaults when nothing else is present",
			want: Config{
				Endpoint: "https://api.example.com",
				Timeout:  5 * time.Second,
				Retries:  3,
			},
			wantOrigins: map[string]Source{
				"endpoint": SourceDefault, "timeout": SourceDefault,
				"retries": SourceDefault, "verbose": SourceDefault,
				"tags": SourceDefault,
			},
		},
		{
			name:     "config file beats defaults",
			withFile: true,
			want: Config{
				Endpoint: "https://file.example",
				Timeout:  time.Second,
				Retries:  1,
				Verbose:  true,
				Tags:     []string{"file-a", "file-b"},
			},
			wantOrigins: map[string]Source{
				"endpoint": SourceFile, "timeout": SourceFile,
				"retries": SourceFile, "verbose": SourceFile,
				"tags": SourceFile,
			},
		},
		{
			name: "environment beats defaults",
			env:  fullEnv(),
			want: Config{
				Endpoint: "https://env.example",
				Timeout:  2 * time.Second,
				Retries:  2,
				Verbose:  false,
				Tags:     []string{"env-a", "env-b"},
			},
			wantOrigins: map[string]Source{
				"endpoint": SourceEnv, "timeout": SourceEnv,
				"retries": SourceEnv, "verbose": SourceEnv,
				"tags": SourceEnv,
			},
		},
		{
			name: "flags beat defaults",
			args: fullArgs(),
			want: Config{
				Endpoint: "https://flag.example",
				Timeout:  3 * time.Second,
				Retries:  4,
				Verbose:  true,
				Tags:     []string{"flag-a", "flag-b"},
			},
			wantOrigins: map[string]Source{
				"endpoint": SourceFlag, "timeout": SourceFlag,
				"retries": SourceFlag, "verbose": SourceFlag,
				"tags": SourceFlag,
			},
		},
		{
			name:     "environment beats the config file",
			withFile: true,
			env: map[string]string{
				"TOOLKIT_ENDPOINT": "https://env.example",
				"TOOLKIT_RETRIES":  "2",
			},
			want: Config{
				Endpoint: "https://env.example",
				Timeout:  time.Second,
				Retries:  2,
				Verbose:  true,
				Tags:     []string{"file-a", "file-b"},
			},
			wantOrigins: map[string]Source{
				"endpoint": SourceEnv, "timeout": SourceFile,
				"retries": SourceEnv, "verbose": SourceFile,
				"tags": SourceFile,
			},
		},
		{
			name:     "flags beat the environment and the config file",
			withFile: true,
			env:      fullEnv(),
			args:     []string{"-timeout", "3s", "-verbose=true"},
			want: Config{
				Endpoint: "https://env.example",
				Timeout:  3 * time.Second,
				Retries:  2,
				Verbose:  true,
				Tags:     []string{"env-a", "env-b"},
			},
			wantOrigins: map[string]Source{
				"endpoint": SourceEnv, "timeout": SourceFlag,
				"retries": SourceEnv, "verbose": SourceFlag,
				"tags": SourceEnv,
			},
		},
		{
			name: "a value equal to the default still records its real origin",
			env:  map[string]string{"TOOLKIT_RETRIES": "3"},
			want: Config{
				Endpoint: "https://api.example.com",
				Timeout:  5 * time.Second,
				Retries:  3,
			},
			wantOrigins: map[string]Source{
				"endpoint": SourceDefault, "timeout": SourceDefault,
				"retries": SourceEnv, "verbose": SourceDefault,
				"tags": SourceDefault,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			loader := Loader{
				Args:      c.args,
				LookupEnv: envFrom(c.env),
				Stderr:    io.Discard,
			}
			if c.withFile {
				loader.ConfigPath = writeConfig(t, fileJSON)
			}

			res, err := loader.Load()
			if err != nil {
				t.Fatalf("Load() = %v, want nil", err)
			}
			if !reflect.DeepEqual(res.Config, c.want) {
				t.Errorf("Config =\n  %+v\nwant\n  %+v", res.Config, c.want)
			}
			if !reflect.DeepEqual(res.Origins, c.wantOrigins) {
				t.Errorf("Origins = %v, want %v", res.Origins, c.wantOrigins)
			}
		})
	}
}

func TestLoadFlagCountsOnlyWhenPresent(t *testing.T) {
	path := writeConfig(t, `{"verbose": true, "retries": 7}`)

	// -verbose=false and -retries=0 carry the zero value, but they were typed,
	// so they must win over the config file.
	res, err := Loader{
		Args:       []string{"-verbose=false", "-retries=0"},
		ConfigPath: path,
		Stderr:     io.Discard,
	}.Load()
	if err != nil {
		t.Fatalf("Load() = %v, want nil", err)
	}
	if res.Config.Verbose {
		t.Error("Verbose = true, want false: -verbose=false must beat the config file")
	}
	if res.Origins["verbose"] != SourceFlag {
		t.Errorf("Origins[verbose] = %v, want %v", res.Origins["verbose"], SourceFlag)
	}
	if res.Config.Retries != 0 {
		t.Errorf("Retries = %d, want 0: -retries=0 must beat the config file", res.Config.Retries)
	}
	if res.Origins["retries"] != SourceFlag {
		t.Errorf("Origins[retries] = %v, want %v", res.Origins["retries"], SourceFlag)
	}

	// Without the flags, the same file wins.
	res, err = Loader{ConfigPath: path, Stderr: io.Discard}.Load()
	if err != nil {
		t.Fatalf("Load() = %v, want nil", err)
	}
	if !res.Config.Verbose || res.Config.Retries != 7 {
		t.Errorf("Config = %+v, want verbose true and retries 7 from the file", res.Config)
	}
}

func TestLoadTagsReplaceLowerLayers(t *testing.T) {
	path := writeConfig(t, `{"tags": ["file-a", "file-b"]}`)

	res, err := Loader{
		Args:       []string{"-tag", "only"},
		LookupEnv:  envFrom(map[string]string{"TOOLKIT_TAGS": "env-a,env-b"}),
		ConfigPath: path,
		Stderr:     io.Discard,
	}.Load()
	if err != nil {
		t.Fatalf("Load() = %v, want nil", err)
	}
	want := []string{"only"}
	if !reflect.DeepEqual(res.Config.Tags, want) {
		t.Errorf("Tags = %v, want %v (a higher layer replaces the list, it does not append)",
			res.Config.Tags, want)
	}
}

func TestLoadConfigFileMissing(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "absent.json")

	// A default path that does not exist is not an error: the tool must run
	// with no config file at all.
	res, err := Loader{ConfigPath: missing, Stderr: io.Discard}.Load()
	if err != nil {
		t.Fatalf("Load() with a missing default config = %v, want nil", err)
	}
	if res.Origins["retries"] != SourceDefault {
		t.Errorf("Origins[retries] = %v, want %v", res.Origins["retries"], SourceDefault)
	}

	// A file the user explicitly named is different: silence would hide a typo.
	_, err = Loader{Args: []string{"-config", missing}, Stderr: io.Discard}.Load()
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Load() with -config pointing at a missing file = %v, want fs.ErrNotExist", err)
	}
}

func TestLoadConfigFlagSelectsFile(t *testing.T) {
	fallback := writeConfig(t, `{"retries": 1}`)
	chosen := writeConfig(t, `{"retries": 9}`)

	res, err := Loader{
		Args:       []string{"-config", chosen},
		ConfigPath: fallback,
		Stderr:     io.Discard,
	}.Load()
	if err != nil {
		t.Fatalf("Load() = %v, want nil", err)
	}
	if res.Config.Retries != 9 {
		t.Errorf("Retries = %d, want 9 from the file named by -config", res.Config.Retries)
	}
}

func TestLoadConfigFileUnknownKey(t *testing.T) {
	path := writeConfig(t, `{"retires": 2}`)

	_, err := Loader{ConfigPath: path, Stderr: io.Discard}.Load()
	if err == nil {
		t.Fatal("Load() = nil, want an error: a misspelled key must not be ignored")
	}
	if !strings.Contains(err.Error(), "retires") {
		t.Errorf("error %q does not name the offending key %q", err, "retires")
	}
}

func TestLoadInvalidValues(t *testing.T) {
	cases := []struct {
		name       string
		file       string
		env        map[string]string
		args       []string
		wantField  string
		wantSource Source
		wantRaw    string
		wantCause  error
	}{
		{
			name:       "retries from the environment is not a number",
			env:        map[string]string{"TOOLKIT_RETRIES": "abc"},
			wantField:  "retries",
			wantSource: SourceEnv,
			wantRaw:    "abc",
		},
		{
			name:       "timeout from the environment is not a duration",
			env:        map[string]string{"TOOLKIT_TIMEOUT": "5 secs"},
			wantField:  "timeout",
			wantSource: SourceEnv,
			wantRaw:    "5 secs",
		},
		{
			name:       "verbose from the environment is not a bool",
			env:        map[string]string{"TOOLKIT_VERBOSE": "yes-please"},
			wantField:  "verbose",
			wantSource: SourceEnv,
			wantRaw:    "yes-please",
		},
		{
			name:       "tags from the environment contain an empty item",
			env:        map[string]string{"TOOLKIT_TAGS": "a,,b"},
			wantField:  "tags",
			wantSource: SourceEnv,
			wantCause:  ErrEmpty,
		},
		{
			name:       "endpoint from the environment is empty",
			env:        map[string]string{"TOOLKIT_ENDPOINT": ""},
			wantField:  "endpoint",
			wantSource: SourceEnv,
			wantCause:  ErrEmpty,
		},
		{
			name:       "timeout in the file is not a duration",
			file:       `{"timeout": "soon"}`,
			wantField:  "timeout",
			wantSource: SourceFile,
			wantRaw:    "soon",
		},
		{
			name:       "endpoint in the file uses the wrong scheme",
			file:       `{"endpoint": "ftp://files.example"}`,
			wantField:  "endpoint",
			wantSource: SourceFile,
			wantCause:  ErrScheme,
		},
		{
			name:       "timeout in the file is zero",
			file:       `{"timeout": "0s"}`,
			wantField:  "timeout",
			wantSource: SourceFile,
			wantCause:  ErrRange,
		},
		{
			name:       "retries in the file is negative",
			file:       `{"retries": -1}`,
			wantField:  "retries",
			wantSource: SourceFile,
			wantCause:  ErrRange,
		},
		{
			name:       "retries flag is above the limit",
			args:       []string{"-retries", "11"},
			wantField:  "retries",
			wantSource: SourceFlag,
			wantCause:  ErrRange,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			loader := Loader{
				Args:      c.args,
				LookupEnv: envFrom(c.env),
				Stderr:    io.Discard,
			}
			if c.file != "" {
				loader.ConfigPath = writeConfig(t, c.file)
			}

			_, err := loader.Load()
			var ve *ValueError
			if !errors.As(err, &ve) {
				t.Fatalf("Load() = %v, want a *ValueError", err)
			}
			if ve.Field != c.wantField {
				t.Errorf("Field = %q, want %q", ve.Field, c.wantField)
			}
			if ve.Source != c.wantSource {
				t.Errorf("Source = %v, want %v", ve.Source, c.wantSource)
			}
			if c.wantRaw != "" && ve.Raw != c.wantRaw {
				t.Errorf("Raw = %q, want %q", ve.Raw, c.wantRaw)
			}
			if c.wantCause != nil && !errors.Is(err, c.wantCause) {
				t.Errorf("errors.Is(%v, %v) = false, want true", err, c.wantCause)
			}
			if !strings.Contains(err.Error(), c.wantField) {
				t.Errorf("error %q does not name the setting %q", err, c.wantField)
			}
		})
	}
}

func TestLoadFlagParseErrorIsReturned(t *testing.T) {
	var stderr bytes.Buffer

	_, err := Loader{Args: []string{"-retries", "abc"}, Stderr: &stderr}.Load()
	if err == nil {
		t.Fatal("Load() = nil, want an error for a malformed flag")
	}
	if stderr.Len() == 0 {
		t.Error("nothing was written to Loader.Stderr; flag output must go to the injected writer")
	}
	if !strings.Contains(stderr.String(), "-retries") {
		t.Errorf("stderr %q does not mention the offending flag", stderr.String())
	}
}

func TestLoadUsage(t *testing.T) {
	var stderr bytes.Buffer

	_, err := Loader{Args: []string{"-h"}, Stderr: &stderr}.Load()
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("Load() with -h = %v, want flag.ErrHelp", err)
	}
	for _, want := range []string{
		"-endpoint", "-timeout", "-retries", "-verbose", "-tag", "-config",
		"TOOLKIT_ENDPOINT", "TOOLKIT_TIMEOUT", "TOOLKIT_RETRIES",
		"TOOLKIT_VERBOSE", "TOOLKIT_TAGS",
		"defaults < config file < environment < flags",
	} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("usage output does not mention %q; got:\n%s", want, stderr.String())
		}
	}
}

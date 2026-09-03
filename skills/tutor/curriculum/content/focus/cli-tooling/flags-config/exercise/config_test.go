package main

import (
	"errors"
	"testing"
	"time"
)

func TestSourceString(t *testing.T) {
	cases := []struct {
		in   Source
		want string
	}{
		{SourceDefault, "default"},
		{SourceFile, "config file"},
		{SourceEnv, "environment"},
		{SourceFlag, "flag"},
		{Source(42), "unknown"},
	}
	for _, c := range cases {
		if got := c.in.String(); got != c.want {
			t.Errorf("Source(%d).String() = %q, want %q", int(c.in), got, c.want)
		}
	}
}

func TestDefaults(t *testing.T) {
	want := Config{
		Endpoint: "https://api.example.com",
		Timeout:  5 * time.Second,
		Retries:  3,
		Verbose:  false,
		Tags:     nil,
	}
	got := Defaults()
	if got.Endpoint != want.Endpoint || got.Timeout != want.Timeout ||
		got.Retries != want.Retries || got.Verbose != want.Verbose ||
		len(got.Tags) != 0 {
		t.Errorf("Defaults() = %+v, want %+v", got, want)
	}
}

func TestResultValidate(t *testing.T) {
	valid := Config{
		Endpoint: "https://api.example.com",
		Timeout:  time.Second,
		Retries:  0,
	}

	cases := []struct {
		name       string
		config     Config
		origins    map[string]Source
		wantField  string
		wantSource Source
		wantCause  error
	}{
		{name: "valid config", config: valid, origins: map[string]Source{}},
		{
			name:       "empty endpoint",
			config:     Config{Endpoint: "  ", Timeout: time.Second},
			origins:    map[string]Source{"endpoint": SourceEnv},
			wantField:  "endpoint",
			wantSource: SourceEnv,
			wantCause:  ErrEmpty,
		},
		{
			name:       "endpoint without http scheme",
			config:     Config{Endpoint: "ftp://files.example", Timeout: time.Second},
			origins:    map[string]Source{"endpoint": SourceFile},
			wantField:  "endpoint",
			wantSource: SourceFile,
			wantCause:  ErrScheme,
		},
		{
			name:       "zero timeout",
			config:     Config{Endpoint: valid.Endpoint},
			origins:    map[string]Source{"timeout": SourceFile},
			wantField:  "timeout",
			wantSource: SourceFile,
			wantCause:  ErrRange,
		},
		{
			name:       "negative retries",
			config:     Config{Endpoint: valid.Endpoint, Timeout: time.Second, Retries: -1},
			origins:    map[string]Source{"retries": SourceFlag},
			wantField:  "retries",
			wantSource: SourceFlag,
			wantCause:  ErrRange,
		},
		{
			name:       "too many retries",
			config:     Config{Endpoint: valid.Endpoint, Timeout: time.Second, Retries: 11},
			origins:    map[string]Source{"retries": SourceEnv},
			wantField:  "retries",
			wantSource: SourceEnv,
			wantCause:  ErrRange,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res := &Result{Config: c.config, Origins: c.origins}
			err := res.Validate()
			if c.wantField == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			var ve *ValueError
			if !errors.As(err, &ve) {
				t.Fatalf("Validate() = %v, want a *ValueError", err)
			}
			if ve.Field != c.wantField {
				t.Errorf("Field = %q, want %q", ve.Field, c.wantField)
			}
			if ve.Source != c.wantSource {
				t.Errorf("Source = %v, want %v", ve.Source, c.wantSource)
			}
			if !errors.Is(err, c.wantCause) {
				t.Errorf("errors.Is(%v, %v) = false, want true", err, c.wantCause)
			}
		})
	}
}

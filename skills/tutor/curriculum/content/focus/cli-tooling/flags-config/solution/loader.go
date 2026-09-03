package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"time"
)

// Loader resolves configuration from four layers, lowest precedence first:
//
//	defaults < config file < environment < flags
//
// Every input is injected so tests can drive the loader without touching the
// real command line, the process environment, or os.Stderr.
type Loader struct {
	// Args are the command-line arguments *without* the program name.
	Args []string

	// LookupEnv reads one environment variable and reports whether it is set
	// at all — os.LookupEnv in production. A nil LookupEnv means "no
	// environment", which is what keeps tests from inheriting your shell.
	LookupEnv func(string) (string, bool)

	// ConfigPath is the config file to read when -config is not given. If it
	// does not exist it is skipped silently; a file named by -config that does
	// not exist is an error. An empty ConfigPath means "no default file".
	ConfigPath string

	// Stderr receives flag usage and flag parse errors. nil discards them.
	Stderr io.Writer
}

// fileConfig mirrors the JSON config file. Every field is a pointer so that
// "absent from the document" is distinguishable from "present and set to the
// zero value" — the JSON lesson's trick, doing real work here.
type fileConfig struct {
	Endpoint *string   `json:"endpoint"`
	Timeout  *string   `json:"timeout"`
	Retries  *int      `json:"retries"`
	Verbose  *bool     `json:"verbose"`
	Tags     *[]string `json:"tags"`
}

// rawFlags holds the flag targets. They start at their zero values on purpose:
// the flag package cannot tell "set to false" from "not set", so the values
// mean nothing until fset.Visit says the flag was actually typed.
type rawFlags struct {
	endpoint   string
	timeout    time.Duration
	retries    int
	verbose    bool
	tags       TagList
	configPath string
}

// registerFlags builds the FlagSet. Every flag is registered with a zero value
// so that fset.Visit can later tell "the user typed this" from "this is merely
// zero" — and because of that, the usage strings carry the real defaults
// themselves: flag's automatic "(default …)" would report the zero values
// registered here.
func (l Loader) registerFlags(raw *rawFlags, out io.Writer) *flag.FlagSet {
	fset := flag.NewFlagSet("toolkit", flag.ContinueOnError)
	fset.SetOutput(out)
	fset.StringVar(&raw.endpoint, "endpoint", "",
		`API endpoint (default "https://api.example.com", env `+EnvPrefix+`ENDPOINT)`)
	fset.DurationVar(&raw.timeout, "timeout", 0,
		"per-request timeout (default 5s, env "+EnvPrefix+"TIMEOUT)")
	fset.IntVar(&raw.retries, "retries", 0,
		"retries per request, 0-10 (default 3, env "+EnvPrefix+"RETRIES)")
	fset.BoolVar(&raw.verbose, "verbose", false,
		"log every request (default false, env "+EnvPrefix+"VERBOSE)")
	fset.Var(&raw.tags, "tag",
		"tag to attach, repeatable (env "+EnvPrefix+"TAGS, comma-separated)")
	fset.StringVar(&raw.configPath, "config", "",
		fmt.Sprintf("config file to read (default %q)", l.ConfigPath))
	fset.Usage = func() { writeUsage(out, fset) }
	return fset
}

// Load resolves the configuration and reports where every setting came from.
//
// It never calls os.Exit: flag parse failures and -h come back as errors
// (compare the latter with errors.Is(err, flag.ErrHelp)).
func (l Loader) Load() (*Result, error) {
	out := l.Stderr
	if out == nil {
		out = io.Discard
	}

	var raw rawFlags
	fset := l.registerFlags(&raw, out)
	if err := fset.Parse(l.Args); err != nil {
		return nil, err
	}
	set := map[string]bool{}
	fset.Visit(func(f *flag.Flag) { set[f.Name] = true })

	res := &Result{Config: Defaults(), Origins: make(map[string]Source, len(Fields))}
	for _, name := range Fields {
		res.Origins[name] = SourceDefault
	}

	if err := l.applyFile(res, raw.configPath, set["config"]); err != nil {
		return nil, err
	}
	if err := l.applyEnv(res); err != nil {
		return nil, err
	}
	applyFlags(res, &raw, set)

	if err := res.Validate(); err != nil {
		return nil, err
	}
	return res, nil
}

// applyFile overlays the JSON config file. A missing file is only an error
// when the user named it: silently ignoring -config would hide a typo, while
// insisting on the default path would break every fresh checkout.
func (l Loader) applyFile(res *Result, flagPath string, explicit bool) error {
	path := l.ConfigPath
	if explicit {
		path = flagPath
	}
	if path == "" {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) && !explicit {
			return nil
		}
		return fmt.Errorf("config file: %w", err)
	}

	var fc fileConfig
	dec := json.NewDecoder(bytes.NewReader(data))
	// A config file is not an API payload: an unknown key is a typo the user
	// wants to hear about, not a forward-compatible extension to ignore.
	dec.DisallowUnknownFields()
	if err := dec.Decode(&fc); err != nil {
		return fmt.Errorf("config file %s: %w", path, err)
	}

	if fc.Endpoint != nil {
		res.Config.Endpoint = *fc.Endpoint
		res.Origins["endpoint"] = SourceFile
	}
	if fc.Timeout != nil {
		d, err := time.ParseDuration(*fc.Timeout)
		if err != nil {
			return &ValueError{Field: "timeout", Source: SourceFile, Raw: *fc.Timeout, Err: err}
		}
		res.Config.Timeout = d
		res.Origins["timeout"] = SourceFile
	}
	if fc.Retries != nil {
		res.Config.Retries = *fc.Retries
		res.Origins["retries"] = SourceFile
	}
	if fc.Verbose != nil {
		res.Config.Verbose = *fc.Verbose
		res.Origins["verbose"] = SourceFile
	}
	if fc.Tags != nil {
		tags, err := cleanTags(*fc.Tags)
		if err != nil {
			return &ValueError{
				Field:  "tags",
				Source: SourceFile,
				Raw:    strings.Join(*fc.Tags, ","),
				Err:    err,
			}
		}
		res.Config.Tags = tags
		res.Origins["tags"] = SourceFile
	}
	return nil
}

// applyEnv overlays the TOOLKIT_* variables. LookupEnv's second return value
// is what makes this layer honest: TOOLKIT_ENDPOINT="" is a deliberate (and
// invalid) empty endpoint, not an absent variable.
func (l Loader) applyEnv(res *Result) error {
	lookup := l.LookupEnv
	if lookup == nil {
		lookup = func(string) (string, bool) { return "", false }
	}

	if v, ok := lookup(EnvPrefix + "ENDPOINT"); ok {
		res.Config.Endpoint = v
		res.Origins["endpoint"] = SourceEnv
	}
	if v, ok := lookup(EnvPrefix + "TIMEOUT"); ok {
		d, err := time.ParseDuration(v)
		if err != nil {
			return &ValueError{Field: "timeout", Source: SourceEnv, Raw: v, Err: err}
		}
		res.Config.Timeout = d
		res.Origins["timeout"] = SourceEnv
	}
	if v, ok := lookup(EnvPrefix + "RETRIES"); ok {
		n, err := strconv.Atoi(v)
		if err != nil {
			return &ValueError{Field: "retries", Source: SourceEnv, Raw: v, Err: err}
		}
		res.Config.Retries = n
		res.Origins["retries"] = SourceEnv
	}
	if v, ok := lookup(EnvPrefix + "VERBOSE"); ok {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return &ValueError{Field: "verbose", Source: SourceEnv, Raw: v, Err: err}
		}
		res.Config.Verbose = b
		res.Origins["verbose"] = SourceEnv
	}
	if v, ok := lookup(EnvPrefix + "TAGS"); ok {
		tags, err := cleanTags(strings.Split(v, ","))
		if err != nil {
			return &ValueError{Field: "tags", Source: SourceEnv, Raw: v, Err: err}
		}
		res.Config.Tags = tags
		res.Origins["tags"] = SourceEnv
	}
	return nil
}

// applyFlags overlays only the flags the user actually typed. Reading the flag
// variables directly instead would overwrite every lower layer with a zero
// value on every run.
func applyFlags(res *Result, raw *rawFlags, set map[string]bool) {
	if set["endpoint"] {
		res.Config.Endpoint = raw.endpoint
		res.Origins["endpoint"] = SourceFlag
	}
	if set["timeout"] {
		res.Config.Timeout = raw.timeout
		res.Origins["timeout"] = SourceFlag
	}
	if set["retries"] {
		res.Config.Retries = raw.retries
		res.Origins["retries"] = SourceFlag
	}
	if set["verbose"] {
		res.Config.Verbose = raw.verbose
		res.Origins["verbose"] = SourceFlag
	}
	if set["tag"] {
		// A list is one setting: the higher layer replaces it rather than
		// appending, so a user can always cut the inherited tags away.
		res.Config.Tags = []string(raw.tags)
		res.Origins["tags"] = SourceFlag
	}
}

func cleanTags(in []string) ([]string, error) {
	out := make([]string, 0, len(in))
	for _, tag := range in {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			return nil, ErrEmpty
		}
		out = append(out, tag)
	}
	return out, nil
}

// writeUsage prints the help text the user sees for -h. The flag package's
// default lists flags only, which answers a third of "how do I configure
// this?" — so print the flags, the TOOLKIT_* variables, and the precedence
// rule.
func writeUsage(w io.Writer, fset *flag.FlagSet) {
	// PrintDefaults writes to the FlagSet's own output, not to w — point it at
	// w so every part of the help text lands in one place, whoever calls this.
	fset.SetOutput(w)
	fmt.Fprint(w, "toolkit — demo client\n\nUsage:\n  toolkit [flags]\n\nFlags:\n")
	fset.PrintDefaults()

	fmt.Fprint(w, "\nEnvironment:\n")
	for _, name := range Fields {
		fmt.Fprintf(w, "  %s%s\n", EnvPrefix, strings.ToUpper(name))
	}

	fmt.Fprint(w, "\nPrecedence (lowest to highest):\n"+
		"  defaults < config file < environment < flags\n")
}

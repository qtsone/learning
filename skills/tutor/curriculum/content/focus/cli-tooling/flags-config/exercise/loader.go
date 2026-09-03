package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
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

// registerFlags is written for you — typing it out teaches nothing the two
// deliberate choices in it do not. First, every flag is registered with a zero
// value, so that fset.Visit can later tell "the user typed this" from "this is
// merely zero". Second, and because of the first, the usage strings carry the
// real defaults themselves: flag's automatic "(default …)" would report the
// zero values registered here, which would be a lie.
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
	// TODO: orchestration only — each layer has its own function below.
	//  1. pick the output writer (a nil Stderr means io.Discard), register the
	//     flags with l.registerFlags, and Parse;
	//  2. record which flags were actually typed, with fset.Visit;
	//  3. start from Defaults(), with every origin at SourceDefault;
	//  4. call applyFile, then applyEnv, then applyFlags;
	//  5. Validate the resolved result and return it.
	return nil, errors.New("Load: not implemented")
}

// applyFile overlays the JSON config file — l.ConfigPath, unless -config named
// another one, which is what explicit reports. A missing file is only an error
// when the user named it: silently ignoring -config would hide a typo, while
// insisting on the default path would break every fresh checkout. An unknown
// key is an error too, because a config file is not an API payload; a
// misspelled key is a typo the user wants to hear about.
func (l Loader) applyFile(res *Result, flagPath string, explicit bool) error {
	// TODO: decode into fileConfig. A non-nil field means the document said
	// something, so write the value and its SourceFile origin together.
	return nil
}

// applyEnv overlays the TOOLKIT_* variables, read through l.LookupEnv (nil
// means "no environment"). That second return value is what makes this layer
// honest: TOOLKIT_ENDPOINT="" is a deliberate — and invalid — empty endpoint,
// not an absent variable.
func (l Loader) applyEnv(res *Result) error {
	// TODO: parse each variable, reporting a failure as a *ValueError carrying
	// the field, SourceEnv and the raw text.
	return nil
}

// applyFlags overlays only the flags the user actually typed, which is what
// set — built from fset.Visit — records. Reading raw directly instead would
// overwrite every lower layer with a zero value on every run.
func applyFlags(res *Result, raw *rawFlags, set map[string]bool) {
	// TODO
}

// writeUsage prints the help text the user sees for -h. The flag package's
// default lists flags only, which answers a third of "how do I configure
// this?" — so print the flags, the TOOLKIT_* variables, and the precedence
// rule.
func writeUsage(w io.Writer, fset *flag.FlagSet) {
	// TODO
}

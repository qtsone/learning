package main

import (
	"errors"
	"fmt"
	"time"
)

// EnvPrefix prefixes every environment variable this tool reads:
// TOOLKIT_ENDPOINT, TOOLKIT_TIMEOUT, TOOLKIT_RETRIES, TOOLKIT_VERBOSE and
// TOOLKIT_TAGS.
const EnvPrefix = "TOOLKIT_"

// Config holds the fully resolved settings for one run of the tool.
type Config struct {
	Endpoint string
	Timeout  time.Duration
	Retries  int
	Verbose  bool
	Tags     []string
}

// Source identifies the layer that supplied a setting's final value.
type Source int

const (
	SourceDefault Source = iota
	SourceFile
	SourceEnv
	SourceFlag
)

// String names the layer for humans: it shows up in error messages and in the
// tool's provenance output, so the words are part of the user interface.
func (s Source) String() string {
	// TODO: "default", "config file", "environment", "flag";
	// any other value is "unknown".
	return ""
}

// Fields lists every setting name used as a key in Result.Origins.
var Fields = []string{"endpoint", "timeout", "retries", "verbose", "tags"}

// Result is a resolved Config plus the provenance of every setting.
type Result struct {
	Config  Config
	Origins map[string]Source
}

// Causes reported inside a *ValueError. Callers match them with errors.Is
// instead of matching on message text.
var (
	ErrEmpty  = errors.New("must not be empty")
	ErrRange  = errors.New("out of allowed range")
	ErrScheme = errors.New("must start with http:// or https://")
)

// ValueError reports a setting that could not be parsed or failed validation,
// and names the layer the offending value came from.
type ValueError struct {
	Field  string
	Source Source
	Raw    string
	Err    error
}

func (e *ValueError) Error() string {
	return fmt.Sprintf("%s: invalid value %q from %s: %v", e.Field, e.Raw, e.Source, e.Err)
}

func (e *ValueError) Unwrap() error { return e.Err }

// Defaults returns the configuration used when no other layer says otherwise.
func Defaults() Config {
	// TODO: see acceptance criterion 2 in LESSON.md.
	return Config{}
}

// Validate reports the first unusable setting in the resolved result,
// attributing it to the layer that supplied the value. It checks endpoint,
// then timeout, then retries.
//
// Rules: the endpoint must be non-blank and use http:// or https://; the
// timeout must be greater than zero; retries must be between 0 and 10.
func (r *Result) Validate() error {
	// TODO: return a *ValueError built from r.Origins, or nil.
	return nil
}

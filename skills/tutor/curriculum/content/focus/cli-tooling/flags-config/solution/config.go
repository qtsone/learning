package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
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
	switch s {
	case SourceDefault:
		return "default"
	case SourceFile:
		return "config file"
	case SourceEnv:
		return "environment"
	case SourceFlag:
		return "flag"
	default:
		return "unknown"
	}
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
	return Config{
		Endpoint: "https://api.example.com",
		Timeout:  5 * time.Second,
		Retries:  3,
		Verbose:  false,
		Tags:     nil,
	}
}

// Validate reports the first unusable setting in the resolved result,
// attributing it to the layer that supplied the value. It checks endpoint,
// then timeout, then retries.
//
// Validation runs on the *resolved* config, not on each layer as it arrives: a
// file may set an out-of-range value that a flag then corrects, and rejecting
// a value nobody ended up using would be wrong.
func (r *Result) Validate() error {
	c := r.Config

	if strings.TrimSpace(c.Endpoint) == "" {
		return &ValueError{Field: "endpoint", Source: r.Origins["endpoint"], Raw: c.Endpoint, Err: ErrEmpty}
	}
	if !strings.HasPrefix(c.Endpoint, "http://") && !strings.HasPrefix(c.Endpoint, "https://") {
		return &ValueError{Field: "endpoint", Source: r.Origins["endpoint"], Raw: c.Endpoint, Err: ErrScheme}
	}
	if c.Timeout <= 0 {
		return &ValueError{Field: "timeout", Source: r.Origins["timeout"], Raw: c.Timeout.String(), Err: ErrRange}
	}
	if c.Retries < 0 || c.Retries > 10 {
		return &ValueError{Field: "retries", Source: r.Origins["retries"], Raw: strconv.Itoa(c.Retries), Err: ErrRange}
	}
	return nil
}

package main

import (
	"errors"
	"strings"
)

// TagList collects a repeatable -tag flag into a list of tags. It implements
// flag.Value, which is how the flag package learns about types it does not
// know: String renders the current value, Set consumes one occurrence.
type TagList []string

// String joins the tags with commas, the way they would be typed back. The
// flag package calls it on a freshly made zero value while printing usage, so
// it must work on an empty list.
func (t *TagList) String() string {
	if t == nil {
		return ""
	}
	return strings.Join(*t, ",")
}

// Set appends one tag; flag calls it once per -tag occurrence on the command
// line. Surrounding whitespace is trimmed, and a tag that is empty after
// trimming is rejected with an error.
func (t *TagList) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return errors.New("tag must not be empty")
	}
	*t = append(*t, value)
	return nil
}

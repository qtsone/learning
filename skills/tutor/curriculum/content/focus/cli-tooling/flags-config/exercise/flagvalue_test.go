package main

import (
	"flag"
	"io"
	"slices"
	"testing"
)

// TagList must satisfy flag.Value; this fails to compile if it does not.
var _ flag.Value = (*TagList)(nil)

func TestTagListSetAppends(t *testing.T) {
	var tags TagList
	for _, in := range []string{"build", "  nightly  ", "arm64"} {
		if err := tags.Set(in); err != nil {
			t.Fatalf("Set(%q) = %v, want nil", in, err)
		}
	}
	want := []string{"build", "nightly", "arm64"}
	if !slices.Equal(tags, want) {
		t.Errorf("after Set calls, TagList = %v, want %v", []string(tags), want)
	}
	if got := tags.String(); got != "build,nightly,arm64" {
		t.Errorf("String() = %q, want %q", got, "build,nightly,arm64")
	}
}

func TestTagListRejectsEmptyTag(t *testing.T) {
	var tags TagList
	for _, in := range []string{"", "   "} {
		if err := tags.Set(in); err == nil {
			t.Errorf("Set(%q) = nil, want an error", in)
		}
	}
	if len(tags) != 0 {
		t.Errorf("rejected tags were stored anyway: %v", []string(tags))
	}
}

func TestTagListStringOnEmptyValue(t *testing.T) {
	// flag calls String on a zero value while printing usage.
	var tags TagList
	if got := tags.String(); got != "" {
		t.Errorf("empty TagList.String() = %q, want %q", got, "")
	}
}

func TestTagListThroughFlagSet(t *testing.T) {
	var tags TagList
	fset := flag.NewFlagSet("toolkit", flag.ContinueOnError)
	fset.SetOutput(io.Discard)
	fset.Var(&tags, "tag", "tag to attach, repeatable")

	if err := fset.Parse([]string{"-tag", "alpha", "-tag", "beta"}); err != nil {
		t.Fatalf("Parse() = %v, want nil", err)
	}
	want := []string{"alpha", "beta"}
	if !slices.Equal(tags, want) {
		t.Errorf("-tag alpha -tag beta gave %v, want %v", []string(tags), want)
	}

	var empty TagList
	fset2 := flag.NewFlagSet("toolkit", flag.ContinueOnError)
	fset2.SetOutput(io.Discard)
	fset2.Var(&empty, "tag", "tag to attach, repeatable")
	if err := fset2.Parse([]string{"-tag", ""}); err == nil {
		t.Error("-tag with an empty value parsed cleanly, want an error")
	}
}

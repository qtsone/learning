package conf

import (
	"errors"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// generatedFile is the file the //go:generate line in config.go writes.
const generatedFile = "conf_gen.go"

// The convention every Go tool uses to recognise generated code.
var generatedHeader = regexp.MustCompile(`(?m)^// Code generated .* DO NOT EDIT\.$`)

func TestGeneratedFileLooksGenerated(t *testing.T) {
	src, err := os.ReadFile(generatedFile)
	if err != nil {
		t.Fatalf("reading %s: %v", generatedFile, err)
	}
	text := string(src)
	if !generatedHeader.MatchString(text) {
		t.Errorf("%s has no `// Code generated ... DO NOT EDIT.` header line", generatedFile)
	}
	if strings.Contains(text, `"reflect"`) {
		t.Errorf("%s imports reflect; the point of generating it was to leave reflection behind", generatedFile)
	}
	if strings.Contains(text, "not generated yet") {
		t.Errorf("%s is still the placeholder — run: go generate ./...", generatedFile)
	}
}

// TestGeneratedMatchesReflect is the real specification for the generator:
// two decoders, one set of tags, indistinguishable behaviour.
func TestGeneratedMatchesReflect(t *testing.T) {
	cases := []struct {
		name string
		src  map[string]string
	}{
		{"every key present", fullSrc()},
		{"only the required keys", map[string]string{"ADDR": ":80", "PORT": "80"}},
		{"skipped keys are ignored", map[string]string{"ADDR": ":80", "PORT": "80", "Comment": "x", "secret": "y"}},
		{"missing required key", map[string]string{"PORT": "80"}},
		{"missing both required keys", map[string]string{"DEBUG": "true"}},
		{"bad int", map[string]string{"ADDR": ":80", "PORT": "eighty"}},
		{"bad bool", map[string]string{"ADDR": ":80", "PORT": "80", "DEBUG": "maybe"}},
		{"empty values", map[string]string{"ADDR": "", "PORT": "0", "REGION": ""}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var viaReflect, viaCodegen Config
			wantErr := Decode(c.src, &viaReflect)
			gotErr := viaCodegen.DecodeMap(c.src)

			if !reflect.DeepEqual(viaCodegen, viaReflect) {
				t.Errorf("decoded values differ:\n  generated %+v\n    reflect %+v", viaCodegen, viaReflect)
			}
			if (gotErr == nil) != (wantErr == nil) {
				t.Fatalf("error mismatch: generated %v, reflect %v", gotErr, wantErr)
			}
			if wantErr == nil {
				return
			}
			var gotField, wantField *FieldError
			if !errors.As(gotErr, &gotField) {
				t.Fatalf("generated decoder returned %v (%T), want a *FieldError like the reflect one", gotErr, gotErr)
			}
			if !errors.As(wantErr, &wantField) {
				t.Fatalf("reflect decoder returned %v (%T), want a *FieldError", wantErr, wantErr)
			}
			if gotField.Field != wantField.Field || gotField.Key != wantField.Key {
				t.Errorf("generated FieldError %+v, reflect %+v", gotField, wantField)
			}
			for _, sentinel := range []error{ErrMissing, strconv.ErrSyntax, strconv.ErrRange} {
				if errors.Is(gotErr, sentinel) != errors.Is(wantErr, sentinel) {
					t.Errorf("errors.Is(err, %v): generated %v, reflect %v", sentinel, errors.Is(gotErr, sentinel), errors.Is(wantErr, sentinel))
				}
			}
		})
	}
}

func TestGeneratedDecoderDoesNotAllocate(t *testing.T) {
	src := fullSrc()
	var c Config
	if err := c.DecodeMap(src); err != nil {
		t.Fatalf("DecodeMap returned error: %v", err)
	}
	allocs := testing.AllocsPerRun(100, func() {
		_ = c.DecodeMap(src)
	})
	// Generated code does map lookups and strconv calls, and nothing else:
	// no boxing, no reflect.Value, no tag re-parsing. The bound is loose;
	// the observed number should be 0.
	if allocs > 2 {
		t.Errorf("DecodeMap made %v allocations per call; generated code should make none", allocs)
	}
}

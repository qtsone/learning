package conf

import (
	"errors"
	"reflect"
	"strconv"
	"testing"
)

func fullSrc() map[string]string {
	return map[string]string{
		"ADDR":    ":8080",
		"PORT":    "8080",
		"DEBUG":   "true",
		"REGION":  "eu-west-1",
		"RETRIES": "3",
		"-":       "ignored",
		"UNKNOWN": "ignored",
	}
}

func TestDecode(t *testing.T) {
	cases := []struct {
		name string
		src  map[string]string
		dst  Config // starting value, so defaults are visible
		want Config
	}{
		{
			name: "every key present",
			src:  fullSrc(),
			want: Config{Addr: ":8080", Port: 8080, Debug: true, Region: "eu-west-1", Retries: 3},
		},
		{
			name: "absent keys leave the field alone",
			src:  map[string]string{"ADDR": ":9000", "PORT": "9000"},
			dst:  Config{Region: "us-east-1", Retries: 5, Debug: true},
			want: Config{Addr: ":9000", Port: 9000, Debug: true, Region: "us-east-1", Retries: 5},
		},
		{
			name: "an empty value is still a value",
			src:  map[string]string{"ADDR": "", "PORT": "0", "REGION": ""},
			dst:  Config{Region: "us-east-1"},
			want: Config{Addr: "", Port: 0, Region: ""},
		},
		{
			name: "skipped fields stay zero",
			src:  map[string]string{"ADDR": ":80", "PORT": "80", "Comment": "hi", "secret": "s"},
			want: Config{Addr: ":80", Port: 80},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := c.dst
			if err := Decode(c.src, &got); err != nil {
				t.Fatalf("Decode returned error: %v", err)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("Decode:\n got %+v\nwant %+v", got, c.want)
			}
		})
	}
}

func TestDecodeFieldErrors(t *testing.T) {
	cases := []struct {
		name      string
		src       map[string]string
		wantField string
		wantKey   string
		wantErr   error
	}{
		{"missing required key", map[string]string{"PORT": "1"}, "Addr", "ADDR", ErrMissing},
		{"missing second required key", map[string]string{"ADDR": ":1"}, "Port", "PORT", ErrMissing},
		{"int is not a number", map[string]string{"ADDR": ":1", "PORT": "eighty"}, "Port", "PORT", strconv.ErrSyntax},
		{"bool is not a bool", map[string]string{"ADDR": ":1", "PORT": "1", "DEBUG": "yes please"}, "Debug", "DEBUG", strconv.ErrSyntax},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got Config
			err := Decode(c.src, &got)
			if err == nil {
				t.Fatalf("Decode(%v) = nil error, want a *FieldError", c.src)
			}
			var fe *FieldError
			if !errors.As(err, &fe) {
				t.Fatalf("Decode returned %v (%T), want a *FieldError", err, err)
			}
			if fe.Field != c.wantField || fe.Key != c.wantKey {
				t.Errorf("FieldError{Field: %q, Key: %q}, want {Field: %q, Key: %q}", fe.Field, fe.Key, c.wantField, c.wantKey)
			}
			if !errors.Is(err, c.wantErr) {
				t.Errorf("errors.Is(err, %v) = false for err %v — wrap the cause so callers can match it", c.wantErr, err)
			}
		})
	}
}

func TestDecodeTagRules(t *testing.T) {
	type sample struct {
		Untagged string
		Skipped  string `conf:"-"`
		Fallback string `conf:",required"`
		Odd      string `conf:"ODD,shiny,required"`
		private  string `conf:"PRIVATE"`
	}
	src := map[string]string{
		"Untagged": "u",
		"-":        "s",
		"Fallback": "f",
		"ODD":      "o",
		"PRIVATE":  "p",
	}
	var got sample
	if err := Decode(src, &got); err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	want := sample{Fallback: "f", Odd: "o"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Decode:\n got %+v\nwant %+v\n(no tag, \"-\" and unexported fields are skipped; an empty key falls back to the field name; unknown options are ignored)", got, want)
	}

	var missing sample
	err := Decode(map[string]string{"ODD": "o"}, &missing)
	if !errors.Is(err, ErrMissing) {
		t.Errorf("Decode without the Fallback key = %v, want an error matching ErrMissing", err)
	}
}

func TestDecodeIntWidth(t *testing.T) {
	type sample struct {
		Small int8 `conf:"SMALL"`
	}
	var ok sample
	if err := Decode(map[string]string{"SMALL": "100"}, &ok); err != nil || ok.Small != 100 {
		t.Fatalf("Decode(SMALL=100) = %v, %+v; want no error and Small == 100", err, ok)
	}
	var over sample
	err := Decode(map[string]string{"SMALL": "9000"}, &over)
	if !errors.Is(err, strconv.ErrRange) {
		t.Errorf("Decode(SMALL=9000) = %v, want an error matching strconv.ErrRange — parse with the field type's own bit width", err)
	}
}

func TestDecodeUnsupportedType(t *testing.T) {
	type sample struct {
		Tags []string `conf:"TAGS"`
	}
	var absent sample
	if err := Decode(map[string]string{"OTHER": "x"}, &absent); err != nil {
		t.Errorf("Decode with the TAGS key absent = %v, want nil: the runtime decoder only meets the type when the key shows up", err)
	}
	var present sample
	err := Decode(map[string]string{"TAGS": "a,b"}, &present)
	if !errors.Is(err, ErrUnsupportedType) {
		t.Errorf("Decode(TAGS=a,b) = %v, want an error matching ErrUnsupportedType", err)
	}
}

func TestDecodeTargetErrors(t *testing.T) {
	var nilCfg *Config
	cases := []struct {
		name string
		dst  any
	}{
		{"nil", nil},
		{"typed nil pointer", nilCfg},
		{"not a pointer", Config{}},
		{"pointer to non-struct", new(int)},
		{"pointer to pointer", &nilCfg},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := Decode(map[string]string{"ADDR": ":1"}, c.dst)
			if !errors.Is(err, ErrNotStructPointer) {
				t.Errorf("Decode(_, %#v) = %v, want an error matching ErrNotStructPointer (and no panic)", c.dst, err)
			}
		})
	}
}

func TestParseTag(t *testing.T) {
	type sample struct {
		Plain    string `conf:"PLAIN"`
		Required string `conf:"REQ,required"`
		Fallback string `conf:",required"`
		Dash     string `conf:"-"`
		Untagged string
		private  string `conf:"PRIVATE"`
	}
	st := reflect.TypeFor[sample]()
	cases := []struct {
		field   string
		wantTag Tag
		wantOK  bool
	}{
		{"Plain", Tag{Key: "PLAIN"}, true},
		{"Required", Tag{Key: "REQ", Required: true}, true},
		{"Fallback", Tag{Key: "Fallback", Required: true}, true},
		{"Dash", Tag{}, false},
		{"Untagged", Tag{}, false},
		{"private", Tag{}, false},
	}
	for _, c := range cases {
		t.Run(c.field, func(t *testing.T) {
			f, found := st.FieldByName(c.field)
			if !found {
				t.Fatalf("no field %q", c.field)
			}
			gotTag, gotOK := ParseTag(f)
			if gotOK != c.wantOK {
				t.Fatalf("ParseTag(%s) ok = %v, want %v", c.field, gotOK, c.wantOK)
			}
			if gotOK && gotTag != c.wantTag {
				t.Errorf("ParseTag(%s) = %+v, want %+v", c.field, gotTag, c.wantTag)
			}
		})
	}
}

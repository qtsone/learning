package conf

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Sentinel errors. Callers match them with errors.Is; both decoders and the
// generator report them, so the failure vocabulary is identical everywhere.
var (
	// ErrNotStructPointer means the decode target was not a non-nil pointer
	// to a struct — the only shape reflection can write through.
	ErrNotStructPointer = errors.New("conf: target must be a non-nil pointer to a struct")
	// ErrMissing means a key marked ",required" was absent from the source.
	ErrMissing = errors.New("conf: required key missing")
	// ErrUnsupportedType means a tagged field has a type no decoder handles.
	ErrUnsupportedType = errors.New("conf: unsupported field type")
)

// FieldError reports a problem with one field. It keeps both names — the Go
// field and the configuration key — because the person reading the message
// may only know one of them.
type FieldError struct {
	Field string
	Key   string
	Err   error
}

func (e *FieldError) Error() string {
	return fmt.Sprintf("conf: field %s (key %q): %v", e.Field, e.Key, e.Err)
}

func (e *FieldError) Unwrap() error { return e.Err }

// Tag is a parsed `conf:"KEY,option,..."` struct tag.
type Tag struct {
	Key      string
	Required bool
}

// ParseTag reports how one struct field is configured.
//
// ok is false when the field must be skipped: it is unexported, it has no
// conf tag at all, or its tag is "-". An empty key ("conf:\",required\"")
// falls back to the Go field name, the way encoding/json does. Unknown
// options are ignored — nothing here is checked by the compiler.
//
// Both the runtime decoder and cmd/confgen call this, so the tag rules exist
// in exactly one place.
func ParseTag(f reflect.StructField) (Tag, bool) {
	if !f.IsExported() {
		return Tag{}, false
	}
	raw, ok := f.Tag.Lookup("conf")
	if !ok || raw == "-" {
		return Tag{}, false
	}
	key, opts, _ := strings.Cut(raw, ",")
	if key == "" {
		key = f.Name
	}
	tag := Tag{Key: key}
	for opts != "" {
		var opt string
		opt, opts, _ = strings.Cut(opts, ",")
		if opt == "required" {
			tag.Required = true
		}
	}
	return tag, true
}

// Decode copies values from src into the struct dst points at, following the
// conf tags. Keys absent from src leave their field untouched, so a
// pre-populated struct acts as a set of defaults.
//
// Supported field types: string, bool, and the signed integer types. A tagged
// field of any other type is an error — but only if its key is actually
// present in src, which is the whole problem with deciding types at runtime.
//
// Decode stops at the first bad field and returns a *FieldError.
func Decode(src map[string]string, dst any) error {
	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Pointer || rv.IsNil() || rv.Elem().Kind() != reflect.Struct {
		return ErrNotStructPointer
	}
	sv := rv.Elem()
	st := sv.Type()

	for i := range st.NumField() {
		tag, ok := ParseTag(st.Field(i))
		if !ok {
			continue
		}
		s, present := src[tag.Key]
		if !present {
			if tag.Required {
				return &FieldError{Field: st.Field(i).Name, Key: tag.Key, Err: ErrMissing}
			}
			continue
		}
		if err := setField(sv.Field(i), s); err != nil {
			return &FieldError{Field: st.Field(i).Name, Key: tag.Key, Err: err}
		}
	}
	return nil
}

// setField converts s according to fv's kind and stores it. fv is known to be
// settable: it came from the addressable struct behind dst.
func setField(fv reflect.Value, s string) error {
	switch fv.Kind() {
	case reflect.String:
		fv.SetString(s)
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return err
		}
		fv.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// The field's own width, so an int8 rejects 9000 instead of
		// wrapping around to 40.
		n, err := strconv.ParseInt(s, 10, fv.Type().Bits())
		if err != nil {
			return err
		}
		fv.SetInt(n)
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedType, fv.Type())
	}
	return nil
}

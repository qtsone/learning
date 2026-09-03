package conf

import (
	"errors"
	"fmt"
	"reflect"
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
	// TODO: implement. reflect.StructField has IsExported(), Name and Tag
	// (a reflect.StructTag with Get and Lookup). strings.Cut is a tidy way
	// to peel the key off the option list.
	return Tag{}, false
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
	// TODO: implement.
	//
	//  1. Validate dst: reflect.ValueOf(dst), Kind() == reflect.Pointer,
	//     not IsNil(), and Elem().Kind() == reflect.Struct. Otherwise
	//     return ErrNotStructPointer.
	//  2. Walk the struct type's fields (Type().NumField() / Field(i)),
	//     call ParseTag, skip what it tells you to skip.
	//  3. Look the key up in src; honour Required; convert with strconv
	//     according to the field's reflect.Kind and set it with SetString /
	//     SetBool / SetInt. For integers, pass the field type's Bits() to
	//     strconv.ParseInt so an int8 cannot silently swallow 9000.
	//  4. Wrap every failure in a *FieldError so the caller learns which
	//     field and which key.
	return errors.New("conf: Decode not implemented")
}

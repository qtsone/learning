package main

import (
	"errors"
	"reflect"
)

// fieldSpec is everything the template needs about one field. Note what it
// is made of: strings and an int. By the time rendering starts, no reflect
// value is left anywhere — the types have been turned into text.
type fieldSpec struct {
	Name     string // Go field name, e.g. "Port"
	Key      string // configuration key, e.g. "PORT"
	Required bool   // the ",required" option was present
	Kind     string // "string", "bool" or "int" — the template branches on this
	GoType   string // for ints, the field's Go type name: "int", "int32", …
	BitSize  int    // for ints, the type's width for strconv.ParseInt
}

// fieldSpecs walks t's fields, applying exactly the same tag rules as the
// runtime decoder (it calls conf.ParseTag), and returns one spec per field
// that should be decoded.
//
// Unlike the runtime decoder it refuses unsupported field types *here* —
// with a *conf.FieldError wrapping conf.ErrUnsupportedType — even for keys
// no one will ever set. That is the trade this whole lesson is about: the
// generator's failure lands on your laptop, the decoder's lands in
// production, on the day someone finally sets that key.
func fieldSpecs(t reflect.Type) ([]fieldSpec, error) {
	// TODO: implement.
	//
	//  1. Iterate t.NumField() / t.Field(i) and call conf.ParseTag(f);
	//     skip the fields it rejects.
	//  2. Classify f.Type.Kind(): reflect.String -> "string",
	//     reflect.Bool -> "bool", the five signed integer kinds -> "int"
	//     with GoType = f.Type.Name() and BitSize = f.Type.Bits().
	//  3. Anything else, including a named type of a supported kind, is an
	//     error: the template can only write a type name it can spell in
	//     the target package, so require f.Type.PkgPath() == "".
	return nil, errors.New("confgen: fieldSpecs not implemented")
}

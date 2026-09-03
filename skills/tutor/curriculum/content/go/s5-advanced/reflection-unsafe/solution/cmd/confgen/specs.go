package main

import (
	"fmt"
	"reflect"

	conf "tutor.local/reflection-unsafe"
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
	var specs []fieldSpec
	for i := range t.NumField() {
		f := t.Field(i)
		tag, ok := conf.ParseTag(f)
		if !ok {
			continue
		}
		spec, err := specFor(f.Type)
		if err != nil {
			return nil, &conf.FieldError{Field: f.Name, Key: tag.Key, Err: err}
		}
		spec.Name, spec.Key, spec.Required = f.Name, tag.Key, tag.Required
		specs = append(specs, spec)
	}
	return specs, nil
}

func specFor(ft reflect.Type) (fieldSpec, error) {
	// A named type (time.Duration, or a local `type level int`) is a type
	// the template cannot reliably spell in the generated file: it may need
	// an import, an alias, or may not be exported at all. Refusing beats
	// emitting code that does not compile.
	if ft.PkgPath() != "" {
		return fieldSpec{}, fmt.Errorf("%w: %s (generated code cannot name it)", conf.ErrUnsupportedType, ft)
	}
	switch ft.Kind() {
	case reflect.String:
		return fieldSpec{Kind: "string"}, nil
	case reflect.Bool:
		return fieldSpec{Kind: "bool"}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fieldSpec{Kind: "int", GoType: ft.Name(), BitSize: ft.Bits()}, nil
	default:
		return fieldSpec{}, fmt.Errorf("%w: %s", conf.ErrUnsupportedType, ft)
	}
}

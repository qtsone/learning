// Command confgen writes a reflection-free decoder for a tagged struct type.
//
// It is invoked by the //go:generate line in config.go:
//
//	go generate ./...
//
// The generator itself uses reflection — once, on your machine, at generate
// time. What it emits is plain Go with every type decision already made.
package main

import (
	"flag"
	"fmt"
	"os"
	"reflect"

	conf "tutor.local/reflection-unsafe"
)

// types the generator knows how to be pointed at. A production generator
// would parse the package with go/ast or golang.org/x/tools/go/packages;
// a registry keeps this one to one screen.
var types = map[string]reflect.Type{
	"Config": reflect.TypeFor[conf.Config](),
}

func main() {
	typeName := flag.String("type", "Config", "struct type to generate a decoder for")
	out := flag.String("out", "conf_gen.go", "file to write")
	flag.Parse()

	if err := run(*typeName, *out); err != nil {
		fmt.Fprintln(os.Stderr, "confgen:", err)
		os.Exit(1)
	}
}

func run(typeName, out string) error {
	t, ok := types[typeName]
	if !ok {
		return fmt.Errorf("unknown type %q (known: %v)", typeName, keys(types))
	}
	fields, err := fieldSpecs(t)
	if err != nil {
		return err
	}
	src, err := render(t, fields)
	if err != nil {
		return err
	}
	// Nothing is written unless the output is valid, formatted Go: a
	// generator that leaves a broken file behind takes the package down
	// with it, including the generator's own build.
	return os.WriteFile(out, src, 0o644)
}

func keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

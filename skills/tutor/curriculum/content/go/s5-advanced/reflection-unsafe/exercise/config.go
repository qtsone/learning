// Package conf decodes string-keyed configuration (environment variables, an
// .env file, a flag map) into structs — twice over: once at runtime with
// reflection, and once with code generated at build time from the same tags.
package conf

//go:generate go run ./cmd/confgen -type Config -out conf_gen.go

// Config is the demo target for both decoders.
//
// The `conf` struct tag is the whole contract: the first comma-separated item
// names the source key (empty means "use the Go field name"), and the option
// ",required" turns a missing key into an error. A field with no conf tag, a
// tag of "-", or an unexported name is skipped.
type Config struct {
	Addr    string `conf:"ADDR,required"`
	Port    int    `conf:"PORT,required"`
	Debug   bool   `conf:"DEBUG"`
	Region  string `conf:"REGION"`
	Retries int    `conf:"RETRIES"`
	Comment string `conf:"-"`
	secret  string
}

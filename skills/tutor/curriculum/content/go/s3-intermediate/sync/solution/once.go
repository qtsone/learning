package syncex

import "sync"

// Config is the expensive-to-build value that Loader caches.
type Config struct {
	Env   string
	Debug bool
}

// Loader builds a Config at most once, no matter how many goroutines ask.
type Loader struct {
	load func() *Config
	once sync.Once
	cfg  *Config
}

// NewLoader returns a Loader that calls load exactly once, on first use.
func NewLoader(load func() *Config) *Loader {
	return &Loader{load: load}
}

// Config returns the loaded configuration, loading it on the first call.
// Every caller — however many goroutines call concurrently — receives the
// same *Config, and load runs exactly once.
func (l *Loader) Config() *Config {
	l.once.Do(func() { l.cfg = l.load() })
	return l.cfg
}

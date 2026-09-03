package syncex

// Config is the expensive-to-build value that Loader caches.
type Config struct {
	Env   string
	Debug bool
}

// Loader builds a Config at most once, no matter how many goroutines ask.
type Loader struct {
	load func() *Config
	// TODO: a sync.Once and the field that caches the loaded *Config.
}

// NewLoader returns a Loader that calls load exactly once, on first use.
func NewLoader(load func() *Config) *Loader {
	return &Loader{load: load}
}

// Config returns the loaded configuration, loading it on the first call.
// Every caller — however many goroutines call concurrently — receives the
// same *Config, and load runs exactly once.
func (l *Loader) Config() *Config {
	// TODO: use the Once so load runs exactly once even under concurrency.
	// An `if l.cfg == nil` check is not enough — the tests run with -race.
	return nil
}

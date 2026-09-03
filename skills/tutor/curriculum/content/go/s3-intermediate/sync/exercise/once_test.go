package syncex

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestLoaderLoadsExactlyOnce(t *testing.T) {
	var calls atomic.Int32
	l := NewLoader(func() *Config {
		calls.Add(1)
		return &Config{Env: "test", Debug: true}
	})

	const goroutines = 16
	results := make([]*Config, goroutines)
	var wg sync.WaitGroup
	for i := range results {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = l.Config()
		}()
	}
	wg.Wait()

	if got := calls.Load(); got != 1 {
		t.Fatalf("load function ran %d times, want exactly 1", got)
	}
	first := results[0]
	if first == nil {
		t.Fatal("Config() = nil, want the loaded *Config")
	}
	for i, r := range results {
		if r != first {
			t.Errorf("goroutine %d got a different *Config pointer — all callers must share one loaded value", i)
		}
	}
	if first.Env != "test" || !first.Debug {
		t.Errorf("loaded Config = %+v, want {Env:test Debug:true}", *first)
	}
}

func TestLoaderRepeatedCallsReturnCache(t *testing.T) {
	var calls atomic.Int32
	l := NewLoader(func() *Config {
		calls.Add(1)
		return &Config{Env: "dev"}
	})
	a, b := l.Config(), l.Config()
	if a == nil || a != b {
		t.Errorf("Config() returned %p then %p, want the same non-nil pointer twice", a, b)
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("load function ran %d times across two calls, want 1", got)
	}
}

func TestLoadersAreIndependent(t *testing.T) {
	var calls atomic.Int32
	load := func() *Config {
		calls.Add(1)
		return &Config{Env: "x"}
	}
	a, b := NewLoader(load), NewLoader(load)
	a.Config()
	b.Config()
	if got := calls.Load(); got != 2 {
		t.Errorf("two independent Loaders ran load %d times, want 2 (Once state belongs to the Loader, not the package)", got)
	}
}

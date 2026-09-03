package syncex

import (
	"fmt"
	"sync"
	"testing"
)

func TestStoreSequential(t *testing.T) {
	s := NewStore()
	if v, ok := s.Get("missing"); ok || v != "" {
		t.Errorf("Get(\"missing\") = (%q, %t), want (\"\", false)", v, ok)
	}
	s.Set("env", "prod")
	if v, ok := s.Get("env"); !ok || v != "prod" {
		t.Errorf("after Set: Get(\"env\") = (%q, %t), want (\"prod\", true)", v, ok)
	}
	s.Set("env", "dev")
	if v, ok := s.Get("env"); !ok || v != "dev" {
		t.Errorf("after overwrite: Get(\"env\") = (%q, %t), want (\"dev\", true)", v, ok)
	}
	if got := s.Len(); got != 1 {
		t.Errorf("Len() = %d, want 1 (overwriting a key must not grow the store)", got)
	}
}

func TestStoreConcurrent(t *testing.T) {
	s := NewStore()
	const writers, keysPerWriter = 4, 50
	var wg sync.WaitGroup
	for w := range writers {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for k := range keysPerWriter {
				s.Set(fmt.Sprintf("w%d-k%d", w, k), "v")
			}
		}()
		go func() {
			defer wg.Done()
			for k := range keysPerWriter {
				s.Get(fmt.Sprintf("w%d-k%d", w, k))
				s.Len()
			}
		}()
	}
	wg.Wait()
	if got, want := s.Len(), writers*keysPerWriter; got != want {
		t.Fatalf("Len() = %d, want %d after all writers finished", got, want)
	}
	for w := range writers {
		for k := range keysPerWriter {
			key := fmt.Sprintf("w%d-k%d", w, k)
			if v, ok := s.Get(key); !ok || v != "v" {
				t.Fatalf("Get(%q) = (%q, %t), want (\"v\", true)", key, v, ok)
			}
		}
	}
}

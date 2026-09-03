package memlab

import (
	"strconv"
	"strings"
	"sync"
	"testing"
)

func TestMeterCountsEveryRecord(t *testing.T) {
	const workers, perWorker = 8, 200
	m := NewMeter()

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			event := "worker-" + strconv.Itoa(id)
			for i := 0; i < perWorker; i++ {
				m.Record(event)
			}
		}(w)
	}

	snapshots := make(chan struct{})
	go func() {
		defer close(snapshots)
		for i := 0; i < 500; i++ {
			hits, last := m.Snapshot()
			if hits < 0 || hits > workers*perWorker {
				t.Errorf("Snapshot() hits = %d, want between 0 and %d", hits, workers*perWorker)
				return
			}
			if hits > 0 && last == "" {
				t.Errorf("Snapshot() = (%d, %q): positive count with empty last event — the two fields were not read as one consistent pair", hits, last)
				return
			}
		}
	}()

	wg.Wait()
	<-snapshots

	hits, last := m.Snapshot()
	if hits != workers*perWorker {
		t.Errorf("after %d concurrent Records: hits = %d, want %d (updates were lost)", workers*perWorker, hits, workers*perWorker)
	}
	if !strings.HasPrefix(last, "worker-") {
		t.Errorf("last event = %q, want one of the recorded %q events", last, "worker-N")
	}
}

func TestStorePublishesTheLatestConfig(t *testing.T) {
	const updates = 300
	s := NewStore(&Config{Version: 0, Endpoint: "https://api.internal"})

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for v := 1; v <= updates; v++ {
			s.Update(&Config{Version: v, Endpoint: "https://api.internal"})
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < updates; i++ {
			c := s.Current()
			if c == nil {
				t.Error("Current() = nil while an Update was in flight")
				return
			}
			if c.Version < 0 || c.Version > updates {
				t.Errorf("Current() returned impossible Version %d", c.Version)
				return
			}
		}
	}()
	wg.Wait()

	if got := s.Current().Version; got != updates {
		t.Errorf("after all updates finished: Current().Version = %d, want %d", got, updates)
	}
}

package patterns

import (
	"sync"
	"testing"
	"time"
)

func TestSemaphoreTryAcquireCapacity(t *testing.T) {
	s := NewSemaphore(2)
	if !s.TryAcquire() {
		t.Fatal("TryAcquire() = false on a fresh semaphore, want true")
	}
	if !s.TryAcquire() {
		t.Fatal("TryAcquire() = false with one of two slots taken, want true")
	}
	if s.TryAcquire() {
		t.Fatal("TryAcquire() = true with all slots taken, want false")
	}
	s.Release()
	if !s.TryAcquire() {
		t.Fatal("TryAcquire() = false after a Release freed a slot, want true")
	}
}

// TestSemaphoreEnforcesLimit runs 4 tasks through a 2-slot semaphore. Each
// task announces itself on `entered`, then holds its slot until the test
// sends on `release`. The semaphore must admit exactly 2 at a time.
func TestSemaphoreEnforcesLimit(t *testing.T) {
	const limit, tasks = 2, 4
	s := NewSemaphore(limit)
	entered := make(chan struct{}, tasks) // buffered: task sends never block
	release := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(tasks)
	for range tasks {
		go func() {
			defer wg.Done()
			s.Acquire()
			entered <- struct{}{}
			<-release
			s.Release()
		}()
	}

	waitEntered := func(msg string) {
		t.Helper()
		select {
		case <-entered:
		case <-time.After(2 * time.Second):
			t.Fatal(msg)
		}
	}
	waitEntered("no task got past Acquire — a fresh semaphore must admit callers")
	waitEntered("only one task got past Acquire — a 2-slot semaphore must admit two")

	if s.TryAcquire() {
		t.Error("TryAcquire() = true while 2 holders are in flight, want false")
	}
	select {
	case <-entered:
		t.Fatal("a third task got past Acquire with all slots held — Acquire must block at the limit")
	case <-time.After(200 * time.Millisecond):
	}

	release <- struct{}{} // one holder releases…
	waitEntered("no waiting task proceeded after a Release — Acquire must unblock when a slot frees")

	for range tasks - 1 {
		release <- struct{}{}
	}
	wg.Wait()
}

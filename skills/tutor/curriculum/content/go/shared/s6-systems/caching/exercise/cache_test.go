package cache

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeClock is the injected time source: tests advance it explicitly, so no
// test sleeps or depends on real time.
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{t: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)}
}

func (f *fakeClock) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.t
}

func (f *fakeClock) Advance(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.t = f.t.Add(d)
}

func TestGetMissing(t *testing.T) {
	c := New[string, int](4, time.Minute, newFakeClock().Now)
	if v, ok := c.Get("absent"); ok {
		t.Fatalf("Get(%q) on empty cache = %d, true; want miss", "absent", v)
	}
}

func TestSetGetOverwrite(t *testing.T) {
	c := New[string, int](4, time.Minute, newFakeClock().Now)
	c.Set("a", 1)
	if v, ok := c.Get("a"); !ok || v != 1 {
		t.Fatalf("after Set(a, 1): Get(a) = %d, %v; want 1, true", v, ok)
	}
	c.Set("a", 2)
	if v, ok := c.Get("a"); !ok || v != 2 {
		t.Fatalf("after Set(a, 2): Get(a) = %d, %v; want 2, true (Set must overwrite)", v, ok)
	}
}

func TestExpiry(t *testing.T) {
	clk := newFakeClock()
	c := New[string, int](4, time.Minute, clk.Now)
	c.Set("a", 1)

	clk.Advance(59 * time.Second)
	if v, ok := c.Get("a"); !ok || v != 1 {
		t.Fatalf("59s into a 60s TTL: Get(a) = %d, %v; want 1, true (entry expired too early)", v, ok)
	}

	clk.Advance(1 * time.Second) // age is now exactly ttl
	if v, ok := c.Get("a"); ok {
		t.Fatalf("at exactly the TTL: Get(a) = %d, true; want miss (an entry is expired once its age reaches ttl)", v)
	}

	c.Set("a", 3)
	if v, ok := c.Get("a"); !ok || v != 3 {
		t.Fatalf("Set after expiry: Get(a) = %d, %v; want 3, true", v, ok)
	}
}

func TestSetRefreshesTTLAndValue(t *testing.T) {
	clk := newFakeClock()
	c := New[string, int](4, time.Minute, clk.Now)
	c.Set("a", 1)

	clk.Advance(40 * time.Second)
	c.Set("a", 2) // rewrite: TTL restarts from here

	clk.Advance(40 * time.Second) // 80s after first Set, 40s after second
	if v, ok := c.Get("a"); !ok || v != 2 {
		t.Fatalf("40s after rewrite: Get(a) = %d, %v; want 2, true (Set must reset the TTL)", v, ok)
	}

	clk.Advance(20 * time.Second) // 60s after second Set
	if v, ok := c.Get("a"); ok {
		t.Fatalf("60s after rewrite: Get(a) = %d, true; want miss", v)
	}
}

func TestLRUEvictionOrder(t *testing.T) {
	c := New[string, int](2, time.Hour, newFakeClock().Now)
	c.Set("a", 1)
	c.Set("b", 2)
	c.Set("c", 3) // full: evicts a, the least recently used

	if v, ok := c.Get("a"); ok {
		t.Fatalf("Get(a) = %d, true; want miss (a was LRU and must be evicted at capacity 2)", v)
	}
	if v, ok := c.Get("b"); !ok || v != 2 {
		t.Fatalf("Get(b) = %d, %v; want 2, true (b was not LRU, must survive)", v, ok)
	}
	if v, ok := c.Get("c"); !ok || v != 3 {
		t.Fatalf("Get(c) = %d, %v; want 3, true", v, ok)
	}
}

func TestGetRefreshesRecency(t *testing.T) {
	c := New[string, int](2, time.Hour, newFakeClock().Now)
	c.Set("a", 1)
	c.Set("b", 2)
	if _, ok := c.Get("a"); !ok {
		t.Fatalf("Get(a) missed; want hit")
	}
	c.Set("c", 3) // b is now LRU (a was touched by Get), so b is evicted

	if v, ok := c.Get("b"); ok {
		t.Fatalf("Get(b) = %d, true; want miss (a Get hit must count as use, making b the LRU)", v)
	}
	if v, ok := c.Get("a"); !ok || v != 1 {
		t.Fatalf("Get(a) = %d, %v; want 1, true (recently used entries must survive eviction)", v, ok)
	}
}

func TestSetRefreshesRecency(t *testing.T) {
	c := New[string, int](2, time.Hour, newFakeClock().Now)
	c.Set("a", 1)
	c.Set("b", 2)
	c.Set("a", 10) // rewrite: a becomes most recently used
	c.Set("c", 3)  // b is LRU, so b is evicted

	if v, ok := c.Get("b"); ok {
		t.Fatalf("Get(b) = %d, true; want miss (Set on an existing key must count as use)", v)
	}
	if v, ok := c.Get("a"); !ok || v != 10 {
		t.Fatalf("Get(a) = %d, %v; want 10, true", v, ok)
	}
}

func TestDelete(t *testing.T) {
	c := New[string, int](4, time.Hour, newFakeClock().Now)
	c.Set("a", 1)
	c.Delete("a")
	if v, ok := c.Get("a"); ok {
		t.Fatalf("Get(a) after Delete = %d, true; want miss", v)
	}
	c.Delete("never-set") // must be a no-op, not a panic
}

func TestGetOrLoadCachesSuccess(t *testing.T) {
	clk := newFakeClock()
	c := New[string, int](4, time.Minute, clk.Now)
	var calls atomic.Int32
	loader := func() (int, error) {
		calls.Add(1)
		return 42, nil
	}

	v, err := c.GetOrLoad("k", loader)
	if err != nil || v != 42 {
		t.Fatalf("first GetOrLoad = %d, %v; want 42, nil", v, err)
	}
	if n := calls.Load(); n != 1 {
		t.Fatalf("loader ran %d times on a miss; want 1", n)
	}

	v, err = c.GetOrLoad("k", loader)
	if err != nil || v != 42 {
		t.Fatalf("second GetOrLoad = %d, %v; want 42, nil", v, err)
	}
	if n := calls.Load(); n != 1 {
		t.Fatalf("loader ran %d times total; want 1 (a hit must not call the loader)", n)
	}

	clk.Advance(time.Minute) // entry expires
	v, err = c.GetOrLoad("k", loader)
	if err != nil || v != 42 {
		t.Fatalf("GetOrLoad after expiry = %d, %v; want 42, nil", v, err)
	}
	if n := calls.Load(); n != 2 {
		t.Fatalf("loader ran %d times total; want 2 (expiry must trigger a reload)", n)
	}
}

func TestGetOrLoadDoesNotCacheError(t *testing.T) {
	c := New[string, int](4, time.Minute, newFakeClock().Now)
	boom := errors.New("origin down")
	var calls atomic.Int32

	_, err := c.GetOrLoad("k", func() (int, error) {
		calls.Add(1)
		return 0, boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("GetOrLoad with failing loader: err = %v; want %v", err, boom)
	}
	if v, ok := c.Get("k"); ok {
		t.Fatalf("Get(k) = %d, true after failed load; errors must not be cached", v)
	}

	v, err := c.GetOrLoad("k", func() (int, error) {
		calls.Add(1)
		return 7, nil
	})
	if err != nil || v != 7 {
		t.Fatalf("GetOrLoad after failed load = %d, %v; want 7, nil (a failure must not poison the key)", v, err)
	}
	if n := calls.Load(); n != 2 {
		t.Fatalf("loaders ran %d times; want 2", n)
	}
}

func TestGetOrLoadSingleflight(t *testing.T) {
	c := New[string, int](8, time.Minute, newFakeClock().Now)

	const callers = 8
	var calls atomic.Int32
	var arrived sync.WaitGroup
	arrived.Add(callers)

	// The loader holds the load open until every caller has reached its
	// GetOrLoad call, guaranteeing the calls really are concurrent. Each
	// caller marks arrived before calling, so the loader never deadlocks.
	loader := func() (int, error) {
		calls.Add(1)
		arrived.Wait()
		return 42, nil
	}

	results := make([]int, callers)
	errs := make([]error, callers)
	var done sync.WaitGroup
	done.Add(callers)
	for i := range callers {
		go func() {
			defer done.Done()
			arrived.Done()
			results[i], errs[i] = c.GetOrLoad("hot", loader)
		}()
	}
	done.Wait()

	if n := calls.Load(); n != 1 {
		t.Fatalf("loader ran %d times for %d concurrent callers of one key; want exactly 1 (collapse the stampede)", n, callers)
	}
	for i := range callers {
		if errs[i] != nil || results[i] != 42 {
			t.Fatalf("caller %d got %d, %v; want 42, nil (every caller shares the one load's result)", i, results[i], errs[i])
		}
	}
}

// A cache that grows without bound is the bug this whole lesson exists to
// prevent, so a nonsense capacity must not produce one.
func TestNewClampsCapacity(t *testing.T) {
	c := New[string, int](0, time.Minute, newFakeClock().Now)
	for i := range 5 {
		c.Set(fmt.Sprintf("k%d", i), i)
	}
	live := 0
	for i := range 5 {
		if _, ok := c.Get(fmt.Sprintf("k%d", i)); ok {
			live++
		}
	}
	if live != 1 {
		t.Fatalf("capacity 0 kept %d entries; want 1 (clamp a sub-1 capacity, never store unbounded)", live)
	}
}

// A slow origin must not freeze the whole cache: while one key is loading,
// every other key stays readable. An implementation that holds its mutex
// across loader() blocks the Get below forever, so this test waits with a
// deliberately huge margin — a correct cache answers in microseconds, and
// the wait is only ever paid by a broken one.
func TestGetOrLoadKeepsOtherKeysReadable(t *testing.T) {
	c := New[string, int](8, time.Minute, newFakeClock().Now)
	c.Set("ready", 1)

	loading := make(chan struct{}) // closed once the load is in flight
	release := make(chan struct{}) // closed once the other key was read
	loaded := make(chan int, 1)

	go func() {
		v, err := c.GetOrLoad("slow", func() (int, error) {
			close(loading)
			<-release
			return 7, nil
		})
		if err != nil {
			t.Errorf("GetOrLoad(slow) error = %v; want nil", err)
		}
		loaded <- v
	}()

	select {
	case <-loading:
	case v := <-loaded:
		t.Fatalf(`GetOrLoad("slow") = %d without ever calling the loader; on a miss it must call loader`, v)
	case <-time.After(10 * time.Second):
		t.Fatal(`GetOrLoad("slow") never called the loader`)
	}

	got := make(chan int, 1)
	go func() {
		v, ok := c.Get("ready")
		if !ok {
			t.Errorf(`Get("ready") missed while another key was loading; want a hit`)
		}
		got <- v
	}()

	select {
	case v := <-got:
		if v != 1 {
			t.Fatalf(`Get("ready") = %d during a load; want 1`, v)
		}
	case <-time.After(10 * time.Second):
		t.Fatal(`Get("ready") blocked while "slow" was loading: you are holding the ` +
			`mutex across loader(). Release it before calling loader, then re-acquire ` +
			`to store the result.`)
	}

	close(release)
	if v := <-loaded; v != 7 {
		t.Fatalf("GetOrLoad(slow) = %d; want 7", v)
	}
}

func TestConcurrentSetGet(t *testing.T) {
	const workers, perWorker = 8, 8
	c := New[string, int](workers*perWorker, time.Hour, newFakeClock().Now)

	var wg sync.WaitGroup
	wg.Add(workers)
	for w := range workers {
		go func() {
			defer wg.Done()
			for j := range perWorker {
				key := fmt.Sprintf("w%d-%d", w, j)
				c.Set(key, w*100+j)
				if v, ok := c.Get(key); !ok || v != w*100+j {
					t.Errorf("Get(%s) = %d, %v; want %d, true", key, v, ok, w*100+j)
				}
			}
		}()
	}
	wg.Wait()

	for w := range workers {
		for j := range perWorker {
			key := fmt.Sprintf("w%d-%d", w, j)
			if v, ok := c.Get(key); !ok || v != w*100+j {
				t.Errorf("after all writers: Get(%s) = %d, %v; want %d, true", key, v, ok, w*100+j)
			}
		}
	}
}

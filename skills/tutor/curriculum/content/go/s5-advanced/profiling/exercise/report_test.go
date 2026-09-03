package report

import (
	"strings"
	"testing"
)

func TestTopUsersAggregation(t *testing.T) {
	entries := []Entry{
		{User: "Alice", Path: "/a", Status: 200, Bytes: 100},
		{User: "BOB", Path: "/b", Status: 200, Bytes: 10},
		{User: "ALICE", Path: "/c", Status: 404, Bytes: 200},
		{User: "alice", Path: "/d", Status: 200, Bytes: 3},
		{User: "bob", Path: "/e", Status: 500, Bytes: 5},
		{User: "Carol", Path: "/f", Status: 200, Bytes: 7},
	}
	got := TopUsers(entries, 10)
	want := []UserStat{
		{User: "alice", Requests: 3, Bytes: 303},
		{User: "bob", Requests: 2, Bytes: 15},
		{User: "carol", Requests: 1, Bytes: 7},
	}
	assertStatsEqual(t, got, want)
}

func TestTopUsersTiesSortByName(t *testing.T) {
	entries := []Entry{
		{User: "zoe", Bytes: 1},
		{User: "Ann", Bytes: 1},
		{User: "mia", Bytes: 1},
	}
	got := TopUsers(entries, 10)
	want := []UserStat{
		{User: "ann", Requests: 1, Bytes: 1},
		{User: "mia", Requests: 1, Bytes: 1},
		{User: "zoe", Requests: 1, Bytes: 1},
	}
	assertStatsEqual(t, got, want)
}

func TestTopUsersTruncatesToN(t *testing.T) {
	entries := []Entry{
		{User: "a"}, {User: "a"}, {User: "a"},
		{User: "b"}, {User: "b"},
		{User: "c"},
	}
	got := TopUsers(entries, 2)
	want := []UserStat{
		{User: "a", Requests: 3},
		{User: "b", Requests: 2},
	}
	assertStatsEqual(t, got, want)
	if got := TopUsers(nil, 5); len(got) != 0 {
		t.Errorf("TopUsers(nil, 5) returned %d stats, want 0", len(got))
	}
}

// TestTopUsersInvariants runs the generated workload and checks the
// aggregate properties any correct implementation must preserve — so an
// "optimization" that drops or double-counts entries cannot pass.
func TestTopUsersInvariants(t *testing.T) {
	entries := genEntries(10_000)
	wantBytes := 0
	for _, e := range entries {
		wantBytes += e.Bytes
	}
	stats := TopUsers(entries, len(entries))

	gotRequests, gotBytes := 0, 0
	seen := make(map[string]bool)
	for i, s := range stats {
		gotRequests += s.Requests
		gotBytes += s.Bytes
		if s.User != strings.ToLower(s.User) {
			t.Errorf("stats[%d].User = %q, want lowercase", i, s.User)
		}
		if seen[s.User] {
			t.Errorf("user %q appears twice — aggregation must merge case-insensitively", s.User)
		}
		seen[s.User] = true
		if i > 0 {
			prev := stats[i-1]
			if prev.Requests < s.Requests ||
				(prev.Requests == s.Requests && prev.User > s.User) {
				t.Errorf("stats[%d..%d] out of order: %+v before %+v", i-1, i, prev, s)
			}
		}
	}
	if gotRequests != len(entries) {
		t.Errorf("total Requests = %d, want %d (every entry counted exactly once)", gotRequests, len(entries))
	}
	if gotBytes != wantBytes {
		t.Errorf("total Bytes = %d, want %d", gotBytes, wantBytes)
	}
	if len(stats) != 100 {
		t.Errorf("distinct users = %d, want 100", len(stats))
	}
}

// TestTopUsersAllocs is the perf gate. The naive scan lowercases inside
// the inner loop: ~half a million allocations for this workload. The
// fixed version needs at most one small allocation per entry plus a map
// — comfortably under the (generous) bound. No wall-clock timing: the
// race detector makes clocks lie, but allocation counts stay honest.
func TestTopUsersAllocs(t *testing.T) {
	entries := genEntries(10_000)
	allocs := testing.AllocsPerRun(5, func() {
		TopUsers(entries, 10)
	})
	const maxAllocs = 50_000
	if allocs > maxAllocs {
		t.Errorf("TopUsers(10k entries) made %.0f allocations, want <= %d\n"+
			"the heap profile knows why — capture one (LESSON.md, 'Reading a profile') "+
			"and look at what runs once per comparison instead of once per entry", allocs, maxAllocs)
	}
}

func TestRenderOutput(t *testing.T) {
	stats := []UserStat{
		{User: "alice", Requests: 3, Bytes: 1200},
		{User: "bob", Requests: 1, Bytes: 50},
	}
	want := "alice requests=3 bytes=1200\nbob requests=1 bytes=50\n"
	if got := Render(stats); got != want {
		t.Errorf("Render() = %q, want %q", got, want)
	}
	if got := Render(nil); got != "" {
		t.Errorf("Render(nil) = %q, want \"\"", got)
	}
}

// TestRenderAllocs: += in a loop allocates a fresh, ever-longer string
// per iteration — thousands of allocations and quadratic bytes copied.
// A strings.Builder plus append-style formatting (strconv.AppendInt into
// a reused []byte) brings the whole render to a handful of allocations.
func TestRenderAllocs(t *testing.T) {
	stats := genStats(2_000)
	allocs := testing.AllocsPerRun(5, func() {
		Render(stats)
	})
	const maxAllocs = 300
	if allocs > maxAllocs {
		t.Errorf("Render(2000 stats) made %.0f allocations, want <= %d\n"+
			"building a string with += re-copies everything on every line; "+
			"see LESSON.md 'The two bottleneck archetypes' for the Builder + AppendInt shape", allocs, maxAllocs)
	}
}

func assertStatsEqual(t *testing.T, got, want []UserStat) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d stats %+v, want %d %+v", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("stats[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

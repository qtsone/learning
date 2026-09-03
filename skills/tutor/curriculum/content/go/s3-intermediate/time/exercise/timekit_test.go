package timekit

import (
	"testing"
	"time"

	// Embed the IANA zone database so LoadLocation works even on
	// machines (minimal containers, stripped CI images) without one.
	_ "time/tzdata"
)

func TestParseLocal(t *testing.T) {
	cases := []struct {
		name  string
		value string
		zone  string
		// wantUTC is the same instant expressed in UTC — the test
		// compares instants, so your result's zone is up to you.
		wantUTC time.Time
	}{
		{
			name:    "bucharest winter time is UTC+2",
			value:   "2026-03-14 09:30",
			zone:    "Europe/Bucharest",
			wantUTC: time.Date(2026, 3, 14, 7, 30, 0, 0, time.UTC),
		},
		{
			name:    "new york summer time is UTC-4",
			value:   "2026-07-04 12:00",
			zone:    "America/New_York",
			wantUTC: time.Date(2026, 7, 4, 16, 0, 0, 0, time.UTC),
		},
		{
			name:    "utc is itself",
			value:   "2026-01-01 00:00",
			zone:    "UTC",
			wantUTC: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ParseLocal(c.value, c.zone)
			if err != nil {
				t.Fatalf("ParseLocal(%q, %q) returned error %v, want nil", c.value, c.zone, err)
			}
			if !got.Equal(c.wantUTC) {
				t.Errorf("ParseLocal(%q, %q) = %v, want the instant %v — was the value parsed in the right zone?",
					c.value, c.zone, got.UTC(), c.wantUTC)
			}
		})
	}
}

func TestParseLocalErrors(t *testing.T) {
	cases := []struct {
		name  string
		value string
		zone  string
	}{
		{"unknown zone", "2026-03-14 09:30", "Mars/Olympus_Mons"},
		{"value does not match layout", "14/03/2026 09:30", "Europe/Bucharest"},
		{"seconds not in layout", "2026-03-14 09:30:00", "UTC"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := ParseLocal(c.value, c.zone); err == nil {
				t.Errorf("ParseLocal(%q, %q) returned nil error, want one naming the bad input", c.value, c.zone)
			}
		})
	}
}

func TestFormatUTC(t *testing.T) {
	eet := time.FixedZone("EET", 2*60*60)
	cases := []struct {
		name string
		in   time.Time
		want string
	}{
		{
			name: "already utc",
			in:   time.Date(2026, 3, 14, 7, 30, 0, 0, time.UTC),
			want: "2026-03-14 07:30 UTC",
		},
		{
			name: "eet instant renders as its utc equivalent",
			in:   time.Date(2026, 3, 14, 9, 30, 0, 0, eet),
			want: "2026-03-14 07:30 UTC",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := FormatUTC(c.in); got != c.want {
				t.Errorf("FormatUTC(%v) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestHalfway(t *testing.T) {
	day := func(h, m, s int) time.Time {
		return time.Date(2026, 1, 1, h, m, s, 0, time.UTC)
	}
	cases := []struct {
		name       string
		start, end time.Time
		want       time.Time
	}{
		{"even hour", day(10, 0, 0), day(11, 0, 0), day(10, 30, 0)},
		{"odd span splits to seconds", day(10, 0, 0), day(10, 3, 0), day(10, 1, 30)},
		{"zero span is start", day(10, 0, 0), day(10, 0, 0), day(10, 0, 0)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Halfway(c.start, c.end); !got.Equal(c.want) {
				t.Errorf("Halfway(%v, %v) = %v, want %v", c.start, c.end, got, c.want)
			}
		})
	}
}

func TestSameInstant(t *testing.T) {
	noon := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	noonInBucharest := noon.In(time.FixedZone("EET", 2*60*60))
	if noon == noonInBucharest {
		t.Fatal("test invariant broken: == should differ across zones")
	}
	cases := []struct {
		name string
		a, b time.Time
		want bool
	}{
		{"same instant different zone", noon, noonInBucharest, true},
		{"identical values", noon, noon, true},
		{"one nanosecond apart", noon, noon.Add(time.Nanosecond), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := SameInstant(c.a, c.b); got != c.want {
				t.Errorf("SameInstant(%v, %v) = %t, want %t — compare the instant, not the representation",
					c.a, c.b, got, c.want)
			}
		})
	}
}

func TestInWindow(t *testing.T) {
	day := func(h, m int) time.Time {
		return time.Date(2026, 1, 1, h, m, 0, 0, time.UTC)
	}
	start, end := day(9, 0), day(17, 0)
	startElsewhere := start.In(time.FixedZone("EET", 2*60*60))
	cases := []struct {
		name string
		t    time.Time
		want bool
	}{
		{"inside", day(12, 0), true},
		{"start is included", day(9, 0), true},
		{"end is excluded", day(17, 0), false},
		{"before start", day(8, 59), false},
		{"after end", day(17, 1), false},
		{"start instant in another zone is still included", startElsewhere, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := InWindow(c.t, start, end); got != c.want {
				t.Errorf("InWindow(%v, %v, %v) = %t, want %t", c.t, start, end, got, c.want)
			}
		})
	}
}

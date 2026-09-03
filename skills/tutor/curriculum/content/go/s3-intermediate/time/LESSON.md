# Time

> `go.intermediate.time` · ~2-3h · Stage: Intermediate Go

## Objectives

By the end of this lesson you can:

- Implement parsing and formatting with Go's reference-time layout
  (`2006-01-02`) and explain why layouts work this way.
- Explain the difference between wall clock and monotonic clock readings in
  `time.Time` and why elapsed-time measurement needs the monotonic clock.
- Choose between `time.Timer`, `time.Ticker`, and `time.After` for a given
  scheduling need, and explain the resource-leak pitfall of `time.After` in
  loops.
- Implement correct duration arithmetic with `time.Duration` and explain why
  comparing times with `Equal` beats `==`.
- Explain how time zones and `Location` affect parsing, formatting, and
  comparisons.

## Two types, one package

Almost everything in `time` revolves around two types:

- `time.Time` — an **instant**: one specific moment in history, accurate to
  the nanosecond. "2026-03-14 07:30:00 UTC" is an instant.
- `time.Duration` — a **span**: an amount of elapsed time with no anchor.
  "90 minutes" is a span.

Keeping instants and spans apart is what makes the arithmetic type-safe.
`Duration` is just an `int64` count of nanoseconds, so you build one by
multiplying the package's constants:

```go
d := 90 * time.Minute          // Duration
timeout := 2500 * time.Millisecond
```

The operations follow from the types. Instant plus span is an instant;
instant minus instant is a span:

```go
later := start.Add(45 * time.Minute)   // Time + Duration = Time
took := end.Sub(start)                 // Time - Time     = Duration
half := took / 2                       // Duration ÷ int  = Duration
```

Two conveniences you will use constantly: `time.Since(start)` is shorthand
for `time.Now().Sub(start)`, and `time.Until(deadline)` is
`deadline.Sub(time.Now())`. You already leaned on `time.Since` in S2's timing
probes — by the end of this lesson you'll know why that was the *correct*
tool and not just the convenient one.

One thing you cannot do: add two `time.Time` values. "March 14 plus March 15"
is meaningless, and the API simply doesn't offer it. When an operation is
absent from `time`, that's usually a hint your model of the problem is off.

## The reference time: why 2006-01-02?

Most languages format dates with cryptic verbs: `%Y-%m-%d`, `YYYY-MM-DD`,
`strftime` tables you look up every single time. Go does something unusual:
you write the layout **as an example**, by formatting one specific reference
moment:

```
Mon Jan 2 15:04:05 MST 2006
```

Why that moment? Line up its components in US date order and you get
1 2 3 4 5 6 7: month **1**, day **2**, hour **3** PM (15 in 24-hour form),
minute **4**, second **5**, year 200**6**, zone UTC-**7**. The layout string
is a picture of your desired output, drawn with that one date:

```go
t := time.Date(2026, 3, 14, 7, 30, 0, 0, time.UTC)
t.Format("2006-01-02 15:04")        // "2026-03-14 07:30"
t.Format("Mon, 02 Jan 2006")        // "Sat, 14 Mar 2026"
t.Format(time.RFC3339)              // "2026-03-14T07:30:00Z"
```

Parsing is the same layout, run in reverse:

```go
t, err := time.Parse("2006-01-02 15:04", "2026-03-14 07:30")
```

The trade: you can no longer mistype a verb (`%m` vs `%M` bugs are extinct),
and layouts read like their output. The new failure mode is writing a
*different* date as the layout — `"2007-01-02"` or `"2006-13-02"` silently
means something else or fails to parse. Rule: never invent the reference
time; copy it, or better, use the named constants the package ships —
`time.RFC3339`, `time.DateOnly` (`"2006-01-02"`), `time.DateTime`,
`time.TimeOnly`. If a constant fits, prefer it over a hand-rolled layout.

## Wall clocks and monotonic clocks

Your machine has two clocks, and `time.Time` quietly carries both.

The **wall clock** tells you *what time it is*. It is allowed to jump: NTP
nudges it to match reality, daylight saving shifts it by an hour, a VM
migration can lurch it seconds either way. Perfect for timestamps, terrible
for measurement — subtract two wall readings across a clock adjustment and
you can get a negative "elapsed" time.

The **monotonic clock** only ever moves forward, at a steady rate, and means
nothing outside the current process. Perfect for measurement, useless for
timestamps.

Go's design decision: `time.Now()` returns a `Time` holding *both* readings.
When you `Sub` two values that both carry a monotonic reading, Go uses the
monotonic parts — so `time.Since(start)` is immune to NTP and DST. You never
choose a clock explicitly; the API chooses correctly for you.

The monotonic reading is fragile, though. It is stripped when it stops making
sense: formatting, marshaling (remember the JSON lesson — a round-trip
through `MarshalJSON` keeps only the wall clock), calling `.UTC()` or
`.In(...)`, or the idiom `t.Round(0)` which strips it on purpose. Parsed and
constructed times never have one — only `time.Now()` does. Consequence:
**measure elapsed time with two `time.Now()` readings in the same process**,
never by subtracting parsed timestamps.

## Comparing times: Equal, not ==

`time.Time` is a struct, so `==` compiles — and then betrays you. It compares
every field: the instant, the `Location` pointer, *and* the monotonic
reading. Two values naming the exact same instant differently are `!=`:

```go
utc := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
buch := utc.In(loc)        // same instant, viewed from Bucharest

utc == buch                // false — different Location
utc.Equal(buch)            // true  — same instant
```

`Equal`, `Before`, and `After` compare the instant and ignore representation.
Use them, always. The same reasoning says: don't use `time.Time` as a map
key (the docs call this out explicitly) — two keys for one instant.

## Time zones and Location

A `time.Time` always carries a `*time.Location` — the lens it is displayed
through. `time.UTC` and `time.Local` are built in; everything else is loaded
by IANA name:

```go
loc, err := time.LoadLocation("Europe/Bucharest")
if err != nil { /* unknown zone, or no zone database on this machine */ }
t := aMoment.In(loc)       // same instant, Bucharest lens
```

`In` and `UTC` **never change the instant** — only how it reads when
formatted. The instant-vs-representation split you just saw with `Equal` is
the same idea again.

Where zones bite is *parsing*. `time.Parse` with a layout that names no zone
assumes **UTC** — not your local time, which surprises nearly everyone once.
When you know which zone a naive timestamp was written in, say so:

```go
t, err := time.ParseInLocation("2006-01-02 15:04", "2026-03-14 09:30", loc)
```

Now `t` is 09:30 *Bucharest* time — 07:30 UTC. Same string, different zone,
different instant: the zone is part of the data, and losing it corrupts the
timestamp as surely as truncating it.

Two operational notes. First, `LoadLocation` reads the IANA database from the
OS; minimal containers often lack it. A blank import of `time/tzdata` embeds
the database in your binary — the exercise's test file does this so it runs
anywhere. Second, the standard architecture for time handling: **store and
compute in UTC, convert at the edges** (parsing input, rendering output).
And beware DST arithmetic: `t.Add(24 * time.Hour)` adds exactly 24 hours of
physics, which lands at a *different wall-clock hour* across a DST change;
"same time tomorrow" is `t.AddDate(0, 0, 1)`.

## Timers, tickers, and After

Sometimes you don't have a time, you want to be *told* when one arrives. The
`time` package delivers these notifications on a **channel** — Go's
concurrency pipe, the centerpiece of a lesson two steps from now. Until
then, the only syntax you need: `<-ch` means "block here until a value
arrives on `ch`".

Three tools, one decision:

- `time.NewTimer(d)` — fires **once**, after `d`. The event arrives on the
  timer's channel `t.C`. You can `Stop` it (cancel) or `Reset` it (reuse).
  Choose it when you may need to cancel or re-arm: timeouts, debouncing.
- `time.NewTicker(d)` — fires **repeatedly**, every `d`, on `t.C`. Choose it
  for periodic work: polling, heartbeats, cleanup sweeps. A ticker owns
  resources until you call `Stop` — `defer t.Stop()` the moment you make one.
- `time.After(d)` — convenience wrapper: one line, gives you just the
  channel of a single-shot timer you can never stop.

```go
t := time.NewTicker(30 * time.Second)
defer t.Stop()
// each <-t.C is one tick — the concurrency lessons make this loop-shaped
```

The classic pitfall lives in `time.After`. Each call allocates a fresh timer
with no handle to stop it, so `time.After` inside a loop historically
*leaked*: on Go 1.22 and earlier, every abandoned timer stayed alive —
channel, callback and all — until it fired, and a tight loop with
`time.After(time.Hour)` pinned an hour of garbage per iteration. Go 1.23
fixed the leak (unreferenced timers are now collectable immediately), but
the habit still stands: in a loop, prefer one `Ticker` or one `Timer` you
`Reset` — one allocation, explicit ownership — and reserve `time.After` for
one-shot waits. You will meet the loop in question (`select` with a timeout)
in the Channels lesson; this is why that code reviews the way it does.

## Exercise

Open [`exercise/`](exercise/) — a Go module with package `timekit`, a small
timestamp toolkit of the kind every ops tool grows eventually. `timekit.go`
has five functions marked `TODO`; `timekit_test.go` is the specification.
Read the tests first — note how they build fixed instants with `time.Date`
so every run is deterministic.

Acceptance criteria:

1. `ParseLocal(value, zone)` loads the IANA zone (e.g.
   `"Europe/Bucharest"`), parses `value` against layout `2006-01-02 15:04`
   *in that zone*, and returns the resulting instant. Both failure paths — an
   unknown zone, a value that doesn't match the layout — return a non-nil
   error. The tests check only that the error is non-nil; naming the
   offending input and wrapping with `%w`, as in S1's errors lesson, is what
   code review expects of an A-grade solution.
2. `FormatUTC(t)` renders any instant as `2006-01-02 15:04 MST` after
   converting it to UTC — the same instant in, say, EET must render as its
   UTC equivalent.
3. `Halfway(start, end)` returns the instant exactly midway between the two
   (duration arithmetic — no float math, no Unix-seconds juggling).
4. `SameInstant(a, b)` reports whether two times name the same instant, even
   when their zones differ and `==` says false.
5. `InWindow(t, start, end)` reports whether `t` is inside `[start, end)` —
   inclusive of `start`, exclusive of `end` — comparing instants, not
   representations.
6. `go test ./...` passes and the code is `gofmt`-clean.

Run the tests from inside `exercise/`:

```sh
cd exercise
go test ./...
```

They fail before you write code — make them green. When they are, reread
your `ParseLocal`: could a caller tell *which* step failed from the error
alone?

## Further reading

- [pkg.go.dev/time](https://pkg.go.dev/time) — the package documentation; the
  overview section on wall vs monotonic clocks is canonical.
- [Layout constants in package time](https://pkg.go.dev/time#pkg-constants) —
  the reference time explained, plus every named layout.
- [Proposal: Monotonic Elapsed Time Measurements in Go](https://go.googlesource.com/proposal/+/master/design/12914-monotonic.md)
  — the design doc behind the two-clock `time.Time`, with the Cloudflare
  leap-second outage that motivated it.
- [pkg.go.dev/time/tzdata](https://pkg.go.dev/time/tzdata) — embedding the
  zone database when the OS may not provide one.

In Go: the standard library has this scar tissue built in. `time.Now()`
returns a value carrying both a wall clock and a monotonic clock reading,
and `time.Since`/`Sub` use the monotonic part — so *durations measured on
one machine* are immune to NTP stepping the clock. That protection ends at
the network: serialize a timestamp, send it to another machine, and the
monotonic part is gone — comparing it there is exactly the cross-node
wall-clock comparison the table forbids. Likewise `context.WithTimeout`,
which you have used since S3, is the "timeouts on every remote call"
defense: the deadline propagates down the call chain, turning an
indistinguishable hang into a bounded, handleable error.

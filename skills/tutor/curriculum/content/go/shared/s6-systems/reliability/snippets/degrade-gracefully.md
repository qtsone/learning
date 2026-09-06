In Go: the deadline tool is one you already use — `context`. A request-scoped
`ctx` with `context.WithTimeout` flows through every downstream call
(`db.QueryContext(ctx, …)`, `req = req.WithContext(ctx)`), so one budget
governs the whole call tree, exactly as S3/S5 drilled. A circuit breaker is a
small state machine (closed → open → half-open) guarded by a mutex; write
one, or reach for a vetted library like `sony/gobreaker` — the states, not
the package, are the concept.

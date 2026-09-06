In Go: `fmt.Printf` with the `%#v` verb prints a value in Go syntax —
`[]float64{1.5, 2}` — which distinguishes `nil` from empty and string from
number at a glance. Inside tests prefer `t.Logf`, which stays silent until a
test fails or you pass `-v`, so diagnostic output never pollutes a green run.
For narration that deserves to survive the debugging session, use `log/slog`
with key-value pairs — a later stage builds full observability on top of it.

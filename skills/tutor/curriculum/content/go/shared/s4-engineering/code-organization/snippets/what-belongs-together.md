In Go: the surface is exactly the capitalized identifiers, and package names
read at the call site — `report.Summary(...)` explains itself, which is why
`report.ReportSummary` would be a bad name (stutter) and `utils.Format` a
useless one. Go also gives hiding teeth one level up: anything under a
directory named `internal/` is importable only within your module.

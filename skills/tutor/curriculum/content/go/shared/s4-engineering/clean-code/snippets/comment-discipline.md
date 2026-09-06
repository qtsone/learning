**In Go:** contracts have a concrete form — doc comments. Every exported
identifier gets a `// Name ...` comment, a complete sentence starting with the
name, and `pkg.go.dev` renders it as the package's documentation; you have
been reading these since S1. Note what the convention does *not* license: a
doc comment that merely echoes the signature (`// GetUser gets a user.`) is
narration in a contract's clothing. State what the caller can rely on —
behavior at the edges, what happens on empty input, what errors mean.

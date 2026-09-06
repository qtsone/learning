**In Go:** the contract has an official format —
[doc comments](https://go.dev/doc/comment). The comment sits immediately
above the declaration, is made of complete sentences, and its first sentence
begins with the identifier's name: `// Open reads the snippet file at path…`.
A `// Package store …` comment introduces the package as a whole. The reward
for the convention is tooling: `go doc`, your editor's hover, and pkg.go.dev
all render these comments as the package's reference documentation — you
have been reading them since S1. Every exported identifier deserves one;
`// Deprecated:` marks the ones callers should migrate away from.

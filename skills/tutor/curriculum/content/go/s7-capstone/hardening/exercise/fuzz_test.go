package capstone

import (
	"fmt"
	"testing"
)

// hostileCorpusFallback is how many committed corpus entries stand in for a
// named hostile-input test when the project has no such test by name.
const hostileCorpusFallback = 3

// Criterion 3: at least one fuzz target exists and at least one of them has a
// committed seed corpus under testdata/fuzz/<Target>/. The corpus is the part
// that keeps paying: those files run on every plain `go test`, on every
// machine, forever.
func TestFuzzTargetHasCommittedCorpus(t *testing.T) {
	dir := project(t)
	targets, err := fuzzTargets(dir)
	if err != nil {
		t.Fatalf("scanning %s for fuzz targets: %v", dir, err)
	}
	if len(targets) == 0 {
		t.Fatalf(`no fuzz target found in %s.

Pick the place where bytes you did not write become a value you trust — a
parser, a decoder, a query or path builder — and write one:

    func FuzzParseLine(f *testing.F) {
            f.Add("id|title|tag")          // a seed
            f.Fuzz(func(t *testing.T, in string) {
                    // a property that must hold for EVERY input
            })
    }

Then run it for real, naming the one package it lives in (-fuzz takes a single
package, never ./...):

    go test -fuzz=FuzzParseLine -fuzztime=60s ./internal/note`, dir)
	}

	var seeded []string
	var bare []string
	for _, target := range targets {
		if len(target.Seeds) > 0 {
			seeded = append(seeded, fmt.Sprintf("%s (%s): %d corpus file(s)",
				target.Name, target.File, len(target.Seeds)))
			continue
		}
		bare = append(bare, fmt.Sprintf("%s (%s): testdata/fuzz/%s/ is empty or missing",
			target.Name, target.File, target.Name))
	}
	if len(seeded) == 0 {
		t.Fatalf(`no fuzz target has a committed seed corpus:
%s

f.Add seeds live in the code; the corpus lives on disk next to the test, and it
is what turns a crash the fuzzer found into a regression test everyone runs.
Run the target with -fuzz until it finds something, commit the file it writes
to testdata/fuzz/<Target>/, or hand-write one:

    printf 'go test fuzz v1\nstring("id|title|tag")\n' > \
      testdata/fuzz/FuzzParseLine/seed-pipe-in-title`, bullets(bare, 10))
	}
	t.Logf("fuzz targets with a committed corpus:\n%s", bullets(seeded, 10))
}

// Criterion 5: if the project accepts input from outside itself, at least one
// test says so in its name. Applies only when an untrusted surface exists;
// a library with no external input surface skips this one.
func TestHostileInputIsTested(t *testing.T) {
	dir := project(t)
	surfaces, err := findUntrustedSurfaces(dir)
	if err != nil {
		t.Fatalf("scanning %s for input surfaces: %v", dir, err)
	}
	if len(surfaces) == 0 {
		t.Skip("no untrusted input surface found (no flags, files, stdin, network or decoders) — " +
			"nothing outside the process reaches this code, so hostile-input tests do not apply")
	}

	names, err := testFunctions(dir)
	if err != nil {
		t.Fatalf("scanning %s for tests: %v", dir, err)
	}
	var hostile []string
	for name, where := range names {
		if hostileRe.MatchString(name) {
			hostile = append(hostile, fmt.Sprintf("%s (%s)", name, where))
		}
	}
	if len(hostile) > 0 {
		return
	}

	targets, err := fuzzTargets(dir)
	if err != nil {
		t.Fatalf("scanning %s for fuzz targets: %v", dir, err)
	}
	corpus := 0
	for _, target := range targets {
		corpus += len(target.Seeds)
	}
	if corpus >= hostileCorpusFallback {
		t.Logf("no test named for a hostile case, but %d committed corpus entries cover it", corpus)
		return
	}

	locations := make([]string, 0, len(surfaces))
	for _, s := range surfaces {
		locations = append(locations, s.String())
	}
	t.Fatalf(`this project takes untrusted input, but no test is named for a hostile case.

input surfaces found:
%s

Write the tests an attacker would write, and name them so a reader can find
them: empty and truncated input, input far past any size limit, wrong types,
duplicate or missing fields, path traversal (../../etc/passwd), embedded NUL
bytes and invalid UTF-8, numbers that overflow, and whatever your own format
makes possible. A name matching /hostile|malicious|malformed|invalid|
traversal|oversize|truncated|overflow|.../ satisfies this check; so does a
committed fuzz corpus of at least %d entries.`,
		bullets(locations, 12), hostileCorpusFallback)
}

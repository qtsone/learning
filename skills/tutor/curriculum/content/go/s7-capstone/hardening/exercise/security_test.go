package capstone

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// minSectionBody is the shortest body that can plausibly say anything. It is
// a filter against empty headings, not a word count to hit.
const minSectionBody = 120

var govulncheckRe = regexp.MustCompile(`(?i)govulncheck`)

// Criterion 8: a security document with the sections that make it useful to
// someone who is not you. The harness can only check that each section exists
// and has prose in it; whether the prose is honest is the review's job.
func TestSecurityDocumentIsComplete(t *testing.T) {
	dir := project(t)
	name, raw, err := findSecurityDoc(dir)
	if err != nil {
		t.Fatalf(`%v (in %s)

Write SECURITY.md at your project root with these sections:
%s`, err, dir, topicList())
	}

	sections := markdownSections(raw)
	var missing, thin []string
	for _, topic := range securityTopics {
		section, ok := findSection(sections, topic.Match)
		if !ok {
			missing = append(missing, fmt.Sprintf("%-20s — %s", topic.Name, topic.Wants))
			continue
		}
		if len(section.Body) < minSectionBody {
			thin = append(thin, fmt.Sprintf("%q (%s:%d) has %d bytes of prose — %s",
				section.Heading, name, section.Line, len(section.Body), topic.Wants))
		}
		if topic.Name == "dependency policy" && !govulncheckRe.MatchString(section.Body) {
			thin = append(thin, fmt.Sprintf("%q (%s:%d) never mentions govulncheck — "+
				"run `govulncheck ./...` yourself, read every finding, and record what you "+
				"found and what you did (the harness cannot run it for you: it needs the "+
				"vulnerability database, and grading is offline)",
				section.Heading, name, section.Line))
		}
	}

	if len(missing) > 0 {
		t.Errorf("%s is missing %d required section(s):\n%s", name, len(missing), bullets(missing, 10))
	}
	for _, problem := range thin {
		t.Errorf("%s", problem)
	}
	if len(missing) > 0 || len(thin) > 0 {
		t.Logf("headings found in %s: %s", name, headingList(sections))
	}
}

// Criterion 9: the pass found something and the fix is pinned by a test. A
// finding with no regression test is a bug waiting for its second appearance.
func TestFixIsPinnedByRegressionTest(t *testing.T) {
	dir := project(t)
	name, raw, err := findSecurityDoc(dir)
	if err != nil {
		t.Fatalf("%v — see TestSecurityDocumentIsComplete", err)
	}
	claimed := regressionTestNames(raw)
	if len(claimed) == 0 {
		t.Fatalf(`%s claims no regression test.

Fix at least one thing this pass turned up — a crash the fuzzer found, an
unvalidated input, a missing timeout, a secret in a log line — and pin it with
a test that fails against the old code. Then name that test in the document:

    Regression test: TestParseLineRejectsHostileInput

If the pass genuinely found nothing, it was not a pass. Look harder at the
input surface, or hand the code to someone whose job is to break it.`, name)
	}

	have, err := testFunctions(dir)
	if err != nil {
		t.Fatalf("scanning %s for tests: %v", dir, err)
	}
	var found, absent []string
	for _, want := range claimed {
		if where, ok := have[want]; ok {
			found = append(found, fmt.Sprintf("%s (%s)", want, where))
			continue
		}
		absent = append(absent, want)
	}
	if len(found) > 0 {
		t.Logf("regression tests claimed and found:\n%s", bullets(found, 10))
	}
	if len(absent) == 0 {
		return
	}
	names := make([]string, 0, len(have))
	for n := range have {
		names = append(names, n)
	}
	sort.Strings(names)
	t.Fatalf(`%s names regression test(s) that do not exist: %s

Tests found in the project:
%s

Either the test was never written, or the name drifted. Both are the same
finding: the document says something the code does not.`,
		name, strings.Join(absent, ", "), bullets(names, 25))
}

func topicList() string {
	items := make([]string, 0, len(securityTopics))
	for _, topic := range securityTopics {
		items = append(items, fmt.Sprintf("## %s — %s", topic.Heading, topic.Wants))
	}
	return bullets(items, len(items))
}

func headingList(sections []mdSection) string {
	if len(sections) == 0 {
		return "(none)"
	}
	items := make([]string, 0, len(sections))
	for _, s := range sections {
		items = append(items, s.Heading)
	}
	return strings.Join(items, " · ")
}

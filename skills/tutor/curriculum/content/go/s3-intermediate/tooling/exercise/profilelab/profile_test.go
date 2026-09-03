package profilelab

import (
	"fmt"
	"testing"
)

var corpus = makeCorpus(2000)

func makeCorpus(n int) []string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = fmt.Sprintf("line %d: some text to pad the line out a bit", i)
	}
	return lines
}

func TestJoinsAgree(t *testing.T) {
	if got, want := JoinNaive(corpus), JoinBuilder(corpus); got != want {
		t.Fatalf("JoinNaive and JoinBuilder disagree: %d vs %d bytes", len(got), len(want))
	}
}

func BenchmarkJoinNaive(b *testing.B) {
	for range b.N {
		JoinNaive(corpus)
	}
}

func BenchmarkJoinBuilder(b *testing.B) {
	for range b.N {
		JoinBuilder(corpus)
	}
}

package report_test

import (
	"testing"

	"tutor.local/code-organization/report"
)

func TestFormatCents(t *testing.T) {
	cases := []struct {
		cents int
		want  string
	}{
		{0, "$0.00"},
		{5, "$0.05"},
		{99, "$0.99"},
		{350, "$3.50"},
		{12345, "$123.45"},
	}
	for _, c := range cases {
		if got := report.FormatCents(c.cents); got != c.want {
			t.Errorf("FormatCents(%d) = %q, want %q", c.cents, got, c.want)
		}
	}
}

func TestSummary(t *testing.T) {
	byCategory := map[string]int{"travel": 9950, "food": 600, "office": 1200}
	want := "food: $6.00\noffice: $12.00\ntravel: $99.50\ntotal: $117.50\n"
	if got := report.Summary(byCategory); got != want {
		t.Errorf("Summary(%v) =\n%q\nwant (categories sorted, total last)\n%q",
			byCategory, got, want)
	}
}

func TestSummaryEmpty(t *testing.T) {
	want := "total: $0.00\n"
	if got := report.Summary(map[string]int{}); got != want {
		t.Errorf("Summary(empty) = %q, want %q", got, want)
	}
}

package report

import "testing"

// Characterization tests: they pin the report's current observable behavior
// so you can refactor with a safety net. Keep them green after every step.
// The only edit allowed in this file is updating call sites when you rename
// the exported function — the expected outputs must never change.

func TestReportOnSampleExport(t *testing.T) {
	lines := []string{
		"alice,books,12.50",
		"bob,tools,99.99",
		"",
		"carol,books,3.25",
		"broken line",
		"dave,games,884.26",
		"",
	}
	want := "records: 4\n" +
		"total: 1000.00\n" +
		"average: 250.00\n" +
		"largest: dave (884.26)\n" +
		"by category:\n" +
		"  books: 15.75\n" +
		"  games: 884.26\n" +
		"  tools: 99.99\n" +
		"skipped: 1\n"
	got, err := DoData(lines)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("report changed — refactors must preserve behavior.\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestReportNoValidRecords(t *testing.T) {
	_, err := DoData([]string{"", "not,enough", "a,b,notanumber"})
	if err == nil {
		t.Fatal("want an error when no line is valid, got nil")
	}
}

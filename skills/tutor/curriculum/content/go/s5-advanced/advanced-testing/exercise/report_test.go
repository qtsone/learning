package logkit

import "testing"

// Small, self-explanatory expectations belong inline: you can read the input
// and the output side by side. The big end-to-end rendering is checked against
// a golden file instead — see cli_test.go.
func TestReport(t *testing.T) {
	cases := []struct {
		name    string
		events  []Event
		skipped int
		want    string
	}{
		{
			name:    "empty",
			events:  nil,
			skipped: 0,
			want:    "level  count  sources\n\ntotal: 0 events, 0 skipped\n",
		},
		{
			name: "severity order, deduplicated sources",
			events: []Event{
				{"info", "api", "a"},
				{"error", "db", "b"},
				{"info", "worker", "c"},
				{"info", "api", "d"},
			},
			skipped: 1,
			want: "level  count  sources\n" +
				"info       3  api, worker\n" +
				"error      1  db\n" +
				"\ntotal: 4 events, 1 skipped\n",
		},
		{
			name: "all four levels",
			events: []Event{
				{"error", "db", "a"},
				{"warn", "db", "b"},
				{"info", "api", "c"},
				{"debug", "api", "d"},
			},
			skipped: 0,
			want: "level  count  sources\n" +
				"debug      1  api\n" +
				"info       1  api\n" +
				"warn       1  db\n" +
				"error      1  db\n" +
				"\ntotal: 4 events, 0 skipped\n",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Report(c.events, c.skipped)
			if got != c.want {
				t.Errorf("Report() mismatch\n--- got ---\n%s\n--- want ---\n%s", got, c.want)
			}
		})
	}
}

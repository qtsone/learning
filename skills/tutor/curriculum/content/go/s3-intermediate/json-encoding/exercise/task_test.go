package tasks

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func str(s string) *string { return &s }

func TestMarshalFullTask(t *testing.T) {
	task := Task{
		ID:         7,
		Title:      "Fix login bug",
		Notes:      "only happens on mobile",
		Done:       false,
		Priority:   High,
		Assignee:   str("ana"),
		AuditToken: "internal-9f2c",
	}
	got, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	want := `{"id":7,"title":"Fix login bug","notes":"only happens on mobile","done":false,"priority":"high","assignee":"ana"}`
	if string(got) != want {
		t.Errorf("Marshal(task) =\n  %s\nwant\n  %s", got, want)
	}
	if strings.Contains(string(got), "internal-9f2c") {
		t.Errorf("AuditToken leaked into the JSON: %s", got)
	}
}

func TestMarshalOmitsEmptyOptionalFields(t *testing.T) {
	task := Task{ID: 1, Title: "Ship it", Done: true, Priority: Low}
	got, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	want := `{"id":1,"title":"Ship it","done":true,"priority":"low"}`
	if string(got) != want {
		t.Errorf("Marshal(task) = %s, want %s (empty notes/assignee must be omitted)", got, want)
	}
}

func TestUnmarshalIgnoresUnknownAndKeepsMissing(t *testing.T) {
	task := Task{Notes: "keep me"}
	input := `{"id":3,"title":"Write docs","done":true,"priority":"medium","reporter":"zoe"}`
	if err := json.Unmarshal([]byte(input), &task); err != nil {
		t.Fatalf("Unmarshal returned error: %v (unknown fields must be ignored)", err)
	}
	if task.ID != 3 || task.Title != "Write docs" || !task.Done || task.Priority != Medium {
		t.Errorf("Unmarshal filled %+v, want ID=3 Title=%q Done=true Priority=Medium",
			task, "Write docs")
	}
	if task.Notes != "keep me" {
		t.Errorf(`Notes = %q, want %q ("notes" missing from the JSON must leave the field untouched)`,
			task.Notes, "keep me")
	}
}

func TestPriorityJSON(t *testing.T) {
	cases := []struct {
		p    Priority
		json string
	}{
		{Low, `"low"`},
		{Medium, `"medium"`},
		{High, `"high"`},
	}
	for _, c := range cases {
		got, err := json.Marshal(c.p)
		if err != nil {
			t.Fatalf("Marshal(Priority %d) returned error: %v", int(c.p), err)
		}
		if string(got) != c.json {
			t.Errorf("Marshal(Priority %d) = %s, want %s", int(c.p), got, c.json)
		}
		var back Priority
		if err := json.Unmarshal([]byte(c.json), &back); err != nil {
			t.Fatalf("Unmarshal(%s) returned error: %v", c.json, err)
		}
		if back != c.p {
			t.Errorf("Unmarshal(%s) = Priority %d, want %d", c.json, int(back), int(c.p))
		}
	}
	var p Priority
	if err := json.Unmarshal([]byte(`"urgent"`), &p); err == nil {
		t.Error(`Unmarshal("urgent") = nil error, want an error for an unknown priority`)
	}
}

func TestApplyPatch(t *testing.T) {
	base := func() Task {
		return Task{ID: 5, Title: "Original", Done: false, Priority: Medium}
	}
	cases := []struct {
		name  string
		patch string
		want  Task
	}{
		{"set done only", `{"done":true}`,
			Task{ID: 5, Title: "Original", Done: true, Priority: Medium}},
		{"set title to empty string", `{"title":""}`,
			Task{ID: 5, Title: "", Done: false, Priority: Medium}},
		{"raise priority", `{"priority":"high"}`,
			Task{ID: 5, Title: "Original", Done: false, Priority: High}},
		{"empty patch changes nothing", `{}`, base()},
		{"unknown fields are ignored", `{"reporter":"zoe"}`, base()},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			task := base()
			if err := ApplyPatch(&task, []byte(c.patch)); err != nil {
				t.Fatalf("ApplyPatch(%s) returned error: %v", c.patch, err)
			}
			if task != c.want {
				t.Errorf("after ApplyPatch(%s):\n  got  %+v\n  want %+v", c.patch, task, c.want)
			}
		})
	}
	task := base()
	if err := ApplyPatch(&task, []byte(`{"title":`)); err == nil {
		t.Error("ApplyPatch with malformed JSON returned nil error, want an error")
	}
}

func TestWriteTasks(t *testing.T) {
	tasks := []Task{
		{ID: 1, Title: "First", Priority: Low},
		{ID: 2, Title: "Second", Done: true, Priority: High},
	}
	var buf bytes.Buffer
	if err := WriteTasks(&buf, tasks); err != nil {
		t.Fatalf("WriteTasks returned error: %v", err)
	}
	want := `{"id":1,"title":"First","done":false,"priority":"low"}` + "\n" +
		`{"id":2,"title":"Second","done":true,"priority":"high"}` + "\n"
	if buf.String() != want {
		t.Errorf("WriteTasks wrote:\n%q\nwant one task per line:\n%q", buf.String(), want)
	}
}

func TestReadTasks(t *testing.T) {
	input := `{"id":1,"title":"First","priority":"low"}
{"id":2,"title":"Second","done":true,"priority":"high"}
`
	got, err := ReadTasks(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ReadTasks returned error: %v", err)
	}
	want := []Task{
		{ID: 1, Title: "First", Priority: Low},
		{ID: 2, Title: "Second", Done: true, Priority: High},
	}
	if len(got) != len(want) {
		t.Fatalf("ReadTasks returned %d task(s), want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("task %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestReadTasksEmptyInput(t *testing.T) {
	got, err := ReadTasks(strings.NewReader(""))
	if err != nil {
		t.Fatalf("ReadTasks on empty input returned error: %v (a clean EOF is not an error)", err)
	}
	if len(got) != 0 {
		t.Errorf("ReadTasks on empty input returned %d task(s), want 0", len(got))
	}
}

func TestReadTasksTruncatedInput(t *testing.T) {
	input := `{"id":1,"title":"ok","priority":"low"}` + "\n" + `{"id":`
	if _, err := ReadTasks(strings.NewReader(input)); err == nil {
		t.Error("ReadTasks with truncated input returned nil error, want an error")
	}
}

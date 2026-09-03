// Package tasks is a tiny task-tracker core: one Task type and the JSON
// plumbing around it.
package tasks

import (
	"io"
)

// Priority is a task's urgency. On the wire it must travel as one of the
// strings "low", "medium", or "high" — never as a bare number.
type Priority int

const (
	Low Priority = iota
	Medium
	High
)

// MarshalJSON encodes the priority as a JSON string: "low", "medium", "high".
func (p Priority) MarshalJSON() ([]byte, error) {
	// TODO: return the JSON string for p. Delegating to json.Marshal gets
	// the quoting right for free.
	return []byte(`""`), nil
}

// UnmarshalJSON accepts "low", "medium", or "high" and rejects anything else
// with an error naming the bad value.
func (p *Priority) UnmarshalJSON(data []byte) error {
	// TODO: decode the JSON string, map it back onto *p, and reject
	// unknown names.
	return nil
}

// Task is one tracked piece of work.
//
// TODO: add struct tags so the JSON matches the acceptance criteria:
// lower-case keys, Notes and Assignee omitted when empty/nil, and
// AuditToken never marshaled even though it is exported.
type Task struct {
	ID         int
	Title      string
	Notes      string
	Done       bool
	Priority   Priority
	Assignee   *string
	AuditToken string
}

// TaskPatch is a partial update. A nil field means "absent from the patch —
// leave the task alone"; a non-nil pointer means "set the field, even to its
// zero value".
type TaskPatch struct {
	Title    *string   `json:"title"`
	Done     *bool     `json:"done"`
	Priority *Priority `json:"priority"`
}

// ApplyPatch parses patchJSON and applies every field present in it to t.
// Fields absent from the JSON are left untouched. Malformed JSON is an error.
func ApplyPatch(t *Task, patchJSON []byte) error {
	// TODO: unmarshal into a TaskPatch, then copy each non-nil field onto t.
	return nil
}

// WriteTasks streams tasks to w as newline-delimited JSON: one task per line.
func WriteTasks(w io.Writer, tasks []Task) error {
	// TODO: use a json.Encoder — one Encode call per task.
	return nil
}

// ReadTasks reads newline-delimited JSON tasks from r until the stream ends.
// A clean end of input is success (an empty stream yields zero tasks);
// truncated or malformed input is an error.
func ReadTasks(r io.Reader) ([]Task, error) {
	// TODO: loop over a json.Decoder; io.EOF at a value boundary means done.
	return nil, nil
}

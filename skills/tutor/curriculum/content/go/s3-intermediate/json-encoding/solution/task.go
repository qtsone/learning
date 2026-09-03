// Package tasks is a tiny task-tracker core: one Task type and the JSON
// plumbing around it.
package tasks

import (
	"encoding/json"
	"errors"
	"fmt"
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
	switch p {
	case Low:
		return json.Marshal("low")
	case Medium:
		return json.Marshal("medium")
	case High:
		return json.Marshal("high")
	default:
		return nil, fmt.Errorf("unknown priority %d", int(p))
	}
}

// UnmarshalJSON accepts "low", "medium", or "high" and rejects anything else
// with an error naming the bad value.
func (p *Priority) UnmarshalJSON(data []byte) error {
	var name string
	if err := json.Unmarshal(data, &name); err != nil {
		return err
	}
	switch name {
	case "low":
		*p = Low
	case "medium":
		*p = Medium
	case "high":
		*p = High
	default:
		return fmt.Errorf("unknown priority %q", name)
	}
	return nil
}

// Task is one tracked piece of work. AuditToken is needed by other Go code
// but must never reach the wire.
type Task struct {
	ID         int      `json:"id"`
	Title      string   `json:"title"`
	Notes      string   `json:"notes,omitempty"`
	Done       bool     `json:"done"`
	Priority   Priority `json:"priority"`
	Assignee   *string  `json:"assignee,omitempty"`
	AuditToken string   `json:"-"`
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
	var patch TaskPatch
	if err := json.Unmarshal(patchJSON, &patch); err != nil {
		return fmt.Errorf("parse patch: %w", err)
	}
	if patch.Title != nil {
		t.Title = *patch.Title
	}
	if patch.Done != nil {
		t.Done = *patch.Done
	}
	if patch.Priority != nil {
		t.Priority = *patch.Priority
	}
	return nil
}

// WriteTasks streams tasks to w as newline-delimited JSON: one task per line.
func WriteTasks(w io.Writer, tasks []Task) error {
	enc := json.NewEncoder(w)
	for _, task := range tasks {
		if err := enc.Encode(task); err != nil {
			return fmt.Errorf("encode task %d: %w", task.ID, err)
		}
	}
	return nil
}

// ReadTasks reads newline-delimited JSON tasks from r until the stream ends.
// A clean end of input is success (an empty stream yields zero tasks);
// truncated or malformed input is an error.
func ReadTasks(r io.Reader) ([]Task, error) {
	dec := json.NewDecoder(r)
	var tasks []Task
	for {
		var t Task
		if err := dec.Decode(&t); err != nil {
			if errors.Is(err, io.EOF) {
				return tasks, nil
			}
			return nil, fmt.Errorf("decode task %d: %w", len(tasks), err)
		}
		tasks = append(tasks, t)
	}
}

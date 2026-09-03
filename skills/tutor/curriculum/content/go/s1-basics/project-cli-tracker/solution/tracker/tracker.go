// Package tracker holds the core logic of the CLI tracker: tasks, the
// operations on them, and (in store.go) saving them to disk. Nothing in
// this package prints or reads command-line arguments — that is main's job.
package tracker

import (
	"errors"
	"fmt"
	"strings"
)

// ErrNotFound is returned (possibly wrapped) when an operation names a
// task id that does not exist.
var ErrNotFound = errors.New("task not found")

// Task is one item in the tracker.
type Task struct {
	ID    int
	Title string
	Done  bool
}

// Tracker holds a set of tasks in the order they were added.
type Tracker struct {
	tasks []Task
}

// New returns an empty, ready-to-use Tracker.
func New() *Tracker {
	return &Tracker{}
}

// Add stores a new task and returns it. The title is trimmed of
// surrounding whitespace; an empty result is an error. The new task's ID
// is one higher than the highest existing ID (the first task gets 1).
func (t *Tracker) Add(title string) (Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Task{}, errors.New("add: title is empty")
	}
	task := Task{ID: t.maxID() + 1, Title: title}
	t.tasks = append(t.tasks, task)
	return task, nil
}

func (t *Tracker) maxID() int {
	highest := 0
	for _, task := range t.tasks {
		highest = max(highest, task.ID)
	}
	return highest
}

// List returns the tasks in insertion order. It returns a copy: mutating
// the result must not change the Tracker's own data.
func (t *Tracker) List() []Task {
	return append([]Task(nil), t.tasks...)
}

// Complete marks the task with the given id as done. Completing an
// already-done task is not an error. An unknown id yields an error that
// wraps ErrNotFound.
func (t *Tracker) Complete(id int) error {
	for i := range t.tasks {
		if t.tasks[i].ID == id {
			t.tasks[i].Done = true
			return nil
		}
	}
	return fmt.Errorf("complete #%d: %w", id, ErrNotFound)
}

// Summary reports how many tasks are open and how many are done. Both
// keys, "open" and "done", are always present — even when zero.
func (t *Tracker) Summary() map[string]int {
	counts := map[string]int{"open": 0, "done": 0}
	for _, task := range t.tasks {
		if task.Done {
			counts["done"]++
		} else {
			counts["open"]++
		}
	}
	return counts
}

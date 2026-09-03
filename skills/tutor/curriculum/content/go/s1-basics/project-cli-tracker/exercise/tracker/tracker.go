// Package tracker holds the core logic of the CLI tracker: tasks, the
// operations on them, and (in store.go) saving them to disk. Nothing in
// this package prints or reads command-line arguments — that is main's job.
package tracker

import "errors"

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
	// TODO: design your own state. You need to remember tasks in
	// insertion order and be able to hand out ids that never repeat.
}

// New returns an empty, ready-to-use Tracker.
func New() *Tracker {
	return &Tracker{}
}

// Add stores a new task and returns it. The title is trimmed of
// surrounding whitespace; an empty result is an error. The new task's ID
// is one higher than the highest existing ID (the first task gets 1).
func (t *Tracker) Add(title string) (Task, error) {
	// TODO: implement per the doc comment and the tests.
	return Task{}, errors.New("TODO: implement Add")
}

// List returns the tasks in insertion order. It returns a copy: mutating
// the result must not change the Tracker's own data.
func (t *Tracker) List() []Task {
	// TODO: implement.
	return nil
}

// Complete marks the task with the given id as done. Completing an
// already-done task is not an error. An unknown id yields an error that
// wraps ErrNotFound.
func (t *Tracker) Complete(id int) error {
	// TODO: implement.
	return errors.New("TODO: implement Complete")
}

// Summary reports how many tasks are open and how many are done. Both
// keys, "open" and "done", are always present — even when zero.
func (t *Tracker) Summary() map[string]int {
	// TODO: implement.
	return nil
}

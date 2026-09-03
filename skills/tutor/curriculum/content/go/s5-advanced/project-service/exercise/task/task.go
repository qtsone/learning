// Package task is the domain of the task service: its types, its rules, its
// error vocabulary, and the storage contract it needs. Nothing in here
// imports net/http or database/sql, and nothing outside it gets to decide
// what a task is.
package task

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"
)

// MaxTitleLen bounds a title in runes, not bytes: 200 characters means 200
// characters even when they are accented or emoji.
const MaxTitleLen = 200

// Status is a task's position in its (very short) lifecycle.
type Status string

const (
	StatusOpen Status = "open"
	StatusDone Status = "done"
)

// Valid reports whether s is a status the domain recognises. The empty
// Status is not valid; callers that mean "no filter" check for "" first.
func (s Status) Valid() bool { return s == StatusOpen || s == StatusDone }

// Task is a stored task. Ids are assigned by storage; timestamps are
// assigned by the service from its clock, never by storage.
type Task struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Status    Status    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Draft is the client-supplied part of a task. A client may not choose an
// id, a status, or a timestamp — that is why this type exists instead of
// decoding straight into Task.
type Draft struct {
	Title string `json:"title"`
}

// StatusPatch is the body of a status change.
type StatusPatch struct {
	Status Status `json:"status"`
}

var (
	// ErrNotFound reports that an id matched no task. Every Store
	// implementation contracts to return it (possibly wrapped) from Get,
	// SetStatus, and Delete.
	ErrNotFound = errors.New("task not found")

	// ErrAlreadyDone reports an attempt to reopen a finished task. It is a
	// rule, not a storage failure, so it lives in the domain.
	ErrAlreadyDone = errors.New("task is already done")
)

// ValidationError maps field names to what is wrong with them. It is data,
// not prose: the HTTP layer renders it as a field-level 400 a client can act
// on.
type ValidationError map[string]string

func (e ValidationError) Error() string {
	fields := make([]string, 0, len(e))
	for f := range e {
		fields = append(fields, f)
	}
	sort.Strings(fields)
	return "invalid fields: " + strings.Join(fields, ", ")
}

// Store is what the domain needs from storage — declared here, on the
// consumer side, so storage depends on the domain and never the reverse.
// Every method takes a context: a cancelled request must be able to stop
// work all the way down to the database driver.
type Store interface {
	Create(ctx context.Context, t Task) (Task, error)
	Get(ctx context.Context, id int64) (Task, error)
	// List returns the tasks with the given status, or every task when
	// status is "". It promises nothing about order.
	List(ctx context.Context, status Status) ([]Task, error)
	// SetStatus writes the new status and stamps updated_at with at.
	SetStatus(ctx context.Context, id int64, status Status, at time.Time) (Task, error)
	Delete(ctx context.Context, id int64) error
	// Ping reports whether storage is currently usable. Readiness checks
	// call it; nothing else should.
	Ping(ctx context.Context) error
}

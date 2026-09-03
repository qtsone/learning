package task

import (
	"context"
	"time"
)

// Clock is the service's only source of "now". Production passes time.Now;
// tests pass a clock they control, so timestamps are assertable values
// instead of whatever the wall clock happened to say.
type Clock func() time.Time

// Service holds the business rules: normalization, validation, ordering,
// and which status transitions are legal. It reaches storage only through
// the Store interface.
type Service struct {
	store Store
	now   Clock
}

// NewService wires a service to its storage and its clock. A nil clock
// means time.Now.
func NewService(store Store, now Clock) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{store: store, now: now}
}

// Create normalizes and validates d, stamps it with the clock, and stores
// it as a new open task.
func (s *Service) Create(ctx context.Context, d Draft) (Task, error) {
	// TODO: trim the title, validate it (exact field messages are in
	// LESSON.md's acceptance criteria), then store a Task that is open and
	// carries the same timestamp in CreatedAt and UpdatedAt. Timestamps are
	// UTC: the wire format and the database both want an unambiguous zone.
	return Task{}, nil
}

// Get returns the task with the given id.
func (s *Service) Get(ctx context.Context, id int64) (Task, error) {
	// TODO: delegate to the store.
	return Task{}, nil
}

// List returns tasks in id order, filtered by status when one is given. It
// never returns a nil slice: an empty listing must encode as [], not null.
func (s *Service) List(ctx context.Context, status Status) ([]Task, error) {
	// TODO: reject a status that is neither "" nor a valid one, ask the
	// store (which promises no order), sort by id, and guarantee non-nil.
	return nil, nil
}

// SetStatus moves a task to target, enforcing the lifecycle rules.
func (s *Service) SetStatus(ctx context.Context, id int64, target Status) (Task, error) {
	// TODO: validate target, read the current task, and decide:
	//   - already in target  -> success with no write at all (idempotent)
	//   - done -> open       -> ErrAlreadyDone
	//   - otherwise          -> store.SetStatus with a fresh timestamp
	return Task{}, nil
}

// Delete removes the task with the given id.
func (s *Service) Delete(ctx context.Context, id int64) error {
	// TODO: delegate to the store.
	return nil
}

// Ping reports whether the service's dependencies are usable. The readiness
// endpoint is its only caller.
func (s *Service) Ping(ctx context.Context) error {
	// TODO: delegate to the store.
	return nil
}

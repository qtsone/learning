package task

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
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
	title := strings.TrimSpace(d.Title)
	if err := validateTitle(title); err != nil {
		return Task{}, err
	}
	// One reading of the clock, used for both stamps: a fresh task whose
	// created_at and updated_at differ by a microsecond is a lie.
	at := s.now().UTC()
	return s.store.Create(ctx, Task{
		Title:     title,
		Status:    StatusOpen,
		CreatedAt: at,
		UpdatedAt: at,
	})
}

// Get returns the task with the given id.
func (s *Service) Get(ctx context.Context, id int64) (Task, error) {
	return s.store.Get(ctx, id)
}

// List returns tasks in id order, filtered by status when one is given. It
// never returns a nil slice: an empty listing must encode as [], not null.
func (s *Service) List(ctx context.Context, status Status) ([]Task, error) {
	if status != "" && !status.Valid() {
		return nil, statusError()
	}
	tasks, err := s.store.List(ctx, status)
	if err != nil {
		return nil, err
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].ID < tasks[j].ID })
	if tasks == nil {
		tasks = []Task{}
	}
	return tasks, nil
}

// SetStatus moves a task to target, enforcing the lifecycle rules.
func (s *Service) SetStatus(ctx context.Context, id int64, target Status) (Task, error) {
	if !target.Valid() {
		return Task{}, statusError()
	}
	current, err := s.store.Get(ctx, id)
	if err != nil {
		return Task{}, err
	}
	switch {
	case current.Status == target:
		// Idempotent: the caller's intent already holds, so there is
		// nothing to write and no timestamp to churn.
		return current, nil
	case current.Status == StatusDone:
		return Task{}, fmt.Errorf("task %d: %w", id, ErrAlreadyDone)
	}
	return s.store.SetStatus(ctx, id, target, s.now().UTC())
}

// Delete removes the task with the given id.
func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.store.Delete(ctx, id)
}

// Ping reports whether the service's dependencies are usable. The readiness
// endpoint is its only caller.
func (s *Service) Ping(ctx context.Context) error {
	return s.store.Ping(ctx)
}

// validateTitle returns a ValidationError, or a literal nil. Returning an
// empty ValidationError instead would hand the caller a non-nil error
// holding an empty map — the typed-nil trap from the interfaces lesson.
func validateTitle(title string) error {
	switch {
	case title == "":
		return ValidationError{"title": "required"}
	case utf8.RuneCountInString(title) > MaxTitleLen:
		return ValidationError{"title": fmt.Sprintf("must be at most %d characters", MaxTitleLen)}
	}
	return nil
}

func statusError() error {
	return ValidationError{"status": fmt.Sprintf("must be %q or %q", StatusOpen, StatusDone)}
}

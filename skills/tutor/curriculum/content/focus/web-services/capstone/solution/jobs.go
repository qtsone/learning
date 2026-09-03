package board

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// JobKindNotify is the one kind of background work this service has: tell a
// task's owner it exists. It stands in for every slow, third-party-dependent
// thing that must not sit in the request path — an e-mail, a webhook, a push.
const JobKindNotify = "notify"

// NotifyHandler applies a notify job.
type NotifyHandler struct {
	Store *Store
	Clock Clock
}

// Handle records the notification for the task named in the payload and returns
// the events the pool should publish once the transaction commits.
//
// Delivery is at-least-once and always will be: "do the work" and "record that
// the work is done" are two operations, and a crash can land between them. The
// pool has already claimed this job id in the dedup ledger on the transaction
// below, so a redelivery never gets here — no second notification row, and no
// second event to announce it. Both halves for free, for every kind of job.
//
// What is left for this handler is the part only it knows: which failures are
// worth retrying.
func (h NotifyHandler) Handle(ctx context.Context, tx *sql.Tx, job Job) ([]Event, error) {
	task, err := h.Store.TaskByID(ctx, job.Payload)
	if errors.Is(err, ErrNotFound) {
		// No retry conjures a deleted row. Burning five attempts and the waits
		// between them would only delay the dead-letter somebody has to read.
		return nil, fmt.Errorf("%w: no task %q", ErrPermanent, job.Payload)
	}
	if err != nil {
		return nil, err
	}

	if err := h.Store.InsertNotification(ctx, tx, task.ID, task.OwnerID, h.Clock.Now()); err != nil {
		return nil, err
	}
	return []Event{taskEvent(EventTaskNotified, task)}, nil
}

// JobHandlers is the registry the pool dispatches on. A job whose kind is not
// in here dead-letters: no retry will conjure a handler, and the row is how
// somebody finds out the deploy was wrong.
func JobHandlers(store *Store, clock Clock) map[string]JobHandler {
	return map[string]JobHandler{
		JobKindNotify: NotifyHandler{Store: store, Clock: clock},
	}
}

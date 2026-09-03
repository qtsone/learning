package board

import (
	"context"
	"database/sql"
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
// pool absorbs that for every kind at once — it claims the job id in the dedup
// ledger on the transaction below and only reaches a handler on the first
// delivery — so write the effect once, plainly, as if it ran exactly once.
//
// TODO:
//   - Load the task (Payload is its id). A task that does not exist is a
//     failure no retry can fix: wrap ErrPermanent and it dead-letters at once
//     instead of burning its whole budget and the waits between.
//   - Write the notification with InsertNotification, and return
//     taskEvent(EventTaskNotified, task) for the pool to publish once the
//     transaction commits.
//
// Everything you write here rolls back if you return an error — including the
// pool's processed marker, which is what keeps a retry a real retry and not a
// silent skip.
func (h NotifyHandler) Handle(ctx context.Context, tx *sql.Tx, job Job) ([]Event, error) {
	return nil, nil
}

// JobHandlers is the registry the pool dispatches on. A job whose kind is not
// in here dead-letters: no retry will conjure a handler, and the row is how
// somebody finds out the deploy was wrong.
func JobHandlers(store *Store, clock Clock) map[string]JobHandler {
	return map[string]JobHandler{
		JobKindNotify: NotifyHandler{Store: store, Clock: clock},
	}
}

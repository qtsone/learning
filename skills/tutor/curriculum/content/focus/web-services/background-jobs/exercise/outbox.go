package jobs

import (
	"context"
)

// CreateOrder writes the order and the work the order implies in one
// transaction — the transactional outbox.
//
// The two-step version everybody writes first:
//
//	db.Exec("INSERT INTO orders ...")   // committed
//	queue.Publish(confirmationJob)      // process dies here → e-mail never sent
//
// There is no ordering of those two statements that is safe. Publish first and
// a rolled-back order still triggers an e-mail (a phantom job, and the worker
// then cannot find the order). Insert first and a crash between them loses the
// job silently, which is worse: nothing failed, nothing retried, nothing
// logged. One transaction has no gap to crash into.
func CreateOrder(ctx context.Context, store *Store, o Order, job NewJob) error {
	// TODO: one transaction — INSERT INTO orders (id, email, total_cents),
	// then store.Enqueue on the same tx, then commit. Anything that fails
	// must leave neither row behind.
	return nil
}

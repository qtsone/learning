package tdd

import (
	"fmt"
	"time"
)

// Task is something with a deadline.
type Task struct {
	Title string
	Due   time.Time
}

// Clock supplies the current time — inject it so tests control "now".
type Clock interface {
	Now() time.Time
}

// Notifier delivers a message — inject it so tests can observe the calls.
type Notifier interface {
	Notify(message string) error
}

// Reminder is the testable redesign of LegacySendOverdue: both hidden
// dependencies are now explicit fields a test can substitute.
type Reminder struct {
	Clock    Clock
	Notifier Notifier
}

// SendOverdue notifies "overdue: <title>" once per overdue task, in input
// order, and returns how many notifications were delivered. A task is
// overdue when its Due is strictly before Clock.Now(). If the notifier
// fails, SendOverdue stops and returns the count delivered so far along
// with the wrapped error.
func (r Reminder) SendOverdue(tasks []Task) (int, error) {
	// One snapshot of now for the whole batch, so a slow notifier can't
	// change which tasks count as overdue mid-run.
	now := r.Clock.Now()
	sent := 0
	for _, task := range tasks {
		if !task.Due.Before(now) {
			continue
		}
		if err := r.Notifier.Notify("overdue: " + task.Title); err != nil {
			return sent, fmt.Errorf("notify %q: %w", task.Title, err)
		}
		sent++
	}
	return sent, nil
}

package tdd

import "time"

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
	// TODO: implement against reminder_test.go.
	return 0, nil
}

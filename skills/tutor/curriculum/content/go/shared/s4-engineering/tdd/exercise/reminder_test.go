package tdd

import (
	"errors"
	"slices"
	"testing"
	"time"
)

// stubClock is a stub: it answers Now() with a canned value and records
// nothing. Tests use it to pin down "now".
type stubClock struct{ now time.Time }

func (c stubClock) Now() time.Time { return c.now }

// errNotifierDown lets tests assert error propagation with errors.Is.
var errNotifierDown = errors.New("notifier down")

// spyNotifier is a spy: it records every delivered message so tests can
// assert on the interaction, and can be told to fail on one message.
type spyNotifier struct {
	messages []string
	failOn   string
}

func (s *spyNotifier) Notify(message string) error {
	if s.failOn != "" && message == s.failOn {
		return errNotifierDown
	}
	s.messages = append(s.messages, message)
	return nil
}

func TestSendOverdueNotifiesOnlyOverdueTasksInOrder(t *testing.T) {
	now := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	spy := &spyNotifier{}
	r := Reminder{Clock: stubClock{now}, Notifier: spy}

	sent, err := r.SendOverdue([]Task{
		{Title: "file taxes", Due: now.Add(-48 * time.Hour)},
		{Title: "water plants", Due: now.Add(24 * time.Hour)},
		{Title: "renew passport", Due: now.Add(-time.Minute)},
	})
	if err != nil {
		t.Fatalf("SendOverdue returned unexpected error: %v", err)
	}
	if sent != 2 {
		t.Errorf("sent = %d, want 2 (only the overdue tasks)", sent)
	}
	want := []string{"overdue: file taxes", "overdue: renew passport"}
	if !slices.Equal(spy.messages, want) {
		t.Errorf("notified %q, want %q", spy.messages, want)
	}
}

func TestSendOverdueWithNoTasksNotifiesNothing(t *testing.T) {
	spy := &spyNotifier{}
	r := Reminder{Clock: stubClock{time.Now()}, Notifier: spy}

	sent, err := r.SendOverdue(nil)
	if err != nil {
		t.Fatalf("SendOverdue returned unexpected error: %v", err)
	}
	if sent != 0 || len(spy.messages) != 0 {
		t.Errorf("sent = %d with %d notifications, want 0 and 0", sent, len(spy.messages))
	}
}

func TestSendOverdueTaskDueExactlyNowIsNotOverdue(t *testing.T) {
	now := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	spy := &spyNotifier{}
	r := Reminder{Clock: stubClock{now}, Notifier: spy}

	sent, err := r.SendOverdue([]Task{{Title: "on the line", Due: now}})
	if err != nil {
		t.Fatalf("SendOverdue returned unexpected error: %v", err)
	}
	if sent != 0 || len(spy.messages) != 0 {
		t.Errorf("task due exactly now was notified (sent=%d, messages=%q); overdue means strictly before now", sent, spy.messages)
	}
}

func TestSendOverdueStopsOnNotifierError(t *testing.T) {
	now := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	spy := &spyNotifier{failOn: "overdue: second"}
	r := Reminder{Clock: stubClock{now}, Notifier: spy}

	sent, err := r.SendOverdue([]Task{
		{Title: "first", Due: now.Add(-time.Hour)},
		{Title: "second", Due: now.Add(-time.Hour)},
		{Title: "third", Due: now.Add(-time.Hour)},
	})
	if !errors.Is(err, errNotifierDown) {
		t.Fatalf("err = %v, want the notifier's error (use %%w if you wrap it)", err)
	}
	if sent != 1 {
		t.Errorf("sent = %d, want 1 (deliveries before the failure)", sent)
	}
	want := []string{"overdue: first"}
	if !slices.Equal(spy.messages, want) {
		t.Errorf("notified %q, want %q — stop at the first failure", spy.messages, want)
	}
}

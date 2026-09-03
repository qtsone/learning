package tdd

import (
	"fmt"
	"time"
)

// LegacySendOverdue is the untestable "before" version of Part B.
// Do not modify it and do not call it from Reminder — it exists so you can
// contrast it with the shape the tests in reminder_test.go force on you.
//
// Two things make it untestable:
//   - time.Now() is a hidden input: the answer depends on when the test runs.
//   - fmt.Println is a hidden output: the result goes to the real terminal,
//     where a test cannot see it.
func LegacySendOverdue(tasks []Task) int {
	sent := 0
	for _, task := range tasks {
		if task.Due.Before(time.Now()) {
			fmt.Println("overdue: " + task.Title)
			sent++
		}
	}
	return sent
}

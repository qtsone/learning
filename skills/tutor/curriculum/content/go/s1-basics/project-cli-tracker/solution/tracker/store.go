package tracker

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ErrCorrupt is wrapped into the error Load returns when the data file
// cannot be parsed.
var ErrCorrupt = errors.New("corrupt tracker file")

// Save writes all tasks to the file at path, one task per line, in the
// format documented in LESSON.md: id|status|title, where status is
// "open" or "done".
func (t *Tracker) Save(path string) error {
	var b strings.Builder
	for _, task := range t.tasks {
		status := "open"
		if task.Done {
			status = "done"
		}
		fmt.Fprintf(&b, "%d|%s|%s\n", task.ID, status, task.Title)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("save tracker: %w", err)
	}
	return nil
}

// Load reads a tracker back from the file at path. A missing file is not
// an error — it means a fresh start, so Load returns an empty tracker.
// A line that cannot be parsed makes Load fail with an error that wraps
// ErrCorrupt and names the offending line number. Blank lines are
// skipped.
func Load(path string) (*Tracker, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return New(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("load tracker: %w", err)
	}
	t := New()
	for i, line := range strings.Split(string(data), "\n") {
		if line == "" {
			continue
		}
		task, err := parseLine(line)
		if err != nil {
			return nil, fmt.Errorf("load tracker: line %d: %w", i+1, err)
		}
		t.tasks = append(t.tasks, task)
	}
	return t, nil
}

func parseLine(line string) (Task, error) {
	parts := strings.SplitN(line, "|", 3)
	if len(parts) != 3 {
		return Task{}, fmt.Errorf("%w: want id|status|title, got %q", ErrCorrupt, line)
	}
	id, err := strconv.Atoi(parts[0])
	if err != nil {
		return Task{}, fmt.Errorf("%w: bad id %q", ErrCorrupt, parts[0])
	}
	var done bool
	switch parts[1] {
	case "open":
	case "done":
		done = true
	default:
		return Task{}, fmt.Errorf("%w: unknown status %q", ErrCorrupt, parts[1])
	}
	return Task{ID: id, Title: parts[2], Done: done}, nil
}

package tracker

import "errors"

// ErrCorrupt is wrapped into the error Load returns when the data file
// cannot be parsed.
var ErrCorrupt = errors.New("corrupt tracker file")

// Save writes all tasks to the file at path, one task per line, in the
// format documented in LESSON.md: id|status|title, where status is
// "open" or "done".
func (t *Tracker) Save(path string) error {
	// TODO: build the file content line by line, then write it in one go.
	return errors.New("TODO: implement Save")
}

// Load reads a tracker back from the file at path. A missing file is not
// an error — it means a fresh start, so Load returns an empty tracker.
// A line that cannot be parsed makes Load fail with an error that wraps
// ErrCorrupt and names the offending line number. Blank lines are
// skipped.
func Load(path string) (*Tracker, error) {
	// TODO: read the file, split it into lines, parse each line.
	return nil, errors.New("TODO: implement Load")
}

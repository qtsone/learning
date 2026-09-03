package composition

// Recorder collects events in the order they happen. It is the reusable base
// behavior for this exercise — complete as given, do not modify.
type Recorder struct {
	events []string
}

// Record appends one event to the history.
func (r *Recorder) Record(event string) {
	r.events = append(r.events, event)
}

// Events returns a copy of everything recorded so far, oldest first.
func (r *Recorder) Events() []string {
	return append([]string(nil), r.events...)
}

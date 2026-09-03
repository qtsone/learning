package realtime

import (
	"errors"
	"time"
)

// ErrInvalidField reports a field value that cannot be put on the wire. In
// text/event-stream a line ends at "\n" or "\r", so a value carrying one could
// close the current field and open another: the SSE equivalent of header
// injection. Anything built from user input goes through here.
var ErrInvalidField = errors.New("realtime: invalid event field")

// Event is one server-sent event.
//
// A frame is a block of "field: value" lines ended by a blank line:
//
//	id: 42
//	event: task.created
//	retry: 3000
//	data: {"id":42}
//
// Every field is optional. A frame with no data dispatches nothing in a
// browser, but its id still updates the client's Last-Event-ID.
type Event struct {
	// ID labels this event so a reconnecting client can say where it
	// stopped. Empty means "unlabelled" — no id: line, and the client keeps
	// whatever id it had.
	ID string

	// Name is the "event:" field, the type a browser dispatches on
	// (addEventListener("task.created", …)). Empty means the default,
	// "message".
	Name string

	// Data is the payload. Newlines are legal here: each line becomes its own
	// data: line and the client rejoins them with "\n".
	Data string

	// Retry asks the client to wait this long before reconnecting. It goes on
	// the wire as whole milliseconds; anything under a millisecond is dropped
	// rather than sent as "retry: 0", which would mean "reconnect at once".
	Retry time.Duration
}

// Frame renders e in the text/event-stream wire format, fields in the order
// id, event, retry, data, followed by the blank line that ends the frame.
// An Event with nothing set renders as the empty string — there is nothing to
// send. Returns ErrInvalidField if ID or Name contains a newline, or if any
// field contains a carriage return or a NUL.
func (e Event) Frame() (string, error) {
	// TODO: build the frame described above.
	//   - validate first: ID and Name may not contain "\n"; ID, Name and Data
	//     may not contain "\r" or "\x00" (wrap ErrInvalidField with %w so
	//     callers can use errors.Is),
	//   - emit each non-empty field on its own line, retry in whole
	//     milliseconds and only when that count is at least 1,
	//   - split Data on "\n" into one data: line per line,
	//   - end with a blank line, unless nothing at all was written.
	return "", nil
}

// Comment renders text as an SSE comment frame — a line starting with ":",
// which every client parses and ignores. It is how a heartbeat is sent without
// the application seeing an event. text must be a single line; it is written
// by the server, never by a client.
func Comment(text string) string {
	// TODO: ": " + text, then the blank line that ends the frame.
	return ""
}

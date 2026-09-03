package realtime

import (
	"errors"
	"testing"
	"time"
)

func TestEventFrame(t *testing.T) {
	tests := []struct {
		name  string
		event Event
		want  string
	}{
		{
			name:  "data only",
			event: Event{Data: "hello"},
			want:  "data: hello\n\n",
		},
		{
			name:  "fields come in wire order",
			event: Event{ID: "42", Name: "task.created", Data: `{"id":42}`},
			want:  "id: 42\nevent: task.created\ndata: {\"id\":42}\n\n",
		},
		{
			name:  "multi-line data is one data line per line",
			event: Event{Data: "line one\nline two"},
			want:  "data: line one\ndata: line two\n\n",
		},
		{
			name:  "retry is whole milliseconds",
			event: Event{Retry: 3 * time.Second},
			want:  "retry: 3000\n\n",
		},
		{
			name:  "retry under a millisecond is dropped, not rounded to zero",
			event: Event{Retry: 500 * time.Microsecond, Data: "x"},
			want:  "data: x\n\n",
		},
		{
			name:  "an id with no data still carries the id",
			event: Event{ID: "7"},
			want:  "id: 7\n\n",
		},
		{
			name:  "an empty event renders nothing",
			event: Event{},
			want:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.event.Frame()
			if err != nil {
				t.Fatalf("Frame() error = %v, want nil", err)
			}
			if got != tc.want {
				t.Errorf("Frame() =\n%q\nwant\n%q", got, tc.want)
			}
		})
	}
}

// A newline in a field would end the line and let the value open a field of
// its own — id: "1\nevent: admin" is an event named admin that you never
// meant to send. Reject it at the frame boundary, where the wire format is
// known, rather than hoping every caller sanitises first.
func TestEventFrameRejectsFieldInjection(t *testing.T) {
	tests := []struct {
		name  string
		event Event
	}{
		{"newline in id", Event{ID: "1\nevent: admin", Data: "x"}},
		{"newline in name", Event{Name: "ping\ndata: forged", Data: "x"}},
		{"carriage return in id", Event{ID: "1\revent: admin", Data: "x"}},
		{"carriage return in data", Event{Data: "one\rtwo"}},
		{"NUL in id", Event{ID: "a\x00b", Data: "x"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.event.Frame()
			if !errors.Is(err, ErrInvalidField) {
				t.Fatalf("Frame() error = %v, want ErrInvalidField", err)
			}
			if got != "" {
				t.Errorf("Frame() = %q, want no frame at all when it is invalid", got)
			}
		})
	}
}

func TestComment(t *testing.T) {
	if got, want := Comment("heartbeat"), ": heartbeat\n\n"; got != want {
		t.Errorf("Comment(\"heartbeat\") = %q, want %q", got, want)
	}
}

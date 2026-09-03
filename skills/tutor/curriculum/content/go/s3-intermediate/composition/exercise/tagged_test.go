package composition

import (
	"slices"
	"testing"
)

// Sink is what the rest of a program would need from any recorder — the
// small, consumer-side interface idea from the interfaces lesson. Both
// recorder types must satisfy it (promotion makes that free).
type Sink interface {
	Record(event string)
}

var (
	_ Sink = (*Recorder)(nil)
	_ Sink = (*TaggedRecorder)(nil)
)

func TestTaggedRecordPrefixesEvents(t *testing.T) {
	tr := &TaggedRecorder{Tag: "auth"}
	tr.Record("login")
	tr.Record("logout")
	want := []string{"[auth] login", "[auth] logout"}
	if got := tr.Events(); !slices.Equal(got, want) {
		t.Errorf("after tr.Record: Events() = %q, want %q\n(is your Record shadowing the promoted one and adding the tag?)", got, want)
	}
}

func TestPromotedEventsOnFreshRecorder(t *testing.T) {
	tr := &TaggedRecorder{Tag: "db"}
	if got := tr.Events(); len(got) != 0 {
		t.Errorf("fresh TaggedRecorder: Events() = %q, want empty (Events must stay the promoted Recorder method)", got)
	}
}

func TestInnerRecorderStillReachable(t *testing.T) {
	tr := &TaggedRecorder{Tag: "auth"}
	tr.Record("login")
	tr.Recorder.Record("raw")
	want := []string{"[auth] login", "raw"}
	if got := tr.Events(); !slices.Equal(got, want) {
		t.Errorf("mixing tr.Record and tr.Recorder.Record: Events() = %q, want %q\n(your Record must delegate to the embedded Recorder so both write the same history)", got, want)
	}
}

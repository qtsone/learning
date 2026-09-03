package composition

// TaggedRecorder is a Recorder whose events carry a tag, e.g. the subsystem
// they came from: with Tag "auth", Record("login") stores "[auth] login".
type TaggedRecorder struct {
	Recorder
	Tag string
}

// Record shadows the promoted Recorder.Record: it prefixes the event with
// the tag, then delegates so the embedded Recorder stays the single source
// of truth for the history.
func (t *TaggedRecorder) Record(event string) {
	t.Recorder.Record("[" + t.Tag + "] " + event)
}

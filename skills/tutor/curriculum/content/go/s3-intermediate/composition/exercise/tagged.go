package composition

// TaggedRecorder is a Recorder whose events carry a tag, e.g. the subsystem
// they came from: with Tag "auth", Record("login") stores "[auth] login".
type TaggedRecorder struct {
	Recorder
	Tag string
}

// TODO: declare a Record method on *TaggedRecorder that shadows the promoted
// Recorder.Record. It must store "[" + Tag + "] " + event by delegating to
// the embedded Recorder — do not touch the events slice directly and do not
// re-implement Events. See LESSON.md acceptance criteria 1-2.

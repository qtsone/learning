package main

import (
	"io"
	"log/slog"
)

// ContextHandler decorates another slog.Handler with the correlation ids
// carried on the context, so no call site has to remember to pass them.
type ContextHandler struct {
	slog.Handler
}

// TODO: give ContextHandler a Handle method. It receives the ctx of the
// logging call, so it can read the request id (RequestIDFrom) and the active
// span (SpanContextFrom) and add request_id / trace_id / span_id to the
// record before delegating to the embedded handler. Ids that are absent must
// not be logged at all — an empty request_id looks like data.

// TODO: give ContextHandler WithAttrs and WithGroup methods. slog calls them
// for every logger.With(...) / logger.WithGroup(...); if the embedded handler
// answers instead, your decorator is dropped from the derived logger and the
// ids silently disappear.

// NewLogger returns the service logger: JSON records on w, filtered by
// level, with correlation ids pulled off the context of every call. Pass a
// *slog.LevelVar as level to raise or lower verbosity while the process
// runs.
func NewLogger(w io.Writer, level slog.Leveler) *slog.Logger {
	// TODO: build a JSON handler that honours level, and wrap it in a
	// ContextHandler.
	return slog.New(slog.NewJSONHandler(w, nil))
}

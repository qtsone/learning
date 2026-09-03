package main

import (
	"context"
	"io"
	"log/slog"
)

// ContextHandler decorates another slog.Handler with the correlation ids
// carried on the context, so no call site has to remember to pass them.
type ContextHandler struct {
	slog.Handler
}

// Handle copies the ids off ctx onto the record and delegates. Ids that are
// not there are not logged at all — an empty request_id is worse than none,
// because it looks like data.
func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := RequestIDFrom(ctx); id != "" {
		r.AddAttrs(slog.String("request_id", id))
	}
	if sc, ok := SpanContextFrom(ctx); ok {
		r.AddAttrs(
			slog.String("trace_id", sc.TraceID),
			slog.String("span_id", sc.SpanID),
		)
	}
	return h.Handler.Handle(ctx, r)
}

// WithAttrs and WithGroup must re-wrap: slog calls them for every
// logger.With(...), and returning the inner handler would silently strip the
// decorator from every derived logger.
func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *ContextHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return &ContextHandler{Handler: h.Handler.WithGroup(name)}
}

// NewLogger returns the service logger: JSON records on w, filtered by
// level, with correlation ids pulled off the context of every call. Pass a
// *slog.LevelVar as level to raise or lower verbosity while the process
// runs.
func NewLogger(w io.Writer, level slog.Leveler) *slog.Logger {
	return slog.New(&ContextHandler{
		Handler: slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level}),
	})
}

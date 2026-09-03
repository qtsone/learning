package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

// records parses every JSON line the logger wrote. Structured logging is only
// worth anything if a machine can read it, so the tests read it as a machine.
func records(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("log output is not one JSON object per line: %q (%v)", buf.String(), err)
		}
		out = append(out, m)
	}
	return out
}

func TestNewLoggerWritesJSONRecords(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, slog.LevelInfo)
	logger.Info("service started", "port", 8080)

	recs := records(t, &buf)
	if len(recs) != 1 {
		t.Fatalf("got %d records, want 1", len(recs))
	}
	rec := recs[0]
	if rec["msg"] != "service started" {
		t.Errorf(`"msg" = %v, want "service started"`, rec["msg"])
	}
	if rec["level"] != "INFO" {
		t.Errorf(`"level" = %v, want "INFO"`, rec["level"])
	}
	if rec["port"] != float64(8080) {
		t.Errorf(`"port" = %v, want 8080 — attrs belong in fields, not in the message`, rec["port"])
	}
	if _, ok := rec["time"]; !ok {
		t.Error(`record has no "time" field`)
	}
}

func TestNewLoggerLevelIsDynamic(t *testing.T) {
	var buf bytes.Buffer
	level := new(slog.LevelVar) // zero value is LevelInfo
	logger := NewLogger(&buf, level)

	logger.Debug("too chatty")
	if buf.Len() != 0 {
		t.Fatalf("a DEBUG record was written while the level was INFO: %s", buf.String())
	}

	level.Set(slog.LevelDebug)
	logger.Debug("now interesting")
	if !strings.Contains(buf.String(), "now interesting") {
		t.Errorf("after level.Set(LevelDebug) the DEBUG record was still dropped: %q — pass the Leveler to the handler, do not read it once", buf.String())
	}
}

func TestContextHandlerAddsCorrelationIDs(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, slog.LevelInfo)

	ctx := WithRequestID(context.Background(), "req-1")
	ctx = ContextWithSpanContext(ctx, SpanContext{
		TraceID: "0af7651916cd43dd8448eb211c80319c",
		SpanID:  "b7ad6b7169203331",
		Sampled: true,
	})
	logger.InfoContext(ctx, "with ids")
	logger.Info("without ids")

	recs := records(t, &buf)
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2", len(recs))
	}
	want := map[string]string{
		"request_id": "req-1",
		"trace_id":   "0af7651916cd43dd8448eb211c80319c",
		"span_id":    "b7ad6b7169203331",
	}
	for k, v := range want {
		if recs[0][k] != v {
			t.Errorf("record logged with a context: %q = %v, want %q", k, recs[0][k], v)
		}
		if got, ok := recs[1][k]; ok {
			t.Errorf("record logged without a context carries %q = %v — omit ids that are not there", k, got)
		}
	}
}

func TestContextHandlerSurvivesWithAndWithGroup(t *testing.T) {
	var buf bytes.Buffer

	logger := NewLogger(&buf, slog.LevelInfo).With("component", "api")
	if _, ok := logger.Handler().(*ContextHandler); !ok {
		t.Fatalf("logger.With(...) produced a %T — WithAttrs must return a *ContextHandler or the decorator is lost", logger.Handler())
	}

	grouped := NewLogger(&buf, slog.LevelInfo).WithGroup("http")
	if _, ok := grouped.Handler().(*ContextHandler); !ok {
		t.Fatalf("logger.WithGroup(...) produced a %T — WithGroup must return a *ContextHandler too", grouped.Handler())
	}

	logger.InfoContext(WithRequestID(context.Background(), "req-2"), "derived")
	rec := records(t, &buf)[0]
	if rec["component"] != "api" {
		t.Errorf(`"component" = %v, want "api" — the attrs from With must survive`, rec["component"])
	}
	if rec["request_id"] != "req-2" {
		t.Errorf(`"request_id" = %v, want "req-2" — the context ids must survive too`, rec["request_id"])
	}
}

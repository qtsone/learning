package config_test

import (
	"errors"
	"log/slog"
	"testing"
	"time"

	"tutor.local/capstone-reference/internal/config"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Addr != ":8080" {
		t.Errorf("Addr = %q, want %q", cfg.Addr, ":8080")
	}
	if cfg.ShutdownTimeout != 15*time.Second {
		t.Errorf("ShutdownTimeout = %v, want 15s", cfg.ShutdownTimeout)
	}
}

func TestLoadFromEnvironment(t *testing.T) {
	t.Setenv("TINYSVC_ADDR", "127.0.0.1:9000")
	t.Setenv("TINYSVC_LOG_LEVEL", "debug")
	t.Setenv("TINYSVC_SHUTDOWN_TIMEOUT", "2s")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Addr != "127.0.0.1:9000" {
		t.Errorf("Addr = %q, want %q", cfg.Addr, "127.0.0.1:9000")
	}
	if cfg.LogLevel != slog.LevelDebug {
		t.Errorf("LogLevel = %v, want debug", cfg.LogLevel)
	}
	if cfg.ShutdownTimeout != 2*time.Second {
		t.Errorf("ShutdownTimeout = %v, want 2s", cfg.ShutdownTimeout)
	}
}

// A bad value stops the process at startup rather than being silently replaced
// by a default: the rollout then fails while the old version is still serving.
func TestLoadRejectsBadValues(t *testing.T) {
	cases := []struct{ name, key, value string }{
		{"address without a port", "TINYSVC_ADDR", "8080"},
		{"address with two colons", "TINYSVC_ADDR", "127.0.0.1:80:80"},
		{"log level", "TINYSVC_LOG_LEVEL", "chatty"},
		{"timeout syntax", "TINYSVC_SHUTDOWN_TIMEOUT", "soon"},
		{"timeout sign", "TINYSVC_SHUTDOWN_TIMEOUT", "-1s"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv(c.key, c.value)
			_, err := config.Load()
			if err == nil {
				t.Fatalf("Load() with %s=%q error = nil, want a startup failure", c.key, c.value)
			}
			if !errors.Is(err, config.ErrInvalidValue) {
				t.Errorf("Load() error = %v, want it to wrap ErrInvalidValue", err)
			}
		})
	}
}

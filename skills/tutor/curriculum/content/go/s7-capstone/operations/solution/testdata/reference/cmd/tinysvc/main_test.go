package main

import (
	"context"
	"io"
	"log/slog"
	"net"
	"strings"
	"testing"
	"time"

	"tutor.local/capstone-reference/internal/config"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

// The rollout case: the signal context ends, readiness is withdrawn, the
// in-flight work drains inside the bounded window, and run reports success.
// main is a shell around this, which is why the root context lives there.
func TestRunDrainsWhenTheSignalContextEnds(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cfg := config.Config{Addr: "127.0.0.1:0", ShutdownTimeout: 2 * time.Second}
	if err := run(ctx, cfg, testLogger()); err != nil {
		t.Fatalf("run() error = %v, want a clean shutdown", err)
	}
}

// A port already in use is the first failure mode in the runbook. It has to
// stop the process with an error that names the address, not leave something
// running that is up and serving nothing.
func TestRunFailsWhenTheAddressIsAlreadyTaken(t *testing.T) {
	taken, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("cannot bind a loopback port on this machine: %v", err)
	}
	defer taken.Close()

	cfg := config.Config{Addr: taken.Addr().String(), ShutdownTimeout: time.Second}
	err = run(context.Background(), cfg, testLogger())
	if err == nil {
		t.Fatal("run() on a taken port error = nil, want a startup failure")
	}
	if !strings.Contains(err.Error(), taken.Addr().String()) {
		t.Errorf("run() error = %v, want it to name the address it could not bind", err)
	}
}

package main

import (
	"bytes"
	"log/slog"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestForwardShutdownSignalsLogsSignal(t *testing.T) {
	var logs bytes.Buffer
	signals := make(chan os.Signal, 1)
	shutdown := make(chan struct{}, 1)
	done := make(chan struct{})
	go func() {
		forwardShutdownSignals(
			signals,
			shutdown,
			slog.New(slog.NewTextHandler(&logs, nil)),
		)
		close(done)
	}()

	signals <- syscall.SIGTERM
	close(signals)

	select {
	case <-shutdown:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for forwarded shutdown")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for signal forwarding to stop")
	}
	if !strings.Contains(logs.String(), "runner received shutdown signal") {
		t.Fatalf("missing shutdown signal log:\n%s", logs.String())
	}
	if !strings.Contains(logs.String(), "signal="+syscall.SIGTERM.String()) {
		t.Fatalf("missing signal name:\n%s", logs.String())
	}
}

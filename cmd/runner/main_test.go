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

func TestPrintVersion(t *testing.T) {
	previousVersion := Version
	Version = "v1.2.3-rc.1"
	t.Cleanup(func() { Version = previousVersion })

	var output bytes.Buffer
	if !printVersion([]string{"--version"}, &output) {
		t.Fatal("expected --version to be handled")
	}
	if output.String() != "runner v1.2.3-rc.1\n" {
		t.Fatalf("version output = %q", output.String())
	}
}

package main

import (
	"bytes"
	"log/slog"
	"os"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/spf13/pflag"
)

func TestRunnerTagsAcceptListAndSingleTag(t *testing.T) {
	t.Setenv("RUNNER_TAGS", `fleet_manager_id=manager-a,"note=hello, world"`)
	flags := pflag.NewFlagSet("runner", pflag.ContinueOnError)
	var tags map[string]string
	if err := registerRunnerTagFlags(flags, &tags); err != nil {
		t.Fatal(err)
	}
	if err := flags.Parse([]string{"--tag", "role=worker", "--tags", "role=builder,zone=us-east-1"}); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"fleet_manager_id": "manager-a", "note": "hello, world", "role": "builder", "zone": "us-east-1"}
	if !reflect.DeepEqual(tags, want) {
		t.Fatalf("tags = %#v, want %#v", tags, want)
	}
}

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

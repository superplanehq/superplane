package runner

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type liveLogReadResult struct {
	result *LiveLogFetchResult
	err    error
}

func TestParseLiveLogRecordKeepsKindPreviewAndTools(t *testing.T) {
	t.Parallel()

	cmd, ok := parseLiveLogRecord(`{"type":"cmd_start","index":1,"text":"Clone","kind":"bash","preview":"git clone"}`)
	require.True(t, ok)
	require.Equal(t, "cmd_start", cmd.Type)
	require.Equal(t, "bash", cmd.Kind)
	require.Equal(t, "git clone", cmd.Preview)

	tool, ok := parseLiveLogRecord(`{"type":"tool_start","id":"toolu_a","kind":"read","text":"pkg/foo.go"}`)
	require.True(t, ok)
	require.Equal(t, "tool_start", tool.Type)
	require.Equal(t, "toolu_a", tool.ID)
	require.Equal(t, "read", tool.Kind)
	require.Equal(t, "pkg/foo.go", tool.Text)
}

func TestReadLiveLogRecordsReturnsAfterLimitOnOpenStream(t *testing.T) {
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()

	done := make(chan liveLogReadResult, 1)
	go func() {
		result, err := readLiveLogRecords(reader, 1)
		done <- liveLogReadResult{result: result, err: err}
	}()

	_, err := writer.Write([]byte(`{"type":"line","text":"first"}` + "\n"))
	require.NoError(t, err)

	select {
	case read := <-done:
		require.NoError(t, read.err)
		require.Len(t, read.result.Records, 1)
		require.Equal(t, "first", read.result.Records[0].Text)
		require.True(t, read.result.Truncated)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected reader to return after reaching record limit")
	}
}

func TestReadLiveLogRecordsStopsAfterLimitEvenWhenNextLineIsInvalid(t *testing.T) {
	result, err := readLiveLogRecords(strings.NewReader(`{"type":"line","text":"first"}`+"\nnot-json\n"), 1)

	require.NoError(t, err)
	require.Len(t, result.Records, 1)
	require.Equal(t, "first", result.Records[0].Text)
	require.True(t, result.Truncated)
}

func TestReadLiveLogRecordsUntilIdleReturnsPartialOpenStream(t *testing.T) {
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()

	done := make(chan liveLogReadResult, 1)
	go func() {
		result, err := readLiveLogRecordsUntilIdle(context.Background(), reader, 10, 20*time.Millisecond)
		done <- liveLogReadResult{result: result, err: err}
	}()

	_, err := writer.Write([]byte(`{"type":"line","text":"first"}` + "\n"))
	require.NoError(t, err)

	select {
	case read := <-done:
		require.NoError(t, read.err)
		require.Len(t, read.result.Records, 1)
		require.Equal(t, "first", read.result.Records[0].Text)
		require.False(t, read.result.Truncated)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected reader to return after idle timeout")
	}
}

func TestReadLiveLogRecordsUntilIdleReturnsContextCancellation(t *testing.T) {
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan liveLogReadResult, 1)
	go func() {
		result, err := readLiveLogRecordsUntilIdle(ctx, reader, 10, time.Minute)
		done <- liveLogReadResult{result: result, err: err}
	}()

	cancel()

	select {
	case read := <-done:
		require.Nil(t, read.result)
		require.ErrorIs(t, read.err, context.Canceled)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected reader to return after context cancellation")
	}
}

func TestDrainReadyLiveLogReadEventsKeepsQueuedRecord(t *testing.T) {
	events := make(chan liveLogReadEvent, 1)
	events <- liveLogReadEvent{record: LiveLogRecord{Type: "line", Text: "second"}}

	result, complete, err := drainReadyLiveLogReadEvents(
		[]LiveLogRecord{{Type: "line", Text: "first"}},
		events,
		10,
	)

	require.NoError(t, err)
	require.False(t, complete)
	require.Len(t, result.Records, 2)
	require.Equal(t, "first", result.Records[0].Text)
	require.Equal(t, "second", result.Records[1].Text)
}

func TestFetchLiveLogRecordsUsesInternalBrokerURL(t *testing.T) {
	internal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/tasks/task-1/live-logs", r.URL.Path)
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = w.Write([]byte(`{"type":"line","text":"from-internal"}` + "\n"))
	}))
	t.Cleanup(internal.Close)

	t.Setenv("TASK_BROKER_BASE_URL", internal.URL)
	t.Setenv("TASK_BROKER_PUBLIC_URL", "http://127.0.0.1:1")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "live-log-secret")

	result, err := FetchLiveLogRecords(context.Background(), "task-1", LiveLogFetchOptions{Limit: 1})
	require.NoError(t, err)
	require.Len(t, result.Records, 1)
	require.Equal(t, "from-internal", result.Records[0].Text)
}

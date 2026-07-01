package runner

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type liveLogReadResult struct {
	result *LiveLogFetchResult
	err    error
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

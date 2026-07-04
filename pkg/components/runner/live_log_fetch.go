package runner

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	DefaultLiveLogRecordLimit = 200
	MaxLiveLogRecordLimit     = 1000
)

type LiveLogRecord struct {
	Type       string `json:"type,omitempty"`
	Text       string `json:"text,omitempty"`
	Message    string `json:"message,omitempty"`
	Index      *int   `json:"index,omitempty"`
	Status     string `json:"status,omitempty"`
	DurationMS *int64 `json:"duration_ms,omitempty"`
	StartedAt  *int64 `json:"started_at,omitempty"`
}

type LiveLogFetchOptions struct {
	Limit       int
	HTTPClient  *http.Client
	Now         time.Time
	IdleTimeout time.Duration
}

type LiveLogFetchResult struct {
	Records   []LiveLogRecord
	Truncated bool
}

func FetchLiveLogRecords(ctx context.Context, brokerTaskID string, opts LiveLogFetchOptions) (*LiveLogFetchResult, error) {
	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}

	session, err := NewLiveLogSession(brokerTaskID, now)
	if err != nil {
		return nil, err
	}

	return FetchLiveLogSessionRecords(ctx, *session, opts)
}

func FetchLiveLogSessionRecords(ctx context.Context, session LiveLogSession, opts LiveLogFetchOptions) (*LiveLogFetchResult, error) {
	limit := normalizeLiveLogRecordLimit(opts.Limit)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, session.StreamURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/x-ndjson")
	request.Header.Set("Accept-Encoding", "identity")
	request.Header.Set("Authorization", "Bearer "+session.Token)

	response, err := liveLogHTTPClient(opts.HTTPClient).Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = response.Status
		}
		return nil, fmt.Errorf("fetch live logs: %s", message)
	}

	if opts.IdleTimeout > 0 {
		return readLiveLogRecordsUntilIdle(ctx, response.Body, limit, opts.IdleTimeout)
	}
	return readLiveLogRecords(response.Body, limit)
}

func normalizeLiveLogRecordLimit(limit int) int {
	if limit <= 0 {
		return DefaultLiveLogRecordLimit
	}
	if limit > MaxLiveLogRecordLimit {
		return MaxLiveLogRecordLimit
	}
	return limit
}

func liveLogHTTPClient(client *http.Client) *http.Client {
	if client != nil {
		return client
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func readLiveLogRecords(reader io.Reader, limit int) (*LiveLogFetchResult, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	records := make([]LiveLogRecord, 0, min(limit, 32))
	for scanner.Scan() {
		if len(records) >= limit {
			return &LiveLogFetchResult{Records: records, Truncated: true}, nil
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		record, ok := parseLiveLogRecord(line)
		if !ok {
			continue
		}

		records = append(records, record)
		if len(records) >= limit {
			return &LiveLogFetchResult{Records: records, Truncated: true}, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read live logs: %w", err)
	}
	return &LiveLogFetchResult{Records: records}, nil
}

type liveLogReadEvent struct {
	record LiveLogRecord
	err    error
	done   bool
}

func readLiveLogRecordsUntilIdle(
	ctx context.Context,
	reader io.Reader,
	limit int,
	idleTimeout time.Duration,
) (*LiveLogFetchResult, error) {
	done := make(chan struct{})
	events := make(chan liveLogReadEvent)

	if closer, ok := reader.(io.Closer); ok {
		defer func() {
			close(done)
			_ = closer.Close()
		}()
	} else {
		defer close(done)
	}

	go streamLiveLogReadEvents(reader, events, done)

	records := make([]LiveLogRecord, 0, min(limit, 32))
	idle := time.NewTimer(idleTimeout)
	defer idle.Stop()

	for {
		select {
		case event := <-events:
			result, complete, err := applyLiveLogReadEvent(records, event, limit)
			if err != nil {
				return nil, err
			}
			records = result.Records
			if complete {
				return result, nil
			}
			resetLiveLogIdleTimer(idle, idleTimeout)
		case <-idle.C:
			result, complete, err := drainReadyLiveLogReadEvents(records, events, limit)
			if err != nil {
				return nil, err
			}
			if complete {
				return result, nil
			}
			if len(result.Records) > len(records) {
				records = result.Records
				resetLiveLogIdleTimer(idle, idleTimeout)
				continue
			}
			return &LiveLogFetchResult{Records: records}, nil
		case <-ctx.Done():
			return nil, fmt.Errorf("read live logs: %w", ctx.Err())
		}
	}
}

func drainReadyLiveLogReadEvents(
	records []LiveLogRecord,
	events <-chan liveLogReadEvent,
	limit int,
) (*LiveLogFetchResult, bool, error) {
	result := &LiveLogFetchResult{Records: records}

	for {
		select {
		case event := <-events:
			next, complete, err := applyLiveLogReadEvent(result.Records, event, limit)
			if err != nil {
				return nil, false, err
			}
			result = next
			if complete {
				return result, true, nil
			}
		default:
			return result, false, nil
		}
	}
}

func applyLiveLogReadEvent(
	records []LiveLogRecord,
	event liveLogReadEvent,
	limit int,
) (*LiveLogFetchResult, bool, error) {
	if event.err != nil {
		return nil, false, fmt.Errorf("read live logs: %w", event.err)
	}
	if event.done {
		return &LiveLogFetchResult{Records: records}, true, nil
	}

	records = append(records, event.record)
	if len(records) >= limit {
		return &LiveLogFetchResult{Records: records, Truncated: true}, true, nil
	}
	return &LiveLogFetchResult{Records: records}, false, nil
}

func streamLiveLogReadEvents(reader io.Reader, events chan<- liveLogReadEvent, done <-chan struct{}) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		record, ok := parseLiveLogRecord(line)
		if !ok {
			continue
		}

		if !sendLiveLogReadEvent(events, done, liveLogReadEvent{record: record}) {
			return
		}
	}

	if err := scanner.Err(); err != nil {
		_ = sendLiveLogReadEvent(events, done, liveLogReadEvent{err: err})
		return
	}
	_ = sendLiveLogReadEvent(events, done, liveLogReadEvent{done: true})
}

func sendLiveLogReadEvent(events chan<- liveLogReadEvent, done <-chan struct{}, event liveLogReadEvent) bool {
	select {
	case events <- event:
		return true
	case <-done:
		return false
	}
}

func resetLiveLogIdleTimer(timer *time.Timer, idleTimeout time.Duration) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(idleTimeout)
}

func parseLiveLogRecord(line string) (LiveLogRecord, bool) {
	var record LiveLogRecord
	if err := json.Unmarshal([]byte(line), &record); err != nil {
		return LiveLogRecord{}, false
	}
	if strings.TrimSpace(record.Type) == "" {
		return LiveLogRecord{}, false
	}
	return record, true
}

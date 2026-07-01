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
	Limit      int
	HTTPClient *http.Client
	Now        time.Time
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
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		record, ok := parseLiveLogRecord(line)
		if !ok {
			continue
		}
		if len(records) >= limit {
			return &LiveLogFetchResult{Records: records, Truncated: true}, nil
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read live logs: %w", err)
	}
	return &LiveLogFetchResult{Records: records}, nil
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

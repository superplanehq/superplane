// Package livelogs streams Amazon CloudWatch Logs task output as NDJSON for the task-broker API.
package livelogs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
)

const (
	// HeartbeatInterval bounds quiet periods on live-log responses.
	HeartbeatInterval = 15 * time.Second
	// pageSize is the max events per GetLogEvents call (CloudWatch allows up to 10_000).
	pageSize int32 = 10_000
	// pollQuiet waits when tailing and no new events have arrived.
	pollQuiet = 750 * time.Millisecond
	// pollActive waits after a partial page before polling again.
	pollActive = 300 * time.Millisecond
	// terminalCatchUp keeps polling after a task is terminal so late CloudWatch events can arrive.
	terminalCatchUp = 10 * time.Second
	// requestBudgetMargin leaves time to write a heartbeat before HeartbeatInterval elapses.
	requestBudgetMargin = 500 * time.Millisecond
	// maxConsecutiveLogTimeouts ends a stream that keeps timing out on the same page.
	// Each attempt is bounded by HeartbeatInterval, so four timeouts are about one minute.
	maxConsecutiveLogTimeouts = 4
)

// heartbeatInterval is the quiet-period bound used by CloudWatch live-log streams.
var heartbeatInterval = HeartbeatInterval

// errLogServiceTimeout is written when CloudWatch keeps timing out.
var errLogServiceTimeout = errors.New("log service did not respond")

type ndjsonWriter struct {
	w         io.Writer
	f         http.Flusher
	lastFlush time.Time
}

func (n *ndjsonWriter) writeRecord(rec map[string]any) error {
	b, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	_, err = n.w.Write(append(b, '\n'))
	return err
}

func (n *ndjsonWriter) flush() {
	if n.f != nil {
		n.f.Flush()
	}
	n.lastFlush = time.Now()
}

func (n *ndjsonWriter) pingIfDue(within time.Duration) error {
	if time.Since(n.lastFlush)+within < heartbeatInterval {
		return nil
	}
	if err := n.writeRecord(map[string]any{"type": "ping"}); err != nil {
		return err
	}
	n.flush()
	return nil
}

func (n *ndjsonWriter) logRequestBudget() time.Duration {
	budget := heartbeatInterval - time.Since(n.lastFlush) - requestBudgetMargin
	if budget < time.Millisecond {
		return time.Millisecond
	}
	return budget
}

func (n *ndjsonWriter) writeStreamError(err error) error {
	_ = n.writeRecord(map[string]any{
		"type":    "error",
		"message": err.Error(),
	})
	n.flush()
	return err
}

// StreamCloudWatchLogToNDJSON tails a CloudWatch Logs stream and writes newline-delimited JSON records:
// {"type":"line","text":"..."} for regular lines, {"type":"cmd_start"...}/{"type":"cmd_end"...}
// for runner command boundaries, {"type":"ping"} during quiet periods, and
// {"type":"error","message":"..."} on fatal errors. A terminal task ends after a short quiet catch-up so late log lines can arrive.
// Repeated CloudWatch timeouts end the stream with an error record instead of retrying for the life of the connection.
func StreamCloudWatchLogToNDJSON(ctx context.Context, w io.Writer, flusher http.Flusher, group, stream, region string, isTaskTerminal func(context.Context) (bool, error)) error {
	group = strings.TrimSpace(group)
	stream = strings.TrimSpace(stream)
	if group == "" || stream == "" {
		return fmt.Errorf("log group and stream are required")
	}

	region = strings.TrimSpace(region)
	if region == "" {
		region = strings.TrimSpace(os.Getenv("AWS_REGION"))
	}
	if region == "" {
		region = strings.TrimSpace(os.Getenv("AWS_DEFAULT_REGION"))
	}
	if region == "" {
		region = "us-east-1"
	}

	awscfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return fmt.Errorf("aws config: %w", err)
	}

	client := cloudwatchlogs.NewFromConfig(awscfg)
	nw := ndjsonWriter{w: w, f: flusher, lastFlush: time.Now()}

	var nextForward *string
	var lastToken string
	var catchUp terminalCatchUpWindow
	var timeouts int

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := nw.pingIfDue(0); err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}

		out, retry, err := fetchLogEvents(ctx, client, &cloudwatchlogs.GetLogEventsInput{
			LogGroupName:  awsString(group),
			LogStreamName: awsString(stream),
			NextToken:     nextForward,
			StartFromHead: awsBool(nextForward == nil),
			Limit:         awsInt32(pageSize),
		}, nw.logRequestBudget())
		if retry {
			timeouts++
			if timeouts >= maxConsecutiveLogTimeouts {
				return nw.writeStreamError(errLogServiceTimeout)
			}
			if err := nw.pingIfDue(requestBudgetMargin); err != nil {
				return err
			}
			continue
		}
		timeouts = 0
		if err != nil {
			return nw.writeStreamError(err)
		}

		token := awsToString(out.NextForwardToken)
		nextForward = out.NextForwardToken

		caughtUp := len(out.Events) == 0 && token != "" && token == lastToken
		if len(out.Events) > 0 || !caughtUp || !taskTerminal(ctx, isTaskTerminal) {
			catchUp.reset()
		} else if catchUp.due(time.Now()) {
			return nil
		}

		lastToken = token

		for _, ev := range out.Events {
			if err := nw.writeRecord(RecordFromLogMessage(awsToString(ev.Message))); err != nil {
				return err
			}
		}
		if len(out.Events) > 0 {
			nw.flush()
		}

		// Full page means more backlog may remain; poll immediately.
		if len(out.Events) >= int(pageSize) {
			continue
		}

		delay := pollActive
		if caughtUp {
			delay = pollQuiet
		}
		// Wake at the heartbeat deadline even when it falls between polls.
		delay = min(delay, max(0, heartbeatInterval-time.Since(nw.lastFlush)))
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
}

// RecordFromLogMessage maps one runner log message to an NDJSON live-log record.
func RecordFromLogMessage(message string) map[string]any {
	if rec, ok := parseRunnerControlRecord(message); ok {
		return rec
	}
	return map[string]any{"type": "line", "text": message}
}

func parseRunnerControlRecord(message string) (map[string]any, bool) {
	var envelope struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal([]byte(message), &envelope); err != nil {
		return nil, false
	}
	switch envelope.Type {
	case "cmd_start":
		var rec struct {
			Type      string `json:"type"`
			Index     int    `json:"index"`
			Text      string `json:"text"`
			Kind      string `json:"kind"`
			Preview   string `json:"preview"`
			StartedAt *int64 `json:"started_at"`
		}
		if err := json.Unmarshal([]byte(message), &rec); err != nil {
			return nil, false
		}
		if rec.Index < 0 {
			return nil, false
		}
		out := map[string]any{
			"type":  "cmd_start",
			"index": rec.Index,
			"text":  rec.Text,
		}
		if kind := strings.TrimSpace(rec.Kind); kind != "" {
			out["kind"] = kind
		}
		if preview := strings.TrimSpace(rec.Preview); preview != "" {
			out["preview"] = preview
		}
		if rec.StartedAt != nil && *rec.StartedAt >= 0 {
			out["started_at"] = *rec.StartedAt
		}
		return out, true
	case "tool_start":
		var rec struct {
			Type      string `json:"type"`
			ID        string `json:"id"`
			Kind      string `json:"kind"`
			Text      string `json:"text"`
			StartedAt *int64 `json:"started_at"`
		}
		if err := json.Unmarshal([]byte(message), &rec); err != nil {
			return nil, false
		}
		out := map[string]any{
			"type": "tool_start",
			"kind": strings.TrimSpace(rec.Kind),
			"text": rec.Text,
		}
		if id := strings.TrimSpace(rec.ID); id != "" {
			out["id"] = id
		}
		if rec.StartedAt != nil && *rec.StartedAt >= 0 {
			out["started_at"] = *rec.StartedAt
		}
		return out, true
	case "tool_end":
		var rec struct {
			Type       string `json:"type"`
			ID         string `json:"id"`
			Kind       string `json:"kind"`
			Status     string `json:"status"`
			DurationMS int64  `json:"duration_ms"`
		}
		if err := json.Unmarshal([]byte(message), &rec); err != nil {
			return nil, false
		}
		if rec.DurationMS < 0 {
			return nil, false
		}
		if rec.Status != "passed" && rec.Status != "failed" {
			return nil, false
		}
		out := map[string]any{
			"type":        "tool_end",
			"status":      rec.Status,
			"duration_ms": rec.DurationMS,
		}
		if id := strings.TrimSpace(rec.ID); id != "" {
			out["id"] = id
		}
		if kind := strings.TrimSpace(rec.Kind); kind != "" {
			out["kind"] = kind
		}
		return out, true
	case "cmd_end":
		var rec struct {
			Type       string `json:"type"`
			Index      int    `json:"index"`
			Status     string `json:"status"`
			DurationMS int64  `json:"duration_ms"`
		}
		if err := json.Unmarshal([]byte(message), &rec); err != nil {
			return nil, false
		}
		if rec.Index < 0 {
			return nil, false
		}
		if rec.DurationMS < 0 {
			return nil, false
		}
		if rec.Status != "passed" && rec.Status != "failed" {
			return nil, false
		}
		return map[string]any{
			"type":        "cmd_end",
			"index":       rec.Index,
			"status":      rec.Status,
			"duration_ms": rec.DurationMS,
		}, true
	default:
		return nil, false
	}
}

func awsString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func awsBool(b bool) *bool { return &b }

func awsInt32(n int32) *int32 { return &n }

func awsToString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

type terminalCatchUpWindow struct {
	quietSince time.Time
}

func (w *terminalCatchUpWindow) reset() {
	w.quietSince = time.Time{}
}

func (w *terminalCatchUpWindow) due(now time.Time) bool {
	if w.quietSince.IsZero() {
		w.quietSince = now
	}
	return now.Sub(w.quietSince) >= terminalCatchUp
}

func taskTerminal(ctx context.Context, isTaskTerminal func(context.Context) (bool, error)) bool {
	if isTaskTerminal == nil {
		return false
	}
	terminal, err := isTaskTerminal(ctx)
	return err == nil && terminal
}

func fetchLogEvents(ctx context.Context, client *cloudwatchlogs.Client, input *cloudwatchlogs.GetLogEventsInput, budget time.Duration) (*cloudwatchlogs.GetLogEventsOutput, bool, error) {
	reqCtx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	out, err := client.GetLogEvents(reqCtx, input)
	if err == nil {
		return out, false, nil
	}
	if ctx.Err() != nil {
		return nil, false, ctx.Err()
	}
	if reqCtx.Err() != nil {
		return nil, true, nil
	}
	return nil, false, err
}

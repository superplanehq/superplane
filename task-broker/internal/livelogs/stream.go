// Package livelogs streams Amazon CloudWatch Logs task output as NDJSON for the task-broker API.
package livelogs

import (
	"context"
	"encoding/json"
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
	// pageSize is the max events per GetLogEvents call (CloudWatch allows up to 10_000).
	pageSize int32 = 10_000
	// pollQuiet waits when tailing and no new events have arrived.
	pollQuiet = 750 * time.Millisecond
	// pollActive waits after a partial page before polling again.
	pollActive = 300 * time.Millisecond
)

type ndjsonWriter struct {
	w io.Writer
	f http.Flusher
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
}

// StreamCloudWatchLogToNDJSON tails a CloudWatch Logs stream and writes newline-delimited JSON records:
// {"type":"line","text":"..."} for regular lines, {"type":"cmd_start"...}/{"type":"cmd_end"...}
// for runner command boundaries, and {"type":"error","message":"..."} on fatal errors.
func StreamCloudWatchLogToNDJSON(ctx context.Context, w io.Writer, flusher http.Flusher, group, stream, region string) error {
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
	nw := ndjsonWriter{w: w, f: flusher}

	var nextForward *string
	var lastToken string

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		out, err := client.GetLogEvents(ctx, &cloudwatchlogs.GetLogEventsInput{
			LogGroupName:  awsString(group),
			LogStreamName: awsString(stream),
			NextToken:     nextForward,
			StartFromHead: awsBool(nextForward == nil),
			Limit:         awsInt32(pageSize),
		})
		if err != nil {
			_ = nw.writeRecord(map[string]any{
				"type":    "error",
				"message": err.Error(),
			})
			nw.flush()
			return err
		}

		token := awsToString(out.NextForwardToken)
		nextForward = out.NextForwardToken

		if len(out.Events) == 0 {
			if token != "" && token == lastToken {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(pollQuiet):
				}
				continue
			}
		}

		lastToken = token

		for _, ev := range out.Events {
			msg := awsToString(ev.Message)
			if rec, ok := parseRunnerControlRecord(msg); ok {
				if err := nw.writeRecord(rec); err != nil {
					return err
				}
				continue
			}
			if err := nw.writeRecord(map[string]any{"type": "line", "text": msg}); err != nil {
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

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollActive):
		}
	}
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
		if rec.StartedAt != nil && *rec.StartedAt >= 0 {
			out["started_at"] = *rec.StartedAt
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

package runnerlive

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

const pollQuiet = 2 * time.Second

type ndjsonWriter struct {
	w io.Writer
	f http.Flusher
}

func (n *ndjsonWriter) writeRecord(rec map[string]any) error {
	b, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	if _, err := n.w.Write(append(b, '\n')); err != nil {
		return err
	}
	if n.f != nil {
		n.f.Flush()
	}
	return nil
}

// StreamCloudWatchLogToNDJSON tails a CloudWatch Logs stream and writes newline-delimited JSON records:
// {"type":"line","text":"..."} for log lines, {"type":"error","message":"..."} on fatal errors.
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
			Limit:         awsInt32(256),
		})
		if err != nil {
			_ = nw.writeRecord(map[string]any{
				"type":    "error",
				"message": err.Error(),
			})
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
			if err := nw.writeRecord(map[string]any{"type": "line", "text": msg}); err != nil {
				return err
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(400 * time.Millisecond):
		}
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

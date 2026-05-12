package runner

import (
	"fmt"
	"net/http"
	"time"
)

// LogEventLine is one line from a CloudWatch log stream.
type LogEventLine struct {
	TimestampMs int64
	Message     string
}

type defaultHTTP struct {
	client *http.Client
}

func (d defaultHTTP) Do(req *http.Request) (*http.Response, error) {
	return d.client.Do(req)
}

// FetchCloudWatchLogEvents fetches a page of log lines for a CodeBuild/CloudWatch stream.
// nextForwardToken should be empty on the first request; use the returned token to poll for newer events.
func FetchCloudWatchLogEvents(groupName, streamName, nextForwardToken string) ([]LogEventLine, string, error) {
	if groupName == "" || streamName == "" {
		return nil, "", fmt.Errorf("log group and stream are required")
	}

	backend, err := loadBackendConfig()
	if err != nil {
		return nil, "", err
	}

	httpCtx := defaultHTTP{client: &http.Client{Timeout: 45 * time.Second}}
	client := newCodeBuildClient(httpCtx, backend.Credentials, backend.Region)

	raw, token, err := client.getLogEventsPaged(groupName, streamName, nextForwardToken)
	if err != nil {
		return nil, "", err
	}

	out := make([]LogEventLine, len(raw))
	for i, e := range raw {
		out[i] = LogEventLine{TimestampMs: e.Timestamp, Message: e.Message}
	}
	return out, token, nil
}

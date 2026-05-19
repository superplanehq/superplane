package superplane

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Client talks to SuperPlane's outbound fleet bridge API.
type Client struct {
	BaseURL    string
	AuthToken  string
	HTTPClient *http.Client
}

// JobSpec mirrors SuperPlane pkg/runners.JobSpec.
type JobSpec struct {
	Command                 []string              `json:"command,omitempty"`
	Commands                []string              `json:"commands,omitempty"`
	Environment             []EnvironmentVariable `json:"environment,omitempty"`
	ExecutionMode           string                `json:"execution_mode,omitempty"`
	DockerImage             string                `json:"docker_image,omitempty"`
	ExecutionTimeoutSeconds *int                  `json:"execution_timeout_seconds,omitempty"`
}

// EnvironmentVariable is one task-scoped environment variable.
type EnvironmentVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// SyncResponse is POST /api/v1/runner-fleets/sync.
type SyncResponse struct {
	Continue bool       `json:"continue"`
	Job      *BridgeJob `json:"job,omitempty"`
}

// BridgeJob is work pulled from SuperPlane.
type BridgeJob struct {
	ID   string  `json:"id"`
	Spec JobSpec `json:"spec"`
}

// CompleteRequest is POST /api/v1/runner-fleets/tasks/{id}/complete.
type CompleteRequest struct {
	ExitCode int             `json:"exit_code"`
	Output   string          `json:"output"`
	Error    string          `json:"error,omitempty"`
	Canceled bool            `json:"canceled,omitempty"`
	Result   json.RawMessage `json:"result,omitempty"`
	TaskLog  *TaskLogSink    `json:"task_log,omitempty"`
}

// TaskLogSink matches SuperPlane task_log JSON.
type TaskLogSink struct {
	Type       string                 `json:"type"`
	CloudWatch *TaskLogSinkCloudWatch `json:"cloudwatch,omitempty"`
}

// TaskLogSinkCloudWatch identifies a CloudWatch Logs stream.
type TaskLogSinkCloudWatch struct {
	LogGroupName  string `json:"log_group_name"`
	LogStreamName string `json:"log_stream_name"`
	Region        string `json:"region,omitempty"`
}

// Sync pulls the next queued job from SuperPlane, if any.
func (c *Client) Sync(ctx context.Context) (*SyncResponse, error) {
	var out SyncResponse
	if err := c.postJSON(ctx, "/api/v1/runner-fleets/sync", map[string]any{}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Complete reports a terminal task outcome to SuperPlane.
func (c *Client) Complete(ctx context.Context, taskID string, req CompleteRequest) error {
	path := "/api/v1/runner-fleets/tasks/" + strings.TrimSpace(taskID) + "/complete"
	return c.postJSON(ctx, path, req, nil)
}

func (c *Client) postJSON(ctx context.Context, path string, body any, out any) error {
	base := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if base == "" {
		return fmt.Errorf("superplane base url is empty")
	}
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if tok := strings.TrimSpace(c.AuthToken); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}

	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(respBody))
		if msg == "" {
			msg = resp.Status
		}
		return fmt.Errorf("superplane POST %s: %s", path, msg)
	}
	if out == nil || len(respBody) == 0 {
		return nil
	}
	return json.Unmarshal(respBody, out)
}

// NewClientFromEnv returns a client when SUPERPLANE_URL and SUPERPLANE_FLEET_AUTH_TOKEN are set.
func NewClientFromEnv() (*Client, bool) {
	base := strings.TrimSpace(os.Getenv("SUPERPLANE_URL"))
	tok := strings.TrimSpace(os.Getenv("SUPERPLANE_FLEET_AUTH_TOKEN"))
	if base == "" || tok == "" {
		return nil, false
	}
	return &Client{
		BaseURL:   base,
		AuthToken: tok,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, true
}

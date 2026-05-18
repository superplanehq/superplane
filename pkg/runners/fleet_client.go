package runners

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

const fleetHTTPTimeout = 30 * time.Second

// FleetClient talks directly to a fleet-manager instance.
type FleetClient struct {
	httpClient core.HTTPContext
	fleet      *RunnerFleet
}

func NewFleetClient(httpClient core.HTTPContext, fleet *RunnerFleet) *FleetClient {
	return &FleetClient{httpClient: httpClient, fleet: fleet}
}

type FleetCreateTaskRequest struct {
	FleetID                 string                     `json:"fleet_id"`
	Commands                []string                   `json:"commands"`
	Environment             []FleetEnvironmentVariable `json:"environment,omitempty"`
	WebhookURL              string                     `json:"webhook_url"`
	ExecutionMode           string                     `json:"execution_mode,omitempty"`
	DockerImage             string                     `json:"docker_image,omitempty"`
	ExecutionTimeoutSeconds *int                       `json:"execution_timeout_seconds,omitempty"`
}

type FleetEnvironmentVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type FleetCreateTaskParams struct {
	Commands       []string
	WebhookURL     string
	Environment    []FleetEnvironmentVariable
	ExecutionMode  string
	DockerImage    string
	TimeoutSeconds int
}

type fleetCreateTaskResponse struct {
	ID string `json:"id"`
}

func (c *FleetClient) CreateTask(p FleetCreateTaskParams) (string, error) {
	req := FleetCreateTaskRequest{
		FleetID:       c.fleet.Name,
		Commands:      p.Commands,
		Environment:   p.Environment,
		WebhookURL:    p.WebhookURL,
		ExecutionMode: p.ExecutionMode,
		DockerImage:   p.DockerImage,
	}
	if p.TimeoutSeconds > 0 {
		t := p.TimeoutSeconds
		req.ExecutionTimeoutSeconds = &t
	}

	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpCtx, cancel := context.WithTimeout(context.Background(), fleetHTTPTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(httpCtx, http.MethodPost, c.fleet.FleetURL+"/v1/tasks", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.fleet.AuthToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.fleet.AuthToken)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("fleet request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("fleet rejected task: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var out fleetCreateTaskResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("unmarshal create task response: %w", err)
	}

	return out.ID, nil
}

// FleetTask is the task payload returned by fleet-manager (GET /v1/tasks/:id and webhook body).
type FleetTask struct {
	TaskID   string          `json:"task_id"`
	Status   string          `json:"status"`
	ExitCode *int            `json:"exit_code,omitempty"`
	Output   string          `json:"output,omitempty"`
	Error    string          `json:"error,omitempty"`
	Result   json.RawMessage `json:"result,omitempty"`

	TaskLog *FleetTaskLog `json:"task_log,omitempty"`

	CloudWatchLogGroup  string `json:"cloudwatch_log_group,omitempty"`
	CloudWatchLogStream string `json:"cloudwatch_log_stream,omitempty"`
}

func (t *FleetTask) EffectiveExitCode() int {
	if t == nil || t.ExitCode == nil {
		return 0
	}
	return *t.ExitCode
}

func (t *FleetTask) IsInTerminalState() bool {
	return t.Status == "succeeded" || t.Status == "failed"
}

// FleetTaskLog matches the fleet-manager JSON shape for CloudWatch-backed live logs.
type FleetTaskLog struct {
	Type       string `json:"type"`
	CloudWatch *struct {
		LogGroupName  string `json:"log_group_name"`
		LogStreamName string `json:"log_stream_name"`
		Region        string `json:"region,omitempty"`
	} `json:"cloudwatch,omitempty"`
}

func (c *FleetClient) FetchTaskStatus(taskID string) (*FleetTask, error) {
	httpCtx, cancel := context.WithTimeout(context.Background(), fleetHTTPTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(httpCtx, http.MethodGet, c.fleet.FleetURL+"/v1/tasks/"+taskID, nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	if c.fleet.AuthToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.fleet.AuthToken)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("fleet request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fleet request failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var out FleetTask
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("unmarshal task response: %w", err)
	}

	return &out, nil
}

func ParseWebhookTask(body []byte) (*FleetTask, error) {
	var out FleetTask
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("unmarshal webhook body: %w", err)
	}
	return &out, nil
}

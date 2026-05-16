package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	brokerHTTPTimeout = 30 * time.Second

	cancel409MaxAttempts  = 3
	cancel409RetryBackoff = 200 * time.Millisecond
)

type BrokerClient struct {
	httpClient core.HTTPContext
	baseURL    string
	fleetID    string
	authToken  string
}

func NewBrokerClient(httpClient core.HTTPContext) (*BrokerClient, error) {
	baseURL := os.Getenv("TASK_BROKER_BASE_URL")
	if baseURL == "" {
		return nil, fmt.Errorf("TASK_BROKER_BASE_URL is not set")
	}

	fleetID := os.Getenv("TASK_BROKER_FLEET_ID")
	if fleetID == "" {
		return nil, fmt.Errorf("TASK_BROKER_FLEET_ID is not set")
	}

	authToken := os.Getenv("TASK_BROKER_AUTH_TOKEN")
	if authToken == "" {
		return nil, fmt.Errorf("TASK_BROKER_AUTH_TOKEN is not set")
	}

	return &BrokerClient{
		httpClient: httpClient,
		baseURL:    strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		fleetID:    fleetID,
		authToken:  authToken,
	}, nil
}

// Create Task
//
// POST /v1/tasks
//
// Example request:
// {
//   "fleet_id": "aws-standard-1",
//   "commands": ["echo \"Hello, World!\""],
//   "environment": [{"name": "APP_ENV", "value": "production"}],
//   "webhook_url": "https://example.com/webhook",
//   "execution_mode": "host",
//   "docker_image": "debian:bookworm-slim",
//   "execution_timeout_seconds": 600
// }
//
// Example response:
// {
//   "id": "1234567890"
// }

type brokerCreateTaskRequest struct {
	FleetID string `json:"fleet_id"`

	Commands                []string                    `json:"commands"`
	Environment             []BrokerEnvironmentVariable `json:"environment,omitempty"`
	WebhookURL              string                      `json:"webhook_url"`
	ExecutionMode           string                      `json:"execution_mode,omitempty"`
	DockerImage             string                      `json:"docker_image,omitempty"`
	ExecutionTimeoutSeconds *int                        `json:"execution_timeout_seconds,omitempty"`
}

// BrokerEnvironmentVariable is forwarded to the task broker as JSON.
type BrokerEnvironmentVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// CreateTaskParams is forwarded to the task broker POST /v1/tasks.
type CreateTaskParams struct {
	Commands       []string
	WebhookURL     string
	Environment    []BrokerEnvironmentVariable
	ExecutionMode  string
	DockerImage    string
	TimeoutSeconds int // 0 = omit (broker / fleet default)
}

type brokerCreateTaskResponse struct {
	ID string `json:"id"`
}

func (b *BrokerClient) CreateTask(p CreateTaskParams) (string, error) {
	mode := strings.ToLower(strings.TrimSpace(p.ExecutionMode))
	if mode == "" {
		mode = ExecutionModeHost
	}

	req := brokerCreateTaskRequest{
		FleetID:       b.fleetID,
		Commands:      p.Commands,
		Environment:   p.Environment,
		WebhookURL:    p.WebhookURL,
		ExecutionMode: mode,
		DockerImage:   strings.TrimSpace(p.DockerImage),
	}
	if p.TimeoutSeconds > 0 {
		t := p.TimeoutSeconds
		req.ExecutionTimeoutSeconds = &t
	}

	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpCtx, cancel := context.WithTimeout(context.Background(), brokerHTTPTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(httpCtx, http.MethodPost, b.baseURL+"/v1/tasks", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+b.authToken)

	resp, err := b.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("broker request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf(
			"broker rejected task: status=%d body=%s",
			resp.StatusCode,
			strings.TrimSpace(string(body)),
		)
	}

	var out brokerCreateTaskResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("unmarshal create task response: %w", err)
	}

	return out.ID, nil
}

// Task is the broker task payload (GET /v1/tasks/:id and webhook body).
type Task struct {
	TaskID   string          `json:"task_id"`
	Status   string          `json:"status"`
	ExitCode *int            `json:"exit_code,omitempty"`
	Output   string          `json:"output,omitempty"`
	Error    string          `json:"error,omitempty"`
	Result   json.RawMessage `json:"result,omitempty"`

	TaskLog *TaskLogSink `json:"task_log,omitempty"`

	CloudWatchLogGroup  string `json:"cloudwatch_log_group,omitempty"`
	CloudWatchLogStream string `json:"cloudwatch_log_stream,omitempty"`
}

func (t *Task) effectiveExitCode() int {
	if t == nil || t.ExitCode == nil {
		return 0
	}
	return *t.ExitCode
}

func (t *Task) IsInTerminalState() bool {
	switch strings.ToLower(strings.TrimSpace(t.Status)) {
	case "succeeded", "failed", "canceled", "cancelled":
		return true
	default:
		return false
	}
}

type brokerErrorResponse struct {
	Error string `json:"error"`
}

// CancelTaskResponse is the JSON body from POST /v1/tasks/{id}/cancel on success (HTTP 200).
type CancelTaskResponse struct {
	ID     string `json:"id"`
	State  string `json:"state"`
	Status string `json:"status"`
}

func readBrokerErrorMessage(body []byte) string {
	var w brokerErrorResponse
	if err := json.Unmarshal(body, &w); err == nil && strings.TrimSpace(w.Error) != "" {
		return w.Error
	}
	return strings.TrimSpace(string(body))
}

// CancelTask requests cancellation for the broker-scoped task id (same id as POST /v1/tasks returns).
// HTTP 404 from the broker is treated as success (no-op). Transient HTTP 409 responses are retried.
func (b *BrokerClient) CancelTask(brokerTaskID string) (*CancelTaskResponse, error) {
	brokerTaskID = strings.TrimSpace(brokerTaskID)
	if brokerTaskID == "" {
		return nil, fmt.Errorf("broker task id is empty")
	}

	cancelPath := b.baseURL + "/v1/tasks/" + url.PathEscape(brokerTaskID) + "/cancel"

	var lastConflictErr error
	for attempt := 0; attempt < cancel409MaxAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(cancel409RetryBackoff)
		}

		httpCtx, cancel := context.WithTimeout(context.Background(), brokerHTTPTimeout)
		httpReq, err := http.NewRequestWithContext(httpCtx, http.MethodPost, cancelPath, http.NoBody)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("new request: %w", err)
		}
		httpReq.Header.Set("Authorization", "Bearer "+b.authToken)

		resp, err := b.httpClient.Do(httpReq)
		cancel()
		if err != nil {
			return nil, fmt.Errorf("broker request: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read response body: %w", err)
		}

		switch resp.StatusCode {
		case http.StatusOK:
			var out CancelTaskResponse
			if err := json.Unmarshal(body, &out); err != nil {
				return nil, fmt.Errorf("unmarshal cancel task response: %w", err)
			}
			return &out, nil
		case http.StatusNotFound:
			return nil, nil
		case http.StatusConflict:
			lastConflictErr = fmt.Errorf(
				"broker cancel conflict (task not yet assigned upstream): status=%d body=%s",
				resp.StatusCode,
				readBrokerErrorMessage(body),
			)
		default:
			return nil, fmt.Errorf(
				"broker rejected cancel: status=%d body=%s",
				resp.StatusCode,
				readBrokerErrorMessage(body),
			)
		}
	}

	return nil, fmt.Errorf("broker cancel: exceeded retries: %w", lastConflictErr)
}

func (b *BrokerClient) FetchTaskStatus(taskID string) (*Task, error) {
	httpCtx, cancel := context.WithTimeout(context.Background(), brokerHTTPTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(httpCtx, http.MethodGet, b.baseURL+"/v1/tasks/"+url.PathEscape(taskID), nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+b.authToken)

	resp, err := b.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("broker request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("broker rejected task: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var out Task
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("unmarshal task response: %w", err)
	}

	return &out, nil
}

func (b *BrokerClient) ProcessWebhook(body []byte) (*Task, error) {
	var out Task
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("unmarshal webhook response: %w", err)
	}

	return &out, nil
}

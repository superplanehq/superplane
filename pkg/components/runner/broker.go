package runner

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

	"github.com/superplanehq/superplane/pkg/tlsclient"
)

const (
	brokerHTTPTimeout = 30 * time.Second
)

type BrokerClient struct {
	httpClient *http.Client
	baseURL    string
	fleetID    string
	authToken  string
}

func NewBrokerClient() (*BrokerClient, error) {
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

	httpClient, err := tlsclient.NewHTTPClientFromEnv(brokerHTTPTimeout)
	if err != nil {
		return nil, fmt.Errorf("broker tls client: %w", err)
	}

	return &BrokerClient{
		httpClient: httpClient,
		baseURL:    baseURL,
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
//   "execution_mode": "host"
// }
//
// Example response:
// {
//   "id": "1234567890"
// }

type brokerCreateTaskRequest struct {
	FleetID string `json:"fleet_id"`

	Commands      []string                    `json:"commands"`
	Environment   []BrokerEnvironmentVariable `json:"environment,omitempty"`
	WebhookURL    string                      `json:"webhook_url"`
	ExecutionMode string                      `json:"execution_mode,omitempty"`
}

type BrokerEnvironmentVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type brokerCreateTaskResponse struct {
	ID string `json:"id"`
}

func (b *BrokerClient) CreateTask(commands []string, webhookURL string, environment []BrokerEnvironmentVariable) (string, error) {
	req := brokerCreateTaskRequest{
		FleetID:       b.fleetID,
		Commands:      commands,
		Environment:   environment,
		WebhookURL:    webhookURL,
		ExecutionMode: "host",
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
	return t.Status == "succeeded" || t.Status == "failed"
}

func (b *BrokerClient) FetchTaskStatus(taskID string) (*Task, error) {
	httpCtx, cancel := context.WithTimeout(context.Background(), brokerHTTPTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(httpCtx, http.MethodGet, b.baseURL+"/v1/tasks/"+taskID, nil)
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

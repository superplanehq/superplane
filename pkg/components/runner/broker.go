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

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/tlsclient"
)

const (
	brokerHTTPTimeout = 30 * time.Second
)

// httpDoer is a minimal interface satisfied by both *http.Client and core.HTTPContext.
// This allows tests to inject a mock HTTP client while production uses a TLS-aware client.
type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type BrokerClient struct {
	httpClient httpDoer
	baseURL    string
	fleetID    string
	authToken  string
}

// NewBrokerClient creates a BrokerClient. In production it builds a TLS-aware
// *http.Client from environment variables (TLS_ROOT_CA_FILE, TLS_CLIENT_CERT_FILE,
// TLS_CLIENT_KEY_FILE, TLS_INSECURE_SKIP_VERIFY). When no TLS env vars are set
// it falls back to the shared core.HTTPContext (which handles SSRF policy checks).
func NewBrokerClient(fallback core.HTTPContext) (*BrokerClient, error) {
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

	// Build a TLS-aware client when any TLS env var is set; otherwise use the
	// shared HTTP context (preserves SSRF policy checks and test mock injection).
	cfg, err := tlsclient.ConfigFromEnv()
	if err != nil {
		return nil, fmt.Errorf("broker tls config: %w", err)
	}

	var doer httpDoer
	if cfg.RootCAFile != "" || cfg.ClientCertFile != "" || cfg.InsecureSkipVerify {
		client, err := tlsclient.NewHTTPClient(cfg, brokerHTTPTimeout)
		if err != nil {
			return nil, fmt.Errorf("broker tls client: %w", err)
		}
		doer = client
	} else {
		doer = fallback
	}

	return &BrokerClient{
		httpClient: doer,
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

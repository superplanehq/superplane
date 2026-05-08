package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	brokerBaseURL   = "http://98.91.210.215:8081"
	brokerFleetID   = "aws-standard-1"
	brokerAuthToken = ""
)

type BrokerClient struct {
	httpClient  core.HTTPContext
	integration core.IntegrationContext
}

func NewBrokerClient(httpClient core.HTTPContext) *BrokerClient {
	return &BrokerClient{
		httpClient: httpClient,
	}
}

func (b *BrokerClient) BaseURL() string {
	return strings.TrimRight(strings.TrimSpace(brokerBaseURL), "/")
}

// Create Task
//
// POST /v1/tasks
//
// Example request:
// {
//   "fleet_id": "aws-standard-1",
//   "commands": ["echo \"Hello, World!\""],
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

	Commands      []string `json:"commands"`
	WebhookURL    string   `json:"webhook_url"`
	ExecutionMode string   `json:"execution_mode,omitempty"`
}

type brokerCreateTaskResponse struct {
	ID string `json:"id"`
}

func (b *BrokerClient) CreateTask(commands []string, webhookURL string) (string, error) {
	req := brokerCreateTaskRequest{
		FleetID:       brokerFleetID,
		Commands:      commands,
		WebhookURL:    webhookURL,
		ExecutionMode: "host",
	}

	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpCtx, cancel := context.WithTimeout(context.Background(), brokerHTTPTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(httpCtx, http.MethodPost, b.BaseURL()+"/v1/tasks", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

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
			strings.TrimSpace(string(b)),
		)
	}

	var out brokerCreateTaskResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("unmarshal create task response: %w", err)
	}

	return out.ID, nil
}

type task struct {
	TaskID   string `json:"task_id"`
	Status   string `json:"status"`
	ExitCode int    `json:"exit_code"`
}

func (t *task) IsInTerminalState() bool {
	return t.Status == "succeeded" || t.Status == "failed"
}

func (b *BrokerClient) FetchTaskStatus(taskID string) (*task, error) {
	httpCtx, cancel := context.WithTimeout(context.Background(), brokerHTTPTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(httpCtx, http.MethodGet, b.BaseURL()+"/v1/tasks/"+taskID, nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

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

	var out task
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("unmarshal task response: %w", err)
	}

	return &out, nil
}

func (b *BrokerClient) ProcessWebhook(body []byte) (*task, error) {
	var out task
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("unmarshal webhook response: %w", err)
	}

	return &out, nil
}

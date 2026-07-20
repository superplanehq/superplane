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

	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
)

// ActiveTask is a non-terminal task from GET /v1/tasks on the task broker.
type ActiveTask struct {
	ID                      string     `json:"id"`
	Status                  string     `json:"status"`
	FleetID                 string     `json:"fleet_id"`
	CreatedAt               time.Time  `json:"created_at"`
	ClaimedAt               *time.Time `json:"claimed_at,omitempty"`
	LeaseUntil              *time.Time `json:"lease_until,omitempty"`
	RunnerID                string     `json:"runner_id,omitempty"`
	ExecutionMode           string     `json:"execution_mode,omitempty"`
	DockerImage             string     `json:"docker_image,omitempty"`
	CancelRequested         bool       `json:"cancel_requested,omitempty"`
	ExecutionTimeoutSeconds *int       `json:"execution_timeout_seconds,omitempty"`
}

const (
	brokerHTTPTimeout = 30 * time.Second

	// Task-broker may return 409 until fleet_task_id is linked (cancel right after create).
	cancel409MaxAttempts  = 3
	cancel409RetryBackoff = 200 * time.Millisecond
)

type BrokerClient struct {
	httpClient core.HTTPContext
	baseURL    string
	authToken  string
}

func NewBrokerClient(httpClient core.HTTPContext) (*BrokerClient, error) {
	baseURL := os.Getenv("TASK_BROKER_BASE_URL")
	if baseURL == "" {
		return nil, fmt.Errorf("TASK_BROKER_BASE_URL is not set")
	}

	authToken := os.Getenv("TASK_BROKER_AUTH_TOKEN")
	if authToken == "" {
		return nil, fmt.Errorf("TASK_BROKER_AUTH_TOKEN is not set")
	}

	return &BrokerClient{
		httpClient: httpClient,
		baseURL:    baseURL,
		authToken:  authToken,
	}, nil
}

// Create Task
//
// POST /v1/tasks
//
// Example request:
// {
//   "fleet_id": "e1-large-amd64",
//   "commands": ["echo \"Hello, World!\""],
//   "environment": [{"name": "APP_ENV", "value": "production"}],
//   "webhook_url": "https://example.com/webhook",
//   "webhook_payload_size_limit": 524288,
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

	RunMode                 string                      `json:"run_mode,omitempty"`
	Script                  string                      `json:"script,omitempty"`
	MessageChain            json.RawMessage             `json:"message_chain,omitempty"`
	Commands                []BrokerCommand             `json:"commands,omitempty"`
	SetupCommands           []string                    `json:"setup_commands,omitempty"`
	Environment             []BrokerEnvironmentVariable `json:"environment,omitempty"`
	WebhookURL              string                      `json:"webhook_url"`
	WebhookPayloadSizeLimit int                         `json:"webhook_payload_size_limit"`
	ExecutionMode           string                      `json:"execution_mode,omitempty"`
	DockerImage             string                      `json:"docker_image,omitempty"`
	ExecutionTimeoutSeconds *int                        `json:"execution_timeout_seconds,omitempty"`
}

// BrokerCommand is one command_list entry. JSON is a plain string when Name is
// empty, or {"name","command"} when Name is set (task-broker accepts both).
type BrokerCommand struct {
	Name    string `json:"name,omitempty"`
	Command string `json:"command"`
}

func (c BrokerCommand) MarshalJSON() ([]byte, error) {
	name := strings.TrimSpace(c.Name)
	command := strings.TrimSpace(c.Command)
	if name == "" {
		return json.Marshal(command)
	}
	return json.Marshal(struct {
		Name    string `json:"name"`
		Command string `json:"command"`
	}{Name: name, Command: command})
}

func (c *BrokerCommand) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || string(data) == "null" {
		*c = BrokerCommand{}
		return nil
	}
	if data[0] == '"' {
		var command string
		if err := json.Unmarshal(data, &command); err != nil {
			return err
		}
		*c = BrokerCommand{Command: strings.TrimSpace(command)}
		return nil
	}
	var raw struct {
		Name    string `json:"name"`
		Command string `json:"command"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*c = BrokerCommand{Name: strings.TrimSpace(raw.Name), Command: strings.TrimSpace(raw.Command)}
	return nil
}

// BrokerCommandsFromLines adapts plain shell lines into unnamed broker commands.
func BrokerCommandsFromLines(lines []string) []BrokerCommand {
	out := make([]BrokerCommand, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, BrokerCommand{Command: line})
	}
	return out
}

// BrokerEnvironmentVariable is forwarded to the task broker as JSON.
type BrokerEnvironmentVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

const (
	RunModeJavaScript = "javascript_script"
	RunModePython     = "python_script"
	RunModeBash       = "bash_script"
)

// CreateTaskParams is forwarded to the task broker POST /v1/tasks.
type CreateTaskParams struct {
	MachineType             string
	RunMode                 string
	Script                  string
	MessageChain            json.RawMessage
	Commands                []BrokerCommand
	SetupCommands           []string
	WebhookURL              string
	WebhookPayloadSizeLimit int
	Environment             []BrokerEnvironmentVariable
	ExecutionMode           string
	DockerImage             string
	TimeoutSeconds          int // 0 = DefaultExecutionTimeoutSeconds
}

type brokerCreateTaskResponse struct {
	ID string `json:"id"`
}

func (b *BrokerClient) CreateTask(p CreateTaskParams) (string, error) {
	mode := strings.ToLower(strings.TrimSpace(p.ExecutionMode))
	if mode == "" {
		mode = ExecutionModeHost
	}

	webhookPayloadSizeLimit := p.WebhookPayloadSizeLimit
	if webhookPayloadSizeLimit <= 0 {
		webhookPayloadSizeLimit = config.MaxWebhookPayloadSize
	}

	fleetID, err := requireMachineType(p.MachineType)
	if err != nil {
		return "", err
	}

	req := brokerCreateTaskRequest{
		FleetID:                 fleetID,
		RunMode:                 strings.TrimSpace(p.RunMode),
		Script:                  strings.TrimSpace(p.Script),
		MessageChain:            p.MessageChain,
		Commands:                p.Commands,
		SetupCommands:           p.SetupCommands,
		Environment:             p.Environment,
		WebhookURL:              p.WebhookURL,
		WebhookPayloadSizeLimit: webhookPayloadSizeLimit,
		ExecutionMode:           mode,
		DockerImage:             strings.TrimSpace(p.DockerImage),
	}
	timeout := p.TimeoutSeconds
	if timeout <= 0 {
		timeout = DefaultExecutionTimeoutSeconds
	}
	req.ExecutionTimeoutSeconds = &timeout

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
	return t.Status == "succeeded" || t.Status == "failed" || t.Status == "canceled"
}

func (b *BrokerClient) CancelTask(brokerTaskID string) error {
	brokerTaskID = strings.TrimSpace(brokerTaskID)
	if brokerTaskID == "" {
		return fmt.Errorf("broker task id is empty")
	}

	cancelPath := b.baseURL + "/v1/tasks/" + url.PathEscape(brokerTaskID) + "/cancel"

	var lastErr error
	for attempt := range cancel409MaxAttempts {
		if attempt > 0 {
			time.Sleep(cancel409RetryBackoff)
		}

		httpCtx, cancel := context.WithTimeout(context.Background(), brokerHTTPTimeout)
		httpReq, err := http.NewRequestWithContext(httpCtx, http.MethodPost, cancelPath, http.NoBody)
		if err != nil {
			cancel()
			return fmt.Errorf("new request: %w", err)
		}
		httpReq.Header.Set("Authorization", "Bearer "+b.authToken)

		resp, err := b.httpClient.Do(httpReq)
		if err != nil {
			cancel()
			return fmt.Errorf("broker request: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		cancel()
		if err != nil {
			return fmt.Errorf("read response body: %w", err)
		}

		switch resp.StatusCode {
		case http.StatusOK, http.StatusNotFound:
			return nil
		case http.StatusConflict:
			lastErr = fmt.Errorf(
				"broker rejected cancel: status=%d body=%s",
				resp.StatusCode,
				strings.TrimSpace(string(body)),
			)
		default:
			return fmt.Errorf(
				"broker rejected cancel: status=%d body=%s",
				resp.StatusCode,
				strings.TrimSpace(string(body)),
			)
		}
	}

	return fmt.Errorf("broker cancel: exceeded retries: %w", lastErr)
}

func (b *BrokerClient) ListActiveTasks() ([]ActiveTask, error) {
	httpCtx, cancel := context.WithTimeout(context.Background(), brokerHTTPTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(httpCtx, http.MethodGet, b.baseURL+"/v1/tasks", nil)
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
		return nil, fmt.Errorf("broker rejected list tasks: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var out struct {
		Tasks []ActiveTask `json:"tasks"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("unmarshal list tasks response: %w", err)
	}

	if out.Tasks == nil {
		return []ActiveTask{}, nil
	}

	return out.Tasks, nil
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

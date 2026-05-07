package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	ComponentName = "runner"

	channelSuccess = "success"
	channelFailure = "failure"

	brokerPollInterval = time.Minute
	brokerHTTPTimeout  = 30 * time.Second

	// defaultBrokerBaseURL is the task-broker HTTP base URL (no trailing slash).
	// Keep in sync with BROKER_PUBLIC_URL in the runner repo: ../runner/scripts/deploy/task-broker.env
	defaultBrokerBaseURL = "http://98.91.210.215:8081"

	// defaultFleetID is the task-broker fleet id for POST /v1/tasks.
	// Keep in sync with TASK_BROKER_FLEET_ID in the runner repo: ../runner/scripts/deploy/task-broker.env
	defaultFleetID = "aws-standard-1"

	// defaultBrokerAuthToken is an optional bearer token for task-broker /v1 when TASK_BROKER_AUTH_TOKEN is unset.
	// Prefer env TASK_BROKER_AUTH_TOKEN (same value as AUTH_TOKEN on the task-broker; see runner/scripts/deploy/task-broker.env).
	// When neither is set, no Authorization header is sent.
	defaultBrokerAuthToken = ""
)

func init() {
	registry.RegisterAction(ComponentName, &RunCommands{})
}

type RunCommands struct{}

type Spec struct {
	Commands      string  `json:"commands" mapstructure:"commands"`
	ExecutionMode string  `json:"executionMode,omitempty" mapstructure:"executionMode"`
	DockerImage   *string `json:"dockerImage,omitempty" mapstructure:"dockerImage"`
}

func (c *RunCommands) Name() string  { return ComponentName }
func (c *RunCommands) Label() string { return "Runner" }
func (c *RunCommands) Icon() string  { return "terminal" }
func (c *RunCommands) Color() string { return "blue" }
func (c *RunCommands) ExampleOutput() map[string]any {
	return map[string]any{"task_id": "…", "status": "succeeded", "exit_code": 0, "output": "…"}
}

func (c *RunCommands) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: channelSuccess, Label: "Success"},
		{Name: channelFailure, Label: "Failure"},
	}
}

func (c *RunCommands) Description() string {
	return "Runs commands by creating a task via task-broker."
}

func (c *RunCommands) Documentation() string {
	return `Creates a task in **task-broker** and resolves when either the completion **webhook** runs or a **polling fallback** observes a terminal status (polling uses **task-broker** ` + "`GET /v1/tasks/{broker_task_id}`" + ` every minute until ` + "`succeeded`" + ` or ` + "`failed`" + `).

## Required configuration
- **Commands**: One command per line

## Output
Emits on **Success** (exit code 0) or **Failure** (non-zero) with ` + "`status`" + `, ` + "`exit_code`" + `, and ` + "`output`" + ` from the runner.`
}

func (c *RunCommands) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "commands",
			Label:       "Commands",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Placeholder: "echo hello\nuname -a",
			Description: "One or more commands, one per line",
		},
		{
			Name:     "executionMode",
			Label:    "Execution mode",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "host",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Host", Value: "host"},
						{Label: "Docker", Value: "docker"},
					},
				},
			},
		},
		{
			Name:                 "dockerImage",
			Label:                "Docker image",
			Type:                 configuration.FieldTypeString,
			Required:             false,
			Description:          "Required when execution mode is docker",
			VisibilityConditions: []configuration.VisibilityCondition{{Field: "executionMode", Values: []string{"docker"}}},
			RequiredConditions:   []configuration.RequiredCondition{{Field: "executionMode", Values: []string{"docker"}}},
		},
	}
}

func (c *RunCommands) Setup(ctx core.SetupContext) error {
	spec := Spec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("decode configuration: %w", err)
	}
	if strings.TrimSpace(spec.Commands) == "" {
		return errors.New("commands is required")
	}
	if strings.TrimSpace(spec.ExecutionMode) == "" {
		spec.ExecutionMode = "host"
	}
	if strings.TrimSpace(spec.ExecutionMode) == "docker" && (spec.DockerImage == nil || strings.TrimSpace(*spec.DockerImage) == "") {
		return errors.New("dockerImage is required when executionMode is docker")
	}

	_, err := ctx.Webhook.Setup()
	return err
}

func (c *RunCommands) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

type brokerCreateTaskRequest struct {
	FleetID string `json:"fleet_id"`

	Command       []string `json:"command,omitempty"`
	Commands      []string `json:"commands,omitempty"`
	WebhookURL    string   `json:"webhook_url"`
	ExecutionMode string   `json:"execution_mode,omitempty"`
	DockerImage   string   `json:"docker_image,omitempty"`
}

type brokerCreateTaskResponse struct {
	ID string `json:"id"`
}

type brokerCompletionPayload struct {
	TaskID      string `json:"task_id"`
	FleetTaskID string `json:"fleet_task_id,omitempty"`
	Status      string `json:"status"`
	ExitCode    int    `json:"exit_code"`
	Output      string `json:"output"`
	Error       string `json:"error,omitempty"`
}

// brokerTaskPollJSON matches task-broker GET /v1/tasks/{id}.
type brokerTaskPollJSON struct {
	TaskID      string `json:"task_id"`
	FleetTaskID string `json:"fleet_task_id,omitempty"`
	Status      string `json:"status"`
	ExitCode    *int   `json:"exit_code,omitempty"`
	Output      string `json:"output"`
	Error       string `json:"error,omitempty"`
}

func emitRunnerFinished(state core.ExecutionStateContext, payload brokerCompletionPayload) error {
	if state.IsFinished() {
		return nil
	}
	channel := channelFailure
	if strings.ToLower(strings.TrimSpace(payload.Status)) == "succeeded" && payload.ExitCode == 0 {
		channel = channelSuccess
	}
	out := map[string]any{
		"task_id":       payload.TaskID,
		"fleet_task_id": payload.FleetTaskID,
		"status":        payload.Status,
		"exit_code":     payload.ExitCode,
		"output":        payload.Output,
		"error":         payload.Error,
	}
	return state.Emit(channel, "runner.finished", []any{out})
}

func isBrokerTerminalStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "succeeded", "failed":
		return true
	default:
		return false
	}
}

func (c *RunCommands) Execute(ctx core.ExecutionContext) error {
	spec := Spec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("decode configuration: %w", err)
	}

	webhookURL, err := ctx.Webhook.Setup()
	if err != nil {
		return fmt.Errorf("webhook setup: %w", err)
	}

	brokerBase := strings.TrimRight(strings.TrimSpace(defaultBrokerBaseURL), "/")

	cmds := normalizeLines(spec.Commands)
	if len(cmds) == 0 {
		return errors.New("commands is required")
	}

	exMode := strings.ToLower(strings.TrimSpace(spec.ExecutionMode))
	if exMode == "" {
		exMode = "host"
	}
	var dockerImage string
	if exMode == "docker" {
		if spec.DockerImage == nil || strings.TrimSpace(*spec.DockerImage) == "" {
			return errors.New("dockerImage is required when executionMode is docker")
		}
		dockerImage = strings.TrimSpace(*spec.DockerImage)
	}

	reqBody := brokerCreateTaskRequest{
		FleetID:       strings.TrimSpace(defaultFleetID),
		Commands:      cmds,
		WebhookURL:    webhookURL,
		ExecutionMode: exMode,
		DockerImage:   dockerImage,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpCtx, cancel := context.WithTimeout(context.Background(), brokerHTTPTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(httpCtx, http.MethodPost, brokerBase+"/v1/tasks", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	setBrokerAuthHeader(req)

	resp, err := ctx.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("broker request: %w", err)
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("broker rejected task: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	out := brokerCreateTaskResponse{}
	if err := json.Unmarshal(b, &out); err != nil {
		return fmt.Errorf("parse broker response: %w", err)
	}
	if strings.TrimSpace(out.ID) == "" {
		return fmt.Errorf("broker response missing id")
	}

	if err := ctx.ExecutionState.SetKV("broker_task_id", out.ID); err != nil {
		return err
	}

	return c.pollBrokerTaskUntilTerminal(ctx, brokerBase, out.ID)
}

func (c *RunCommands) pollBrokerTaskUntilTerminal(ctx core.ExecutionContext, brokerBase, brokerTaskID string) error {
	for attempt := 0; ; attempt++ {
		if ctx.ExecutionState.IsFinished() {
			return nil
		}
		if attempt > 0 {
			time.Sleep(brokerPollInterval)
			if ctx.ExecutionState.IsFinished() {
				return nil
			}
		}

		poll, err := c.fetchBrokerTaskStatus(ctx, brokerBase, brokerTaskID)
		if err != nil {
			if ctx.Logger != nil {
				ctx.Logger.WithError(err).Warn("runner: broker poll failed, will retry")
			}
			continue
		}
		if !isBrokerTerminalStatus(poll.Status) {
			continue
		}
		exit := 0
		if poll.ExitCode != nil {
			exit = *poll.ExitCode
		}
		payload := brokerCompletionPayload{
			TaskID:      brokerTaskID,
			FleetTaskID: poll.FleetTaskID,
			Status:      poll.Status,
			ExitCode:    exit,
			Output:      poll.Output,
			Error:       poll.Error,
		}
		if ctx.ExecutionState.IsFinished() {
			return nil
		}
		if err := emitRunnerFinished(ctx.ExecutionState, payload); err != nil && ctx.Logger != nil {
			ctx.Logger.WithError(err).Warn("runner: emit after poll failed")
		}
		return nil
	}
}

func (c *RunCommands) fetchBrokerTaskStatus(ctx core.ExecutionContext, brokerBase, brokerTaskID string) (*brokerTaskPollJSON, error) {
	httpCtx, cancel := context.WithTimeout(context.Background(), brokerHTTPTimeout)
	defer cancel()

	url := brokerBase + "/v1/tasks/" + brokerTaskID
	req, err := http.NewRequestWithContext(httpCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("new poll request: %w", err)
	}
	setBrokerAuthHeader(req)

	resp, err := ctx.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("broker poll: %w", err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read poll response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("broker poll: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var out brokerTaskPollJSON
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("parse poll json: %w", err)
	}
	return &out, nil
}

func (c *RunCommands) Hooks() []core.Hook { return nil }
func (c *RunCommands) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (c *RunCommands) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	payload := brokerCompletionPayload{}
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("invalid json: %w", err)
	}
	if strings.TrimSpace(payload.TaskID) == "" {
		return http.StatusBadRequest, nil, fmt.Errorf("task_id is required")
	}

	execCtx, err := ctx.FindExecutionByKV("broker_task_id", payload.TaskID)
	if err != nil {
		return http.StatusOK, nil, nil
	}
	if execCtx == nil {
		return http.StatusOK, nil, nil
	}
	if execCtx.ExecutionState.IsFinished() {
		return http.StatusOK, nil, nil
	}

	if err := emitRunnerFinished(execCtx.ExecutionState, payload); err != nil {
		return http.StatusOK, nil, nil
	}
	return http.StatusOK, nil, nil
}

func (c *RunCommands) Cancel(ctx core.ExecutionContext) error { return nil }
func (c *RunCommands) Cleanup(ctx core.SetupContext) error    { return nil }

func effectiveBrokerAuthToken() string {
	if t := strings.TrimSpace(os.Getenv("TASK_BROKER_AUTH_TOKEN")); t != "" {
		return t
	}
	return strings.TrimSpace(defaultBrokerAuthToken)
}

func setBrokerAuthHeader(req *http.Request) {
	tok := effectiveBrokerAuthToken()
	if tok == "" {
		return
	}
	req.Header.Set("Authorization", "Bearer "+tok)
}

func normalizeLines(s string) []string {
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		out = append(out, l)
	}
	return out
}

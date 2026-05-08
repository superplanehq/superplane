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

	PassedOutputChannel = "passed"
	FailedOutputChannel = "failed"

	RunnerFinishedEventType = "runner.finished"

	brokerPollInterval   = 30 * time.Second
	runnerFirstPollDelay = 2 * time.Second
	brokerHTTPTimeout    = 30 * time.Second

	runnerBrokerPollHook = "brokerPoll"

	paramKeyBrokerTaskID = "broker_task_id"
	paramKeyBrokerBase   = "broker_base"

	metaKeyRunnerBrokerTaskID = "runner_broker_task_id"
	metaKeyRunnerBrokerBase   = "runner_broker_base"

	defaultBrokerBaseURL   = "http://98.91.210.215:8081"
	defaultFleetID         = "aws-standard-1"
	defaultBrokerAuthToken = ""
)

func init() {
	registry.RegisterAction(ComponentName, &RunCommands{})
}

type RunCommands struct{}

type Spec struct {
	Commands string `json:"commands" mapstructure:"commands"`
}

func (c *RunCommands) Name() string  { return ComponentName }
func (c *RunCommands) Label() string { return "Runner" }
func (c *RunCommands) Icon() string  { return "terminal" }
func (c *RunCommands) Color() string { return "blue" }
func (c *RunCommands) ExampleOutput() map[string]any {
	return map[string]any{"status": "succeeded", "exit_code": 0}
}

func (c *RunCommands) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: PassedOutputChannel, Label: "Passed"},
		{Name: FailedOutputChannel, Label: "Failed"},
	}
}

func (c *RunCommands) Description() string {
	return "Runs bash commands on a dedicated machine"
}

func (c *RunCommands) Documentation() string {
	return `Runs bash commands on a dedicated machine.

## Configuration
- **Commands**: One or more shell commands, one per line.

## Output channels
- **Passed**: The commands finished with exit code **0**.
- **Failed**: The commands finished with non-zero exit code.
`
}

func (c *RunCommands) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "commands",
			Label:       "Commands",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Placeholder: "echo \"Hello, World!\"",
			Description: "One or more shell commands, one per line",
		},
	}
}

func (c *RunCommands) Setup(ctx core.SetupContext) error {
	spec := Spec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("decode configuration: %w", err)
	}

	if err := validateCommands(spec.Commands); err != nil {
		return err
	}

	_, err := ctx.Webhook.Setup()
	return err
}

func validateCommands(commands string) error {
	lines := normalizeLines(commands)
	if len(lines) == 0 {
		return errors.New("at least one command is required")
	}
	return nil
}

func (c *RunCommands) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

type brokerCreateTaskRequest struct {
	FleetID string `json:"fleet_id"`

	Commands      []string `json:"commands"`
	WebhookURL    string   `json:"webhook_url"`
	ExecutionMode string   `json:"execution_mode,omitempty"`
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

	channel := FailedOutputChannel
	if strings.ToLower(strings.TrimSpace(payload.Status)) == "succeeded" && payload.ExitCode == 0 {
		channel = PassedOutputChannel
	}

	out := map[string]any{"status": payload.Status, "exit_code": payload.ExitCode}
	return state.Emit(channel, RunnerFinishedEventType, []any{out})
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

	reqBody := brokerCreateTaskRequest{
		FleetID:       strings.TrimSpace(defaultFleetID),
		Commands:      cmds,
		WebhookURL:    webhookURL,
		ExecutionMode: "host",
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

	if err := setRunnerPollMetadata(ctx.Metadata, brokerBase, out.ID); err != nil {
		return err
	}

	return ctx.Requests.ScheduleActionCall(runnerBrokerPollHook, brokerPollHookParams(brokerBase, out.ID), runnerFirstPollDelay)
}

func brokerPollHookParams(brokerBase, taskID string) map[string]any {
	return map[string]any{
		paramKeyBrokerTaskID: taskID,
		paramKeyBrokerBase:   brokerBase,
	}
}

func coerceString(v any) string {
	if v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return strings.TrimSpace(s)
	case json.Number:
		return strings.TrimSpace(s.String())
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func normalizedBrokerBase(raw string) string {
	base := strings.TrimRight(strings.TrimSpace(raw), "/")
	if base == "" {
		base = strings.TrimRight(strings.TrimSpace(defaultBrokerBaseURL), "/")
	}
	return base
}

func setRunnerPollMetadata(metadata core.MetadataWriter, brokerBase, taskID string) error {
	md := map[string]any{}
	if existing := metadata.Get(); existing != nil {
		if typed, ok := existing.(map[string]any); ok {
			for k, v := range typed {
				md[k] = v
			}
		}
	}
	md[metaKeyRunnerBrokerTaskID] = taskID
	md[metaKeyRunnerBrokerBase] = brokerBase
	return metadata.Set(md)
}

func runnerBrokerPollArgsFromMetadata(existing any) (brokerBase string, brokerTaskID string, err error) {
	if existing == nil {
		return "", "", fmt.Errorf("runner poll: execution metadata missing")
	}
	m, ok := existing.(map[string]any)
	if !ok {
		return "", "", fmt.Errorf("runner poll: execution metadata has unexpected shape")
	}
	tid := coerceString(m[metaKeyRunnerBrokerTaskID])
	if tid == "" {
		return "", "", fmt.Errorf("runner poll: %s missing in metadata", metaKeyRunnerBrokerTaskID)
	}
	return normalizedBrokerBase(coerceString(m[metaKeyRunnerBrokerBase])), tid, nil
}

func resolveBrokerPollArgs(ctx core.ActionHookContext) (brokerBase string, brokerTaskID string, err error) {
	if ctx.Parameters != nil {
		tid := coerceString(ctx.Parameters[paramKeyBrokerTaskID])
		if tid != "" {
			return normalizedBrokerBase(coerceString(ctx.Parameters[paramKeyBrokerBase])), tid, nil
		}
	}
	return runnerBrokerPollArgsFromMetadata(ctx.Metadata.Get())
}

func (c *RunCommands) handleBrokerPoll(ctx core.ActionHookContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	brokerBase, brokerTaskID, argErr := resolveBrokerPollArgs(ctx)
	if argErr != nil {
		return argErr
	}
	next := brokerPollHookParams(brokerBase, brokerTaskID)

	poll, err := c.fetchBrokerTaskStatus(ctx.HTTP, brokerBase, brokerTaskID)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.WithError(err).Warn("runner: broker poll failed, will retry")
		}
		return ctx.Requests.ScheduleActionCall(runnerBrokerPollHook, next, brokerPollInterval)
	}
	if !isBrokerTerminalStatus(poll.Status) {
		return ctx.Requests.ScheduleActionCall(runnerBrokerPollHook, next, brokerPollInterval)
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

func (c *RunCommands) fetchBrokerTaskStatus(httpCtx core.HTTPContext, brokerBase, brokerTaskID string) (*brokerTaskPollJSON, error) {
	reqTimeout, cancel := context.WithTimeout(context.Background(), brokerHTTPTimeout)
	defer cancel()

	url := brokerBase + "/v1/tasks/" + brokerTaskID
	req, err := http.NewRequestWithContext(reqTimeout, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("new poll request: %w", err)
	}
	setBrokerAuthHeader(req)

	resp, err := httpCtx.Do(req)
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

func (c *RunCommands) Hooks() []core.Hook {
	return []core.Hook{{Name: runnerBrokerPollHook, Type: core.HookTypeInternal}}
}

func (c *RunCommands) HandleHook(ctx core.ActionHookContext) error {
	switch ctx.Name {
	case runnerBrokerPollHook:
		return c.handleBrokerPoll(ctx)
	default:
		return fmt.Errorf("unknown hook: %s", ctx.Name)
	}
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

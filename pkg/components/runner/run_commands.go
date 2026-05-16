package runner

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	PassedOutputChannel     = "passed"
	FailedOutputChannel     = "failed"
	RunnerFinishedEventType = "runner.finished"

	pollInterval   = 2 * time.Minute
	hookActionPoll = "poll"
)

func init() {
	registry.RegisterAction("runner", &Runner{})
}

type Runner struct{}

type EnvironmentVariable struct {
	Name        string                     `json:"name" mapstructure:"name"`
	ValueSource string                     `json:"valueSource" mapstructure:"valueSource"`
	Value       *string                    `json:"value,omitempty" mapstructure:"value"`
	Secret      configuration.SecretKeyRef `json:"secret,omitempty" mapstructure:"secret"`
}

type Spec struct {
	Commands    string                `json:"commands" mapstructure:"commands"`
	Environment []EnvironmentVariable `json:"environment,omitempty" mapstructure:"environment"`
}

func (c *Runner) Name() string  { return "runner" }
func (c *Runner) Label() string { return "Runner" }
func (c *Runner) Icon() string  { return "terminal" }
func (c *Runner) Color() string { return "blue" }

func (c *Runner) ExampleOutput() map[string]any {
	return map[string]any{
		"type":      RunnerFinishedEventType,
		"timestamp": "2026-01-16T17:56:16.680755501Z",
		"data": []any{map[string]any{
			"status":    "succeeded",
			"exit_code": 0,
			"result":    map[string]any{"example": "value"},
		}},
	}
}

func (c *Runner) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: PassedOutputChannel, Label: "Passed"},
		{Name: FailedOutputChannel, Label: "Failed"},
	}
}

func (c *Runner) Description() string {
	return "Runs bash commands on a dedicated machine"
}

func (c *Runner) Documentation() string {
	return `Runs bash commands on a dedicated machine.

## Configuration
- **Commands**: One or more shell commands, one per line.
- **Environment variables**: Optional key/value pairs available during command execution. Values can be literal strings (with expression support) or organization secret keys.

## Output channels
- **Passed**: The commands finished with exit code **0**.
- **Failed**: The commands finished with non-zero exit code.

## Structured result
When the remote task writes valid JSON to **SUPERPLANE_RESULT_FILE** before exit, that object is returned as **result** on the finished event payload (alongside **status** and **exit_code**).
`
}

func (c *Runner) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "commands",
			Label:       "Commands",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Placeholder: "echo \"Hello, World!\"",
			Description: "One or more shell commands, one per line",
		},
		{
			Name:        "environment",
			Label:       "Environment variables",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Optional key/value pairs available to the commands",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Variable",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "name",
								Label:       "Name",
								Type:        configuration.FieldTypeString,
								Description: "Environment variable name (letters, numbers, underscore)",
								Placeholder: "e.g. COMMIT_AUTHOR",
								Required:    true,
							},
							{
								Name:        "valueSource",
								Label:       "Value source",
								Type:        configuration.FieldTypeSelect,
								Description: "Where this variable value comes from",
								Required:    true,
								Default:     EnvironmentValueSourceLiteral,
								TypeOptions: &configuration.TypeOptions{
									Select: &configuration.SelectTypeOptions{
										Options: []configuration.FieldOption{
											{Label: "Literal value", Value: EnvironmentValueSourceLiteral},
											{Label: "Secret key", Value: EnvironmentValueSourceSecret},
										},
									},
								},
							},
							{
								Name:                 "value",
								Label:                "Value",
								Type:                 configuration.FieldTypeString,
								Description:          "Literal value. Supports expressions such as {{ previous().data.author.email }}",
								Placeholder:          "e.g. production",
								Required:             false,
								VisibilityConditions: []configuration.VisibilityCondition{{Field: "valueSource", Values: []string{EnvironmentValueSourceLiteral}}},
								RequiredConditions:   []configuration.RequiredCondition{{Field: "valueSource", Values: []string{EnvironmentValueSourceLiteral}}},
							},
							{
								Name:                 "secret",
								Label:                "Secret key",
								Type:                 configuration.FieldTypeSecretKey,
								Description:          "Stored credential key to use as the variable value",
								Required:             false,
								VisibilityConditions: []configuration.VisibilityCondition{{Field: "valueSource", Values: []string{EnvironmentValueSourceSecret}}},
								RequiredConditions:   []configuration.RequiredCondition{{Field: "valueSource", Values: []string{EnvironmentValueSourceSecret}}},
							},
						},
					},
				},
			},
		},
	}
}

func (c *Runner) Setup(ctx core.SetupContext) error {
	spec := Spec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("decode configuration: %w", err)
	}

	if err := validateCommands(spec.Commands); err != nil {
		return err
	}

	if err := validateEnvironment(spec.Environment); err != nil {
		return err
	}

	_, err := ctx.Webhook.Setup()
	return err
}

func (c *Runner) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *Runner) Execute(ctx core.ExecutionContext) error {
	spec := Spec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("decode configuration: %w", err)
	}

	if err := validateCommands(spec.Commands); err != nil {
		return err
	}

	if err := validateEnvironment(spec.Environment); err != nil {
		return err
	}

	environment, err := resolveEnvironment(ctx.Secrets, spec.Environment)
	if err != nil {
		return err
	}

	webhookURL, err := ctx.Webhook.Setup()
	if err != nil {
		return fmt.Errorf("webhook setup: %w", err)
	}

	cmds := normalizeCommands(spec.Commands)
	broker, err := NewBrokerClient(ctx.HTTP)
	if err != nil {
		return fmt.Errorf("new broker client: %w", err)
	}

	taskID, err := broker.CreateTask(cmds, webhookURL, environment)
	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}

	params := map[string]any{"task_id": taskID}

	err = ctx.ExecutionState.SetKV("task_id", taskID)
	if err != nil {
		return fmt.Errorf("set task id in kv: %w", err)
	}

	if err := mergeRunnerBrokerTaskID(ctx.Metadata, taskID); err != nil {
		return fmt.Errorf("runner execution metadata: %w", err)
	}

	return ctx.Requests.ScheduleActionCall(hookActionPoll, params, pollInterval)
}

func (c *Runner) Hooks() []core.Hook {
	return []core.Hook{{Name: hookActionPoll, Type: core.HookTypeInternal}}
}

func (c *Runner) HandleHook(ctx core.ActionHookContext) error {
	switch ctx.Name {
	case hookActionPoll:
		return c.handlePoll(ctx)
	default:
		return fmt.Errorf("unknown hook: %s", ctx.Name)
	}
}

func (c *Runner) handlePoll(ctx core.ActionHookContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	taskID, ok := ctx.Parameters["task_id"].(string)
	if !ok {
		return fmt.Errorf("task_id is missing from parameters")
	}

	broker, err := NewBrokerClient(ctx.HTTP)
	if err != nil {
		return fmt.Errorf("new broker client: %w", err)
	}

	task, err := broker.FetchTaskStatus(taskID)
	if err != nil {
		ctx.Logger.WithError(err).Warn("runner: broker poll failed, will retry")
		return ctx.Requests.ScheduleActionCall(hookActionPoll, map[string]any{"task_id": taskID}, pollInterval)
	}

	sink := taskLogFromBrokerTask(task)
	if err := mergeRunnerTaskLog(ctx.Metadata, taskID, sink); err != nil {
		ctx.Logger.WithError(err).Warn("runner: execution metadata update failed")
	}

	if task.IsInTerminalState() {
		return c.processTaskStatus(ctx.ExecutionState, task)
	}

	return ctx.Requests.ScheduleActionCall(hookActionPoll, map[string]any{"task_id": taskID}, pollInterval)
}

func (c *Runner) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	broker, err := NewBrokerClient(ctx.HTTP)
	if err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("new broker client: %w", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(ctx.Body, &raw); err != nil {
		raw = nil
	}

	task, err := broker.ProcessWebhook(ctx.Body)
	if err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("process webhook: %w", err)
	}

	if !task.IsInTerminalState() {
		ctx.Logger.Warn("runner: broker webhook received non-terminal state")
	}

	executionCtx, err := ctx.FindExecutionByKV("task_id", task.TaskID)
	if err != nil {
		return http.StatusNotFound, nil, nil
	}

	sink := taskLogFromBrokerTask(task)
	if sink == nil {
		sink = taskLogFromRawWebhook(raw)
	}
	if executionCtx.Metadata != nil {
		if err := mergeRunnerTaskLog(executionCtx.Metadata, task.TaskID, sink); err != nil {
			ctx.Logger.WithError(err).Warn("runner: execution metadata update failed")
		}
	}

	err = c.processTaskStatus(executionCtx.ExecutionState, task)
	if err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("process task status: %w", err)
	}

	return http.StatusOK, nil, nil
}

func (c *Runner) processTaskStatus(state core.ExecutionStateContext, task *Task) error {
	if state.IsFinished() {
		return nil
	}

	if !task.IsInTerminalState() {
		return fmt.Errorf("task is not in terminal state")
	}

	channel := FailedOutputChannel
	if strings.ToLower(strings.TrimSpace(task.Status)) == "succeeded" && task.effectiveExitCode() == 0 {
		channel = PassedOutputChannel
	}

	out := map[string]any{"status": task.Status, "exit_code": task.effectiveExitCode()}
	if v := brokerResultAsAny(task.Result); v != nil {
		out["result"] = v
	}
	return state.Emit(channel, RunnerFinishedEventType, []any{out})
}

func brokerResultAsAny(raw json.RawMessage) any {
	b := bytes.TrimSpace(raw)
	if len(b) == 0 || !json.Valid(b) {
		return nil
	}
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return nil
	}
	return v
}

func (c *Runner) Cancel(ctx core.ExecutionContext) error {
	if ctx.ExecutionState == nil {
		return nil
	}
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	taskID, err := ctx.ExecutionState.GetKV("task_id")
	if err != nil {
		if errors.Is(err, core.ErrExecutionKVNotFound) {
			return nil
		}
		return fmt.Errorf("runner cancel: get task_id kv: %w", err)
	}

	broker, err := NewBrokerClient(ctx.HTTP)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.WithError(err).Debug("runner: cancel skipped, task broker not configured")
		}
		return nil
	}

	out, err := broker.CancelTask(taskID)
	if err != nil {
		return fmt.Errorf("runner cancel: %w", err)
	}
	if out != nil && ctx.Logger != nil {
		ctx.Logger.Infof(
			"runner: broker cancel accepted fleet_task_id=%s state=%s status=%s",
			out.ID,
			out.State,
			out.Status,
		)
	}
	return nil
}
func (c *Runner) Cleanup(ctx core.SetupContext) error { return nil }

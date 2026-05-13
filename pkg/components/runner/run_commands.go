package runner

import (
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

type Spec struct {
	Commands string `json:"commands" mapstructure:"commands"`
}

func (c *Runner) Name() string  { return "runner" }
func (c *Runner) Label() string { return "Runner" }
func (c *Runner) Icon() string  { return "terminal" }
func (c *Runner) Color() string { return "blue" }

func (c *Runner) ExampleOutput() map[string]any {
	return map[string]any{"status": "succeeded", "exit_code": 0}
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

## Output channels
- **Passed**: The commands finished with exit code **0**.
- **Failed**: The commands finished with non-zero exit code.
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

	webhookURL, err := ctx.Webhook.Setup()
	if err != nil {
		return fmt.Errorf("webhook setup: %w", err)
	}

	cmds := normalizeCommands(spec.Commands)
	broker, err := NewBrokerClient(ctx.HTTP)
	if err != nil {
		return fmt.Errorf("new broker client: %w", err)
	}

	taskID, err := broker.CreateTask(cmds, webhookURL)
	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}

	params := map[string]any{"task_id": taskID}

	err = ctx.ExecutionState.SetKV("task_id", taskID)
	if err != nil {
		return fmt.Errorf("set task id in kv: %w", err)
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

	task, err := broker.ProcessWebhook(ctx.Body)
	if err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("process webhook: %w", err)
	}

	if !task.IsInTerminalState() {
		ctx.Logger.WithError(err).Warn("runner: broker webhook received non-terminal state")
	}

	executionCtx, err := ctx.FindExecutionByKV("task_id", task.TaskID)
	if err != nil {
		return http.StatusNotFound, nil, nil
	}

	err = c.processTaskStatus(executionCtx.ExecutionState, task)
	if err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("process task status: %w", err)
	}

	return http.StatusOK, nil, nil
}

func (c *Runner) processTaskStatus(state core.ExecutionStateContext, task *task) error {
	if state.IsFinished() {
		return nil
	}

	if !task.IsInTerminalState() {
		return fmt.Errorf("task is not in terminal state")
	}

	channel := FailedOutputChannel
	if strings.ToLower(strings.TrimSpace(task.Status)) == "succeeded" && task.ExitCode == 0 {
		channel = PassedOutputChannel
	}

	out := map[string]any{"status": task.Status, "exit_code": task.ExitCode}
	return state.Emit(channel, RunnerFinishedEventType, []any{out})
}

func (c *Runner) Cancel(ctx core.ExecutionContext) error { return nil }
func (c *Runner) Cleanup(ctx core.SetupContext) error    { return nil }

package runner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
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

var dockerExecutionOnly = []configuration.VisibilityCondition{
	{Field: "execution_mode", Values: []string{ExecutionModeDocker}},
}

var dockerImageCustomOnly = []configuration.VisibilityCondition{
	{Field: "execution_mode", Values: []string{ExecutionModeDocker}},
	{Field: "docker_image_preset", Values: []string{DockerImagePresetCustom}},
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
	return "Runs shell commands on a fleet runner (host or Docker container)"
}

func (c *Runner) Documentation() string {
	return `Runs shell commands on a fleet runner.

## Execution
- **Host**: Commands run directly on the runner machine (Bash with a PTY).
- **Docker**: Commands run inside a container started from **Docker image**. The runner pulls the image, starts a long-lived container, and executes your script via ` + "`docker exec`" + `. The image must include a usable ` + "`sleep`" + ` (common base images do).

## Configuration
- **Execution mode**: Host (default) or Docker.
- **Container base image**: Choose a common public image, or **Other (custom image)** to enter any OCI reference.
- **Custom container image**: Shown only for **Other**; use a normal reference (` + "`my.registry.example.com/org/repo:1.2.3`" + ` or ` + "`debian:bookworm-slim@sha256:…`" + `). Private registries require the runner to be configured with registry credentials.
- **Execution timeout**: Optional wall-clock limit in seconds (1–86400). Leave at **0** to use the broker default.
- **Commands**: One or more shell commands, one per line.
- **Environment variables**: Optional key/value pairs available during command execution. Values can be literal strings (with expression support) or organization secret keys.

## Output channels
- **Passed**: The commands finished with exit code **0**.
- **Failed**: The commands finished with non-zero exit code.

## Structured result
If the completed broker task includes valid JSON in **result**, SuperPlane includes it on the ` + "`runner.finished`" + ` event payload next to **status** and **exit_code** (the exact shape depends on your runner / task implementation).
`
}

func (c *Runner) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "execution_mode",
			Label:       "Execution mode",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     ExecutionModeHost,
			Description: "Where the shell commands run: on the runner machine, or inside a container.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{
							Label:       "Host",
							Value:       ExecutionModeHost,
							Description: "Runs in a Bash session on the runner (PTY). Best when the workflow should use tools already installed on the runner.",
						},
						{
							Label:       "Docker",
							Value:       ExecutionModeDocker,
							Description: "Runs in an isolated container started from the image below. The runner must have Docker and (for private registries) pull credentials.",
						},
					},
				},
			},
		},
		{
			Name:                 "docker_image_preset",
			Label:                "Container base image",
			Type:                 configuration.FieldTypeSelect,
			Required:             false,
			Default:              "debian:bookworm-slim",
			Description:          "Pick a common image, or choose Other to type your own registry reference.",
			VisibilityConditions: dockerExecutionOnly,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Debian Bookworm (slim)", Value: "debian:bookworm-slim"},
						{Label: "Ubuntu 24.04", Value: "ubuntu:24.04"},
						{Label: "Alpine 3.20", Value: "alpine:3.20"},
						{Label: "Node.js 22 (Bookworm)", Value: "node:22-bookworm"},
						{Label: "Python 3.12 (slim)", Value: "python:3.12-slim"},
						{Label: "Other (custom image)", Value: DockerImagePresetCustom},
					},
				},
			},
		},
		{
			Name:                 "docker_image",
			Label:                "Custom container image",
			Type:                 configuration.FieldTypeString,
			Required:             false,
			Placeholder:          "e.g. debian:bookworm-slim",
			Description:          "Full OCI image reference when you chose Other above. Pin with a tag or digest for reproducible runs.",
			VisibilityConditions: dockerImageCustomOnly,
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "docker_image_preset", Values: []string{DockerImagePresetCustom}},
			},
			TypeOptions: &configuration.TypeOptions{
				String: &configuration.StringTypeOptions{
					MaxLength: intPtr(maxDockerImageReferenceChars),
				},
			},
		},
		{
			Name:        "commands",
			Label:       "Commands",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Placeholder: "echo \"Hello, World!\"",
			Description: "One or more shell commands, one per line. In Docker mode these run inside the container (after image entrypoint behavior; use an image that stays alive long enough for your script).",
		},
		{
			Name:        "environment",
			Label:       "Environment variables",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Optional key/value pairs passed into the command environment",
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
		{
			Name:        "execution_timeout_seconds",
			Label:       "Execution timeout (seconds)",
			Type:        configuration.FieldTypeNumber,
			Required:    true,
			Default:     0,
			Description: "Hard time limit for the whole task, including image pull and command run. Use 0 for the broker default.",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: intPtr(0),
					Max: intPtr(maxExecutionTimeoutSecondsRequest),
				},
			},
		},
	}
}

func intPtr(v int) *int {
	return &v
}

func (c *Runner) Setup(ctx core.SetupContext) error {
	spec, err := decodeRunnerSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	if err := validateRunnerSpec(spec); err != nil {
		return err
	}

	_, err = ctx.Webhook.Setup()
	return err
}

func (c *Runner) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *Runner) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeRunnerSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	if err := validateRunnerSpec(spec); err != nil {
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

	mode := normalizeExecutionMode(spec.ExecutionMode)
	params := CreateTaskParams{
		Commands:       cmds,
		WebhookURL:     webhookURL,
		Environment:    environment,
		ExecutionMode:  mode,
		DockerImage:    resolvedDockerImageRef(spec),
		TimeoutSeconds: spec.ExecutionTimeoutSeconds,
	}

	taskID, err := broker.CreateTask(params)
	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}

	hookParams := map[string]any{"task_id": taskID}

	err = ctx.ExecutionState.SetKV("task_id", taskID)
	if err != nil {
		return fmt.Errorf("set task id in kv: %w", err)
	}

	if err := mergeRunnerBrokerTaskID(ctx.Metadata, taskID); err != nil {
		return fmt.Errorf("runner execution metadata: %w", err)
	}

	return ctx.Requests.ScheduleActionCall(hookActionPoll, hookParams, pollInterval)
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

func (c *Runner) Cancel(ctx core.ExecutionContext) error { return nil }
func (c *Runner) Cleanup(ctx core.SetupContext) error    { return nil }

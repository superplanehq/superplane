package runner

import (
	"bytes"
	"encoding/json"
	"fmt"
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

	pollInterval = 2 * time.Minute

	// HookActionPoll is the internal hook name used to poll broker task status.
	// Components that offload compute to a runner register a hook with this name
	// and forward it to PollTask.
	HookActionPoll = "poll"
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
func (c *Runner) Label() string { return "Run Shell Commands" }
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
- **Machine type**: Runner fleet registered on the task-broker (required).
- **Execution mode**: Host (default) or Docker.
- **Container base image**: Choose a common public image, or **Other (custom image)** to enter any OCI reference.
- **Custom container image**: Shown only for **Other**; use a normal reference (` + "`my.registry.example.com/org/repo:1.2.3`" + ` or ` + "`debian:bookworm-slim@sha256:…`" + `). Private registries require the runner to be configured with registry credentials.
- **Execution timeout**: Optional wall-clock limit in seconds (1–86400). Defaults to **3600** (1 hour) when unset or **0**.
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
			Name:     configurationFieldMachineType,
			Label:    "Machine type",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: machineTypeSelectOptions,
				},
			},
		},
		{
			Name:        "execution_mode",
			Label:       "Execution mode",
			Type:        configuration.FieldTypeSelect,
			Required:    false, // legacy nodes omit this; defaults applied in decodeRunnerSpec / normalizeExecutionMode
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
			Description: "One shell command per line.",
			TypeOptions: &configuration.TypeOptions{
				Text: &configuration.TextTypeOptions{
					Language: "shell",
				},
			},
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
			Required:    false, // legacy nodes omit this; 0 means DefaultExecutionTimeoutSeconds
			Default:     DefaultExecutionTimeoutSeconds,
			Description: "Hard time limit for the whole task, including image pull and command run. Defaults to 3600 seconds (1 hour).",
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
		MachineType:    spec.MachineType,
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

	return AfterTaskCreated(ctx, taskID)
}

func (c *Runner) Hooks() []core.Hook {
	return []core.Hook{{Name: HookActionPoll, Type: core.HookTypeInternal}}
}

func (c *Runner) HandleHook(ctx core.ActionHookContext) error {
	switch ctx.Name {
	case HookActionPoll:
		return PollTask(ctx, c.taskOutcome())
	default:
		return fmt.Errorf("unknown hook: %s", ctx.Name)
	}
}

func (c *Runner) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return HandleTaskWebhook(ctx, c.taskOutcome())
}

func (c *Runner) taskOutcome() TaskOutcome {
	return TaskOutcome{
		FinishedEventType: RunnerFinishedEventType,
		PassedChannel:     PassedOutputChannel,
		FailedChannel:     FailedOutputChannel,
	}
}

func (c *Runner) processTaskStatus(state core.ExecutionStateContext, task *Task) error {
	return processBrokerTaskStatus(state, task, c.taskOutcome())
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
	return CancelBrokerTask(ctx)
}

func (c *Runner) Cleanup(ctx core.SetupContext) error { return nil }

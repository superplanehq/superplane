package runner

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	RunJSComponentName       = "runnerJS"
	RunJSFinishedEventType   = "runnerJS.finished"
	runJSDefaultDockerPreset = "node:22-bookworm"
	defaultRunJSScript       = "function main() {\n  console.log(\"Hello world\");\n\n  return { \"example\": \"output\" };\n}"
)

func init() {
	registry.RegisterAction(RunJSComponentName, &RunJS{})
}

type RunJS struct{}

func (c *RunJS) Name() string  { return RunJSComponentName }
func (c *RunJS) Label() string { return "Run JavaScript" }
func (c *RunJS) Icon() string  { return "code" }
func (c *RunJS) Color() string { return "blue" }

func (c *RunJS) ExampleOutput() map[string]any {
	return map[string]any{
		"type":      RunJSFinishedEventType,
		"timestamp": "2026-01-16T17:56:16.680755501Z",
		"data": []any{map[string]any{
			"status":    "succeeded",
			"exit_code": 0,
			"result":    map[string]any{"example": "value"},
		}},
	}
}

func (c *RunJS) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: PassedOutputChannel, Label: "Passed"},
		{Name: FailedOutputChannel, Label: "Failed"},
	}
}

func (c *RunJS) Description() string {
	return "Runs JavaScript on a fleet runner with access to upstream node data via $"
}

func (c *RunJS) Documentation() string {
	return `Runs JavaScript on a fleet runner.

## Execution
- **Host**: Script runs with Node.js on the runner machine.
- **Docker**: Script runs inside a container started from **Docker image**. Use a Node.js image (for example **Node.js 22 (Bookworm)**) so ` + "`node`" + ` is available.

## Script contract
Your script must define ` + "`function main()`" + `. The runner injects upstream canvas data as the global ` + "`$`" + ` object (same shape as workflow expressions). Return a JSON-serializable value from ` + "`main()`" + `; it is emitted on the finished event as **result**.

Example:

` + "```javascript" + `
function main() {
  return { pr: $['GitHub PR'].data.number };
}
` + "```" + `

## Configuration
- **Machine type**: Runner fleet registered on the task-broker (required).
- **Execution mode**: Host (default) or Docker.
- **Container base image**: Defaults to a Node.js image in Docker mode.
- **Execution timeout**: Optional wall-clock limit in seconds (1–86400). Defaults to **3600** (1 hour) when unset or **0**.
- **Script**: JavaScript source executed by Node.js.
- **Setup commands**: Optional shell commands (one per line) run before the script in the same environment and working directory.
- **Environment variables**: Optional key/value pairs available during execution.

## Output channels
- **Passed**: The script finished with exit code **0**.
- **Failed**: The script finished with non-zero exit code.
`
}

var setupCommandsEnabledOnly = []configuration.VisibilityCondition{
	{Field: "enable_setup_commands", Values: []string{"true"}},
}

func (c *RunJS) Configuration() []configuration.Field {
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
			Required:    false,
			Default:     ExecutionModeHost,
			Description: "Where the script runs: on the runner machine, or inside a container.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{
							Label:       "Host",
							Value:       ExecutionModeHost,
							Description: "Runs with Node.js on the runner. The fleet image must include Node.",
						},
						{
							Label:       "Docker",
							Value:       ExecutionModeDocker,
							Description: "Runs in an isolated container from the image below. Pick a Node.js image.",
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
			Default:              runJSDefaultDockerPreset,
			Description:          "Pick a Node.js image, or choose Other to type your own registry reference.",
			VisibilityConditions: dockerExecutionOnly,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Node.js 22 (Bookworm)", Value: "node:22-bookworm"},
						{Label: "Node.js 20 (Bookworm)", Value: "node:20-bookworm"},
						{Label: "Node.js 18 (Bookworm)", Value: "node:18-bookworm"},
						{Label: "Debian Bookworm (slim)", Value: "debian:bookworm-slim"},
						{Label: "Ubuntu 24.04", Value: "ubuntu:24.04"},
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
			Placeholder:          "e.g. node:22-bookworm",
			Description:          "Full OCI image reference when you chose Other above.",
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
			Name:        "enable_setup_commands",
			Label:       "Run setup commands",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Run shell commands before the JavaScript script.",
		},
		{
			Name:                 "setup_commands",
			Label:                "Setup commands",
			Type:                 configuration.FieldTypeText,
			Required:             false,
			Placeholder:          "npm ci",
			Description:          "One shell command per line. Runs before the script with the same environment variables.",
			VisibilityConditions: setupCommandsEnabledOnly,
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "enable_setup_commands", Values: []string{"true"}},
			},
		},
		{
			Name:        "script",
			Label:       "Script",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Default:     defaultRunJSScript,
			Description: "JavaScript executed by Node.js. Define function main() and return a JSON-serializable value.",
			TypeOptions: &configuration.TypeOptions{
				Text: &configuration.TextTypeOptions{
					Language: "javascript",
				},
			},
		},
		{
			Name:        "environment",
			Label:       "Environment variables",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Optional key/value pairs passed into the script environment",
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
			Required:    false,
			Default:     DefaultExecutionTimeoutSeconds,
			Description: "Hard time limit for the whole task, including image pull and script run. Defaults to 3600 seconds (1 hour).",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: intPtr(0),
					Max: intPtr(maxExecutionTimeoutSecondsRequest),
				},
			},
		},
	}
}

func (c *RunJS) Setup(ctx core.SetupContext) error {
	spec, err := decodeRunJSSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	if err := validateRunJSSpec(spec); err != nil {
		return err
	}

	_, err = ctx.Webhook.Setup()
	return err
}

func (c *RunJS) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *RunJS) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeRunJSSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	if err := validateRunJSSpec(spec); err != nil {
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

	messageChain, err := messageChainJSON(ctx.Expressions)
	if err != nil {
		return err
	}

	broker, err := NewBrokerClient(ctx.HTTP)
	if err != nil {
		return fmt.Errorf("new broker client: %w", err)
	}

	mode := normalizeExecutionMode(spec.ExecutionMode)
	var setupCommands []string
	if spec.EnableSetupCommands {
		setupCommands = normalizeCommands(spec.SetupCommands)
	}

	params := CreateTaskParams{
		MachineType:    spec.MachineType,
		RunMode:        RunModeJavaScript,
		Script:         spec.Script,
		MessageChain:   messageChain,
		SetupCommands:  setupCommands,
		WebhookURL:     webhookURL,
		Environment:    environment,
		ExecutionMode:  mode,
		DockerImage:    resolvedRunJSDockerImageRef(spec),
		TimeoutSeconds: spec.ExecutionTimeoutSeconds,
	}

	taskID, err := broker.CreateTask(params)
	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}

	return AfterTaskCreated(ctx, taskID)
}

func (c *RunJS) Hooks() []core.Hook {
	return []core.Hook{{Name: HookActionPoll, Type: core.HookTypeInternal}}
}

func (c *RunJS) HandleHook(ctx core.ActionHookContext) error {
	switch ctx.Name {
	case HookActionPoll:
		return PollTask(ctx, TaskOutcome{FinishedEventType: RunJSFinishedEventType})
	default:
		return fmt.Errorf("unknown hook: %s", ctx.Name)
	}
}

func (c *RunJS) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return HandleTaskWebhook(ctx, TaskOutcome{FinishedEventType: RunJSFinishedEventType})
}

func (c *RunJS) Cancel(ctx core.ExecutionContext) error {
	return CancelBrokerTask(ctx)
}

func (c *RunJS) Cleanup(ctx core.SetupContext) error { return nil }

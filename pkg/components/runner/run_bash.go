package runner

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	RunBashComponentName       = "runnerBash"
	RunBashFinishedEventType   = "runnerBash.finished"
	runBashDefaultDockerPreset = "debian:bookworm-slim"
	defaultRunBashScript       = "#!/usr/bin/env bash\nset -euo pipefail\n\necho \"Hello world\"\necho '{\"example\":\"output\"}' > \"$SUPERPLANE_RESULT_FILE\"\n"
)

func init() {
	registry.RegisterAction(RunBashComponentName, &RunBash{})
}

type RunBash struct{}

func (c *RunBash) Name() string  { return RunBashComponentName }
func (c *RunBash) Label() string { return "Run Bash" }
func (c *RunBash) Icon() string  { return "code" }
func (c *RunBash) Color() string { return "blue" }

func (c *RunBash) ExampleOutput() map[string]any {
	return map[string]any{
		"type":      RunBashFinishedEventType,
		"timestamp": "2026-01-16T17:56:16.680755501Z",
		"data": []any{map[string]any{
			"status":    "succeeded",
			"exit_code": 0,
			"result":    map[string]any{"example": "value"},
		}},
	}
}

func (c *RunBash) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: PassedOutputChannel, Label: "Passed"},
		{Name: FailedOutputChannel, Label: "Failed"},
	}
}

func (c *RunBash) Description() string {
	return "Runs a Bash script on a fleet runner with upstream node data in SUPERPLANE_PAYLOAD_FILE"
}

func (c *RunBash) Documentation() string {
	return `Runs a Bash script on a fleet runner.

## Execution
- **Host**: Script runs with Bash on the runner machine.
- **Docker**: Script runs inside a container started from **Docker image**. Use an image that includes Bash (for example **Debian Bookworm (slim)**).

## Script contract
Your script runs as-is. The runner sets:

- ` + "`SUPERPLANE_PAYLOAD_FILE`" + ` — path to a JSON file with upstream canvas data (same shape as workflow expressions)
- ` + "`SUPERPLANE_RESULT_FILE`" + ` — path where your script must write a JSON-serializable **result**

Stdout and stderr (for example ` + "`echo`" + `) stream to **View logs**. Write the structured **result** to ` + "`SUPERPLANE_RESULT_FILE`" + `; it is emitted on the finished event as **result**.

Example:

` + "```bash" + `
#!/usr/bin/env bash
set -euo pipefail

num=$(jq -r '."GitHub PR".data.number' "$SUPERPLANE_PAYLOAD_FILE")
echo "Processing PR #$num"
printf '{"pr":%s}\n' "$num" > "$SUPERPLANE_RESULT_FILE"
` + "```" + `

## Configuration
- **Machine type**: Runner fleet registered on the task-broker (required).
- **Execution mode**: Host (default) or Docker.
- **Container base image**: Defaults to a Debian image in Docker mode.
- **Execution timeout**: Optional wall-clock limit in seconds (1–86400). Defaults to **3600** (1 hour) when unset or **0**.
- **Script**: Bash source executed by the runner.
- **Setup commands**: Optional shell commands (one per line) run before the script in the same environment and working directory.
- **Environment variables**: Optional key/value pairs available during execution.

## Output channels
- **Passed**: The script finished with exit code **0**.
- **Failed**: The script finished with non-zero exit code.
`
}

func (c *RunBash) Configuration() []configuration.Field {
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
							Description: "Runs with Bash on the runner. The fleet image must include Bash.",
						},
						{
							Label:       "Docker",
							Value:       ExecutionModeDocker,
							Description: "Runs in an isolated container from the image below.",
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
			Default:              runBashDefaultDockerPreset,
			Description:          "Pick a base image with Bash, or choose Other to type your own registry reference.",
			VisibilityConditions: dockerExecutionOnly,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Debian Bookworm (slim)", Value: "debian:bookworm-slim"},
						{Label: "Ubuntu 24.04", Value: "ubuntu:24.04"},
						{Label: "Alpine 3.20", Value: "alpine:3.20"},
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
			Description: "Run shell commands before the Bash script.",
		},
		{
			Name:                 "setup_commands",
			Label:                "Setup commands",
			Type:                 configuration.FieldTypeText,
			Required:             false,
			Placeholder:          "apt-get update",
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
			Default:     defaultRunBashScript,
			Description: "Bash executed by the runner. Write JSON to SUPERPLANE_RESULT_FILE; read upstream data from SUPERPLANE_PAYLOAD_FILE.",
			TypeOptions: &configuration.TypeOptions{
				Text: &configuration.TextTypeOptions{
					Language:         "shell",
					AllowExpressions: boolPtr(false),
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

func (c *RunBash) Setup(ctx core.SetupContext) error {
	spec, err := decodeRunBashSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	if err := validateRunBashSpec(spec); err != nil {
		return err
	}

	_, err = ctx.Webhook.Setup()
	return err
}

func (c *RunBash) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *RunBash) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeRunBashSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	if err := validateRunBashSpec(spec); err != nil {
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
		RunMode:        RunModeBash,
		Script:         spec.Script,
		MessageChain:   messageChain,
		SetupCommands:  setupCommands,
		WebhookURL:     webhookURL,
		Environment:    environment,
		ExecutionMode:  mode,
		DockerImage:    resolvedRunBashDockerImageRef(spec),
		TimeoutSeconds: spec.ExecutionTimeoutSeconds,
	}

	taskID, err := broker.CreateTask(params)
	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}

	return afterRunnerTaskCreated(ctx, taskID)
}

func (c *RunBash) Hooks() []core.Hook {
	return []core.Hook{{Name: hookActionPoll, Type: core.HookTypeInternal}}
}

func (c *RunBash) HandleHook(ctx core.ActionHookContext) error {
	switch ctx.Name {
	case hookActionPoll:
		return pollBrokerTask(ctx, RunBashFinishedEventType)
	default:
		return fmt.Errorf("unknown hook: %s", ctx.Name)
	}
}

func (c *RunBash) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return handleBrokerWebhook(ctx, RunBashFinishedEventType)
}

func (c *RunBash) Cancel(ctx core.ExecutionContext) error {
	return cancelBrokerTask(ctx)
}

func (c *RunBash) Cleanup(ctx core.SetupContext) error { return nil }

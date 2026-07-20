package runner

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	RunClaudeCodeComponentName     = "runnerClaudeCode"
	RunClaudeCodeFinishedEventType = "runnerClaudeCode.finished"
)

func init() {
	registry.RegisterAction(RunClaudeCodeComponentName, &RunClaudeCode{})
}

type RunClaudeCode struct{}

func (c *RunClaudeCode) Name() string  { return RunClaudeCodeComponentName }
func (c *RunClaudeCode) Label() string { return "Run Claude Code" }
func (c *RunClaudeCode) Icon() string  { return "code" }
func (c *RunClaudeCode) Color() string { return "#C9784D" }

func (c *RunClaudeCode) ExampleOutput() map[string]any {
	return map[string]any{
		"type":      RunClaudeCodeFinishedEventType,
		"timestamp": "2026-01-16T17:56:16.680755501Z",
		"data": []any{map[string]any{
			"status":    "succeeded",
			"exit_code": 0,
			"result": map[string]any{
				"type":       "result",
				"result":     "Done.",
				"session_id": "session-123",
			},
		}},
	}
}

func (c *RunClaudeCode) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: PassedOutputChannel, Label: "Passed"},
		{Name: FailedOutputChannel, Label: "Failed"},
	}
}

func (c *RunClaudeCode) Description() string {
	return "Runs the Claude Code CLI on a fleet runner (claude must already be installed on the machine)"
}

func (c *RunClaudeCode) Documentation() string {
	return `Runs [Claude Code](https://code.claude.com/docs/en/headless) in non-interactive mode on a fleet runner.

## Prerequisites
- The ` + "`claude`" + ` CLI is installed on the runner machine and available on ` + "`PATH`" + `.
- An Anthropic API key stored as a SuperPlane secret (passed as ` + "`ANTHROPIC_API_KEY`" + `).

## Steps
Configure an ordered list of **bash** and **prompt** steps. Each step runs as its own broker command so **View logs** shows a separate section per step:

- **bash** — shell commands (clone a repo, install deps, run tests, push).
- **prompt** — a Claude Code headless turn (` + "`claude --bare -p`" + `). After the first prompt, later prompts use ` + "`--continue`" + ` so they share the same Claude session.

Example:

1. bash — ` + "`git clone … && cd repo`" + `
2. prompt — implement the feature
3. prompt — run tests and fix failures
4. bash — ` + "`git push`" + `

## Configuration
- **Machine type**: Runner fleet registered on the task-broker (required).
- **Steps**: Ordered bash/prompt actions (at least one prompt required).
- **Anthropic API Key**: SuperPlane secret used as ` + "`ANTHROPIC_API_KEY`" + `.
- **Model**: Optional model id or alias (for example ` + "`sonnet`" + `).
- **Working directory**: Optional directory each step starts in.
- **Execution timeout**: Optional wall-clock limit in seconds (1–86400). Defaults to **3600** (1 hour).

## Output
Prompt steps stream readable agent activity to **View logs** (Claude text, tool calls, and truncated tool results). The latest stream ` + "`result`" + ` event is emitted as **result** on the finished event.

## Output channels
- **Passed**: All steps finished with exit code **0**.
- **Failed**: A bash or prompt step failed (non-zero exit).
`
}

func (c *RunClaudeCode) Configuration() []configuration.Field {
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
			Name:        "anthropicApiKey",
			Label:       "Anthropic API Key",
			Type:        configuration.FieldTypeSecretKey,
			Required:    true,
			Description: "SuperPlane secret holding an Anthropic API key. Exposed to the runner as ANTHROPIC_API_KEY.",
		},
		{
			Name:        "model",
			Label:       "Model",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional Claude model id or alias (for example sonnet, opus, or claude-sonnet-4-6).",
			Placeholder: "sonnet",
		},
		{
			Name:        "steps",
			Label:       "Steps",
			Type:        configuration.FieldTypeList,
			Required:    true,
			Default:     defaultClaudeCodeSteps(),
			Description: "Ordered bash commands and Claude Code prompts. Add, reorder, and mix freely.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel:   "Step",
					Accordion:   true,
					Reorderable: true,
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "name",
								Label:       "Name",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Placeholder: "e.g. Clone repo",
							},
							{
								Name:     "type",
								Label:    "Type",
								Type:     configuration.FieldTypeSelect,
								Required: true,
								Default:  claudeStepPrompt,
								TypeOptions: &configuration.TypeOptions{
									Select: &configuration.SelectTypeOptions{
										Options: []configuration.FieldOption{
											{Label: "Prompt", Value: claudeStepPrompt, Description: "Run a Claude Code headless turn"},
											{Label: "Bash", Value: claudeStepBash, Description: "Run shell commands on the runner"},
										},
									},
								},
							},
							{
								Name:        "prompt",
								Label:       "Prompt",
								Type:        configuration.FieldTypeText,
								Required:    false,
								Placeholder: "Fix the failing tests and commit the changes.",
								VisibilityConditions: []configuration.VisibilityCondition{
									{Field: "type", Values: []string{claudeStepPrompt}},
								},
								RequiredConditions: []configuration.RequiredCondition{
									{Field: "type", Values: []string{claudeStepPrompt}},
								},
							},
							{
								Name:        "command",
								Label:       "Command",
								Type:        configuration.FieldTypeText,
								Required:    false,
								Placeholder: "git clone https://github.com/org/repo.git /tmp/repo",
								VisibilityConditions: []configuration.VisibilityCondition{
									{Field: "type", Values: []string{claudeStepBash}},
								},
								RequiredConditions: []configuration.RequiredCondition{
									{Field: "type", Values: []string{claudeStepBash}},
								},
								TypeOptions: &configuration.TypeOptions{
									Text: &configuration.TextTypeOptions{
										Language: "shell",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Name:        "workingDirectory",
			Label:       "Working directory",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional directory the script starts in. Prefer a bash step to clone or prepare the workspace.",
			Placeholder: "/tmp/repo",
		},
		{
			Name:        "environment",
			Label:       "Environment variables",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Optional key/value pairs passed into the Claude Code environment (in addition to ANTHROPIC_API_KEY)",
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
								Placeholder: "e.g. GITHUB_TOKEN",
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
			Description: "Hard time limit for the whole task, including all steps. Defaults to 3600 seconds (1 hour).",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: intPtr(0),
					Max: intPtr(maxExecutionTimeoutSecondsRequest),
				},
			},
		},
	}
}

func (c *RunClaudeCode) Setup(ctx core.SetupContext) error {
	spec, err := decodeRunClaudeCodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	if err := validateRunClaudeCodeSpec(spec); err != nil {
		return err
	}

	_, err = ctx.Webhook.Setup()
	return err
}

func (c *RunClaudeCode) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *RunClaudeCode) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeRunClaudeCodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	if err := validateRunClaudeCodeSpec(spec); err != nil {
		return err
	}

	environment, err := resolveEnvironment(ctx.Secrets, spec.Environment)
	if err != nil {
		return err
	}

	if ctx.Secrets == nil {
		return fmt.Errorf("resolve anthropic API key: secrets context is unavailable")
	}
	apiKey, err := ctx.Secrets.GetKey(spec.AnthropicAPIKey.Secret, spec.AnthropicAPIKey.Key)
	if err != nil {
		return fmt.Errorf("resolve anthropic API key: %w", err)
	}
	environment = append(environment, BrokerEnvironmentVariable{
		Name:  envAnthropicAPIKey,
		Value: string(apiKey),
	})

	webhookURL, err := ctx.Webhook.Setup()
	if err != nil {
		return fmt.Errorf("webhook setup: %w", err)
	}

	broker, err := NewBrokerClient(ctx.HTTP)
	if err != nil {
		return fmt.Errorf("new broker client: %w", err)
	}

	// command_list tasks only accept commands (no message_chain / script fields).
	task := buildClaudeCodeBrokerTask(spec)
	params := CreateTaskParams{
		MachineType:    spec.MachineType,
		Commands:       task.Commands,
		WebhookURL:     webhookURL,
		Environment:    environment,
		ExecutionMode:  ExecutionModeHost,
		TimeoutSeconds: spec.ExecutionTimeoutSeconds,
	}

	taskID, err := broker.CreateTask(params)
	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}

	return afterRunnerTaskCreated(ctx, taskID)
}

func (c *RunClaudeCode) Hooks() []core.Hook {
	return []core.Hook{{Name: hookActionPoll, Type: core.HookTypeInternal}}
}

func (c *RunClaudeCode) HandleHook(ctx core.ActionHookContext) error {
	switch ctx.Name {
	case hookActionPoll:
		return pollBrokerTask(ctx, RunClaudeCodeFinishedEventType)
	default:
		return fmt.Errorf("unknown hook: %s", ctx.Name)
	}
}

func (c *RunClaudeCode) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return handleBrokerWebhook(ctx, RunClaudeCodeFinishedEventType)
}

func (c *RunClaudeCode) Cancel(ctx core.ExecutionContext) error {
	return cancelBrokerTask(ctx)
}

func (c *RunClaudeCode) Cleanup(ctx core.SetupContext) error { return nil }

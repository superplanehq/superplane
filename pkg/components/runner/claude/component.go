package claude

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	ComponentName     = "runnerClaudeCode"
	FinishedEventType = "runnerClaudeCode.finished"
)

func init() {
	registry.RegisterAction(ComponentName, &RunClaudeCode{})
	runner.RegisterRunnerComponent(ComponentName)
}

type RunClaudeCode struct{}

func (c *RunClaudeCode) Name() string  { return ComponentName }
func (c *RunClaudeCode) Label() string { return "Run Claude Code" }
func (c *RunClaudeCode) Icon() string  { return "code" }
func (c *RunClaudeCode) Color() string { return "#C9784D" }

func (c *RunClaudeCode) ExampleOutput() map[string]any {
	return map[string]any{
		"type":      FinishedEventType,
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
		{Name: runner.PassedOutputChannel, Label: "Passed"},
		{Name: runner.FailedOutputChannel, Label: "Failed"},
	}
}

func (c *RunClaudeCode) Description() string {
	return "Runs Claude Code on a fleet runner"
}

func (c *RunClaudeCode) Documentation() string {
	return `Runs [Claude Code](https://code.claude.com/docs/en/headless) in non-interactive mode on a fleet runner.

## Prerequisites
- The ` + "`claude`" + ` CLI is installed on the runner machine and available on ` + "`PATH`" + `.
- An Anthropic API key stored as a SuperPlane secret.

## Steps
Configure an ordered list of **bash** and **prompt** steps:

- **bash** — shell commands (clone a repo, install deps, run tests, push).
- **prompt** — a Claude Code turn. Later prompts continue the same session.

Example:

1. bash — ` + "`git clone …`" + `
2. prompt — implement the feature
3. prompt — run tests and fix failures
4. bash — ` + "`git push`" + `

## Configuration
- **Machine type**: Runner fleet registered on the task-broker (required).
- **Steps**: Ordered bash/prompt actions (at least one prompt required).
- **Anthropic API Key**: SuperPlane secret used as ` + "`ANTHROPIC_API_KEY`" + `.
- **Model**: Optional model id or alias (for example ` + "`sonnet`" + `).
- **Working directory**: Optional starting directory.
- **Execution timeout**: Optional wall-clock limit in seconds (1–86400). Defaults to **3600** (1 hour).

## Output
Prompt steps stream agent activity to **View logs**. The finished event includes the latest Claude ` + "`result`" + `.

## Output channels
- **Passed**: All steps finished with exit code **0**.
- **Failed**: A bash or prompt step failed (non-zero exit).
`
}

func (c *RunClaudeCode) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "machineType",
			Label:    "Machine type",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: runner.MachineTypeOptions(),
				},
			},
		},
		{
			Name:        "credentials",
			Label:       "Credentials",
			Type:        configuration.FieldTypeObject,
			Required:    true,
			Description: "Anthropic API key or Claude integration to use.",
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: []configuration.Field{
						{
							Name:     "source",
							Label:    "Source",
							Type:     configuration.FieldTypeSelect,
							Required: true,
							TypeOptions: &configuration.TypeOptions{
								Select: &configuration.SelectTypeOptions{
									Options: []configuration.FieldOption{
										{Label: "Secret", Value: "secret"},
										{Label: "Integration", Value: "integration"},
									},
								},
							},
						},
						{
							Name:  "secret",
							Label: "Anthropic API Key",
							Type:  configuration.FieldTypeSecretKey,
							VisibilityConditions: []configuration.VisibilityCondition{
								{Field: "source", Values: []string{"secret"}},
							},
							RequiredConditions: []configuration.RequiredCondition{
								{Field: "source", Values: []string{"secret"}},
							},
						},
						{
							Name:  "integration",
							Label: "Integration",
							Type:  configuration.FieldTypeIntegration,
							VisibilityConditions: []configuration.VisibilityCondition{
								{Field: "source", Values: []string{"integration"}},
							},
							RequiredConditions: []configuration.RequiredCondition{
								{Field: "source", Values: []string{"integration"}},
							},
							TypeOptions: &configuration.TypeOptions{
								Integration: &configuration.IntegrationTypeOptions{
									Integration: "claude",
								},
							},
						},
					},
				},
			},
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
			Description: "Optional starting directory.",
			Placeholder: "/tmp/repo",
		},
		runner.EnvironmentFromConfigurationField(),
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
								Default:     runner.EnvironmentValueSourceLiteral,
								TypeOptions: &configuration.TypeOptions{
									Select: &configuration.SelectTypeOptions{
										Options: []configuration.FieldOption{
											{Label: "Literal value", Value: runner.EnvironmentValueSourceLiteral},
											{Label: "Secret key", Value: runner.EnvironmentValueSourceSecret},
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
								VisibilityConditions: []configuration.VisibilityCondition{{Field: "valueSource", Values: []string{runner.EnvironmentValueSourceLiteral}}},
								RequiredConditions:   []configuration.RequiredCondition{{Field: "valueSource", Values: []string{runner.EnvironmentValueSourceLiteral}}},
							},
							{
								Name:                 "secret",
								Label:                "Secret key",
								Type:                 configuration.FieldTypeSecretKey,
								Description:          "Stored credential key to use as the variable value",
								Required:             false,
								VisibilityConditions: []configuration.VisibilityCondition{{Field: "valueSource", Values: []string{runner.EnvironmentValueSourceSecret}}},
								RequiredConditions:   []configuration.RequiredCondition{{Field: "valueSource", Values: []string{runner.EnvironmentValueSourceSecret}}},
							},
						},
					},
				},
			},
		},
		{
			Name:        "executionTimeoutSeconds",
			Label:       "Execution timeout (seconds)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     runner.DefaultExecutionTimeoutSeconds,
			Description: "Hard time limit for the whole task, including all steps. Defaults to 3600 seconds (1 hour).",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: runner.IntPtr(0),
					Max: runner.IntPtr(runner.MaxExecutionTimeoutSecondsRequest),
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

	environment, err := runner.ResolveEnvironment(ctx.Secrets, spec.EnvironmentFrom, spec.Environment)
	if err != nil {
		return err
	}

	environment, err = c.injectCredentials(ctx, environment, spec.Credentials)
	if err != nil {
		return err
	}

	webhookURL, err := ctx.Webhook.Setup()
	if err != nil {
		return fmt.Errorf("webhook setup: %w", err)
	}

	broker, err := runner.NewBrokerClient(ctx.HTTP)
	if err != nil {
		return fmt.Errorf("new broker client: %w", err)
	}

	// command_list tasks only accept commands (+ optional files).
	task := buildClaudeCodeBrokerTask(spec)
	params := runner.CreateTaskParams{
		MachineType:    spec.MachineType,
		Commands:       task.Commands,
		Files:          task.Files,
		WebhookURL:     webhookURL,
		Environment:    environment,
		ExecutionMode:  runner.ExecutionModeHost,
		TimeoutSeconds: spec.ExecutionTimeoutSeconds,
	}

	taskID, err := broker.CreateTask(params)
	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}

	return runner.AfterRunnerTaskCreated(ctx, taskID)
}

func (c *RunClaudeCode) injectCredentials(ctx core.ExecutionContext, environment []runner.BrokerEnvironmentVariable, credentials ClaudeCodeCredentials) ([]runner.BrokerEnvironmentVariable, error) {
	switch credentials.Source {
	case "secret":
		apiKey, err := ctx.Secrets.GetKey(credentials.Secret.Secret, credentials.Secret.Key)
		if err != nil {
			return nil, fmt.Errorf("resolve anthropic API key: %w", err)
		}

		return append(environment, runner.BrokerEnvironmentVariable{
			Name:  envAnthropicAPIKey,
			Value: string(apiKey),
		}), nil

	case "integration":
		keys, err := ctx.Secrets.GetIntegrationKeys(credentials.Integration.Name)
		if err != nil {
			return nil, fmt.Errorf("resolve integration: %w", err)
		}
		for name, value := range keys {
			environment = append(environment, runner.BrokerEnvironmentVariable{
				Name:  name,
				Value: string(value),
			})
		}
		return environment, nil
	default:
		return nil, fmt.Errorf("invalid credentials source: %s", credentials.Source)
	}
}

func (c *RunClaudeCode) Hooks() []core.Hook {
	return []core.Hook{{Name: runner.HookPoll, Type: core.HookTypeInternal}}
}

func (c *RunClaudeCode) HandleHook(ctx core.ActionHookContext) error {
	switch ctx.Name {
	case runner.HookPoll:
		return runner.PollBrokerTask(ctx, FinishedEventType)
	default:
		return fmt.Errorf("unknown hook: %s", ctx.Name)
	}
}

func (c *RunClaudeCode) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return runner.HandleBrokerWebhook(ctx, FinishedEventType)
}

func (c *RunClaudeCode) Cancel(ctx core.ExecutionContext) error {
	return runner.CancelBrokerTask(ctx)
}

func (c *RunClaudeCode) Cleanup(ctx core.SetupContext) error { return nil }

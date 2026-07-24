package opencode

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	ComponentName     = "runnerOpenCode"
	FinishedEventType = "runnerOpenCode.finished"
)

func init() {
	registry.RegisterAction(ComponentName, &RunOpenCode{})
	runner.RegisterRunnerComponent(ComponentName)
}

type RunOpenCode struct{}

func (c *RunOpenCode) Name() string  { return ComponentName }
func (c *RunOpenCode) Label() string { return "Run OpenCode" }
func (c *RunOpenCode) Icon() string  { return "code" }
func (c *RunOpenCode) Color() string { return "#0B7285" }

func (c *RunOpenCode) ExampleOutput() map[string]any {
	return map[string]any{
		"type":      FinishedEventType,
		"timestamp": "2026-01-16T17:56:16.680755501Z",
		"data": []any{map[string]any{
			"status":    "succeeded",
			"exit_code": 0,
			"result": map[string]any{
				"type":       "result",
				"result":     "Done.",
				"session_id": "ses_123",
			},
		}},
	}
}

func (c *RunOpenCode) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: runner.PassedOutputChannel, Label: "Passed"},
		{Name: runner.FailedOutputChannel, Label: "Failed"},
	}
}

func (c *RunOpenCode) Description() string {
	return "Runs OpenCode on a fleet runner"
}

func (c *RunOpenCode) Documentation() string {
	return `Runs [OpenCode](https://opencode.ai/docs/cli/) in non-interactive mode on a fleet runner.

OpenCode is provider-agnostic: point it at any supported ` + "`provider/model`" + ` (OpenAI, Anthropic, Google, OpenRouter, Groq, and more) and supply the matching provider API key.

## Prerequisites
- The ` + "`opencode`" + ` CLI is installed on the runner machine and available on ` + "`PATH`" + ` (see the [OpenCode install docs](https://opencode.ai/docs/)).
- ` + "`node`" + ` is available on ` + "`PATH`" + ` (used to format live logs).
- A provider API key stored as a SuperPlane secret.

## Steps
Configure an ordered list of **bash** and **prompt** steps:

- **bash** — shell commands (clone a repo, install deps, run tests, push).
- **prompt** — an OpenCode turn. Later prompts continue the same session.

Example:

1. bash — ` + "`git clone …`" + `
2. prompt — implement the feature
3. prompt — run tests and fix failures
4. bash — ` + "`git push`" + `

## Configuration
- **Machine type**: Runner fleet registered on the task-broker (required).
- **Model**: Required ` + "`provider/model`" + ` (for example ` + "`openai/gpt-4.1`" + ` or ` + "`anthropic/claude-sonnet-4-5`" + `).
- **Provider API keys**: One or more curated providers, each mapped to a SuperPlane secret. OpenCode reads each key from its well-known environment variable.
- **Steps**: Ordered bash/prompt actions (at least one prompt required).
- **Working directory**: Optional starting directory.
- **Execution timeout**: Optional wall-clock limit in seconds (1–86400). Defaults to **3600** (1 hour).

## Output
Prompt steps stream agent activity to **View logs**. The finished event includes the latest OpenCode result and session id.

## Output channels
- **Passed**: All steps finished with exit code **0**.
- **Failed**: A bash or prompt step failed (non-zero exit) or OpenCode reported an error.
`
}

func (c *RunOpenCode) Configuration() []configuration.Field {
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
			Name:        "model",
			Label:       "Model",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "OpenCode model in provider/model form (for example openai/gpt-4.1 or anthropic/claude-sonnet-4-5).",
			Placeholder: "provider/model",
		},
		{
			Name:        "credentials",
			Label:       "Provider API keys",
			Type:        configuration.FieldTypeList,
			Required:    true,
			Description: "Provider API keys OpenCode reads from the environment. Each entry maps a provider to a stored secret.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Provider API key",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:     "provider",
								Label:    "Provider",
								Type:     configuration.FieldTypeSelect,
								Required: true,
								TypeOptions: &configuration.TypeOptions{
									Select: &configuration.SelectTypeOptions{
										Options: providerFieldOptions(),
									},
								},
							},
							{
								Name:        "secret",
								Label:       "API key",
								Type:        configuration.FieldTypeSecretKey,
								Required:    true,
								Description: "Stored credential key used as the provider API key.",
							},
						},
					},
				},
			},
		},
		{
			Name:        "steps",
			Label:       "Steps",
			Type:        configuration.FieldTypeList,
			Required:    true,
			Default:     defaultOpenCodeSteps(),
			Description: "Ordered bash commands and OpenCode prompts. Add, reorder, and mix freely.",
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
								Default:  openCodeStepPrompt,
								TypeOptions: &configuration.TypeOptions{
									Select: &configuration.SelectTypeOptions{
										Options: []configuration.FieldOption{
											{Label: "Prompt", Value: openCodeStepPrompt, Description: "Run an OpenCode headless turn"},
											{Label: "Bash", Value: openCodeStepBash, Description: "Run shell commands on the runner"},
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
									{Field: "type", Values: []string{openCodeStepPrompt}},
								},
								RequiredConditions: []configuration.RequiredCondition{
									{Field: "type", Values: []string{openCodeStepPrompt}},
								},
							},
							{
								Name:        "command",
								Label:       "Command",
								Type:        configuration.FieldTypeText,
								Required:    false,
								Placeholder: "git clone https://github.com/org/repo.git /tmp/repo",
								VisibilityConditions: []configuration.VisibilityCondition{
									{Field: "type", Values: []string{openCodeStepBash}},
								},
								RequiredConditions: []configuration.RequiredCondition{
									{Field: "type", Values: []string{openCodeStepBash}},
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
			Description: "Optional key/value pairs passed into the OpenCode environment (in addition to provider API keys)",
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

func (c *RunOpenCode) Setup(ctx core.SetupContext) error {
	spec, err := decodeRunOpenCodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	if err := validateRunOpenCodeSpec(spec); err != nil {
		return err
	}

	_, err = ctx.Webhook.Setup()
	return err
}

func (c *RunOpenCode) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *RunOpenCode) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeRunOpenCodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	if err := validateRunOpenCodeSpec(spec); err != nil {
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
	task := buildOpenCodeBrokerTask(spec)
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

func (c *RunOpenCode) injectCredentials(ctx core.ExecutionContext, environment []runner.BrokerEnvironmentVariable, credentials []OpenCodeCredential) ([]runner.BrokerEnvironmentVariable, error) {
	for _, credential := range credentials {
		provider, ok := providerByValue(strings.TrimSpace(credential.Provider))
		if !ok {
			return nil, fmt.Errorf("unsupported provider: %s", credential.Provider)
		}

		apiKey, err := ctx.Secrets.GetKey(credential.Secret.Secret, credential.Secret.Key)
		if err != nil {
			return nil, fmt.Errorf("resolve %s API key: %w", provider.Value, err)
		}

		environment = append(environment, runner.BrokerEnvironmentVariable{
			Name:  provider.EnvVar,
			Value: string(apiKey),
		})
	}
	return environment, nil
}

func (c *RunOpenCode) Hooks() []core.Hook {
	return []core.Hook{{Name: runner.HookPoll, Type: core.HookTypeInternal}}
}

func (c *RunOpenCode) HandleHook(ctx core.ActionHookContext) error {
	switch ctx.Name {
	case runner.HookPoll:
		return runner.PollBrokerTask(ctx, FinishedEventType)
	default:
		return fmt.Errorf("unknown hook: %s", ctx.Name)
	}
}

func (c *RunOpenCode) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return runner.HandleBrokerWebhook(ctx, FinishedEventType)
}

func (c *RunOpenCode) Cancel(ctx core.ExecutionContext) error {
	return runner.CancelBrokerTask(ctx)
}

func (c *RunOpenCode) Cleanup(ctx core.SetupContext) error { return nil }

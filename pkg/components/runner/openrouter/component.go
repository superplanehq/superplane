package openrouter

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	ComponentName        = "runnerOpenRouter"
	FinishedEventType    = "runnerOpenRouter.finished"
	envOpenRouterAPIKey  = "OPENROUTER_API_KEY"
	envOpenRouterBaseURL = "OPENROUTER_BASE_URL"
)

func init() {
	registry.RegisterAction(ComponentName, &RunOpenRouter{})
	runner.RegisterRunnerComponent(ComponentName)
}

type RunOpenRouter struct{}

func (c *RunOpenRouter) Name() string  { return ComponentName }
func (c *RunOpenRouter) Label() string { return "Run OpenRouter Agent" }
func (c *RunOpenRouter) Icon() string  { return "code" }
func (c *RunOpenRouter) Color() string { return "#6566F1" }

func (c *RunOpenRouter) ExampleOutput() map[string]any {
	return map[string]any{
		"type":      FinishedEventType,
		"timestamp": "2026-01-16T17:56:16.680755501Z",
		"data": []any{map[string]any{
			"status":    "succeeded",
			"exit_code": 0,
			"result":    map[string]any{"type": "result", "result": "Done."},
		}},
	}
}

func (c *RunOpenRouter) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: runner.PassedOutputChannel, Label: "Passed"},
		{Name: runner.FailedOutputChannel, Label: "Failed"},
	}
}

func (c *RunOpenRouter) Description() string {
	return "Runs a SuperPlane OpenRouter agent on a fleet runner"
}

func (c *RunOpenRouter) Documentation() string {
	return `Runs a SuperPlane-shipped OpenRouter agent on a fleet runner. Prompt turns can use bash, read, edit, and write tools.

## Prerequisites
- Node.js on the runner ` + "`PATH`" + `.
- An OpenRouter API key stored as a SuperPlane secret or an OpenRouter integration.

## Steps
Configure an ordered list of **bash** and **prompt** steps:

- **bash** — shell commands (clone a repo, install deps, run tests, push).
- **prompt** — one OpenRouter agent loop in the same working directory (up to max turns).

## Configuration
- **Machine type**: Runner fleet registered on the task-broker (required).
- **Steps**: Ordered bash/prompt actions (at least one prompt required).
- **Credentials**: SuperPlane secret or OpenRouter integration used as ` + "`OPENROUTER_API_KEY`" + `.
- **Model**: Required OpenRouter model id (` + "`provider/model`" + `).
- **Working directory**: Optional starting directory.
- **Execution timeout**: Optional wall-clock limit in seconds (1–86400). Defaults to **3600** (1 hour).
- **Max turns per prompt**: Optional limit on model turns for each prompt step (1–256). Defaults to **128**. After this limit, SuperPlane asks for a final reply without tools.

Use **Run SuperPlane Agent** for SuperPlane-hosted credentials.

## Output channels
- **Passed**: All steps finished with exit code **0**.
- **Failed**: A bash or prompt step failed (non-zero exit).
`
}

func (c *RunOpenRouter) Configuration() []configuration.Field {
	model := runner.AgentModelField("openrouter", "OpenRouter model id (provider/model). Required.", "anthropic/claude-sonnet-4-6")
	model.Required = true
	return []configuration.Field{
		runner.AgentMachineTypeField(),
		runner.AgentCredentialsField(runner.AgentCredentialsOptions{
			SecretLabel:      "OpenRouter API Key",
			IntegrationName:  "openrouter",
			IntegrationLabel: "Integration",
		}),
		model,
		runner.AgentStepsField(
			"Ordered bash commands and OpenRouter agent prompts. Add, reorder, and mix freely.",
			"Fix the failing tests and commit the changes.",
			"git clone https://github.com/org/repo.git /tmp/repo",
		),
		runner.AgentWorkingDirectoryField(),
		runner.EnvironmentFromConfigurationField(),
		runner.AgentEnvironmentField(envOpenRouterAPIKey),
		runner.AgentTimeoutField(),
		maxTurnsField(),
	}
}

func maxTurnsField() configuration.Field {
	return configuration.Field{
		Name:        "maxTurns",
		Label:       "Max turns per prompt",
		Type:        configuration.FieldTypeNumber,
		Required:    false,
		Default:     DefaultMaxTurns,
		Description: "Maximum model turns for each prompt step. Defaults to 128. After this limit, SuperPlane asks for a final reply without tools.",
		TypeOptions: &configuration.TypeOptions{
			Number: &configuration.NumberTypeOptions{
				Min: runner.IntPtr(0),
				Max: runner.IntPtr(MaxTurnsLimit),
			},
		},
	}
}

func (c *RunOpenRouter) Setup(ctx core.SetupContext) error {
	spec, err := decodeRunOpenRouterSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateRunOpenRouterSpec(spec); err != nil {
		return err
	}
	_, err = ctx.Webhook.Setup()
	return err
}

func (c *RunOpenRouter) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeRunOpenRouterSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateRunOpenRouterSpec(spec); err != nil {
		return err
	}

	resolved, err := runner.ResolveEnvironment(ctx.Secrets, spec.EnvironmentFrom, spec.Environment)
	if err != nil {
		return err
	}
	environment, err := injectOpenRouterCredentials(ctx, resolved.Variables, spec.Credentials)
	if err != nil {
		return err
	}

	webhookURL, err := ctx.Webhook.Setup()
	if err != nil {
		return fmt.Errorf("webhook setup: %w", err)
	}
	if err := runner.EnsureRunnerMinutesAvailable(ctx); err != nil {
		return err
	}

	broker, err := runner.NewBrokerClient(ctx.HTTP)
	if err != nil {
		return fmt.Errorf("new broker client: %w", err)
	}

	environment = runner.AttachPlanningSessionEnv(ctx, environment, spec.ExecutionTimeoutSeconds)

	task := buildOpenRouterBrokerTask(spec, resolved.Usage, resolved.Setups)
	task = applyPlanningFollowUp(task, environment, spec)
	if runner.HasPlanningSessionToken(environment) {
		task.Files = append(task.Files, runner.PlanningSessionMCPScriptFile())
	}
	taskID, err := broker.CreateTask(runner.CreateTaskParams{
		MachineType:    spec.MachineType,
		Commands:       task.Commands,
		Files:          task.Files,
		WebhookURL:     webhookURL,
		Environment:    environment,
		ExecutionMode:  runner.ExecutionModeHost,
		TimeoutSeconds: spec.ExecutionTimeoutSeconds,
		Labels:         runner.OriginLabelsForTask(ctx),
	})
	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}
	return runner.AfterRunnerTaskCreated(ctx, taskID)
}

func injectOpenRouterCredentials(ctx core.ExecutionContext, environment []runner.BrokerEnvironmentVariable, credentials runner.AgentCredentials) ([]runner.BrokerEnvironmentVariable, error) {
	switch credentials.Source {
	case runner.CredentialsSourceSecret:
		return runner.InjectSecretAPIKey(ctx, environment, envOpenRouterAPIKey, credentials.Secret)
	case runner.CredentialsSourceIntegration:
		return runner.InjectIntegrationKeys(ctx, environment, credentials.Integration)
	case runner.CredentialsSourceHosted:
		return nil, runner.RejectHostedCredentials(credentials)
	default:
		return nil, fmt.Errorf("invalid credentials source: %s", credentials.Source)
	}
}

func (c *RunOpenRouter) Hooks() []core.Hook {
	return []core.Hook{{Name: runner.HookPoll, Type: core.HookTypeInternal}}
}

func (c *RunOpenRouter) HandleHook(ctx core.ActionHookContext) error {
	if ctx.Name == runner.HookPoll {
		return runner.PollBrokerTask(ctx, FinishedEventType)
	}
	return fmt.Errorf("unknown hook: %s", ctx.Name)
}

func (c *RunOpenRouter) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return runner.HandleBrokerWebhook(ctx, FinishedEventType)
}

func (c *RunOpenRouter) Cancel(ctx core.ExecutionContext) error {
	return runner.CancelBrokerTask(ctx)
}

func (c *RunOpenRouter) Cleanup(ctx core.SetupContext) error { return nil }

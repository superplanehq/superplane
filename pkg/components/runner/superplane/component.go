package superplane

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/components/runner/claude"
	"github.com/superplanehq/superplane/pkg/components/runner/codex"
	"github.com/superplanehq/superplane/pkg/components/runner/openrouter"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	ComponentName     = models.SuperPlaneRunnerComponent
	FinishedEventType = "runnerSuperPlane.finished"
)

func init() {
	registry.RegisterAction(ComponentName, &RunSuperPlane{})
	runner.RegisterRunnerComponent(ComponentName)
}

type RunSuperPlane struct{}

func (c *RunSuperPlane) Name() string  { return ComponentName }
func (c *RunSuperPlane) Label() string { return "Run SuperPlane Agent" }
func (c *RunSuperPlane) Icon() string  { return "code" }
func (c *RunSuperPlane) Color() string { return "#1D4ED8" }

func (c *RunSuperPlane) ExampleOutput() map[string]any {
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

func (c *RunSuperPlane) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: runner.PassedOutputChannel, Label: "Passed"},
		{Name: runner.FailedOutputChannel, Label: "Failed"},
	}
}

func (c *RunSuperPlane) Description() string {
	return "Runs a SuperPlane-hosted coding agent on a fleet runner. This agent uses the SuperPlane-hosted model from instance settings."
}

func (c *RunSuperPlane) Documentation() string {
	return `Runs a SuperPlane-hosted coding agent on a fleet runner.

This agent uses the SuperPlane-hosted model from instance settings. It does not store a provider, model, or API key on the node.

## Prerequisites
- The organization has hosted credit.
- An installation admin has set a SuperPlane agent model.

## Steps
Configure an ordered list of **bash** and **prompt** steps:

- **bash** — shell commands (clone a repo, install deps, run tests, push).
- **prompt** — an agent turn. Later prompts continue the same session.

## Configuration
- **Machine type**: Runner fleet registered on the task-broker (required).
- **Steps**: Ordered bash/prompt actions (at least one prompt required).
- **Working directory**: Optional starting directory.
- **Execution timeout**: Optional wall-clock limit in seconds (1–86400). Defaults to **3600** (1 hour).

## Output
Prompt steps stream agent activity to **View logs**. The finished event includes the latest agent result.

## Output channels
- **Passed**: All steps finished with exit code **0**.
- **Failed**: A bash or prompt step failed (non-zero exit).
`
}

func (c *RunSuperPlane) Configuration() []configuration.Field {
	machineType := runner.AgentMachineTypeField()
	machineType.Description = "This agent uses the SuperPlane-hosted model from instance settings."
	return []configuration.Field{
		machineType,
		runner.AgentStepsField(
			"Ordered bash commands and SuperPlane agent prompts. Add, reorder, and mix freely.",
			"Fix the failing tests and commit the changes.",
			"git clone https://github.com/org/repo.git /tmp/repo",
		),
		runner.AgentWorkingDirectoryField(),
		runner.EnvironmentFromConfigurationField(),
		runner.AgentEnvironmentField(""),
		runner.AgentTimeoutField(),
	}
}

func (c *RunSuperPlane) ValidateNodeConfiguration(config map[string]any) error {
	spec, err := decodeRunSuperPlaneSpec(config)
	if err != nil {
		return err
	}
	return validateRunSuperPlaneSpec(spec)
}

func (c *RunSuperPlane) Setup(ctx core.SetupContext) error {
	spec, err := decodeRunSuperPlaneSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateRunSuperPlaneSpec(spec); err != nil {
		return err
	}
	_, err = ctx.Webhook.Setup()
	return err
}

func (c *RunSuperPlane) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeRunSuperPlaneSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateRunSuperPlaneSpec(spec); err != nil {
		return err
	}

	defaultModel, err := resolveSuperPlaneDefaultModel(ctx)
	if err != nil {
		return err
	}

	access, err := runner.PrepareHostedRun(ctx, defaultModel.Provider, defaultModel.Model)
	if err != nil {
		return err
	}

	recordSuperPlaneRunOnConfiguration(ctx.Configuration, defaultModel)

	resolved, err := runner.ResolveEnvironment(ctx.Secrets, spec.EnvironmentFrom, spec.Environment)
	if err != nil {
		return err
	}

	environment, err := injectSuperPlaneCredentials(resolved.Variables, defaultModel.Provider, access)
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
	commands, files, err := buildSuperPlaneBrokerTask(defaultModel.Provider, spec, defaultModel.Model, resolved.Usage, resolved.Setups, environment)
	if err != nil {
		return err
	}

	taskID, err := broker.CreateTask(runner.CreateTaskParams{
		MachineType:    spec.MachineType,
		Commands:       commands,
		Files:          files,
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

func (c *RunSuperPlane) Hooks() []core.Hook {
	return []core.Hook{{Name: runner.HookPoll, Type: core.HookTypeInternal}}
}

func (c *RunSuperPlane) HandleHook(ctx core.ActionHookContext) error {
	if ctx.Name == runner.HookPoll {
		return runner.PollBrokerTask(ctx, FinishedEventType)
	}
	return fmt.Errorf("unknown hook: %s", ctx.Name)
}

func (c *RunSuperPlane) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return runner.HandleBrokerWebhook(ctx, FinishedEventType)
}

func (c *RunSuperPlane) Cancel(ctx core.ExecutionContext) error {
	return runner.CancelBrokerTask(ctx)
}

func (c *RunSuperPlane) Cleanup(ctx core.SetupContext) error { return nil }

func resolveSuperPlaneDefaultModel(ctx core.ExecutionContext) (core.DefaultHostedLLMModel, error) {
	if ctx.HostedLLM == nil {
		return core.DefaultHostedLLMModel{}, fmt.Errorf("hosted credentials are not available")
	}
	defaultModel, err := ctx.HostedLLM.DefaultModel()
	if err != nil {
		return core.DefaultHostedLLMModel{}, err
	}
	if !defaultModel.IsSet() {
		return core.DefaultHostedLLMModel{}, models.ErrSuperPlaneRunnerNoModel
	}
	return defaultModel, nil
}

func recordSuperPlaneRunOnConfiguration(configuration any, defaultModel core.DefaultHostedLLMModel) {
	cfg, ok := configuration.(map[string]any)
	if !ok {
		return
	}
	cfg["hostedProvider"] = defaultModel.Provider
	cfg["model"] = defaultModel.Model
	cfg["credentials"] = map[string]any{"source": runner.CredentialsSourceHosted}
}

func injectSuperPlaneCredentials(
	environment []runner.BrokerEnvironmentVariable,
	provider string,
	access core.HostedLLMAccess,
) ([]runner.BrokerEnvironmentVariable, error) {
	switch provider {
	case models.UsageProviderAnthropic:
		return runner.InjectHostedCredentials(environment, envAnthropicAPIKey, access.APIKey, envAnthropicBaseURL, access.BaseURL), nil
	case models.UsageProviderOpenAI:
		return runner.InjectHostedCredentials(environment, envOpenAIAPIKey, access.APIKey, envOpenAIBaseURL, access.BaseURL), nil
	case models.UsageProviderOpenRouter:
		return runner.InjectHostedCredentials(environment, envOpenRouterAPIKey, access.APIKey, envOpenRouterBaseURL, access.BaseURL), nil
	default:
		return nil, fmt.Errorf("unsupported SuperPlane agent provider: %s", provider)
	}
}

func buildSuperPlaneBrokerTask(
	provider string,
	spec RunSuperPlaneSpec,
	model string,
	usage string,
	setups []runner.IntegrationSetup,
	environment []runner.BrokerEnvironmentVariable,
) ([]runner.BrokerCommand, []runner.BrokerTaskFile, error) {
	switch provider {
	case models.UsageProviderAnthropic:
		claudeSpec := claude.RunClaudeCodeSpec{
			MachineType:             spec.MachineType,
			Steps:                   spec.Steps,
			Model:                   model,
			WorkingDirectory:        spec.WorkingDirectory,
			ExecutionTimeoutSeconds: spec.ExecutionTimeoutSeconds,
		}
		task := claude.BuildBrokerTask(claudeSpec, usage, setups)
		task = claude.ApplyPlanningFollowUp(task, environment, claudeSpec)
		if runner.HasPlanningSessionToken(environment) {
			task.Files = append(task.Files, runner.PlanningSessionMCPFiles()...)
		}
		return task.Commands, task.Files, nil
	case models.UsageProviderOpenAI:
		codexSpec := codex.RunCodexSpec{
			MachineType:             spec.MachineType,
			Steps:                   spec.Steps,
			Model:                   model,
			WorkingDirectory:        spec.WorkingDirectory,
			ExecutionTimeoutSeconds: spec.ExecutionTimeoutSeconds,
		}
		task := codex.BuildBrokerTask(codexSpec, usage, setups)
		task = codex.ApplyPlanningFollowUp(task, environment, codexSpec)
		if runner.HasPlanningSessionToken(environment) {
			task.Files = append(task.Files, runner.PlanningSessionMCPFiles()...)
		}
		return task.Commands, task.Files, nil
	case models.UsageProviderOpenRouter:
		openRouterSpec := openrouter.RunOpenRouterSpec{
			MachineType:             spec.MachineType,
			Steps:                   spec.Steps,
			Model:                   model,
			WorkingDirectory:        spec.WorkingDirectory,
			ExecutionTimeoutSeconds: spec.ExecutionTimeoutSeconds,
		}
		task := openrouter.BuildBrokerTask(openRouterSpec, usage, setups)
		task = openrouter.ApplyPlanningFollowUp(task, environment, openRouterSpec)
		if runner.HasPlanningSessionToken(environment) {
			task.Files = append(task.Files, runner.PlanningSessionMCPScriptFile())
		}
		return task.Commands, task.Files, nil
	default:
		return nil, nil, fmt.Errorf("unsupported SuperPlane agent provider: %s", provider)
	}
}

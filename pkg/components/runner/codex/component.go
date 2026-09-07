package codex

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	ComponentName     = "runnerCodex"
	FinishedEventType = "runnerCodex.finished"
	envOpenAIAPIKey   = "OPENAI_API_KEY"
	envOpenAIBaseURL  = "OPENAI_BASE_URL"
)

func init() {
	registry.RegisterAction(ComponentName, &RunCodex{})
	runner.RegisterRunnerComponent(ComponentName)
}

type RunCodex struct{}

func (c *RunCodex) Name() string  { return ComponentName }
func (c *RunCodex) Label() string { return "Run Codex" }
func (c *RunCodex) Icon() string  { return "code" }
func (c *RunCodex) Color() string { return "#10A37F" }

func (c *RunCodex) ExampleOutput() map[string]any {
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

func (c *RunCodex) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: runner.PassedOutputChannel, Label: "Passed"},
		{Name: runner.FailedOutputChannel, Label: "Failed"},
	}
}

func (c *RunCodex) Description() string {
	return "Runs the Codex CLI on a fleet runner"
}

func (c *RunCodex) Documentation() string {
	return `Runs the OpenAI Codex CLI in non-interactive mode on a fleet runner.

## Prerequisites
- The ` + "`codex`" + ` CLI is installed on the runner machine and available on ` + "`PATH`" + `.
- An OpenAI API key stored as a SuperPlane secret or an OpenAI integration.

## Steps
Configure an ordered list of **bash** and **prompt** steps:

- **bash** — shell commands (clone a repo, install deps, run tests, push).
- **prompt** — a Codex turn. Later prompts continue in the same working directory.

## Configuration
- **Machine type**: Runner fleet registered on the task-broker (required).
- **Steps**: Ordered bash/prompt actions (at least one prompt required).
- **Credentials**: SuperPlane secret or OpenAI integration used as ` + "`OPENAI_API_KEY`" + `.
- **Model**: Select a model from Organization LLM Models.
- **Working directory**: Optional starting directory.
- **Execution timeout**: Optional wall-clock limit in seconds (1–86400). Defaults to **3600** (1 hour).

Use **Run SuperPlane Agent** for SuperPlane-hosted credentials.

## Output channels
- **Passed**: All steps finished with exit code **0**.
- **Failed**: A bash or prompt step failed (non-zero exit).
`
}

func (c *RunCodex) Configuration() []configuration.Field {
	return []configuration.Field{
		runner.AgentMachineTypeField(),
		runner.AgentCredentialsField(runner.AgentCredentialsOptions{
			SecretLabel:      "OpenAI API Key",
			IntegrationName:  "openai",
			IntegrationLabel: "Integration",
		}),
		runner.AgentModelField("openai"),
		runner.AgentStepsField(
			"Ordered bash commands and Codex prompts. Add, reorder, and mix freely.",
			"Fix the failing tests and commit the changes.",
			"git clone https://github.com/org/repo.git /tmp/repo",
		),
		runner.AgentWorkingDirectoryField(),
		runner.EnvironmentFromConfigurationField(),
		runner.AgentEnvironmentField(envOpenAIAPIKey),
		runner.AgentTimeoutField(),
	}
}

func (c *RunCodex) Setup(ctx core.SetupContext) error {
	spec, err := decodeRunCodexSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateRunCodexSpec(spec); err != nil {
		return err
	}
	_, err = ctx.Webhook.Setup()
	return err
}

func (c *RunCodex) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeRunCodexSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateRunCodexSpec(spec); err != nil {
		return err
	}

	resolved, err := runner.ResolveEnvironment(ctx.Secrets, spec.EnvironmentFrom, spec.Environment)
	if err != nil {
		return err
	}
	environment, err := injectCodexCredentials(ctx, resolved.Variables, spec.Credentials)
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

	task := buildCodexBrokerTask(spec, resolved.Usage, resolved.Setups)
	task = applyPlanningFollowUp(task, environment, spec)
	if runner.HasPlanningSessionToken(environment) {
		task.Files = append(task.Files, runner.PlanningSessionMCPFiles()...)
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

func injectCodexCredentials(ctx core.ExecutionContext, environment []runner.BrokerEnvironmentVariable, credentials runner.AgentCredentials) ([]runner.BrokerEnvironmentVariable, error) {
	switch credentials.Source {
	case runner.CredentialsSourceSecret:
		return runner.InjectSecretAPIKey(ctx, environment, envOpenAIAPIKey, credentials.Secret)
	case runner.CredentialsSourceIntegration:
		return runner.InjectIntegrationKeys(ctx, environment, credentials.Integration)
	case runner.CredentialsSourceHosted:
		return nil, runner.RejectHostedCredentials(credentials)
	default:
		return nil, fmt.Errorf("invalid credentials source: %s", credentials.Source)
	}
}

func (c *RunCodex) Hooks() []core.Hook {
	return []core.Hook{{Name: runner.HookPoll, Type: core.HookTypeInternal}}
}

func (c *RunCodex) HandleHook(ctx core.ActionHookContext) error {
	if ctx.Name == runner.HookPoll {
		return runner.PollBrokerTask(ctx, FinishedEventType)
	}
	return fmt.Errorf("unknown hook: %s", ctx.Name)
}

func (c *RunCodex) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return runner.HandleBrokerWebhook(ctx, FinishedEventType)
}

func (c *RunCodex) Cancel(ctx core.ExecutionContext) error {
	return runner.CancelBrokerTask(ctx)
}

func (c *RunCodex) Cleanup(ctx core.SetupContext) error { return nil }

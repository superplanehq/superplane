package claude

import (
	"fmt"

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
- An Anthropic API key stored as a SuperPlane secret, a Claude integration, or SuperPlane-hosted credentials.

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
- **Anthropic API Key**: SuperPlane secret, Claude integration, or SuperPlane-hosted credentials used as ` + "`ANTHROPIC_API_KEY`" + `.
- **Model**: Optional model id or alias (for example ` + "`sonnet`" + `). SuperPlane-hosted credentials require a model from the installation allowlist.
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
		runner.AgentMachineTypeField(),
		runner.AgentCredentialsField(runner.AgentCredentialsOptions{
			SecretLabel:       "Anthropic API Key",
			IntegrationName:   "claude",
			IntegrationLabel:  "Integration",
			AllowHosted:       true,
			HostedDescription: "Anthropic API key, Claude integration, or SuperPlane-hosted credentials.",
		}),
		runner.AgentModelField("anthropic", "Claude model id. SuperPlane-hosted credentials use the installation allowlist.", "sonnet"),
		runner.AgentStepsField(
			"Ordered bash commands and Claude Code prompts. Add, reorder, and mix freely.",
			"Fix the failing tests and commit the changes.",
			"git clone https://github.com/org/repo.git /tmp/repo",
		),
		runner.AgentWorkingDirectoryField(),
		runner.EnvironmentFromConfigurationField(),
		runner.AgentEnvironmentField(envAnthropicAPIKey),
		runner.AgentTimeoutField(),
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

func (c *RunClaudeCode) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeRunClaudeCodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	if err := validateRunClaudeCodeSpec(spec); err != nil {
		return err
	}

	resolved, err := runner.ResolveEnvironment(ctx.Secrets, spec.EnvironmentFrom, spec.Environment)
	if err != nil {
		return err
	}

	environment, err := c.injectCredentials(ctx, resolved.Variables, spec.Credentials, spec.Model)
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

	// command_list tasks only accept commands (+ optional files).
	task := buildClaudeCodeBrokerTask(spec, resolved.Usage, resolved.Setups)
	params := runner.CreateTaskParams{
		MachineType:    spec.MachineType,
		Commands:       task.Commands,
		Files:          task.Files,
		WebhookURL:     webhookURL,
		Environment:    environment,
		ExecutionMode:  runner.ExecutionModeHost,
		TimeoutSeconds: spec.ExecutionTimeoutSeconds,
		Labels:         runner.OriginLabelsForTask(ctx),
	}

	taskID, err := broker.CreateTask(params)
	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}

	return runner.AfterRunnerTaskCreated(ctx, taskID)
}

func (c *RunClaudeCode) injectCredentials(ctx core.ExecutionContext, environment []runner.BrokerEnvironmentVariable, credentials runner.AgentCredentials, model string) ([]runner.BrokerEnvironmentVariable, error) {
	switch credentials.Source {
	case runner.CredentialsSourceSecret:
		return runner.InjectSecretAPIKey(ctx, environment, envAnthropicAPIKey, credentials.Secret)
	case runner.CredentialsSourceIntegration:
		return runner.InjectIntegrationKeys(ctx, environment, credentials.Integration)
	case runner.CredentialsSourceHosted:
		access, err := runner.PrepareHostedRun(ctx, "anthropic", model)
		if err != nil {
			return nil, err
		}
		return runner.InjectHostedCredentials(environment, envAnthropicAPIKey, access.APIKey, envAnthropicBaseURL, access.BaseURL), nil
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

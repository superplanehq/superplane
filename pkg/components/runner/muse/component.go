package muse

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	ComponentName      = "runnerMuse"
	FinishedEventType  = "runnerMuse.finished"
	envMetaAPIKey      = "META_API_KEY"
	envCustomLLMAPIKey = "CUSTOM_LLM_API_KEY"
)

func init() {
	registry.RegisterAction(ComponentName, &RunMuse{})
	runner.RegisterRunnerComponent(ComponentName)
}

type RunMuse struct{}

func (c *RunMuse) Name() string  { return ComponentName }
func (c *RunMuse) Label() string { return "Run Muse Code" }
func (c *RunMuse) Icon() string  { return "code" }
func (c *RunMuse) Color() string { return "#0866FF" }

func (c *RunMuse) ExampleOutput() map[string]any {
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

func (c *RunMuse) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: runner.PassedOutputChannel, Label: "Passed"},
		{Name: runner.FailedOutputChannel, Label: "Failed"},
	}
}

func (c *RunMuse) Description() string {
	return "Runs the Meta Muse Code CLI on a fleet runner"
}

func (c *RunMuse) Documentation() string {
	return `Runs the Meta Muse Code CLI in non-interactive mode on a fleet runner.

## Prerequisites
- The ` + "`muse`" + ` CLI is installed on the runner machine and available on ` + "`PATH`" + `.
- A Meta API key stored as a SuperPlane secret, or a Custom provider integration configured for the Meta Model API.

Browser sign-in is not copied to remote runners. Meta recommends an API key for non-interactive use.

## Steps
Configure an ordered list of **bash** and **prompt** steps:

- **bash** — shell commands (clone a repo, install dependencies, run tests, push).
- **prompt** — a Muse Code turn. Later prompts continue the same Muse session.

## Configuration
- **Machine type**: Runner fleet registered on the task-broker (required).
- **Steps**: Ordered bash/prompt actions (at least one prompt required).
- **Credentials**: SuperPlane secret used as ` + "`META_API_KEY`" + `, or a Custom provider integration. Set the integration base URL to ` + "`https://api.meta.ai/v1`" + `.
- **Model**: Select a model from Organization LLM Models. Muse uses its default when omitted.
- **Working directory**: Optional starting directory.
- **Execution timeout**: Optional wall-clock limit in seconds (1–86400). Defaults to **3600** (1 hour).

## Output channels
- **Passed**: All steps finished with exit code **0**.
- **Failed**: A bash or prompt step failed (non-zero exit).
`
}

func (c *RunMuse) Configuration() []configuration.Field {
	return []configuration.Field{
		runner.AgentMachineTypeField(),
		runner.AgentCredentialsField(runner.AgentCredentialsOptions{
			SecretLabel:      "Meta API Key",
			IntegrationName:  "customLlm",
			IntegrationLabel: "Custom provider",
		}),
		runner.AgentModelField("custom"),
		runner.AgentStepsField(
			"Ordered bash commands and Muse Code prompts. Add, reorder, and mix freely.",
			"Fix the failing tests and commit the changes.",
			"git clone https://github.com/org/repo.git /tmp/repo",
		),
		runner.AgentWorkingDirectoryField(),
		runner.EnvironmentFromConfigurationField(),
		runner.AgentEnvironmentField(envMetaAPIKey),
		runner.AgentTimeoutField(),
	}
}

func (c *RunMuse) Setup(ctx core.SetupContext) error {
	spec, err := decodeRunMuseSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateRunMuseSpec(spec); err != nil {
		return err
	}
	_, err = ctx.Webhook.Setup()
	return err
}

func (c *RunMuse) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeRunMuseSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateRunMuseSpec(spec); err != nil {
		return err
	}

	resolved, err := runner.ResolveEnvironment(ctx.Secrets, spec.EnvironmentFrom, spec.Environment)
	if err != nil {
		return err
	}
	environment, err := injectMuseCredentials(ctx, resolved.Variables, spec.Credentials)
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

	environment = runner.AttachPlanningSessionEnv(ctx, environment, spec.ExecutionTimeoutSeconds)
	environment = runner.AttachArtifactUploadEnv(ctx, environment, spec.ExecutionTimeoutSeconds, spec.IncludeVisualEvidence)
	dispatched, err := runner.MintDispatchForRun(ctx, spec.ExecutionTimeoutSeconds, spec.Steps)
	if err != nil {
		return err
	}
	dispatched.Steps = runner.AppendVisualEvidenceProtocol(dispatched.Steps, runner.HasArtifactUploadToken(environment))
	dispatched.Steps = runner.AppendFactoryImaginedLimitPrompt(ctx, dispatched.Steps)
	task := buildMuseBrokerTask(spec, resolved.Usage, resolved.Setups, dispatched.Steps, dispatched.Attachments, runner.HasPlanningSessionToken(environment))
	task = applyPlanningFollowUp(task, environment, spec)
	task.Files = runner.AppendTaskArtifactMCP(environment, task.Files)
	task.Files = runner.AppendPlanningSessionContinuation(ctx, environment, task.Files)
	environment, task.Files = runner.AttachWorkspaceAgentResources(ctx, environment, task.Files)
	environment, task.Files = runner.AttachFactoryCommitIdentity(ctx, environment, task.Files)
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

func injectMuseCredentials(ctx core.ExecutionContext, environment []runner.BrokerEnvironmentVariable, credentials runner.AgentCredentials) ([]runner.BrokerEnvironmentVariable, error) {
	switch credentials.Source {
	case runner.CredentialsSourceSecret:
		return runner.InjectSecretAPIKey(ctx, environment, envMetaAPIKey, credentials.Secret)
	case runner.CredentialsSourceIntegration:
		return runner.InjectIntegrationKeys(ctx, environment, credentials.Integration)
	case runner.CredentialsSourceHosted:
		return nil, runner.RejectHostedCredentials(credentials)
	default:
		return nil, fmt.Errorf("invalid credentials source: %s", credentials.Source)
	}
}

func (c *RunMuse) Hooks() []core.Hook {
	return []core.Hook{{Name: runner.HookPoll, Type: core.HookTypeInternal}}
}

func (c *RunMuse) HandleHook(ctx core.ActionHookContext) error {
	if ctx.Name == runner.HookPoll {
		return runner.PollBrokerTask(ctx, FinishedEventType)
	}
	return fmt.Errorf("unknown hook: %s", ctx.Name)
}

func (c *RunMuse) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return runner.HandleBrokerWebhook(ctx, FinishedEventType)
}

func (c *RunMuse) Cancel(ctx core.ExecutionContext) error {
	return runner.CancelBrokerTask(ctx, FinishedEventType)
}

func (c *RunMuse) Cleanup(ctx core.SetupContext) error { return nil }

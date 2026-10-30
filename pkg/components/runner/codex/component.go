package codex

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/jwt"
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
- **Machine type**: Runner fleet available to the organization (required).
- **Steps**: One or more ordered bash commands or agent prompts.
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
		runner.MachineTypeField("machineType"),
		runner.AgentCredentialsField(runner.AgentCredentialsOptions{
			SecretLabel:       "OpenAI API Key",
			IntegrationName:   "openai",
			IntegrationLabel:  "Integration",
			AllowSubscription: true,
		}),
		runner.AgentModelField("openai"),
		runner.AgentStepsField(
			"Ordered bash commands and Codex prompts. Add, reorder, and mix freely.",
			"Fix the failing tests and commit the changes.",
			"git clone https://github.com/org/repo.git /tmp/repo",
		),
		runner.AgentWorkingDirectoryField(),
		runner.AgentAdditionalInstructionsField(),
		runner.EnvironmentFromConfigurationField(),
		runner.AgentEnvironmentField(envOpenAIAPIKey),
		runner.AgentSkipSetupCommandsField(),
		runner.AgentTimeoutField(),
	}
}

func (c *RunCodex) ValidateNodeConfiguration(config map[string]any) error {
	spec, err := decodeRunCodexSpec(config)
	if err != nil {
		return err
	}
	return validateRunCodexSpec(spec)
}

func (c *RunCodex) Setup(ctx core.SetupContext) error {
	spec, err := decodeRunCodexSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	return validateRunCodexSpec(spec)
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
	taskID := uuid.New()

	environment := runner.AttachPlanningSessionEnv(ctx, resolved.Variables, spec.ExecutionTimeoutSeconds)
	environment = runner.AttachArtifactUploadEnv(ctx, environment, spec.ExecutionTimeoutSeconds, spec.IncludeVisualEvidence)
	environment, err = runner.AttachMergeConfidenceEnv(ctx, environment, spec.Steps, spec.ExecutionTimeoutSeconds)
	if err != nil {
		return err
	}
	environment, err = injectCodexCredentials(ctx, environment, spec.Credentials, taskID, spec.ExecutionTimeoutSeconds, spec.Model)
	if err != nil {
		return err
	}

	dispatched, err := runner.MintDispatchForRun(ctx, spec.ExecutionTimeoutSeconds, spec.Steps)
	if err != nil {
		return err
	}
	dispatched.Steps = runner.AppendVisualEvidenceProtocol(dispatched.Steps, runner.HasVisualEvidence(environment))
	dispatched.Steps = runner.AppendPublishLinkProtocol(dispatched.Steps, runner.HasPublishLinks(environment))
	dispatched.Steps = runner.AppendFactoryImaginedLimitPrompt(ctx, dispatched.Steps)
	environment = runner.AttachRunnerEnv(ctx, environment, spec.ExecutionTimeoutSeconds)
	task := buildCodexTask(spec, resolved.Usage, resolved.Setups, dispatched.Steps, dispatched.Attachments, runner.HasPlanningSessionToken(environment))
	task = applyPlanningFollowUp(task, environment, spec)
	if runner.HasPlanningSessionToken(environment) {
		task.Files = runner.AppendPlanningSessionMCPFiles(task.Files)
	}
	task.Files = runner.AppendTaskArtifactMCP(environment, task.Files)
	task.Files = runner.AppendMergeConfidenceMCP(environment, task.Files)
	task.Files = runner.AppendGlobalMCP(environment, task.Files)
	task.Files = runner.AppendPlanningSessionContinuation(ctx, environment, task.Files)
	environment, task.Files = runner.AttachWorkspaceAgentResources(ctx, environment, task.Files)
	task.Commands = runner.ApplyFactoryCheckout(task.Files, task.Commands, spec.Steps)
	environment, task.Files = runner.AttachFactoryCommitIdentity(ctx, environment, task.Files)
	createdTaskID, err := runner.CreateTask(ctx.RunnerTasks, runner.CreateTaskParams{
		TaskID:         taskID.String(),
		MachineType:    spec.MachineType,
		Commands:       task.Commands,
		Files:          task.Files,
		Environment:    environment,
		ExecutionMode:  runner.ExecutionModeHost,
		TimeoutSeconds: spec.ExecutionTimeoutSeconds,
	})
	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}
	return runner.AfterRunnerTaskCreated(ctx, createdTaskID)
}

func injectCodexCredentials(ctx core.ExecutionContext, environment []runner.TaskEnvironmentVariable, credentials runner.AgentCredentials, taskID uuid.UUID, timeoutSeconds int, model string) ([]runner.TaskEnvironmentVariable, error) {
	switch credentials.Source {
	case runner.CredentialsSourceSecret:
		return runner.InjectSecretAPIKey(ctx, environment, envOpenAIAPIKey, credentials.Secret)
	case runner.CredentialsSourceIntegration:
		return runner.InjectIntegrationKeys(ctx, environment, credentials.Integration)
	case runner.CredentialsSourceHosted:
		return nil, runner.RejectHostedCredentials(credentials)
	case runner.CredentialsSourceSubscription:
		return injectSubscriptionCredentials(ctx, environment, taskID, timeoutSeconds, model)
	default:
		return nil, fmt.Errorf("invalid credentials source: %s", credentials.Source)
	}
}

func injectSubscriptionCredentials(ctx core.ExecutionContext, environment []runner.TaskEnvironmentVariable, taskID uuid.UUID, timeoutSeconds int, model string) ([]runner.TaskEnvironmentVariable, error) {
	if ctx.CodexSubscription == nil {
		return nil, fmt.Errorf("connect ChatGPT before this task can run")
	}
	lease, err := ctx.CodexSubscription.Reserve(taskID, model)
	if err != nil {
		return nil, err
	}
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if secret == "" {
		return nil, fmt.Errorf("chatgpt subscription token is not configured")
	}
	orgID, err := uuid.Parse(strings.TrimSpace(ctx.OrganizationID))
	if err != nil {
		return nil, fmt.Errorf("chatgpt subscription organization is missing")
	}
	token, err := runner.MintCodexSubscriptionToken(jwt.NewSigner(secret), runner.CodexSubscriptionScope{
		OrganizationID: orgID,
		FactoryID:      lease.FactoryID,
		ConnectionID:   lease.ConnectionID,
		TaskID:         taskID,
		Generation:     lease.Generation,
	}, time.Duration(timeoutSeconds)*time.Second)
	if err != nil {
		return nil, err
	}
	baseURL := runner.RunnerSuperplaneBaseURL(ctx.BaseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("chatgpt subscription endpoint is not configured")
	}
	environment = withoutCodexCredentials(environment)
	return append(environment,
		runner.TaskEnvironmentVariable{Name: runner.EnvCodexSubscriptionToken, Value: token},
		runner.TaskEnvironmentVariable{Name: runner.EnvSuperplaneBaseURL, Value: baseURL},
		runner.TaskEnvironmentVariable{Name: runner.EnvCodexRefreshTokenURL, Value: baseURL + runner.CodexSubscriptionRefreshPath},
	), nil
}

func withoutCodexCredentials(environment []runner.TaskEnvironmentVariable) []runner.TaskEnvironmentVariable {
	filtered := make([]runner.TaskEnvironmentVariable, 0, len(environment))
	for _, item := range environment {
		if item.Name == "CODEX_API_KEY" || item.Name == envOpenAIAPIKey || item.Name == envOpenAIBaseURL || item.Name == runner.EnvSuperplaneBaseURL || item.Name == runner.EnvCodexSubscriptionToken || item.Name == runner.EnvCodexRefreshTokenURL {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func (c *RunCodex) Hooks() []core.Hook {
	return []core.Hook{{Name: runner.HookPoll, Type: core.HookTypeInternal}}
}

func (c *RunCodex) HandleHook(ctx core.ActionHookContext) error {
	if ctx.Name == runner.HookPoll {
		return runner.PollTask(ctx, FinishedEventType)
	}
	return fmt.Errorf("unknown hook: %s", ctx.Name)
}

func (c *RunCodex) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusNotFound, nil, nil
}

func (c *RunCodex) Cancel(ctx core.ExecutionContext) error {
	return runner.CancelTask(ctx, FinishedEventType)
}

func (c *RunCodex) Cleanup(ctx core.SetupContext) error { return nil }

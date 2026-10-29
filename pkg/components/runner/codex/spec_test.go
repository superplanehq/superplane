package codex

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

func TestDecodeRunCodexSpecAllowsBashOnly(t *testing.T) {
	t.Parallel()

	spec, err := decodeRunCodexSpec(map[string]any{
		"machineType":           "e1-large-amd64",
		"includeVisualEvidence": true,
		"steps": []map[string]any{
			{"name": "Clone", "type": "bash", "command": "git clone"},
		},
		"credentials": map[string]any{
			"source": "secret",
			"secret": map[string]any{"secret": "openai", "key": "api_key"},
		},
	})
	require.NoError(t, err)
	assert.True(t, spec.IncludeVisualEvidence)
	require.NoError(t, validateRunCodexSpec(spec))
}

func TestValidateRunCodexSpecRejectsHostedCredentials(t *testing.T) {
	t.Parallel()

	prompt := "fix tests"
	spec := RunCodexSpec{
		MachineType: "e1-large-amd64",
		Steps: []runner.AgentStep{
			{Name: "Prompt", Type: runner.AgentStepPrompt, Prompt: &prompt},
		},
		Credentials: runner.AgentCredentials{Source: runner.CredentialsSourceHosted},
		Model:       "gpt-5",
	}
	err := validateRunCodexSpec(spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Run SuperPlane Agent")
}

func TestValidateCodexConfigurationAcceptsSubscription(t *testing.T) {
	t.Parallel()
	err := configuration.ValidateConfiguration((&RunCodex{}).Configuration(), map[string]any{
		"machineType":   "e1-large-amd64",
		"credentials":   map[string]any{"source": runner.CredentialsSourceSubscription},
		"model":         "gpt-5.4",
		"thinkingLevel": "xhigh",
		"steps": []any{
			map[string]any{"name": "Plan task", "type": runner.AgentStepPrompt, "prompt": "Plan this task"},
		},
	})
	require.NoError(t, err)
}

func TestInjectSubscriptionCredentialsPreservesPlanningEnvironment(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("RUNNER_API_BASE_URL", "http://host.docker.internal:8000")
	t.Setenv("BASE_URL", "http://localhost:8000")
	environment, err := injectSubscriptionCredentials(core.ExecutionContext{
		OrganizationID:    uuid.NewString(),
		CodexSubscription: subscriptionLease{},
	}, []runner.TaskEnvironmentVariable{
		{Name: runner.EnvSuperplaneBaseURL, Value: "http://host.docker.internal:8000"},
		{Name: runner.EnvSuperplanePlanningID, Value: "planning-session"},
		{Name: runner.EnvSuperplaneRunToken, Value: "planning-token"},
		{Name: envOpenAIAPIKey, Value: "api-key"},
		{Name: envOpenAIBaseURL, Value: "https://api.example.com"},
	}, uuid.New(), 60, "gpt-5.4")
	require.NoError(t, err)

	values := map[string]string{}
	for _, item := range environment {
		require.NotContains(t, values, item.Name, "duplicate environment variable")
		values[item.Name] = item.Value
	}
	assert.Equal(t, "planning-session", values[runner.EnvSuperplanePlanningID])
	assert.Equal(t, "planning-token", values[runner.EnvSuperplaneRunToken])
	assert.Equal(t, "http://host.docker.internal:8000", values[runner.EnvSuperplaneBaseURL])
	assert.NotEmpty(t, values[runner.EnvCodexSubscriptionToken])
	assert.Equal(t, values[runner.EnvSuperplaneBaseURL]+runner.CodexSubscriptionRefreshPath, values[runner.EnvCodexRefreshTokenURL])
	assert.NotContains(t, values, envOpenAIAPIKey)
	assert.NotContains(t, values, envOpenAIBaseURL)
}

type subscriptionLease struct{}

func (subscriptionLease) Reserve(uuid.UUID, string) (core.CodexSubscriptionLease, error) {
	return core.CodexSubscriptionLease{ConnectionID: uuid.New(), FactoryID: uuid.New(), Generation: 1}, nil
}

func TestValidateRunCodexSpecRejectsReservedEnv(t *testing.T) {
	t.Parallel()

	prompt := "fix tests"
	value := "x"
	spec := RunCodexSpec{
		MachineType: "e1-large-amd64",
		Steps: []runner.AgentStep{
			{Name: "Prompt", Type: runner.AgentStepPrompt, Prompt: &prompt},
		},
		Credentials: runner.AgentCredentials{
			Source: runner.CredentialsSourceSecret,
			Secret: configuration.SecretKeyRef{Secret: "openai", Key: "api_key"},
		},
		Environment: []runner.EnvironmentVariable{
			{Name: envOpenAIAPIKey, ValueSource: runner.EnvironmentValueSourceLiteral, Value: &value},
		},
	}
	err := validateRunCodexSpec(spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), envOpenAIAPIKey)
}

func strPtr(v string) *string { return &v }

func TestBuildCodexTaskRunsOrderedSteps(t *testing.T) {
	t.Parallel()

	spec := RunCodexSpec{
		Model:            "gpt-5",
		WorkingDirectory: "/tmp/workspace",
		Steps: []runner.AgentStep{
			{Name: "Clone repo", Type: runner.AgentStepBash, Command: strPtr("git clone https://github.com/acme/widgets.git repo")},
			{Name: "Fix panic", Type: runner.AgentStepPrompt, Prompt: strPtr("Fix auth.py's nil panic")},
		},
	}

	task := buildCodexTask(spec, "", nil, nil, nil, false)
	require.Len(t, task.Commands, 3)
	assert.Equal(t, "Prepare Codex", task.Commands[0].Name)
	assert.Equal(t, "Clone repo", task.Commands[1].Name)
	assert.Equal(t, "Fix panic", task.Commands[2].Name)
	assert.Contains(t, task.Commands[2].Command, `node "$SUPERPLANE_TASK_DIR/run.js" "$SUPERPLANE_TASK_DIR/prompts/02-fix-panic.txt" 'gpt-5'`)
	assert.Equal(t, runScript, requireTaskFile(t, task.Files, "run.js").Content)
	assert.Equal(t, "Fix auth.py's nil panic", requireTaskFile(t, task.Files, "prompts/02-fix-panic.txt").Content)
}

func TestBuildCodexTaskRestoresSubscriptionWithSelectedModel(t *testing.T) {
	t.Parallel()

	spec := RunCodexSpec{
		Credentials: runner.AgentCredentials{Source: runner.CredentialsSourceSubscription},
		Model:       "gpt-5.4",
		Steps: []runner.AgentStep{
			{Name: "Fix panic", Type: runner.AgentStepPrompt, Prompt: strPtr("Fix it")},
		},
	}
	task := buildCodexTask(spec, "", nil, nil, nil, false)
	require.GreaterOrEqual(t, len(task.Commands), 4)
	assert.Equal(t, "Prepare Codex", task.Commands[0].Name)
	assert.Equal(t, "Restore ChatGPT sign-in", task.Commands[1].Name)
	assert.Equal(t, "Save ChatGPT usage", task.Commands[len(task.Commands)-1].Name)
	assert.Contains(t, task.Commands[len(task.Commands)-1].Command, "usage")
	assert.Contains(t, task.Commands[2].Command, `node "$SUPERPLANE_TASK_DIR/run.js"`)
	assert.Contains(t, task.Commands[2].Command, "gpt-5.4")
	assert.Contains(t, requireTaskFile(t, task.Files, "codex_subscription.js").Content, "auth.json")
}

func TestBuildCodexTaskPassesThinking(t *testing.T) {
	t.Parallel()

	spec := RunCodexSpec{
		Model:         "gpt-5",
		ThinkingLevel: "medium",
		Steps: []runner.AgentStep{
			{Name: "Fix panic", Type: runner.AgentStepPrompt, Prompt: strPtr("Fix it")},
		},
	}
	task := buildCodexTask(spec, "", nil, nil, nil, false)
	assert.Contains(t, task.Commands[1].Command, `node "$SUPERPLANE_TASK_DIR/run.js" "$SUPERPLANE_TASK_DIR/prompts/01-fix-panic.txt" 'gpt-5' 'medium'`)
}

func TestApplyPlanningFollowUpLeavesLineAutomationsUnchanged(t *testing.T) {
	t.Parallel()

	spec := RunCodexSpec{
		Model: "gpt-5",
		Steps: []runner.AgentStep{
			{Name: "Fix tests", Type: runner.AgentStepPrompt, Prompt: strPtr("fix"), WorkingDirectory: "repo"},
		},
	}
	base := buildCodexTask(spec, "", nil, nil, nil, false)
	got := applyPlanningFollowUp(base, nil, spec)
	assert.Len(t, got.Commands, len(base.Commands))
	assert.Len(t, got.Files, len(base.Files))
}

func TestApplyPlanningFollowUpAppendsWaitLoopForPlanningToken(t *testing.T) {
	t.Parallel()

	spec := RunCodexSpec{
		Model: "o3",
		Steps: []runner.AgentStep{
			{Name: "Clone", Type: runner.AgentStepBash, Command: strPtr("git clone")},
			{Name: "Hello", Type: runner.AgentStepPrompt, Prompt: strPtr("greet"), WorkingDirectory: "repo"},
		},
	}
	base := buildCodexTask(spec, "", nil, nil, nil, false)
	got := applyPlanningFollowUp(base, []runner.TaskEnvironmentVariable{{
		Name:  runner.EnvSuperplanePlanningID,
		Value: "session-1",
	}}, spec)

	require.Len(t, got.Commands, len(base.Commands)+1)
	last := got.Commands[len(got.Commands)-1]
	assert.Equal(t, "Wait for the next message", last.Name)
	assert.Equal(t, runner.LiveLogKindPrompt, last.Kind)
	assert.Contains(t, last.Command, `node "$SUPERPLANE_TASK_DIR/follow_up_loop.js" 'o3'`)
	assert.Contains(t, last.Command, `cd "$_sp_root"/'repo'`)
	assert.Equal(t, runner.FollowUpLoopFile().Content, requireTaskFile(t, got.Files, "follow_up_loop.js").Content)
	assert.NotEmpty(t, requireTaskFile(t, got.Files, runner.AttachmentFetchScriptPath).Content)
	assert.NotEmpty(t, requireTaskFile(t, got.Files, runner.AttachmentProcessScriptPath).Content)
}

func TestApplyPlanningFollowUpKeepsSubscriptionCheckpointLast(t *testing.T) {
	t.Parallel()

	spec := RunCodexSpec{
		Credentials: runner.AgentCredentials{Source: runner.CredentialsSourceSubscription},
		Steps: []runner.AgentStep{
			{Name: "Hello", Type: runner.AgentStepPrompt, Prompt: strPtr("greet")},
		},
	}
	base := buildCodexTask(spec, "", nil, nil, nil, false)
	got := applyPlanningFollowUp(base, []runner.TaskEnvironmentVariable{{
		Name:  runner.EnvSuperplanePlanningID,
		Value: "session-1",
	}}, spec)

	require.Len(t, got.Commands, len(base.Commands)+1)
	assert.Equal(t, "Wait for the next message", got.Commands[len(got.Commands)-2].Name)
	assert.Equal(t, "Save ChatGPT usage", got.Commands[len(got.Commands)-1].Name)
}

func requireTaskFile(t *testing.T, files []runner.TaskFile, path string) runner.TaskFile {
	t.Helper()
	for _, file := range files {
		if file.Path == path {
			return file
		}
	}
	t.Fatalf("missing task file %q", path)
	return runner.TaskFile{}
}

package codex

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/configuration"
)

func TestDecodeRunCodexSpecRequiresPromptStep(t *testing.T) {
	t.Parallel()

	spec, err := decodeRunCodexSpec(map[string]any{
		"machineType": "e1-large-amd64",
		"steps": []map[string]any{
			{"name": "Clone", "type": "bash", "command": "git clone"},
		},
		"credentials": map[string]any{
			"source": "secret",
			"secret": map[string]any{"secret": "openai", "key": "api_key"},
		},
	})
	require.NoError(t, err)
	err = validateRunCodexSpec(spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "prompt")
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

func TestBuildCodexBrokerTaskRunsOrderedSteps(t *testing.T) {
	t.Parallel()

	spec := RunCodexSpec{
		Model:            "gpt-5",
		WorkingDirectory: "/tmp/workspace",
		Steps: []runner.AgentStep{
			{Name: "Clone repo", Type: runner.AgentStepBash, Command: strPtr("git clone https://github.com/acme/widgets.git repo")},
			{Name: "Fix panic", Type: runner.AgentStepPrompt, Prompt: strPtr("Fix auth.py's nil panic")},
		},
	}

	task := buildCodexBrokerTask(spec, "", nil)
	require.Len(t, task.Commands, 3)
	assert.Equal(t, "Prepare Codex", task.Commands[0].Name)
	assert.Equal(t, "Clone repo", task.Commands[1].Name)
	assert.Equal(t, "Fix panic", task.Commands[2].Name)
	assert.Contains(t, task.Commands[2].Command, `node "$SUPERPLANE_TASK_DIR/run.js" "$SUPERPLANE_TASK_DIR/prompts/02-fix-panic.txt" 'gpt-5'`)
	assert.Equal(t, runScript, requireTaskFile(t, task.Files, "run.js").Content)
}

func TestApplyPlanningFollowUpLeavesLineAutomationsUnchanged(t *testing.T) {
	t.Parallel()

	spec := RunCodexSpec{
		Model: "gpt-5",
		Steps: []runner.AgentStep{
			{Name: "Fix tests", Type: runner.AgentStepPrompt, Prompt: strPtr("fix"), WorkingDirectory: "repo"},
		},
	}
	base := buildCodexBrokerTask(spec, "", nil)
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
	base := buildCodexBrokerTask(spec, "", nil)
	got := applyPlanningFollowUp(base, []runner.BrokerEnvironmentVariable{{
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
}

func requireTaskFile(t *testing.T, files []runner.BrokerTaskFile, path string) runner.BrokerTaskFile {
	t.Helper()
	for _, file := range files {
		if file.Path == path {
			return file
		}
	}
	t.Fatalf("missing task file %q", path)
	return runner.BrokerTaskFile{}
}

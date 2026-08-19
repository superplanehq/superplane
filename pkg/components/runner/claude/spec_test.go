package claude

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/configuration"
)

func strPtr(v string) *string { return &v }

func secretRef(secret, key string) configuration.SecretKeyRef {
	return configuration.SecretKeyRef{Secret: secret, Key: key}
}

func TestDecodeRunClaudeCodeSpecAppliesDefaults(t *testing.T) {
	t.Parallel()

	spec, err := decodeRunClaudeCodeSpec(map[string]any{
		"machineType": testRunnerMachineType,
		"steps": []map[string]any{
			{"name": "Fix bug", "type": "prompt", "prompt": "fix the bug"},
		},
		"anthropicApiKey": map[string]any{
			"secret": "anthropic",
			"key":    "api_key",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, runner.DefaultExecutionTimeoutSeconds, spec.ExecutionTimeoutSeconds)
	require.Len(t, spec.Steps, 1)
	assert.Equal(t, "Fix bug", spec.Steps[0].Name)
	assert.Equal(t, claudeStepPrompt, spec.Steps[0].Type)
}

func TestDecodeRunClaudeCodeSpecMigratesLegacyFields(t *testing.T) {
	t.Parallel()

	spec, err := decodeRunClaudeCodeSpec(map[string]any{
		"machineType":           testRunnerMachineType,
		"prompt":                "implement the issue",
		"enable_setup_commands": true,
		"setup_commands":        "git clone https://github.com/acme/widgets.git /tmp/repo",
		"enable_after_commands": true,
		"after_commands":        "git push",
		"anthropicApiKey": map[string]any{
			"secret": "anthropic",
			"key":    "api_key",
		},
	})
	require.NoError(t, err)
	require.Len(t, spec.Steps, 3)
	assert.Equal(t, "Setup", spec.Steps[0].Name)
	assert.Equal(t, claudeStepBash, spec.Steps[0].Type)
	assert.Equal(t, "Prompt", spec.Steps[1].Name)
	assert.Equal(t, claudeStepPrompt, spec.Steps[1].Type)
	assert.Equal(t, "After", spec.Steps[2].Name)
	assert.Equal(t, claudeStepBash, spec.Steps[2].Type)
}

func TestValidateRunClaudeCodeSpec(t *testing.T) {
	t.Parallel()

	valid := RunClaudeCodeSpec{
		MachineType: testRunnerMachineType,
		Steps: []ClaudeCodeStep{
			{Name: "Do the thing", Type: claudeStepPrompt, Prompt: strPtr("do the thing")},
		},
		Credentials: ClaudeCodeCredentials{
			Source: "secret",
			Secret: secretRef("anthropic", "api_key"),
		},
	}
	require.NoError(t, validateRunClaudeCodeSpec(valid))

	t.Run("requires step name", func(t *testing.T) {
		spec := valid
		spec.Steps = []ClaudeCodeStep{{Type: claudeStepPrompt, Prompt: strPtr("go")}}
		require.Error(t, validateRunClaudeCodeSpec(spec))
	})

	t.Run("requires steps", func(t *testing.T) {
		spec := valid
		spec.Steps = nil
		require.Error(t, validateRunClaudeCodeSpec(spec))
	})

	t.Run("requires at least one prompt", func(t *testing.T) {
		spec := valid
		spec.Steps = []ClaudeCodeStep{{Name: "Echo", Type: claudeStepBash, Command: strPtr("echo hi")}}
		require.Error(t, validateRunClaudeCodeSpec(spec))
	})
}

func TestBuildClaudeCodeBrokerTaskRunsOrderedSteps(t *testing.T) {
	t.Parallel()

	spec := RunClaudeCodeSpec{
		Model:            "sonnet",
		WorkingDirectory: "/tmp/workspace",
		Steps: []ClaudeCodeStep{
			{Name: "Clone repo", Type: claudeStepBash, Command: strPtr("git clone https://github.com/acme/widgets.git repo")},
			{Name: "Fix panic", Type: claudeStepPrompt, Prompt: strPtr("Fix auth.py's nil panic")},
			{Name: "Fix tests", Type: claudeStepPrompt, Prompt: strPtr("Run the tests and fix failures")},
			{Name: "Push", Type: claudeStepBash, Command: strPtr("git push")},
		},
	}

	task := buildClaudeCodeBrokerTask(spec)
	require.Len(t, task.Commands, 5)
	assert.Equal(t, "Prepare Claude Code", task.Commands[0].Name)
	assert.Equal(t, `source "$SUPERPLANE_TASK_DIR/prepare.sh"`, task.Commands[0].Command)

	assert.Equal(t, runner.BrokerCommand{Name: "Clone repo", Command: `source "$SUPERPLANE_TASK_DIR/steps/01-clone-repo.sh"`}, task.Commands[1])
	assert.Equal(t, runner.BrokerCommand{
		Name:    "Fix panic",
		Command: `node "$SUPERPLANE_TASK_DIR/run.js" "$SUPERPLANE_TASK_DIR/prompts/02-fix-panic.txt" 'sonnet'`,
	}, task.Commands[2])
	assert.Equal(t, runner.BrokerCommand{
		Name:    "Fix tests",
		Command: `node "$SUPERPLANE_TASK_DIR/run.js" "$SUPERPLANE_TASK_DIR/prompts/03-fix-tests.txt" 'sonnet'`,
	}, task.Commands[3])
	assert.Equal(t, runner.BrokerCommand{Name: "Push", Command: `source "$SUPERPLANE_TASK_DIR/steps/04-push.sh"`}, task.Commands[4])

	require.Len(t, task.Files, 6)
	assert.Equal(t, runScript, requireTaskFile(t, task.Files, "run.js").Content)
	prepare := requireTaskFile(t, task.Files, "prepare.sh").Content
	assert.Contains(t, prepare, "claude CLI not found")
	assert.Contains(t, prepare, "node not found")
	assert.Contains(t, prepare, "cd '/tmp/workspace'")
	assert.Contains(t, prepare, `echo "Claude Code ready"`)
	assert.Contains(t, prepare, "claude --version")
	assert.Contains(t, prepare, "node --version")

	assert.Equal(t, "git clone https://github.com/acme/widgets.git repo", requireTaskFile(t, task.Files, "steps/01-clone-repo.sh").Content)

	assert.Equal(t, "Fix auth.py's nil panic", requireTaskFile(t, task.Files, "prompts/02-fix-panic.txt").Content)
	assert.Equal(t, "Run the tests and fix failures", requireTaskFile(t, task.Files, "prompts/03-fix-tests.txt").Content)

	assert.Contains(t, runScript, "stream-json")
	assert.Contains(t, runScript, "--append-system-prompt")
	assert.Contains(t, runScript, "plain terminal text")
	assert.Contains(t, runScript, "--continue")
	assert.Contains(t, runScript, "SUPERPLANE_RESULT_FILE")
	assert.NotContains(t, runScript, "workdir")
}

func TestClaudeStepSlug(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "01-clone-repo", claudeStepSlug(1, "Clone repo"))
	assert.Equal(t, "02-step", claudeStepSlug(2, "!!!"))
	assert.Equal(t, "03-step", claudeStepSlug(3, "   "))
}

func TestShellSingleQuote(t *testing.T) {
	t.Parallel()

	assert.Equal(t, `'hello'`, shellSingleQuote("hello"))
	assert.Equal(t, `'it'\''s fine'`, shellSingleQuote("it's fine"))
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

package opencode

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

func TestDecodeRunOpenCodeSpecAppliesDefaults(t *testing.T) {
	t.Parallel()

	spec, err := decodeRunOpenCodeSpec(map[string]any{
		"machineType": testRunnerMachineType,
		"provider":    "openai",
		"secret":      map[string]any{"secret": "openai", "key": "api_key"},
		"model":       "gpt-4.1",
		"steps": []map[string]any{
			{"name": "Fix bug", "type": "prompt", "prompt": "fix the bug"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, runner.DefaultExecutionTimeoutSeconds, spec.ExecutionTimeoutSeconds)
	require.Len(t, spec.Steps, 1)
	assert.Equal(t, "Fix bug", spec.Steps[0].Name)
	assert.Equal(t, openCodeStepPrompt, spec.Steps[0].Type)
	assert.Equal(t, "openai", spec.Provider)
	assert.Equal(t, "gpt-4.1", spec.Model)
	assert.Equal(t, "openai/gpt-4.1", spec.modelRef())
}

func TestValidateRunOpenCodeSpec(t *testing.T) {
	t.Parallel()

	valid := RunOpenCodeSpec{
		MachineType: testRunnerMachineType,
		Provider:    "anthropic",
		Secret:      secretRef("anthropic", "api_key"),
		Model:       "claude-sonnet-4-5",
		Steps: []OpenCodeStep{
			{Name: "Do the thing", Type: openCodeStepPrompt, Prompt: strPtr("do the thing")},
		},
	}
	require.NoError(t, validateRunOpenCodeSpec(valid))

	t.Run("requires model", func(t *testing.T) {
		spec := valid
		spec.Model = ""
		require.Error(t, validateRunOpenCodeSpec(spec))
	})

	t.Run("accepts a bare model name", func(t *testing.T) {
		spec := valid
		spec.Model = "gpt-4.1"
		require.NoError(t, validateRunOpenCodeSpec(spec))
	})

	t.Run("requires provider", func(t *testing.T) {
		spec := valid
		spec.Provider = ""
		require.Error(t, validateRunOpenCodeSpec(spec))
	})

	t.Run("rejects unknown provider", func(t *testing.T) {
		spec := valid
		spec.Provider = "acme"
		require.Error(t, validateRunOpenCodeSpec(spec))
	})

	t.Run("requires API key", func(t *testing.T) {
		spec := valid
		spec.Secret = configuration.SecretKeyRef{}
		require.Error(t, validateRunOpenCodeSpec(spec))
	})

	t.Run("requires step name", func(t *testing.T) {
		spec := valid
		spec.Steps = []OpenCodeStep{{Type: openCodeStepPrompt, Prompt: strPtr("go")}}
		require.Error(t, validateRunOpenCodeSpec(spec))
	})

	t.Run("requires steps", func(t *testing.T) {
		spec := valid
		spec.Steps = nil
		require.Error(t, validateRunOpenCodeSpec(spec))
	})

	t.Run("requires at least one prompt", func(t *testing.T) {
		spec := valid
		spec.Steps = []OpenCodeStep{{Name: "Echo", Type: openCodeStepBash, Command: strPtr("echo hi")}}
		require.Error(t, validateRunOpenCodeSpec(spec))
	})

	t.Run("rejects reserved env var names", func(t *testing.T) {
		spec := valid
		value := "x"
		spec.Environment = []runner.EnvironmentVariable{
			{Name: "ANTHROPIC_API_KEY", ValueSource: runner.EnvironmentValueSourceLiteral, Value: &value},
		}
		require.Error(t, validateRunOpenCodeSpec(spec))
	})
}

func TestBuildOpenCodeBrokerTaskRunsOrderedSteps(t *testing.T) {
	t.Parallel()

	spec := RunOpenCodeSpec{
		Provider:         "openai",
		Model:            "gpt-4.1",
		WorkingDirectory: "/tmp/workspace",
		Steps: []OpenCodeStep{
			{Name: "Clone repo", Type: openCodeStepBash, Command: strPtr("git clone https://github.com/acme/widgets.git repo")},
			{Name: "Fix panic", Type: openCodeStepPrompt, Prompt: strPtr("Fix auth.py's nil panic")},
			{Name: "Fix tests", Type: openCodeStepPrompt, Prompt: strPtr("Run the tests and fix failures")},
			{Name: "Push", Type: openCodeStepBash, Command: strPtr("git push")},
		},
	}

	task := buildOpenCodeBrokerTask(spec)
	require.Len(t, task.Commands, 5)
	assert.Equal(t, "Prepare OpenCode", task.Commands[0].Name)
	assert.Equal(t, `source "$SUPERPLANE_TASK_DIR/prepare.sh"`, task.Commands[0].Command)

	assert.Equal(t, runner.BrokerCommand{Name: "Clone repo", Command: `source "$SUPERPLANE_TASK_DIR/steps/01-clone-repo.sh"`}, task.Commands[1])
	assert.Equal(t, runner.BrokerCommand{
		Name:    "Fix panic",
		Command: `node "$SUPERPLANE_TASK_DIR/run.js" "$SUPERPLANE_TASK_DIR/prompts/02-fix-panic.txt" 'openai/gpt-4.1'`,
	}, task.Commands[2])
	assert.Equal(t, runner.BrokerCommand{
		Name:    "Fix tests",
		Command: `node "$SUPERPLANE_TASK_DIR/run.js" "$SUPERPLANE_TASK_DIR/prompts/03-fix-tests.txt" 'openai/gpt-4.1'`,
	}, task.Commands[3])
	assert.Equal(t, runner.BrokerCommand{Name: "Push", Command: `source "$SUPERPLANE_TASK_DIR/steps/04-push.sh"`}, task.Commands[4])

	require.Len(t, task.Files, 6)
	assert.Equal(t, runScript, requireTaskFile(t, task.Files, "run.js").Content)
	prepare := requireTaskFile(t, task.Files, "prepare.sh").Content
	assert.Contains(t, prepare, "opencode CLI not found")
	assert.Contains(t, prepare, "node not found")
	assert.Contains(t, prepare, "cd '/tmp/workspace'")
	assert.Contains(t, prepare, `echo "OpenCode ready"`)
	assert.Contains(t, prepare, "opencode --version")
	assert.Contains(t, prepare, "node --version")
	assert.Contains(t, prepare, "rm -f \"$SUPERPLANE_TASK_DIR/session_id\"")

	assert.Equal(t, "git clone https://github.com/acme/widgets.git repo", requireTaskFile(t, task.Files, "steps/01-clone-repo.sh").Content)
	assert.Equal(t, "Fix auth.py's nil panic", requireTaskFile(t, task.Files, "prompts/02-fix-panic.txt").Content)
	assert.Equal(t, "Run the tests and fix failures", requireTaskFile(t, task.Files, "prompts/03-fix-tests.txt").Content)

	assert.Contains(t, runScript, "--format")
	assert.Contains(t, runScript, "--session")
	assert.Contains(t, runScript, "session_id")
	assert.Contains(t, runScript, "SUPERPLANE_RESULT_FILE")
}

func TestOpenCodeStepSlug(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "01-clone-repo", openCodeStepSlug(1, "Clone repo"))
	assert.Equal(t, "02-step", openCodeStepSlug(2, "!!!"))
	assert.Equal(t, "03-step", openCodeStepSlug(3, "   "))
}

func TestShellSingleQuote(t *testing.T) {
	t.Parallel()

	assert.Equal(t, `'hello'`, shellSingleQuote("hello"))
	assert.Equal(t, `'it'\''s fine'`, shellSingleQuote("it's fine"))
}

func TestProviderEnvVarsUnique(t *testing.T) {
	t.Parallel()

	seen := map[string]struct{}{}
	for _, provider := range openCodeProviders {
		assert.NotEmpty(t, provider.EnvVar)
		_, dup := seen[provider.EnvVar]
		assert.False(t, dup, "duplicate env var %s", provider.EnvVar)
		seen[provider.EnvVar] = struct{}{}
	}
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

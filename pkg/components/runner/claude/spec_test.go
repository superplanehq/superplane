package claude

import (
	"encoding/base64"
	"strings"
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
		"machine_type": testRunnerMachineType,
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
		"machine_type":          testRunnerMachineType,
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
		AnthropicAPIKey: secretRef("anthropic", "api_key"),
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
	assert.True(t, strings.HasPrefix(task.Commands[0].Command, "bash -c "))
	assert.Contains(t, task.Commands[0].Command, "claude CLI not found")
	assert.Contains(t, task.Commands[0].Command, "node not found")
	assert.Contains(t, task.Commands[0].Command, "/tmp/workspace")
	assert.Contains(t, task.Commands[0].Command, "01-clone-repo.sh")
	assert.Contains(t, task.Commands[0].Command, "02-fix-panic.sh")
	assert.Contains(t, task.Commands[0].Command, base64.StdEncoding.EncodeToString([]byte(streamFormatJS)))
	assert.Contains(t, task.Commands[0].Command, base64.StdEncoding.EncodeToString([]byte(claudeWriteResultScript())))

	assert.Equal(t, runner.BrokerCommand{Name: "Clone repo", Command: `bash "$(dirname "$SUPERPLANE_RESULT_FILE")/claude-code/steps/01-clone-repo.sh"`}, task.Commands[1])
	assert.Equal(t, runner.BrokerCommand{Name: "Fix panic", Command: `bash "$(dirname "$SUPERPLANE_RESULT_FILE")/claude-code/steps/02-fix-panic.sh"`}, task.Commands[2])
	assert.Equal(t, runner.BrokerCommand{Name: "Fix tests", Command: `bash "$(dirname "$SUPERPLANE_RESULT_FILE")/claude-code/steps/03-fix-tests.sh"`}, task.Commands[3])
	assert.Equal(t, runner.BrokerCommand{Name: "Push", Command: `bash "$(dirname "$SUPERPLANE_RESULT_FILE")/claude-code/steps/04-push.sh"`}, task.Commands[4])

	cloneScript := buildClaudeBashStepScript("git clone https://github.com/acme/widgets.git repo")
	assert.Contains(t, task.Commands[0].Command, base64.StdEncoding.EncodeToString([]byte(cloneScript)))
	assert.Contains(t, cloneScript, "git clone https://github.com/acme/widgets.git repo\n")
	assert.Contains(t, cloneScript, `pwd -P >"$SP/workdir"`)
	assert.NotContains(t, cloneScript, "bash -c ")

	promptScript := buildClaudePromptStepScript("Fix auth.py's nil panic", "sonnet")
	assert.Contains(t, task.Commands[0].Command, base64.StdEncoding.EncodeToString([]byte(promptScript)))
	assert.Contains(t, promptScript, "--output-format stream-json")
	assert.Contains(t, promptScript, "--append-system-prompt")
	assert.Contains(t, promptScript, "plain terminal text")
	assert.Contains(t, promptScript, "--continue")
	assert.Contains(t, promptScript, `node "$SP/format.js"`)
	assert.Contains(t, promptScript, base64.StdEncoding.EncodeToString([]byte("Fix auth.py's nil panic")))
	assert.Contains(t, promptScript, "--model 'sonnet'")
	assert.Contains(t, promptScript, "write-result.sh")
}

func TestClaudeStepScriptName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "01-clone-repo.sh", claudeStepScriptName(1, "Clone repo"))
	assert.Equal(t, "02-step.sh", claudeStepScriptName(2, "!!!"))
	assert.Equal(t, "03-step.sh", claudeStepScriptName(3, "   "))
}

func TestShellSingleQuote(t *testing.T) {
	t.Parallel()

	assert.Equal(t, `'hello'`, shellSingleQuote("hello"))
	assert.Equal(t, `'it'\''s fine'`, shellSingleQuote("it's fine"))
}

package runner

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func strPtr(v string) *string { return &v }

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
	assert.Equal(t, DefaultExecutionTimeoutSeconds, spec.ExecutionTimeoutSeconds)
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

func TestBuildClaudeCodeScriptRunsOrderedSteps(t *testing.T) {
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

	script := buildClaudeCodeScript(spec)
	assert.Contains(t, script, "cd '/tmp/workspace'")
	assert.Contains(t, script, "==> Step 1: bash (Clone repo)")
	assert.Contains(t, script, "git clone https://github.com/acme/widgets.git repo")
	assert.Contains(t, script, "==> Step 2: prompt (Fix panic)")
	assert.Contains(t, script, "==> Step 3: prompt (Fix tests)")
	assert.Contains(t, script, "==> Step 4: bash (Push)")
	assert.Contains(t, script, "git push")
	assert.Contains(t, script, "--continue")
	assert.Contains(t, script, "--output-format stream-json")
	assert.Contains(t, script, "--verbose")
	assert.Contains(t, script, "--include-partial-messages")
	assert.Contains(t, script, "python3 -u")
	assert.Contains(t, script, "write_claude_result")
	assert.Contains(t, script, base64.StdEncoding.EncodeToString([]byte("Fix auth.py's nil panic")))
}

func TestShellSingleQuote(t *testing.T) {
	t.Parallel()

	assert.Equal(t, `'hello'`, shellSingleQuote("hello"))
	assert.Equal(t, `'it'\''s fine'`, shellSingleQuote("it's fine"))
}

package runner

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeRunClaudeCodeSpecAppliesDefaults(t *testing.T) {
	t.Parallel()

	spec, err := decodeRunClaudeCodeSpec(map[string]any{
		"machine_type": testRunnerMachineType,
		"prompt":       "fix the bug",
		"anthropicApiKey": map[string]any{
			"secret": "anthropic",
			"key":    "api_key",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, DefaultExecutionTimeoutSeconds, spec.ExecutionTimeoutSeconds)
}

func TestValidateRunClaudeCodeSpec(t *testing.T) {
	t.Parallel()

	valid := RunClaudeCodeSpec{
		MachineType:     testRunnerMachineType,
		Prompt:          "do the thing",
		AnthropicAPIKey: secretRef("anthropic", "api_key"),
	}
	require.NoError(t, validateRunClaudeCodeSpec(valid))

	t.Run("requires prompt", func(t *testing.T) {
		spec := valid
		spec.Prompt = "  "
		require.Error(t, validateRunClaudeCodeSpec(spec))
	})

	t.Run("requires api key", func(t *testing.T) {
		spec := valid
		spec.AnthropicAPIKey = secretRef("", "")
		require.Error(t, validateRunClaudeCodeSpec(spec))
	})

	t.Run("rejects ANTHROPIC_API_KEY in environment", func(t *testing.T) {
		value := "should-not-use"
		spec := valid
		spec.Environment = []EnvironmentVariable{{
			Name:        envAnthropicAPIKey,
			ValueSource: EnvironmentValueSourceLiteral,
			Value:       &value,
		}}
		require.Error(t, validateRunClaudeCodeSpec(spec))
	})

	t.Run("requires setup commands when enabled", func(t *testing.T) {
		spec := valid
		spec.EnableSetupCommands = true
		spec.SetupCommands = "   \n  "
		require.Error(t, validateRunClaudeCodeSpec(spec))
	})

	t.Run("requires after commands when enabled", func(t *testing.T) {
		spec := valid
		spec.EnableAfterCommands = true
		spec.AfterCommands = "   \n  "
		require.Error(t, validateRunClaudeCodeSpec(spec))
	})
}

func TestBuildClaudeCodeScript(t *testing.T) {
	t.Parallel()

	spec := RunClaudeCodeSpec{
		Prompt:           "Fix auth.py's nil panic",
		Model:            "sonnet",
		WorkingDirectory: "/tmp/repo",
	}

	script := buildClaudeCodeScript(spec)
	assert.Contains(t, script, "command -v claude")
	assert.Contains(t, script, "--bare -p --output-format json")
	assert.Contains(t, script, "--permission-mode 'acceptEdits'")
	assert.Contains(t, script, "--model 'sonnet'")
	assert.Contains(t, script, "--allowedTools 'Bash,Read,Edit,Write'")
	assert.Contains(t, script, "cd '/tmp/repo'")
	assert.Contains(t, script, "SUPERPLANE_RESULT_FILE")
	assert.Contains(t, script, base64.StdEncoding.EncodeToString([]byte(spec.Prompt)))
	assert.NotContains(t, script, "git push")
}

func TestBuildClaudeCodeScriptIncludesAfterCommands(t *testing.T) {
	t.Parallel()

	script := buildClaudeCodeScript(RunClaudeCodeSpec{
		Prompt:              "ship it",
		EnableAfterCommands: true,
		AfterCommands:       "git push\ngh pr create --fill",
	})
	assert.Contains(t, script, "claude")
	assert.Contains(t, script, "git push")
	assert.Contains(t, script, "gh pr create --fill")
	assert.Greater(t, strings.Index(script, "git push"), strings.Index(script, "SUPERPLANE_RESULT_FILE"))
}

func TestShellSingleQuote(t *testing.T) {
	t.Parallel()

	assert.Equal(t, `'hello'`, shellSingleQuote("hello"))
	assert.Equal(t, `'it'\''s fine'`, shellSingleQuote("it's fine"))
}

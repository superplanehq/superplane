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

func TestValidateRunCodexSpecAcceptsHostedCredentials(t *testing.T) {
	t.Parallel()

	prompt := "fix tests"
	spec := RunCodexSpec{
		MachineType: "e1-large-amd64",
		Steps: []runner.AgentStep{
			{Name: "Prompt", Type: runner.AgentStepPrompt, Prompt: &prompt},
		},
		Credentials: runner.AgentCredentials{Source: runner.CredentialsSourceHosted},
	}
	require.NoError(t, validateRunCodexSpec(spec))
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

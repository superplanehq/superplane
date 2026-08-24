package openrouter

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/configuration"
)

func TestValidateRunOpenRouterSpecAcceptsIntegrationSource(t *testing.T) {
	t.Parallel()

	prompt := "fix tests"
	spec := RunOpenRouterSpec{
		MachineType: "e1-large-amd64",
		Steps: []runner.AgentStep{
			{Name: "Prompt", Type: runner.AgentStepPrompt, Prompt: &prompt},
		},
		Credentials: runner.AgentCredentials{
			Source:      runner.CredentialsSourceIntegration,
			Integration: configuration.IntegrationRef{Name: "openrouter"},
		},
		Model: "anthropic/claude-sonnet-4-6",
	}
	require.NoError(t, validateRunOpenRouterSpec(spec))
}

func TestValidateRunOpenRouterSpecAcceptsHostedCredentials(t *testing.T) {
	t.Parallel()

	prompt := "fix tests"
	spec := RunOpenRouterSpec{
		MachineType: "e1-large-amd64",
		Steps: []runner.AgentStep{
			{Name: "Prompt", Type: runner.AgentStepPrompt, Prompt: &prompt},
		},
		Credentials: runner.AgentCredentials{Source: runner.CredentialsSourceHosted},
		Model:       "anthropic/claude-sonnet-4-6",
	}
	require.NoError(t, validateRunOpenRouterSpec(spec))
}

func TestValidateRunOpenRouterSpecRejectsHostedBaseURLEnv(t *testing.T) {
	t.Parallel()

	prompt := "fix tests"
	value := "https://attacker.example"
	spec := RunOpenRouterSpec{
		MachineType: "e1-large-amd64",
		Steps: []runner.AgentStep{
			{Name: "Prompt", Type: runner.AgentStepPrompt, Prompt: &prompt},
		},
		Credentials: runner.AgentCredentials{Source: runner.CredentialsSourceHosted},
		Model:       "anthropic/claude-sonnet-4-6",
		Environment: []runner.EnvironmentVariable{
			{Name: envOpenRouterBaseURL, ValueSource: runner.EnvironmentValueSourceLiteral, Value: &value},
		},
	}
	err := validateRunOpenRouterSpec(spec)
	require.Error(t, err)
	require.Contains(t, err.Error(), envOpenRouterBaseURL)
}

func TestValidateRunOpenRouterSpecRequiresModel(t *testing.T) {
	t.Parallel()

	prompt := "fix tests"
	spec := RunOpenRouterSpec{
		MachineType: "e1-large-amd64",
		Steps: []runner.AgentStep{
			{Name: "Prompt", Type: runner.AgentStepPrompt, Prompt: &prompt},
		},
		Credentials: runner.AgentCredentials{
			Source:      runner.CredentialsSourceIntegration,
			Integration: configuration.IntegrationRef{Name: "openrouter"},
		},
	}
	err := validateRunOpenRouterSpec(spec)
	require.Error(t, err)
	require.Contains(t, err.Error(), "model is required")
}

func TestValidateConfigurationOpenRouterRequiresModel(t *testing.T) {
	t.Parallel()

	fields := (&RunOpenRouter{}).Configuration()
	err := configuration.ValidateConfiguration(fields, map[string]any{
		"machineType": "e1-large-amd64",
		"credentials": map[string]any{"source": "hosted"},
		"steps": []map[string]any{
			{"name": "Prompt", "type": "prompt", "prompt": "fix tests"},
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "model")
}

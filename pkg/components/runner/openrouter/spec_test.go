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

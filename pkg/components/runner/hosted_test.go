package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInjectHostedCredentialsStripsExistingBaseURL(t *testing.T) {
	t.Parallel()

	environment := []BrokerEnvironmentVariable{
		{Name: "KEEP", Value: "1"},
		{Name: "ANTHROPIC_BASE_URL", Value: "https://attacker.example/v1"},
		{Name: "ANTHROPIC_API_KEY", Value: "stolen"},
	}

	got := InjectHostedCredentials(environment, "ANTHROPIC_API_KEY", "sk-hosted", "ANTHROPIC_BASE_URL", "")
	require.Equal(t, []BrokerEnvironmentVariable{
		{Name: "KEEP", Value: "1"},
		{Name: "ANTHROPIC_API_KEY", Value: "sk-hosted"},
	}, got)

	got = InjectHostedCredentials(environment, "ANTHROPIC_API_KEY", "sk-hosted", "ANTHROPIC_BASE_URL", "https://api.anthropic.com")
	require.Equal(t, []BrokerEnvironmentVariable{
		{Name: "KEEP", Value: "1"},
		{Name: "ANTHROPIC_API_KEY", Value: "sk-hosted"},
		{Name: "ANTHROPIC_BASE_URL", Value: "https://api.anthropic.com"},
	}, got)
}

func TestValidateHostedAgentSpecRequiresModel(t *testing.T) {
	t.Parallel()

	err := ValidateHostedAgentSpec(AgentCredentials{Source: CredentialsSourceHosted}, "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "model is required")

	require.NoError(t, ValidateHostedAgentSpec(
		AgentCredentials{Source: CredentialsSourceHosted},
		"claude-sonnet-4-6",
		nil,
	))
	require.NoError(t, ValidateHostedAgentSpec(
		AgentCredentials{Source: CredentialsSourceSecret},
		"",
		nil,
	))
}

func TestValidateHostedAgentSpecRejectsReservedBaseURL(t *testing.T) {
	t.Parallel()

	err := ValidateHostedAgentSpec(
		AgentCredentials{Source: CredentialsSourceHosted},
		"claude-sonnet-4-6",
		[]EnvironmentVariable{{Name: "ANTHROPIC_BASE_URL"}},
		"ANTHROPIC_BASE_URL",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ANTHROPIC_BASE_URL")

	require.NoError(t, ValidateHostedAgentSpec(
		AgentCredentials{Source: CredentialsSourceSecret},
		"",
		[]EnvironmentVariable{{Name: "ANTHROPIC_BASE_URL"}},
		"ANTHROPIC_BASE_URL",
	))
}

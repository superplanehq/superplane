package runner

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/core"
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

func TestPrepareHostedRunChecksFactoryAllowlist(t *testing.T) {
	t.Parallel()

	stub := &hostedAllowlistLLM{
		access: core.HostedLLMAccess{
			APIKey:        "sk-hosted",
			AllowedModels: []string{"claude-sonnet-4-6", "claude-opus-4-6"},
		},
		selectable: map[string]bool{"claude-sonnet-4-6": true},
	}

	_, err := PrepareHostedRun(core.ExecutionContext{HostedLLM: stub}, "anthropic", "claude-opus-4-6")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "selected-model list")
	assert.True(t, stub.asserted)

	access, err := PrepareHostedRun(core.ExecutionContext{HostedLLM: stub}, "anthropic", "claude-sonnet-4-6")
	require.NoError(t, err)
	assert.Equal(t, "sk-hosted", access.APIKey)
}

type hostedAllowlistLLM struct {
	access     core.HostedLLMAccess
	selectable map[string]bool
	asserted   bool
}

func (c *hostedAllowlistLLM) Resolve(string) (core.HostedLLMAccess, error) {
	return c.access, nil
}

func (c *hostedAllowlistLLM) AssertCreditAvailable() error {
	return nil
}

func (c *hostedAllowlistLLM) AssertModelSelectable(_, _, model string) error {
	c.asserted = true
	if c.selectable[strings.TrimSpace(model)] {
		return nil
	}
	return fmt.Errorf("model %s is not on the selected-model list", model)
}

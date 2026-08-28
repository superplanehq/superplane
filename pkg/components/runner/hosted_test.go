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

func TestPrepareBYOKRunChecksAllowlistWhenModelIsEmpty(t *testing.T) {
	t.Parallel()

	require.NoError(t, PrepareBYOKRun(core.ExecutionContext{}, "anthropic", ""))

	stub := &byokHostedLLM{}
	err := PrepareBYOKRun(core.ExecutionContext{HostedLLM: stub}, "anthropic", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "model is required")
	assert.True(t, stub.called)

	require.NoError(t, PrepareBYOKRun(core.ExecutionContext{HostedLLM: stub}, "anthropic", "claude-sonnet-4-6"))
}

type byokHostedLLM struct {
	called bool
}

func (c *byokHostedLLM) Resolve(string) (core.HostedLLMAccess, error) {
	return core.HostedLLMAccess{}, nil
}

func (c *byokHostedLLM) AssertCreditAvailable() error {
	return nil
}

func (c *byokHostedLLM) AssertModelSelectable(_, _, model string) error {
	c.called = true
	if strings.TrimSpace(model) == "" {
		return fmt.Errorf("model is required")
	}
	return nil
}

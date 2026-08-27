package factories

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__ResolveIntakeAgent(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	newFactoryIn := func(t *testing.T, organizationID uuid.UUID) *models.Factory {
		t.Helper()
		factory, err := models.CreateFactory(db, organizationID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		return factory
	}

	t.Run("the intake scores with the agent the setup recorded", func(t *testing.T) {
		organization := support.CreateOrganization(t, r, r.User)
		factory := newFactoryIn(t, organization.ID)
		agentID := createReadyOnboardingIntegration(t, organization.ID, "openai")
		require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
			AgentIntegrationID: &agentID,
		}))

		agent := resolveIntakeAgent(db, factory)
		require.NotNil(t, agent)
		assert.Equal(t, "runnerCodex", agent.Component)
		assert.Equal(t, runner.CredentialsSourceIntegration, agent.Credentials["source"])
		assert.Equal(t, integrationName(t, organization.ID, agentID), integrationRefName(t, agent))
	})

	t.Run("OpenRouter needs the model the runner asks for", func(t *testing.T) {
		organization := support.CreateOrganization(t, r, r.User)
		factory := newFactoryIn(t, organization.ID)
		agentID := createReadyOnboardingIntegration(t, organization.ID, "openrouter")
		require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
			AgentIntegrationID: &agentID,
		}))

		agent := resolveIntakeAgent(db, factory)
		require.NotNil(t, agent)
		assert.Equal(t, "runnerOpenRouter", agent.Component)
		assert.NotEmpty(t, agent.Model)
	})

	t.Run("an agent without a runner falls back to the installations", func(t *testing.T) {
		organization := support.CreateOrganization(t, r, r.User)
		factory := newFactoryIn(t, organization.ID)
		cursorID := createReadyOnboardingIntegration(t, organization.ID, "cursor")
		claudeID := createReadyOnboardingIntegration(t, organization.ID, "claude")
		require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
			AgentIntegrationID: &cursorID,
		}))

		agent := resolveIntakeAgent(db, factory)
		require.NotNil(t, agent)
		assert.Equal(t, "runnerClaudeCode", agent.Component)
		assert.Equal(t, integrationName(t, organization.ID, claudeID), integrationRefName(t, agent))
	})

	t.Run("a setup without an agent uses the installations of the organization", func(t *testing.T) {
		organization := support.CreateOrganization(t, r, r.User)
		factory := newFactoryIn(t, organization.ID)
		claudeID := createReadyOnboardingIntegration(t, organization.ID, "claude")

		agent := resolveIntakeAgent(db, factory)
		require.NotNil(t, agent)
		assert.Equal(t, "runnerClaudeCode", agent.Component)
		assert.Equal(t, integrationName(t, organization.ID, claudeID), integrationRefName(t, agent))
	})

	t.Run("an organization without an installation runs on hosted credentials", func(t *testing.T) {
		organization := support.CreateOrganization(t, r, r.User)
		factory := newFactoryIn(t, organization.ID)
		clearHostedLLMProviders(t, db)
		_, err := models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
			Provider:      models.UsageProviderAnthropic,
			Enabled:       true,
			APIKey:        []byte("test-hosted-key"),
			AllowedModels: datatypes.JSONSlice[string]{"claude-haiku-4-6", "claude-sonnet-4-6"},
		})
		require.NoError(t, err)

		agent := resolveIntakeAgent(db, factory)
		require.NotNil(t, agent)
		assert.Equal(t, "runnerClaudeCode", agent.Component)
		assert.Equal(t, map[string]any{"source": runner.CredentialsSourceHosted}, agent.Credentials)
		assert.Equal(t, "claude-sonnet-4-6", agent.Model)
	})

	t.Run("hosted fallback prefers OpenRouter when several providers are enabled", func(t *testing.T) {
		organization := support.CreateOrganization(t, r, r.User)
		factory := newFactoryIn(t, organization.ID)
		clearHostedLLMProviders(t, db)
		_, err := models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
			Provider:      models.UsageProviderAnthropic,
			Enabled:       true,
			APIKey:        []byte("test-hosted-key"),
			AllowedModels: datatypes.JSONSlice[string]{"claude-sonnet-4-6"},
		})
		require.NoError(t, err)
		_, err = models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
			Provider:      models.UsageProviderOpenRouter,
			Enabled:       true,
			APIKey:        []byte("test-openrouter-key"),
			AllowedModels: datatypes.JSONSlice[string]{"openai/gpt-4.1", "anthropic/claude-sonnet-4-6"},
		})
		require.NoError(t, err)

		agent := resolveIntakeAgent(db, factory)
		require.NotNil(t, agent)
		assert.Equal(t, "runnerOpenRouter", agent.Component)
		assert.Equal(t, map[string]any{"source": runner.CredentialsSourceHosted}, agent.Credentials)
		assert.Equal(t, "anthropic/claude-sonnet-4-6", agent.Model)
	})

	t.Run("a workspace with no agent at all leaves the node incomplete", func(t *testing.T) {
		organization := support.CreateOrganization(t, r, r.User)
		factory := newFactoryIn(t, organization.ID)
		clearHostedLLMProviders(t, db)

		assert.Nil(t, resolveIntakeAgent(db, factory))
	})

	t.Run("an installation that is not ready cannot authenticate the runner", func(t *testing.T) {
		organization := support.CreateOrganization(t, r, r.User)
		factory := newFactoryIn(t, organization.ID)
		clearHostedLLMProviders(t, db)
		integration, err := models.CreateIntegration(
			uuid.New(),
			organization.ID,
			"claude",
			support.RandomName("claude"),
			map[string]any{},
		)
		require.NoError(t, err)
		agentID := integration.ID.String()
		require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
			AgentIntegrationID: &agentID,
		}))

		assert.Nil(t, resolveIntakeAgent(db, factory))
	})
}

func Test__HostedIntakeModel(t *testing.T) {
	t.Run("prefers the model the hint names", func(t *testing.T) {
		provider := models.HostedLLMProvider{
			AllowedModels: datatypes.JSONSlice[string]{"gpt-5-mini", "gpt-4.1"},
		}
		assert.Equal(t, "gpt-5-mini", hostedIntakeModel(provider, "gpt-5"))
	})

	t.Run("takes the first model when the hint misses", func(t *testing.T) {
		provider := models.HostedLLMProvider{
			AllowedModels: datatypes.JSONSlice[string]{"  ", "z-model", "a-model"},
		}
		assert.Equal(t, "a-model", hostedIntakeModel(provider, "sonnet"))
	})

	t.Run("reports no model for an empty allowlist", func(t *testing.T) {
		assert.Empty(t, hostedIntakeModel(models.HostedLLMProvider{}, "sonnet"))
	})
}

func integrationName(t *testing.T, organizationID uuid.UUID, integrationID string) string {
	t.Helper()

	integration, err := models.FindIntegrationInTransaction(
		database.DB(t.Context()),
		organizationID,
		uuid.MustParse(integrationID),
	)
	require.NoError(t, err)

	return integration.InstallationName
}

func integrationRefName(t *testing.T, agent *intakeAgent) string {
	t.Helper()

	ref, ok := agent.Credentials["integration"].(map[string]any)
	require.True(t, ok, "credentials have no integration reference")
	name, ok := ref["name"].(string)
	require.True(t, ok, "integration reference has no name")

	return name
}

package models_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ParseSelectableLLMModelKey(t *testing.T) {
	t.Parallel()

	got, err := models.ParseSelectableLLMModelKey("hosted::anthropic::claude-sonnet-4-6")
	require.NoError(t, err)
	assert.Equal(t, "hosted", got.Source.ID)
	assert.Equal(t, "SuperPlane", got.Source.Name)
	assert.Equal(t, "anthropic", got.Provider.ID)
	assert.Equal(t, "Anthropic", got.Provider.Name)
	assert.Equal(t, "claude-sonnet-4-6", got.Model.ID)
	assert.Equal(t, "hosted::anthropic::claude-sonnet-4-6", got.Key)
	assert.Equal(t, "anthropic/claude-sonnet-4-6", got.Label)

	got, err = models.ParseSelectableLLMModelKey("byok::openrouter::moonshotai/kimi-k2.6")
	require.NoError(t, err)
	assert.Equal(t, "byok", got.Source.ID)
	assert.Equal(t, "Your keys", got.Source.Name)
	assert.Equal(t, "moonshotai/kimi-k2.6", got.Label)

	_, err = models.ParseSelectableLLMModelKey("anthropic::claude-sonnet-4-6")
	assert.ErrorIs(t, err, models.ErrSelectableLLMModelIncomplete)
}

func Test__ParseHostedLLMModelKey_AcceptsThreePartHostedKey(t *testing.T) {
	t.Parallel()

	got, err := models.ParseHostedLLMModelKey("hosted::openrouter::anthropic/claude-sonnet-4-6")
	require.NoError(t, err)
	assert.Equal(t, models.DefaultHostedLLMModel{Provider: "openrouter", Model: "anthropic/claude-sonnet-4-6"}, got)

	_, err = models.ParseHostedLLMModelKey("byok::anthropic::claude-sonnet-4-6")
	assert.ErrorIs(t, err, models.ErrDefaultHostedModelIncomplete)
}

func Test__HostedLLMTechnicalName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "anthropic/claude-sonnet-4-6", models.HostedLLMTechnicalName("anthropic", "claude-sonnet-4-6"))
	assert.Equal(t, "moonshotai/kimi-k2.6", models.HostedLLMTechnicalName("openrouter", "moonshotai/kimi-k2.6"))
}

func Test__ListSelectableLLMModels(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = database.Conn().Where("provider <> ?", "").Delete(&models.HostedLLMProvider{})
		_ = database.Conn().Where("organization_id = ?", r.Organization.ID).Delete(&models.OrganizationBYOKModelAllowlist{})
		_ = database.Conn().Where("factory_id = ?", factory.ID).Delete(&models.FactoryLLMModelAllowlist{})
	})

	listed, err := models.ListSelectableLLMModels(db, r.Organization.ID, &factory.ID)
	require.NoError(t, err)
	assert.Empty(t, listed)

	_, err = models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider:      models.UsageProviderAnthropic,
		Enabled:       true,
		APIKey:        []byte("encrypted"),
		AllowedModels: datatypes.JSONSlice[string]{"claude-sonnet-4-6", "claude-opus-4-6"},
	})
	require.NoError(t, err)
	_, err = models.UpsertOrganizationBYOKModelAllowlist(db, r.Organization.ID, models.UsageProviderAnthropic, datatypes.JSONSlice[string]{
		"claude-sonnet-4-6",
	})
	require.NoError(t, err)

	listed, err = models.ListSelectableLLMModels(db, r.Organization.ID, nil)
	require.NoError(t, err)
	require.Len(t, listed, 3)
	assert.Equal(t, "hosted::anthropic::claude-opus-4-6", listed[0].Key)
	assert.Equal(t, "byok::anthropic::claude-sonnet-4-6", listed[1].Key)
	assert.Equal(t, "hosted::anthropic::claude-sonnet-4-6", listed[2].Key)
	assert.Equal(t, "hosted", listed[2].Source.ID)
	assert.Equal(t, "byok", listed[1].Source.ID)
	assert.Equal(t, listed[1].Label, listed[2].Label)

	_, err = models.UpsertFactoryLLMModelAllowlist(db, r.Organization.ID, factory.ID, models.UsageProviderAnthropic, models.UsageFundingSourceHosted, datatypes.JSONSlice[string]{
		"claude-sonnet-4-6",
	})
	require.NoError(t, err)

	listed, err = models.ListSelectableLLMModels(db, r.Organization.ID, &factory.ID)
	require.NoError(t, err)
	require.Len(t, listed, 2)
	assert.Equal(t, []string{"byok::anthropic::claude-sonnet-4-6", "hosted::anthropic::claude-sonnet-4-6"}, selectableKeys(listed))

	found, err := models.FindSelectableLLMModel(db, r.Organization.ID, &factory.ID, "hosted::anthropic::claude-opus-4-6")
	assert.ErrorIs(t, err, models.ErrSelectableLLMModelNotAllowed)
	assert.Equal(t, models.SelectableLLMModel{}, found)

	found, err = models.FindSelectableLLMModel(db, r.Organization.ID, &factory.ID, "hosted::anthropic::claude-sonnet-4-6")
	require.NoError(t, err)
	assert.Equal(t, "hosted::anthropic::claude-sonnet-4-6", found.Key)
}

func Test__SelectableLLMRunnerComponent(t *testing.T) {
	t.Parallel()

	hosted, err := models.ParseSelectableLLMModelKey("hosted::openai::gpt-5")
	require.NoError(t, err)
	component, err := models.SelectableLLMRunnerComponent(hosted)
	require.NoError(t, err)
	assert.Equal(t, models.SuperPlaneRunnerComponent, component)
	assert.Equal(t, "hosted::openai::gpt-5", models.SelectableLLMRunnerModel(hosted))
	credentials, err := models.SelectableLLMRunnerCredentials(nil, uuid.Nil, hosted)
	require.NoError(t, err)
	assert.Nil(t, credentials)

	byok, err := models.ParseSelectableLLMModelKey("byok::anthropic::claude-sonnet-4-6")
	require.NoError(t, err)
	component, err = models.SelectableLLMRunnerComponent(byok)
	require.NoError(t, err)
	assert.Equal(t, "runnerClaudeCode", component)
}

func Test__SelectableLLMRunnerCredentialsUsesInstallationName(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	installationName := support.RandomName("claude")
	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "claude", installationName, map[string]any{})
	require.NoError(t, err)
	require.NoError(t, db.Model(integration).Update("state", models.IntegrationStateReady).Error)

	byok, err := models.ParseSelectableLLMModelKey("byok::anthropic::claude-sonnet-4-6")
	require.NoError(t, err)
	credentials, err := models.SelectableLLMRunnerCredentials(db, r.Organization.ID, byok)
	require.NoError(t, err)
	assert.Equal(t, "integration", credentials["source"])
	ref, ok := credentials["integration"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, installationName, ref["name"])
	assert.NotEqual(t, "claude", ref["name"])
}

func selectableKeys(models []models.SelectableLLMModel) []string {
	out := make([]string, 0, len(models))
	for _, model := range models {
		out = append(out, model.Key)
	}
	return out
}

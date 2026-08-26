package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__OrganizationBYOKModelAllowlist(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()

	empty, err := models.FindOrganizationBYOKModelAllowlist(db, r.Organization.ID, models.UsageProviderAnthropic)
	require.NoError(t, err)
	assert.Empty(t, empty.AllowedModels)

	saved, err := models.UpsertOrganizationBYOKModelAllowlist(db, r.Organization.ID, models.UsageProviderAnthropic, datatypes.JSONSlice[string]{
		"claude-sonnet-4-6",
		"claude-opus-4-6",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"claude-sonnet-4-6", "claude-opus-4-6"}, []string(saved.AllowedModels))

	_, err = models.UpsertOrganizationBYOKModelAllowlist(db, r.Organization.ID, models.UsageProviderAnthropic, datatypes.JSONSlice[string]{
		"claude-sonnet-4-6",
		"claude-sonnet-4-6",
	})
	require.Error(t, err)
}

func Test__ResolveSelectableLLMModels(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = database.Conn().Where("provider = ?", models.UsageProviderAnthropic).Delete(&models.HostedLLMProvider{})
		_ = database.Conn().Where("organization_id = ?", r.Organization.ID).Delete(&models.OrganizationBYOKModelAllowlist{})
		_ = database.Conn().Where("factory_id = ?", factory.ID).Delete(&models.FactoryLLMModelAllowlist{})
	})

	_, err = models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider:      models.UsageProviderAnthropic,
		Enabled:       true,
		APIKey:        []byte("encrypted"),
		AllowedModels: datatypes.JSONSlice[string]{"claude-sonnet-4-6", "claude-opus-4-6"},
	})
	require.NoError(t, err)

	hosted, err := models.ResolveSelectableLLMModels(db, r.Organization.ID, &factory.ID, models.UsageProviderAnthropic, models.UsageFundingSourceHosted)
	require.NoError(t, err)
	assert.Equal(t, []string{"claude-sonnet-4-6", "claude-opus-4-6"}, hosted)

	_, err = models.UpsertFactoryLLMModelAllowlist(db, r.Organization.ID, factory.ID, models.UsageProviderAnthropic, models.UsageFundingSourceHosted, datatypes.JSONSlice[string]{
		"claude-sonnet-4-6",
		"ignored-model",
	})
	require.Error(t, err)

	_, err = models.UpsertFactoryLLMModelAllowlist(db, r.Organization.ID, factory.ID, models.UsageProviderAnthropic, models.UsageFundingSourceHosted, datatypes.JSONSlice[string]{
		"claude-sonnet-4-6",
	})
	require.NoError(t, err)

	hosted, err = models.ResolveSelectableLLMModels(db, r.Organization.ID, &factory.ID, models.UsageProviderAnthropic, models.UsageFundingSourceHosted)
	require.NoError(t, err)
	assert.Equal(t, []string{"claude-sonnet-4-6"}, hosted)

	_, err = models.UpsertOrganizationBYOKModelAllowlist(db, r.Organization.ID, models.UsageProviderOpenAI, datatypes.JSONSlice[string]{
		"gpt-4.1",
		"gpt-4o",
	})
	require.NoError(t, err)

	byok, err := models.ResolveSelectableLLMModels(db, r.Organization.ID, nil, models.UsageProviderOpenAI, models.UsageFundingSourceBYOK)
	require.NoError(t, err)
	assert.Equal(t, []string{"gpt-4.1", "gpt-4o"}, byok)

	_, err = models.UpsertFactoryLLMModelAllowlist(db, r.Organization.ID, factory.ID, models.UsageProviderOpenAI, models.UsageFundingSourceBYOK, nil)
	require.NoError(t, err)

	byok, err = models.ResolveSelectableLLMModels(db, r.Organization.ID, &factory.ID, models.UsageProviderOpenAI, models.UsageFundingSourceBYOK)
	require.NoError(t, err)
	assert.Equal(t, []string{"gpt-4.1", "gpt-4o"}, byok)

	allowed, err := models.ModelIsSelectable(db, r.Organization.ID, &factory.ID, models.UsageProviderAnthropic, models.UsageFundingSourceHosted, "claude-opus-4-6")
	require.NoError(t, err)
	assert.False(t, allowed)

	allowed, err = models.ModelIsSelectable(db, r.Organization.ID, &factory.ID, models.UsageProviderAnthropic, models.UsageFundingSourceHosted, "")
	require.NoError(t, err)
	assert.False(t, allowed)
}

func Test__CompactModelIDs(t *testing.T) {
	assert.Equal(t, []string{"gpt-4o", "gpt-4.1"}, models.CompactModelIDs([]string{" gpt-4o ", "", "gpt-4o", "gpt-4.1"}))
}

func Test__IntersectModelIDs(t *testing.T) {
	assert.Equal(t, []string{"b"}, models.IntersectModelIDs([]string{"a", "b", "c"}, []string{"gone", "b", "b", ""}))
}

func Test__BYOKIntegrationAppName(t *testing.T) {
	name, err := models.BYOKIntegrationAppName(models.UsageProviderAnthropic)
	require.NoError(t, err)
	assert.Equal(t, "claude", name)
}

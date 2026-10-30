package contexts

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/llm"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__HostedLLMContext__AssertModelSelectable(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	_, err = models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{Provider: models.UsageProviderOpenAI, Enabled: true, APIKey: []byte("key"), AllowedModels: []string{"gpt-4.1", "gpt-4o"}})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Where("provider = ?", models.UsageProviderOpenAI).Delete(&models.HostedLLMProvider{}).Error
	})
	_, err = models.UpsertFactoryLLMModelAllowlist(db, r.Organization.ID, factory.ID, models.UsageProviderOpenAI, models.UsageFundingSourceHosted, []string{"gpt-4.1"})
	require.NoError(t, err)
	hosted := NewHostedLLMContext(db, r.Encryptor, r.Organization.ID, &factory.ID)
	require.ErrorContains(t, hosted.AssertModelSelectable(models.UsageProviderOpenAI, models.UsageFundingSourceHosted, ""), "model is required")
	require.NoError(t, hosted.AssertModelSelectable(models.UsageProviderOpenAI, models.UsageFundingSourceHosted, "gpt-4.1"))
	require.Error(t, hosted.AssertModelSelectable(models.UsageProviderOpenAI, models.UsageFundingSourceHosted, "gpt-4o"))
}

func Test__HostedLLMContext__ResolveOpenRouterRequiresManagementKey(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureOrganizationBYOK))
	db := database.Conn()
	t.Cleanup(func() {
		_ = db.Where("provider = ?", models.UsageProviderOpenRouter).Delete(&models.HostedLLMProvider{})
		_ = db.Where("provider = ?", models.UsageProviderAnthropic).Delete(&models.HostedLLMProvider{})
	})

	apiKey, err := llm.EncryptAPIKey(t.Context(), r.Encryptor, models.UsageProviderOpenRouter, "sk-or")
	require.NoError(t, err)
	_, err = models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider:      models.UsageProviderOpenRouter,
		Enabled:       true,
		APIKey:        apiKey,
		AllowedModels: datatypes.JSONSlice[string]{"anthropic/claude-sonnet-4-6"},
	})
	require.NoError(t, err)

	hosted := NewHostedLLMContext(db, r.Encryptor, r.Organization.ID, nil)
	_, err = hosted.Resolve(models.UsageProviderOpenRouter)
	require.Error(t, err)
	assert.ErrorIs(t, err, models.ErrHostedLLMProviderNoManagementKey)
}

func Test__HostedLLMContext__ResolveDecryptsOpenRouterManagementKey(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureOrganizationBYOK))
	db := database.Conn()
	t.Cleanup(func() {
		_ = db.Where("provider = ?", models.UsageProviderOpenRouter).Delete(&models.HostedLLMProvider{})
	})

	apiKey, err := llm.EncryptAPIKey(t.Context(), r.Encryptor, models.UsageProviderOpenRouter, "sk-or")
	require.NoError(t, err)
	mgmtKey, err := llm.EncryptManagementKey(t.Context(), r.Encryptor, models.UsageProviderOpenRouter, "sk-or-mgmt")
	require.NoError(t, err)
	_, err = models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider:      models.UsageProviderOpenRouter,
		Enabled:       true,
		APIKey:        apiKey,
		ManagementKey: mgmtKey,
		AllowedModels: datatypes.JSONSlice[string]{"anthropic/claude-sonnet-4-6"},
	})
	require.NoError(t, err)

	hosted := NewHostedLLMContext(db, r.Encryptor, r.Organization.ID, nil)
	access, err := hosted.Resolve(models.UsageProviderOpenRouter)
	require.NoError(t, err)
	assert.Equal(t, "sk-or", access.APIKey)
	assert.Equal(t, "sk-or-mgmt", access.ManagementKey)
}

func Test__HostedLLMContext__ResolveLeavesAnthropicManagementKeyEmpty(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureOrganizationBYOK))
	db := database.Conn()
	t.Cleanup(func() {
		_ = db.Where("provider = ?", models.UsageProviderAnthropic).Delete(&models.HostedLLMProvider{})
	})

	apiKey, err := llm.EncryptAPIKey(t.Context(), r.Encryptor, models.UsageProviderAnthropic, "sk-ant")
	require.NoError(t, err)
	_, err = models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider:      models.UsageProviderAnthropic,
		Enabled:       true,
		APIKey:        apiKey,
		AllowedModels: datatypes.JSONSlice[string]{"claude-sonnet-4-6"},
	})
	require.NoError(t, err)

	hosted := NewHostedLLMContext(db, r.Encryptor, r.Organization.ID, nil)
	access, err := hosted.Resolve(models.UsageProviderAnthropic)
	require.NoError(t, err)
	assert.Equal(t, "sk-ant", access.APIKey)
	assert.Empty(t, access.ManagementKey)
}

func Test__HostedLLMContext__DefaultModelUsesAnAllowedModelWhenInstallationDefaultIsExcluded(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureOrganizationBYOK))
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Where("provider = ?", models.UsageProviderAnthropic).Delete(&models.HostedLLMProvider{})
		_, _ = models.UpdateInstallationLLMSettings(db, models.InstallationLLMSettings{})
	})

	_, err = models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider:      models.UsageProviderAnthropic,
		Enabled:       true,
		APIKey:        []byte("test-hosted-key"),
		AllowedModels: datatypes.JSONSlice[string]{"claude-sonnet-4-6", "claude-opus-4-6"},
	})
	require.NoError(t, err)
	_, err = models.UpdateInstallationLLMSettings(db, models.InstallationLLMSettings{
		DefaultHostedProvider: stringPointer("anthropic"),
		DefaultHostedModel:    stringPointer("claude-sonnet-4-6"),
	})
	require.NoError(t, err)
	_, err = models.UpsertFactoryLLMModelAllowlist(db, r.Organization.ID, factory.ID, models.UsageProviderAnthropic, models.UsageFundingSourceHosted, datatypes.JSONSlice[string]{"claude-opus-4-6"})
	require.NoError(t, err)

	hosted := NewHostedLLMContext(db, r.Encryptor, r.Organization.ID, &factory.ID)
	model, err := hosted.DefaultModel()
	require.NoError(t, err)
	assert.Equal(t, "anthropic", model.Provider)
	assert.Equal(t, "claude-opus-4-6", model.Model)
}

func stringPointer(value string) *string {
	return &value
}

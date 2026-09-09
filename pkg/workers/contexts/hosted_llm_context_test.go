package contexts

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/llm"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__HostedLLMContext__AssertModelSelectable(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	t.Cleanup(func() {
		_ = database.Conn().Where("organization_id = ?", r.Organization.ID).Delete(&models.OrganizationBYOKModelAllowlist{})
	})

	hosted := NewHostedLLMContext(db, nil, r.Organization.ID, nil)
	err := hosted.AssertModelSelectable(models.UsageProviderOpenAI, models.UsageFundingSourceBYOK, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "model is required")

	_, err = models.UpsertOrganizationBYOKModelAllowlist(db, r.Organization.ID, models.UsageProviderOpenAI, datatypes.JSONSlice[string]{
		"gpt-4.1",
	})
	require.NoError(t, err)

	require.NoError(t, hosted.AssertModelSelectable(models.UsageProviderOpenAI, models.UsageFundingSourceBYOK, "gpt-4.1"))
	err = hosted.AssertModelSelectable(models.UsageProviderOpenAI, models.UsageFundingSourceBYOK, "gpt-4o")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "selected-model list")
}

func Test__HostedLLMContext__ResolveOpenRouterRequiresManagementKey(t *testing.T) {
	r := support.Setup(t)
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

package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__HostedLLMProviderAllowlist(t *testing.T) {
	_ = support.Setup(t)
	db := database.Conn()
	t.Cleanup(func() {
		_ = database.Conn().Where("provider = ?", models.UsageProviderAnthropic).Delete(&models.HostedLLMProvider{})
		_ = database.Conn().Where("provider = ?", models.UsageProviderOpenRouter).Delete(&models.HostedLLMProvider{})
	})

	saved, err := models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider:      models.UsageProviderAnthropic,
		Enabled:       true,
		APIKey:        []byte("encrypted"),
		AllowedModels: datatypes.JSONSlice[string]{"claude-sonnet-4-6", "claude-opus-4-6"},
	})
	require.NoError(t, err)
	assert.True(t, saved.AllowsModel("claude-sonnet-4-6"))
	assert.False(t, saved.AllowsModel("gpt-5-mini"))

	disabled, err := models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider:      models.UsageProviderOpenRouter,
		Enabled:       false,
		APIKey:        []byte("encrypted"),
		AllowedModels: datatypes.JSONSlice[string]{"openai/gpt-4.1"},
	})
	require.NoError(t, err)
	assert.True(t, disabled.OffersHostedModels())
	_, err = models.RequireEnabledHostedLLMProvider(db, models.UsageProviderOpenRouter)
	require.NoError(t, err)

	_, err = models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{Provider: "bedrock"})
	require.Error(t, err)
}

func Test__HasOfferedHostedLLMProvider(t *testing.T) {
	_ = support.Setup(t)
	db := database.Conn()
	existing, err := models.ListHostedLLMProviders(db)
	require.NoError(t, err)
	require.NoError(t, db.Where("provider <> ?", "").Delete(&models.HostedLLMProvider{}).Error)
	t.Cleanup(func() {
		_ = database.Conn().Where("provider <> ?", "").Delete(&models.HostedLLMProvider{}).Error
		for _, provider := range existing {
			_, _ = models.UpsertHostedLLMProvider(database.Conn(), provider)
		}
	})

	offered, err := models.HasOfferedHostedLLMProvider(db)
	require.NoError(t, err)
	assert.False(t, offered)

	_, err = models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider: models.UsageProviderAnthropic,
		Enabled:  true,
		APIKey:   []byte("encrypted"),
	})
	require.NoError(t, err)
	offered, err = models.HasOfferedHostedLLMProvider(db)
	require.NoError(t, err)
	assert.False(t, offered)

	_, err = models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider:      models.UsageProviderAnthropic,
		Enabled:       true,
		APIKey:        []byte("encrypted"),
		AllowedModels: datatypes.JSONSlice[string]{"claude-sonnet-4-6"},
	})
	require.NoError(t, err)
	offered, err = models.HasOfferedHostedLLMProvider(db)
	require.NoError(t, err)
	assert.True(t, offered)
}

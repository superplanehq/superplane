package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	eebilling "github.com/superplanehq/superplane/ee/billing"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/billing"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestInstallationCreditPurchaseFees(t *testing.T) {
	restoreInstallationLLMSettings(t)
	db := database.Conn()
	settings, err := billingmodels.GetInstallationLLMSettings(db)
	require.NoError(t, err)
	assert.Zero(t, settings.CreditPurchaseFeeBPS)
	assert.Zero(t, settings.CreditPurchaseFixedFeeCents)

	settings.CreditPurchaseFeeBPS = 500
	settings.CreditPurchaseFixedFeeCents = 50
	_, err = billingmodels.UpdateInstallationLLMSettings(db, *settings)
	require.NoError(t, err)
	saved, err := billingmodels.GetInstallationLLMSettings(db)
	require.NoError(t, err)
	assert.Equal(t, 500, saved.CreditPurchaseFeeBPS)
	assert.Equal(t, int64(50), saved.CreditPurchaseFixedFeeCents)

	for _, update := range []map[string]any{
		{"credit_purchase_fee_bps": -1},
		{"credit_purchase_fee_bps": 10000},
		{"credit_purchase_fixed_fee_cents": -1},
	} {
		t.Run("database rejects invalid fees", func(t *testing.T) {
			err := db.Model(&billingmodels.InstallationLLMSettings{}).Where("id = ?", saved.ID).Updates(update).Error
			require.Error(t, err)
		})
	}
}

// Billing and provider policy share a singleton table. A write through either
// owner must preserve changes made after the other owner read its columns.
func TestSharedLLMSettingsPreserveBillingAndProviderPolicies(t *testing.T) {
	tx := database.Conn().Begin()
	require.NoError(t, tx.Error)
	defer tx.Rollback()

	billingPolicy, err := billingmodels.GetInstallationLLMSettings(tx)
	require.NoError(t, err)
	providerPolicy, err := models.GetInstallationLLMSettings(tx)
	require.NoError(t, err)
	_, err = models.UpsertHostedLLMProvider(tx, models.HostedLLMProvider{
		Provider: models.UsageProviderAnthropic, Enabled: true, APIKey: []byte("encrypted"),
		AllowedModels: []string{"claude-sonnet-4-6"},
	})
	require.NoError(t, err)
	providerName, modelName := models.UsageProviderAnthropic, "claude-sonnet-4-6"
	providerPolicy.DefaultHostedProvider = &providerName
	providerPolicy.DefaultHostedModel = &modelName
	_, err = models.UpdateInstallationLLMSettings(tx, *providerPolicy)
	require.NoError(t, err)

	billingPolicy.MarkupBPS = 4321
	billingPolicy.CreditPurchaseFeeBPS = 321
	_, err = billingmodels.UpdateInstallationLLMSettings(tx, *billingPolicy)
	require.NoError(t, err)
	savedProvider, err := models.GetInstallationLLMSettings(tx)
	require.NoError(t, err)
	require.Equal(t, &providerName, savedProvider.DefaultHostedProvider)
	require.Equal(t, &modelName, savedProvider.DefaultHostedModel)

	// This provider policy was loaded before the billing write.
	providerPolicy.DefaultHostedProvider = nil
	providerPolicy.DefaultHostedModel = nil
	_, err = models.UpdateInstallationLLMSettings(tx, *providerPolicy)
	require.NoError(t, err)
	savedBilling, err := billingmodels.GetInstallationLLMSettings(tx)
	require.NoError(t, err)
	require.Equal(t, 4321, savedBilling.MarkupBPS)
	require.Equal(t, 321, savedBilling.CreditPurchaseFeeBPS)
}

func TestSharedInstallationMetadataPreservesBillingGracePeriod(t *testing.T) {
	tx := database.Conn().Begin()
	require.NoError(t, tx.Error)
	defer tx.Rollback()

	metadata, err := models.GetInstallationMetadata(tx)
	require.NoError(t, err)
	require.NoError(t, billingmodels.UpdatePastDueGraceDays(tx, 7))
	metadata.SignupsEnabled = !metadata.SignupsEnabled
	require.NoError(t, models.UpdateInstallationMetadata(tx, metadata))
	graceDays, err := billingmodels.GetPastDueGraceDays(tx)
	require.NoError(t, err)
	require.Equal(t, 7, graceDays)

	require.NoError(t, billingmodels.UpdatePastDueGraceDays(tx, 14))
	saved, err := models.GetInstallationMetadata(tx)
	require.NoError(t, err)
	require.Equal(t, metadata.SignupsEnabled, saved.SignupsEnabled)
	require.Equal(t, metadata.InstallationID, saved.InstallationID)
}

func TestProviderSettingsFirstPreservesBillingSingletonDefaults(t *testing.T) {
	restore := billing.Bind(eebilling.New(nil))
	defer restore()
	tx := database.Conn().Begin()
	require.NoError(t, tx.Error)
	defer tx.Rollback()
	require.NoError(t, tx.Where("id = ?", 1).Delete(&models.InstallationLLMSettings{}).Error)
	provider, err := models.GetInstallationLLMSettings(tx)
	require.NoError(t, err)
	require.Nil(t, provider.DefaultHostedProvider)
	require.Nil(t, provider.DefaultHostedModel)
	policy, err := billingmodels.GetInstallationLLMSettings(tx)
	require.NoError(t, err)
	require.Equal(t, billingmodels.DefaultWelcomeGrantCents, policy.WelcomeGrantCents)
	require.Equal(t, billingmodels.DefaultMarkupBPS, policy.MarkupBPS)
	require.Equal(t, billingmodels.DefaultWarningThresholdBPS, policy.WarningThresholdBPS)
	require.Zero(t, policy.CreditPurchaseFeeBPS)
	require.Zero(t, policy.CreditPurchaseFixedFeeCents)
}

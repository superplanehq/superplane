package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestInstallationCreditPurchaseFees(t *testing.T) {
	restoreInstallationLLMSettings(t)
	db := database.Conn()
	settings, err := models.GetInstallationLLMSettings(db)
	require.NoError(t, err)
	assert.Zero(t, settings.CreditPurchaseFeeBPS)
	assert.Zero(t, settings.CreditPurchaseFixedFeeCents)

	settings.CreditPurchaseFeeBPS = 500
	settings.CreditPurchaseFixedFeeCents = 50
	_, err = models.UpdateInstallationLLMSettings(db, *settings)
	require.NoError(t, err)
	saved, err := models.GetInstallationLLMSettings(db)
	require.NoError(t, err)
	assert.Equal(t, 500, saved.CreditPurchaseFeeBPS)
	assert.Equal(t, int64(50), saved.CreditPurchaseFixedFeeCents)

	for _, update := range []map[string]any{
		{"credit_purchase_fee_bps": -1},
		{"credit_purchase_fee_bps": 10000},
		{"credit_purchase_fixed_fee_cents": -1},
	} {
		t.Run("database rejects invalid fees", func(t *testing.T) {
			err := db.Model(&models.InstallationLLMSettings{}).Where("id = ?", saved.ID).Updates(update).Error
			require.Error(t, err)
		})
	}
}

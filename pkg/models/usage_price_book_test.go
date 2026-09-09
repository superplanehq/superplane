package models_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/usage/pricebook"
	"github.com/superplanehq/superplane/test/support"
)

func Test__LoadCurrentPriceBook__MatchesHardcodedRates(t *testing.T) {
	_ = support.Setup(t)
	require.NoError(t, models.LoadCurrentPriceBook(database.Conn()))
	t.Cleanup(pricebook.Reset)

	assert.Equal(t, int64(3_000_000), pricebook.EstimateMicros("anthropic", "claude-sonnet-4-6", 1_000_000, 0, 0, 0, 0))
	assert.Equal(t, int64(2_500_000), pricebook.EstimateMicros("openai", "gpt-4o", 1_000_000, 0, 0, 0, 0))
	assert.Equal(t, int64(15_000_000), pricebook.EstimateMicros("anthropic", "claude-3-opus-20240229", 1_000_000, 0, 0, 0, 0))
	assert.Equal(t, 10*pricebook.MicrosPerSecondE1Large, pricebook.EstimateComputeMicros("e1-large-amd64", "e1-large-amd64", 10))
	assert.Equal(t, int64(0), pricebook.EstimateComputeMicros("e1-large-amd64", "local", 10))
}

func Test__LoadCurrentPriceBook__LoadsExactModelRates(t *testing.T) {
	_ = support.Setup(t)
	t.Cleanup(pricebook.Reset)

	now := time.Now().UTC()
	version, err := models.NextPriceBookVersion(database.Conn(), now)
	require.NoError(t, err)
	err = models.InsertPriceBook(database.Conn(), models.UsagePriceBook{
		Version:     version,
		EffectiveAt: now,
	}, []models.UsagePriceBookRate{{
		UsageKind:            models.UsageKindModel,
		MatchKey:             "openai/gpt-test-exact",
		MatchMode:            models.UsagePriceBookMatchExact,
		InputCentsPerMillion: 42,
	}})
	require.NoError(t, err)
	require.NoError(t, models.LoadCurrentPriceBook(database.Conn()))

	assert.Equal(t, int64(420_000), pricebook.EstimateMicros("openrouter", "openai/gpt-test-exact", 1_000_000, 0, 0, 0, 0))

	t.Cleanup(func() {
		_ = database.Conn().Where("version = ?", version).Delete(&models.UsagePriceBookRate{})
		_ = database.Conn().Where("version = ?", version).Delete(&models.UsagePriceBook{})
	})
}

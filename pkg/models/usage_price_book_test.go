package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/usage/pricebook"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ListUsagePriceBooks__OrdersByEffectiveAtDesc(t *testing.T) {
	_ = support.Setup(t)
	db := database.DB(t.Context())

	books, err := models.ListUsagePriceBooks(db)
	require.NoError(t, err)
	require.NotEmpty(t, books)
	assert.Equal(t, "2026-09-09.1", books[0].Version)

	for i := 1; i < len(books); i++ {
		assert.False(t, books[i].EffectiveAt.After(books[i-1].EffectiveAt))
	}
}

func Test__FindUsagePriceBook(t *testing.T) {
	_ = support.Setup(t)
	db := database.DB(t.Context())

	book, err := models.FindUsagePriceBook(db, "2026-09-09.1")
	require.NoError(t, err)
	assert.Equal(t, "2026-09-09.1", book.Version)

	_, err = models.FindUsagePriceBook(db, "does-not-exist")
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func Test__ListUsagePriceBookRates(t *testing.T) {
	_ = support.Setup(t)
	db := database.DB(t.Context())

	rows, err := models.ListUsagePriceBookRates(db, "2026-09-09.1")
	require.NoError(t, err)
	require.NotEmpty(t, rows)

	hasModel := false
	hasCompute := false
	for i, row := range rows {
		assert.Equal(t, "2026-09-09.1", row.Version)
		if row.UsageKind == models.UsageKindModel {
			hasModel = true
		}
		if row.UsageKind == models.UsageKindCompute {
			hasCompute = true
		}
		if i == 0 {
			continue
		}
		previous := rows[i-1]
		if previous.MatchKey == row.MatchKey {
			assert.LessOrEqual(t, previous.MatchMode, row.MatchMode)
			continue
		}
		assert.Less(t, previous.MatchKey, row.MatchKey)
	}
	assert.True(t, hasModel)
	assert.True(t, hasCompute)
}

func Test__LoadCurrentPriceBook__MatchesHardcodedRates(t *testing.T) {
	_ = support.Setup(t)
	require.NoError(t, models.LoadCurrentPriceBook(database.Conn()))
	t.Cleanup(pricebook.Reset)

	assert.Equal(t, int64(3_000_000), pricebook.EstimateMicros("anthropic", "claude-sonnet-4-6", 1_000_000, 0, 0, 0, 0))
	assert.Equal(t, int64(2_500_000), pricebook.EstimateMicros("openai", "gpt-4o", 1_000_000, 0, 0, 0, 0))
	assert.Equal(t, int64(15_000_000), pricebook.EstimateMicros("anthropic", "claude-3-opus-20240229", 1_000_000, 0, 0, 0, 0))
	assert.True(t, pricebook.IsPriced("claude-sonnet-4-6"))
	assert.True(t, pricebook.IsPriced("gpt-4o"))
	assert.True(t, pricebook.IsPriced("openrouter/anthropic/claude-sonnet-4-6"))
	assert.Equal(t, 10*pricebook.MicrosPerSecondE1Large, pricebook.EstimateComputeMicros("e1-large-amd64", "e1-large-amd64", 10))
	assert.Equal(t, int64(0), pricebook.EstimateComputeMicros("e1-large-amd64", "local", 10))
}

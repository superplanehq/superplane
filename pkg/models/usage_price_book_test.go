package models_test

import (
	"testing"
	"time"

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

func Test__FindCurrentUsagePriceBook(t *testing.T) {
	_ = support.Setup(t)
	db := database.DB(t.Context())

	book, err := models.FindCurrentUsagePriceBook(db)
	require.NoError(t, err)
	assert.Equal(t, "2026-09-09.1", book.Version)
	assert.True(t, book.IsCurrent)
}

func Test__ApplyCatalogPrices__UpdatesPrefixAndKeepsMiniSeparate(t *testing.T) {
	rows := []models.UsagePriceBookRate{
		{UsageKind: models.UsageKindModel, MatchKey: "claude-sonnet", MatchMode: models.UsagePriceBookMatchPrefix, InputCentsPerMillion: 300, OutputCentsPerMillion: 1500},
		{UsageKind: models.UsageKindModel, MatchKey: "gpt-4o", MatchMode: models.UsagePriceBookMatchPrefix, InputCentsPerMillion: 250, OutputCentsPerMillion: 1000},
		{UsageKind: models.UsageKindModel, MatchKey: "gpt-4o-mini", MatchMode: models.UsagePriceBookMatchPrefix, InputCentsPerMillion: 15, OutputCentsPerMillion: 60},
		{UsageKind: models.UsageKindModel, MatchKey: "sonnet", MatchMode: models.UsagePriceBookMatchFamily, InputCentsPerMillion: 300, OutputCentsPerMillion: 1500},
		{UsageKind: models.UsageKindCompute, MatchKey: "e1-large-amd64", MatchMode: models.UsagePriceBookMatchExact, MicrosPerSecond: 70},
	}

	next, updated, added := models.ApplyCatalogPrices(rows, []models.CatalogModelPrice{
		{ModelID: "anthropic/claude-sonnet-4", Rate: pricebook.Rate{Input: 100, Output: 500}},
		{ModelID: "anthropic/claude-sonnet-4-6", Rate: pricebook.Rate{Input: 400, Output: 2000, CacheRead: 40, CacheWrite: 500}},
		{ModelID: "gpt-4o-mini", Rate: pricebook.Rate{Input: 20, Output: 80, CacheRead: 2}},
		{ModelID: "gpt-4o", Rate: pricebook.Rate{Input: 275, Output: 1100, CacheRead: 28}},
	})

	assert.Equal(t, 3, updated)
	assert.Equal(t, 0, added)
	assert.Equal(t, int64(70), next[findTestRate(t, next, models.UsageKindCompute, "e1-large-amd64")].MicrosPerSecond)
	assert.Equal(t, int64(400), next[findTestRate(t, next, models.UsageKindModel, "claude-sonnet")].InputCentsPerMillion)
	assert.Equal(t, int64(20), next[findTestRate(t, next, models.UsageKindModel, "gpt-4o-mini")].InputCentsPerMillion)
	assert.Equal(t, int64(275), next[findTestRate(t, next, models.UsageKindModel, "gpt-4o")].InputCentsPerMillion)
	assert.Equal(t, int64(300), next[findTestRate(t, next, models.UsageKindModel, "sonnet")].InputCentsPerMillion)
}

func Test__ApplyCatalogPrices__AddsUnknownPrefix(t *testing.T) {
	rows := []models.UsagePriceBookRate{
		{UsageKind: models.UsageKindModel, MatchKey: "claude-sonnet", MatchMode: models.UsagePriceBookMatchPrefix, InputCentsPerMillion: 300},
	}

	next, updated, added := models.ApplyCatalogPrices(rows, []models.CatalogModelPrice{
		{ModelID: "acme/lab-model-1", Rate: pricebook.Rate{Input: 12, Output: 24}},
	})

	assert.Equal(t, 0, updated)
	assert.Equal(t, 1, added)
	index := findTestRate(t, next, models.UsageKindModel, "lab-model-1")
	assert.Equal(t, models.UsagePriceBookMatchPrefix, next[index].MatchMode)
	assert.Equal(t, int64(12), next[index].InputCentsPerMillion)
}

func Test__NextUsagePriceBookVersion(t *testing.T) {
	_ = support.Setup(t)
	db := database.DB(t.Context())

	version, err := models.NextUsagePriceBookVersion(db, time.Date(2099, 1, 2, 15, 4, 5, 0, time.UTC))
	require.NoError(t, err)
	assert.Equal(t, "2099-01-02.1", version)
}

func Test__PublishAndActivateUsagePriceBook(t *testing.T) {
	_ = support.Setup(t)
	db := database.DB(t.Context())
	t.Cleanup(pricebook.Reset)
	t.Cleanup(func() {
		_ = models.ActivateUsagePriceBook(database.DB(t.Context()), "2026-09-09.1")
	})

	current, err := models.FindCurrentUsagePriceBook(db)
	require.NoError(t, err)
	rows, err := models.ListUsagePriceBookRates(db, current.Version)
	require.NoError(t, err)

	cloned := models.CloneUsagePriceBookRates(rows)
	cloned[findTestRate(t, cloned, models.UsageKindModel, "claude-sonnet")].InputCentsPerMillion = 350

	published, err := models.PublishUsagePriceBook(db, cloned)
	require.NoError(t, err)
	assert.True(t, published.IsCurrent)
	assert.NotEqual(t, "2026-09-09.1", published.Version)

	reloaded, err := models.FindCurrentUsagePriceBook(db)
	require.NoError(t, err)
	assert.Equal(t, published.Version, reloaded.Version)

	publishedRows, err := models.ListUsagePriceBookRates(db, published.Version)
	require.NoError(t, err)
	assert.Equal(t, int64(350), publishedRows[findTestRate(t, publishedRows, models.UsageKindModel, "claude-sonnet")].InputCentsPerMillion)
	assert.Equal(t, int64(3_500_000), pricebook.EstimateMicros("anthropic", "claude-sonnet-4-6", 1_000_000, 0, 0, 0, 0))

	require.NoError(t, models.ActivateUsagePriceBook(db, "2026-09-09.1"))
	restored, err := models.FindCurrentUsagePriceBook(db)
	require.NoError(t, err)
	assert.Equal(t, "2026-09-09.1", restored.Version)
	assert.Equal(t, int64(3_000_000), pricebook.EstimateMicros("anthropic", "claude-sonnet-4-6", 1_000_000, 0, 0, 0, 0))
}

func findTestRate(t *testing.T, rows []models.UsagePriceBookRate, kind, matchKey string) int {
	t.Helper()
	for i, row := range rows {
		if row.UsageKind == kind && row.MatchKey == matchKey {
			return i
		}
	}
	t.Fatalf("rate %s %s not found", kind, matchKey)
	return -1
}

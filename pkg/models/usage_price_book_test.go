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

const seedUsagePriceBookVersion = "2026-09-23.1"

func Test__ListUsagePriceBooks__OrdersByEffectiveAtDesc(t *testing.T) {
	_ = support.Setup(t)
	db := database.DB(t.Context())

	books, err := models.ListUsagePriceBooks(db)
	require.NoError(t, err)
	require.NotEmpty(t, books)
	assert.Equal(t, seedUsagePriceBookVersion, books[0].Version)

	for i := 1; i < len(books); i++ {
		assert.False(t, books[i].EffectiveAt.After(books[i-1].EffectiveAt))
	}
}

func Test__FindUsagePriceBook(t *testing.T) {
	_ = support.Setup(t)
	db := database.DB(t.Context())

	book, err := models.FindUsagePriceBook(db, seedUsagePriceBookVersion)
	require.NoError(t, err)
	assert.Equal(t, seedUsagePriceBookVersion, book.Version)

	_, err = models.FindUsagePriceBook(db, "does-not-exist")
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func Test__ListUsagePriceBookRates(t *testing.T) {
	_ = support.Setup(t)
	db := database.DB(t.Context())

	rows, err := models.ListUsagePriceBookRates(db, seedUsagePriceBookVersion)
	require.NoError(t, err)
	require.NotEmpty(t, rows)

	hasModel := false
	hasCompute := false
	for i, row := range rows {
		assert.Equal(t, seedUsagePriceBookVersion, row.Version)
		if row.UsageKind == models.UsageKindModel {
			hasModel = true
			assert.NotEmpty(t, row.Provider)
			assert.Equal(t, models.UsagePriceBookMatchExact, row.MatchMode)
		}
		if row.UsageKind == models.UsageKindCompute {
			hasCompute = true
		}
		if i == 0 {
			continue
		}
		previous := rows[i-1]
		if previous.Provider != row.Provider {
			assert.Less(t, previous.Provider, row.Provider)
			continue
		}
		if previous.MatchKey == row.MatchKey {
			assert.LessOrEqual(t, previous.MatchMode, row.MatchMode)
			continue
		}
		assert.Less(t, previous.MatchKey, row.MatchKey)
	}
	assert.True(t, hasModel)
	assert.True(t, hasCompute)
}

func Test__LoadCurrentPriceBook__MatchesSeededRates(t *testing.T) {
	_ = support.Setup(t)
	require.NoError(t, models.LoadCurrentPriceBook(database.Conn()))
	t.Cleanup(pricebook.Reset)

	assert.Equal(t, int64(3_000_000), pricebook.EstimateMicros("anthropic", "claude-sonnet-4-6", 1_000_000, 0, 0, 0, 0))
	assert.Equal(t, int64(2_500_000), pricebook.EstimateMicros("openai", "gpt-4o", 1_000_000, 0, 0, 0, 0))
	assert.Equal(t, int64(2_000_000), pricebook.EstimateMicros("openrouter", "x-ai/grok-4.6", 1_000_000, 0, 0, 0, 0))
	assert.Equal(t, int64(2_000_000), pricebook.EstimateMicros("openrouter", "openrouter/x-ai/grok-4.6", 1_000_000, 0, 0, 0, 0))
	assert.Equal(t, int64(15_000_000), pricebook.EstimateMicros("anthropic", "claude-3-opus-20240229", 1_000_000, 0, 0, 0, 0))
	assert.True(t, pricebook.IsPriced("anthropic", "claude-sonnet-4-6"))
	assert.True(t, pricebook.IsPriced("openai", "gpt-4o"))
	assert.True(t, pricebook.IsPriced("openrouter", "anthropic/claude-sonnet-4.6"))
	assert.Equal(t, 10*pricebook.MicrosPerSecondE1Large, pricebook.EstimateComputeMicros("e1-large-amd64", "e1-large-amd64", 10))
	assert.Equal(t, int64(0), pricebook.EstimateComputeMicros("e1-large-amd64", "local", 10))
}

func Test__FindCurrentUsagePriceBook(t *testing.T) {
	_ = support.Setup(t)
	db := database.DB(t.Context())

	book, err := models.FindCurrentUsagePriceBook(db)
	require.NoError(t, err)
	assert.Equal(t, seedUsagePriceBookVersion, book.Version)
	assert.True(t, book.IsCurrent)
}

func Test__ApplyCatalogPrices__UpdatesExactAndKeepsOtherProviders(t *testing.T) {
	rows := []models.UsagePriceBookRate{
		{UsageKind: models.UsageKindModel, Provider: "openrouter", MatchKey: "anthropic/claude-sonnet-4.6", MatchMode: models.UsagePriceBookMatchExact, InputCentsPerMillion: 300, OutputCentsPerMillion: 1500},
		{UsageKind: models.UsageKindModel, Provider: "anthropic", MatchKey: "claude-sonnet-4-6", MatchMode: models.UsagePriceBookMatchExact, InputCentsPerMillion: 300, OutputCentsPerMillion: 1500},
		{UsageKind: models.UsageKindCompute, MatchKey: "e1-large-amd64", MatchMode: models.UsagePriceBookMatchExact, MicrosPerSecond: 70},
	}

	next, updated, added := models.ApplyCatalogPrices(rows, []models.CatalogModelPrice{
		{Provider: "openrouter", ModelID: "anthropic/claude-sonnet-4.6", Rate: pricebook.Rate{Input: 400, Output: 2000, CacheRead: 40, CacheWrite: 500}},
		{Provider: "openrouter", ModelID: "x-ai/grok-4.6", Rate: pricebook.Rate{Input: 200, Output: 600}},
	})

	assert.Equal(t, 1, updated)
	assert.Equal(t, 1, added)
	assert.Equal(t, int64(70), next[findTestRate(t, next, models.UsageKindCompute, "", "e1-large-amd64")].MicrosPerSecond)
	assert.Equal(t, int64(400), next[findTestRate(t, next, models.UsageKindModel, "openrouter", "anthropic/claude-sonnet-4.6")].InputCentsPerMillion)
	assert.Equal(t, int64(300), next[findTestRate(t, next, models.UsageKindModel, "anthropic", "claude-sonnet-4-6")].InputCentsPerMillion)
	assert.Equal(t, int64(200), next[findTestRate(t, next, models.UsageKindModel, "openrouter", "x-ai/grok-4.6")].InputCentsPerMillion)
}

func Test__ApplyCatalogPrices__AddsUnknownCatalogID(t *testing.T) {
	rows := []models.UsagePriceBookRate{
		{UsageKind: models.UsageKindModel, Provider: "openrouter", MatchKey: "anthropic/claude-sonnet-4.6", MatchMode: models.UsagePriceBookMatchExact, InputCentsPerMillion: 300},
	}

	next, updated, added := models.ApplyCatalogPrices(rows, []models.CatalogModelPrice{
		{Provider: "openrouter", ModelID: "acme/lab-model-1", Rate: pricebook.Rate{Input: 12, Output: 24}},
	})

	assert.Equal(t, 0, updated)
	assert.Equal(t, 1, added)
	index := findTestRate(t, next, models.UsageKindModel, "openrouter", "acme/lab-model-1")
	assert.Equal(t, models.UsagePriceBookMatchExact, next[index].MatchMode)
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

	current, err := models.FindCurrentUsagePriceBook(db)
	require.NoError(t, err)
	rows, err := models.ListUsagePriceBookRates(db, current.Version)
	require.NoError(t, err)

	cloned := models.CloneUsagePriceBookRates(rows)
	cloned[findTestRate(t, cloned, models.UsageKindModel, "anthropic", "claude-sonnet-4-6")].InputCentsPerMillion = 350

	published, err := models.PublishUsagePriceBook(db, cloned, current.Version)
	require.NoError(t, err)
	require.NoError(t, models.LoadCurrentPriceBook(db))
	assert.True(t, published.IsCurrent)
	assert.NotEqual(t, seedUsagePriceBookVersion, published.Version)

	reloaded, err := models.FindCurrentUsagePriceBook(db)
	require.NoError(t, err)
	assert.Equal(t, published.Version, reloaded.Version)

	publishedRows, err := models.ListUsagePriceBookRates(db, published.Version)
	require.NoError(t, err)
	assert.Equal(t, int64(350), publishedRows[findTestRate(t, publishedRows, models.UsageKindModel, "anthropic", "claude-sonnet-4-6")].InputCentsPerMillion)
	assert.Equal(t, int64(3_500_000), pricebook.EstimateMicros("anthropic", "claude-sonnet-4-6", 1_000_000, 0, 0, 0, 0))

	require.NoError(t, models.ActivateUsagePriceBook(db, seedUsagePriceBookVersion))
	require.NoError(t, models.LoadCurrentPriceBook(db))
	restored, err := models.FindCurrentUsagePriceBook(db)
	require.NoError(t, err)
	assert.Equal(t, seedUsagePriceBookVersion, restored.Version)
	assert.Equal(t, int64(3_000_000), pricebook.EstimateMicros("anthropic", "claude-sonnet-4-6", 1_000_000, 0, 0, 0, 0))
}

func Test__PublishUsagePriceBook__RejectsStaleBaseVersion(t *testing.T) {
	_ = support.Setup(t)
	db := database.DB(t.Context())
	t.Cleanup(pricebook.Reset)

	current, err := models.FindCurrentUsagePriceBook(db)
	require.NoError(t, err)
	rows, err := models.ListUsagePriceBookRates(db, current.Version)
	require.NoError(t, err)

	cloned := models.CloneUsagePriceBookRates(rows)
	_, err = models.PublishUsagePriceBook(db, cloned, "missing-base")
	require.ErrorIs(t, err, models.ErrUsagePriceBookConflict)

	reloaded, err := models.FindCurrentUsagePriceBook(db)
	require.NoError(t, err)
	assert.Equal(t, current.Version, reloaded.Version)
}

func Test__DeleteUsagePriceBook(t *testing.T) {
	_ = support.Setup(t)
	db := database.DB(t.Context())
	t.Cleanup(pricebook.Reset)

	current, err := models.FindCurrentUsagePriceBook(db)
	require.NoError(t, err)
	rows, err := models.ListUsagePriceBookRates(db, current.Version)
	require.NoError(t, err)

	published, err := models.PublishUsagePriceBook(db, models.CloneUsagePriceBookRates(rows), current.Version)
	require.NoError(t, err)

	err = models.DeleteUsagePriceBook(db, published.Version)
	require.ErrorIs(t, err, models.ErrUsagePriceBookCurrent)

	require.NoError(t, models.ActivateUsagePriceBook(db, current.Version))
	require.NoError(t, models.DeleteUsagePriceBook(db, published.Version))

	_, err = models.FindUsagePriceBook(db, published.Version)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	err = models.DeleteUsagePriceBook(db, current.Version)
	require.ErrorIs(t, err, models.ErrUsagePriceBookCurrent)
}

func findTestRate(t *testing.T, rows []models.UsagePriceBookRate, kind, provider, matchKey string) int {
	t.Helper()
	for i, row := range rows {
		if row.UsageKind == kind && row.Provider == provider && row.MatchKey == matchKey {
			return i
		}
	}
	t.Fatalf("rate %s %s %s not found", kind, provider, matchKey)
	return -1
}

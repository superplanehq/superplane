package pricebooksync_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/llm"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/usage/pricebook"
	"github.com/superplanehq/superplane/pkg/usage/pricebooksync"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/contexts"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__FilterCatalogPrices__MatchesAllowlistAndNormalizedIDs(t *testing.T) {
	prices := []llm.CatalogPrice{
		{ID: "anthropic/claude-sonnet-4-6", Rate: pricebook.Rate{Input: 400}},
		{ID: "openai/gpt-4o", Rate: pricebook.Rate{Input: 250}},
		{ID: "openai/gpt-4o-mini", Rate: pricebook.Rate{Input: 15}},
	}

	filtered := pricebooksync.FilterCatalogPrices(prices, []string{"anthropic/claude-sonnet-4-6", "gpt-4o-mini"})
	require.Len(t, filtered, 2)
	assert.Equal(t, "anthropic/claude-sonnet-4-6", filtered[0].ModelID)
	assert.Equal(t, "openai/gpt-4o-mini", filtered[1].ModelID)
}

func Test__Sync__SkipsPublishWhenRatesUnchanged(t *testing.T) {
	_ = support.Setup(t)
	db := database.DB(t.Context())
	t.Cleanup(func() {
		_ = models.ActivateUsagePriceBook(database.DB(t.Context()), "2026-09-09.1")
	})

	current, err := models.FindCurrentUsagePriceBook(db)
	require.NoError(t, err)

	upsertOpenRouterProvider(t, db, nil)

	svc := pricebooksync.New(crypto.NewNoOpEncryptor(), openRouterHTTPContext(sonnetCatalogBody("0.000003", "0.000015")))
	result, err := svc.Sync(t.Context(), db, pricebooksync.Options{SkipWhenUnchanged: true})
	require.NoError(t, err)
	assert.False(t, result.Published)
	assert.Equal(t, current.Version, result.Book.Version)

	books, err := models.ListUsagePriceBooks(db)
	require.NoError(t, err)
	assert.Equal(t, current.Version, books[0].Version)
}

func Test__Sync__PublishesWhenRatesChange(t *testing.T) {
	_ = support.Setup(t)
	db := database.DB(t.Context())
	t.Cleanup(func() {
		_ = models.ActivateUsagePriceBook(database.DB(t.Context()), "2026-09-09.1")
	})
	t.Cleanup(pricebook.Reset)

	current, err := models.FindCurrentUsagePriceBook(db)
	require.NoError(t, err)

	upsertOpenRouterProvider(t, db, nil)

	svc := pricebooksync.New(crypto.NewNoOpEncryptor(), openRouterHTTPContext(sonnetCatalogBody("0.000004", "0.000015")))
	result, err := svc.Sync(t.Context(), db, pricebooksync.Options{SkipWhenUnchanged: true})
	require.NoError(t, err)
	assert.True(t, result.Published)
	assert.NotEqual(t, current.Version, result.Book.Version)
}

func Test__Sync__PublishesWhenModelAdded(t *testing.T) {
	_ = support.Setup(t)
	db := database.DB(t.Context())
	t.Cleanup(func() {
		_ = models.ActivateUsagePriceBook(database.DB(t.Context()), "2026-09-09.1")
	})
	t.Cleanup(pricebook.Reset)

	current, err := models.FindCurrentUsagePriceBook(db)
	require.NoError(t, err)

	upsertOpenRouterProvider(t, db, datatypes.NewJSONSlice([]string{"acme/lab-model-1"}))

	svc := pricebooksync.New(crypto.NewNoOpEncryptor(), openRouterHTTPContext(`{
		"data":[
			{
				"id":"acme/lab-model-1",
				"pricing":{"prompt":"0.000012","completion":"0.000024"}
			}
		]
	}`))
	result, err := svc.Sync(t.Context(), db, pricebooksync.Options{SkipWhenUnchanged: true})
	require.NoError(t, err)
	assert.True(t, result.Published)
	assert.Equal(t, 1, result.AddedCount)
	assert.NotEqual(t, current.Version, result.Book.Version)
}

func upsertOpenRouterProvider(t *testing.T, db *gorm.DB, allowlist datatypes.JSONSlice[string]) {
	t.Helper()

	_, err := models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider:      models.UsageProviderOpenRouter,
		Enabled:       true,
		APIKey:        []byte("sk-or-test"),
		AllowedModels: allowlist,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Delete(&models.HostedLLMProvider{}, "provider = ?", models.UsageProviderOpenRouter)
	})
}

func sonnetCatalogBody(prompt, completion string) string {
	return `{
		"data":[
			{
				"id":"anthropic/claude-sonnet-4-6",
				"pricing":{"prompt":"` + prompt + `","completion":"` + completion + `"}
			}
		]
	}`
}

func openRouterHTTPContext(body string) *contexts.HTTPContext {
	return &contexts.HTTPContext{
		Responses: []*http.Response{jsonResponse(http.StatusOK, body)},
	}
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

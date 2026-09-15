package public

import (
	"encoding/json"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/llm"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/usage/pricebook"
)

func TestAdminGetPriceBooks(t *testing.T) {
	server, _, token := setupAdminTestServer(t)

	t.Run("non-admin gets 404", func(t *testing.T) {
		account, err := models.CreateAccount("Regular User", "regular-price-books@example.com")
		require.NoError(t, err)
		signer := jwt.NewSigner("test-client-secret")
		regularToken, err := authentication.GenerateAccountToken(signer, account.ID.String(), time.Now(), time.Hour)
		require.NoError(t, err)

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/price-books",
			authCookie: regularToken,
		})
		assert.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("admin loads the current price book", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/price-books",
			authCookie: token,
		})
		assert.Equal(t, http.StatusOK, response.Code)

		var body adminPriceBooksResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		assert.Equal(t, "2026-09-09.1", body.CurrentVersion)
		assert.Equal(t, "2026-09-09.1", body.Version)
		require.GreaterOrEqual(t, len(body.Versions), 2)
		assert.Equal(t, "2026-09-09.1", body.Versions[0].Version)
		require.NotEmpty(t, body.Models)
		require.NotEmpty(t, body.VMs)

		assert.True(t, containsModelRate(body.Models, "claude-sonnet"))
		assert.True(t, containsVMRate(body.VMs, "e1-large-amd64", 70))
		assert.True(t, modelRatesAreSorted(body.Models))
		assert.True(t, vmRatesAreSorted(body.VMs))
	})

	t.Run("admin loads an explicit older version", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/price-books?version=2026-08-31.1",
			authCookie: token,
		})
		assert.Equal(t, http.StatusOK, response.Code)

		var body adminPriceBooksResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		assert.Equal(t, "2026-09-09.1", body.CurrentVersion)
		assert.Equal(t, "2026-08-31.1", body.Version)
		require.NotEmpty(t, body.Models)
		require.NotEmpty(t, body.VMs)
		assert.True(t, containsModelRate(body.Models, "claude-sonnet"))
		assert.True(t, containsVMRate(body.VMs, "e1-large-amd64", 0))
		assert.False(t, containsModelRate(body.Models, "gemini-2.5-pro"))
	})

	t.Run("unknown version returns 404", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/price-books?version=does-not-exist",
			authCookie: token,
		})
		assert.Equal(t, http.StatusNotFound, response.Code)
	})
}

func TestAdminSaveAndActivatePriceBooks(t *testing.T) {
	server, _, token := setupAdminTestServer(t)
	t.Cleanup(func() {
		_ = models.ActivateUsagePriceBook(database.Conn(), "2026-09-09.1")
	})

	getBody := execRequest(server, requestParams{
		method:     "GET",
		path:       "/admin/api/price-books",
		authCookie: token,
	})
	require.Equal(t, http.StatusOK, getBody.Code)
	var current adminPriceBooksResponse
	require.NoError(t, json.Unmarshal(getBody.Body.Bytes(), &current))

	current.Models[0].InputCentsPerMillion = current.Models[0].InputCentsPerMillion + 1
	current.VMs = append(current.VMs, adminPriceBookVMRate{
		MatchKey:        "e1-test-amd64",
		MatchMode:       "exact",
		MicrosPerSecond: 9,
	})

	payload, err := json.Marshal(adminPriceBooksSaveRequest{
		BaseVersion: current.Version,
		Models:      current.Models,
		VMs:         current.VMs,
	})
	require.NoError(t, err)

	save := execRequest(server, requestParams{
		method:      "PUT",
		path:        "/admin/api/price-books",
		authCookie:  token,
		body:        payload,
		contentType: "application/json",
	})
	require.Equal(t, http.StatusOK, save.Code)

	var saved adminPriceBooksResponse
	require.NoError(t, json.Unmarshal(save.Body.Bytes(), &saved))
	assert.NotEqual(t, "2026-09-09.1", saved.CurrentVersion)
	assert.Equal(t, saved.CurrentVersion, saved.Version)
	assert.True(t, containsVMRate(saved.VMs, "e1-test-amd64", 9))

	activate := execRequest(server, requestParams{
		method:      "PUT",
		path:        "/admin/api/price-books/current",
		authCookie:  token,
		body:        []byte(`{"version":"2026-09-09.1"}`),
		contentType: "application/json",
	})
	require.Equal(t, http.StatusOK, activate.Code)
	var restored adminPriceBooksResponse
	require.NoError(t, json.Unmarshal(activate.Body.Bytes(), &restored))
	assert.Equal(t, "2026-09-09.1", restored.CurrentVersion)
	assert.Equal(t, "2026-09-09.1", restored.Version)
	assert.False(t, containsVMRate(restored.VMs, "e1-test-amd64", 9))
}

func TestAdminSavePriceBooks_RejectsStaleBaseVersion(t *testing.T) {
	server, _, token := setupAdminTestServer(t)
	t.Cleanup(func() {
		_ = models.ActivateUsagePriceBook(database.Conn(), "2026-09-09.1")
	})

	getBody := execRequest(server, requestParams{
		method:     "GET",
		path:       "/admin/api/price-books",
		authCookie: token,
	})
	require.Equal(t, http.StatusOK, getBody.Code)
	var current adminPriceBooksResponse
	require.NoError(t, json.Unmarshal(getBody.Body.Bytes(), &current))

	payload, err := json.Marshal(adminPriceBooksSaveRequest{
		BaseVersion: "2026-08-31.1",
		Models:      current.Models,
		VMs:         current.VMs,
	})
	require.NoError(t, err)

	save := execRequest(server, requestParams{
		method:      "PUT",
		path:        "/admin/api/price-books",
		authCookie:  token,
		body:        payload,
		contentType: "application/json",
	})
	require.Equal(t, http.StatusConflict, save.Code)
	assert.Contains(t, save.Body.String(), "The current price book changed")

	after := execRequest(server, requestParams{
		method:     "GET",
		path:       "/admin/api/price-books",
		authCookie: token,
	})
	require.Equal(t, http.StatusOK, after.Code)
	var stillCurrent adminPriceBooksResponse
	require.NoError(t, json.Unmarshal(after.Body.Bytes(), &stillCurrent))
	assert.Equal(t, current.CurrentVersion, stillCurrent.CurrentVersion)
}

func TestAdminSavePriceBooks_RequiresBaseVersion(t *testing.T) {
	server, _, token := setupAdminTestServer(t)

	payload, err := json.Marshal(adminPriceBooksSaveRequest{
		Models: []adminPriceBookModelRate{{
			MatchKey:  "claude-sonnet",
			MatchMode: "prefix",
		}},
	})
	require.NoError(t, err)

	save := execRequest(server, requestParams{
		method:      "PUT",
		path:        "/admin/api/price-books",
		authCookie:  token,
		body:        payload,
		contentType: "application/json",
	})
	require.Equal(t, http.StatusBadRequest, save.Code)
	assert.Contains(t, save.Body.String(), "Base version is required")
}

func TestAdminSyncPriceBooks_RequiresPricedProvider(t *testing.T) {
	server, _, token := setupAdminTestServer(t)

	response := execRequest(server, requestParams{
		method:     "POST",
		path:       "/admin/api/price-books/sync",
		authCookie: token,
	})
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Contains(t, response.Body.String(), "No enabled provider publishes catalog prices")
}

func TestFilterCatalogPrices_MatchesAllowlistAndNormalizedIDs(t *testing.T) {
	prices := []llm.CatalogPrice{
		{ID: "anthropic/claude-sonnet-4-6", Rate: pricebook.Rate{Input: 400}},
		{ID: "openai/gpt-4o", Rate: pricebook.Rate{Input: 250}},
		{ID: "openai/gpt-4o-mini", Rate: pricebook.Rate{Input: 15}},
	}

	filtered := filterCatalogPrices(prices, []string{"anthropic/claude-sonnet-4-6", "gpt-4o-mini"})
	require.Len(t, filtered, 2)
	assert.Equal(t, "anthropic/claude-sonnet-4-6", filtered[0].ModelID)
	assert.Equal(t, "openai/gpt-4o-mini", filtered[1].ModelID)
}

func containsModelRate(rates []adminPriceBookModelRate, matchKey string) bool {
	return slices.ContainsFunc(rates, func(rate adminPriceBookModelRate) bool {
		return rate.MatchKey == matchKey
	})
}

func containsVMRate(rates []adminPriceBookVMRate, matchKey string, microsPerSecond int64) bool {
	return slices.ContainsFunc(rates, func(rate adminPriceBookVMRate) bool {
		return rate.MatchKey == matchKey && rate.MicrosPerSecond == microsPerSecond
	})
}

func modelRatesAreSorted(rates []adminPriceBookModelRate) bool {
	for i := 1; i < len(rates); i++ {
		previous := rates[i-1]
		current := rates[i]
		if previous.MatchKey == current.MatchKey {
			if previous.MatchMode > current.MatchMode {
				return false
			}
			continue
		}
		if previous.MatchKey > current.MatchKey {
			return false
		}
	}
	return true
}

func vmRatesAreSorted(rates []adminPriceBookVMRate) bool {
	for i := 1; i < len(rates); i++ {
		if rates[i-1].MatchKey > rates[i].MatchKey {
			return false
		}
	}
	return true
}

package public

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/usage/pricebook"
	"github.com/superplanehq/superplane/pkg/usage/pricesync"
)

func TestAdminPriceBook(t *testing.T) {
	server, _, token := setupAdminTestServer(t)
	t.Cleanup(pricebook.Reset)

	t.Run("non-admin gets 404", func(t *testing.T) {
		account, err := models.CreateAccount("Regular User", "regular-price-book@example.com")
		require.NoError(t, err)
		signer := jwt.NewSigner("test-client-secret")
		regularToken, err := authentication.GenerateAccountToken(signer, account.ID.String(), time.Now(), time.Hour)
		require.NoError(t, err)

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/installation/price-book",
			authCookie: regularToken,
		})
		assert.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("admin can load the current price book", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/installation/price-book",
			authCookie: token,
		})
		assert.Equal(t, http.StatusOK, response.Code)

		var book priceBookResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &book))
		assert.NotEmpty(t, book.Version)
		assert.Greater(t, book.ModelRateCount, 0)
		assert.Greater(t, book.ComputeRateCount, 0)
		assert.Equal(t, book.ModelRateCount+book.ComputeRateCount, len(book.Rates))
	})

	t.Run("admin can scan and publish a new price book", func(t *testing.T) {
		now := time.Date(2099, 1, 2, 16, 0, 0, 0, time.UTC)
		server.newPriceBookScanner = func() pricesync.Scanner {
			return pricesync.NewScanner(pricesync.Options{
				Now: func() time.Time { return now },
				Models: []pricesync.ModelSource{
					stubAdminModelSource{rates: []pricesync.ModelRate{{
						MatchKey:  "openai/gpt-scan",
						MatchMode: models.UsagePriceBookMatchExact,
						Rate:      pricebook.Rate{Input: 12, Output: 24},
					}}},
				},
				Compute: pricesync.CatalogSource{},
			})
		}
		t.Cleanup(func() { server.newPriceBookScanner = nil })

		response := execRequest(server, requestParams{
			method:     "POST",
			path:       "/admin/api/installation/price-book/scan",
			authCookie: token,
		})
		assert.Equal(t, http.StatusOK, response.Code)

		var book priceBookResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &book))
		assert.Equal(t, "2099-01-02.1", book.Version)
		assert.Equal(t, 1, book.ModelRateCount)
		assert.Greater(t, book.ComputeRateCount, 0)
		assert.Contains(t, book.Sources, "superplane-catalog")
		assert.Equal(t, int64(120_000), pricebook.EstimateMicros("openrouter", "openai/gpt-scan", 1_000_000, 0, 0, 0, 0))
		t.Cleanup(func() {
			_ = database.Conn().Where("version LIKE ?", "2099-01-02.%").Delete(&models.UsagePriceBookRate{})
			_ = database.Conn().Where("version LIKE ?", "2099-01-02.%").Delete(&models.UsagePriceBook{})
		})
	})
}

type stubAdminModelSource struct {
	rates []pricesync.ModelRate
}

func (stubAdminModelSource) Name() string { return "openrouter" }

func (s stubAdminModelSource) ScanModels(context.Context) ([]pricesync.ModelRate, error) {
	return s.rates, nil
}

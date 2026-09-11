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
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
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

package public

import (
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
)

func TestAdminIntakeCatalog(t *testing.T) {
	server, r, token := setupAdminTestServer(t)
	db := database.Conn()

	request := func(method, path string, body any) (int, []byte) {
		params := requestParams{method: method, path: path, authCookie: token}
		if body != nil {
			payload, err := json.Marshal(body)
			require.NoError(t, err)
			params.body = payload
			params.contentType = "application/json"
		}
		response := execRequest(server, params)
		return response.Code, response.Body.Bytes()
	}

	decodeEntry := func(t *testing.T, body []byte) adminIntakeCatalogEntry {
		var entry adminIntakeCatalogEntry
		require.NoError(t, json.Unmarshal(body, &entry))
		return entry
	}

	restoreDatadog := func(t *testing.T) {
		original, err := models.FindIntakeCatalogEntry(db, models.FactoryIntakeSourceDatadog)
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, db.Model(&models.IntakeCatalogEntry{}).Where("key = ?", original.Key).Updates(map[string]any{
				"status":      original.Status,
				"status_note": original.StatusNote,
			}).Error)
		})
	}

	t.Run("non-admin gets 404", func(t *testing.T) {
		account, err := models.CreateAccount("Regular User", "regular-intakes@example.com")
		require.NoError(t, err)
		signer := jwt.NewSigner("test-client-secret")
		regularToken, err := authentication.GenerateAccountToken(signer, account.ID.String(), time.Now(), time.Hour)
		require.NoError(t, err)

		for _, params := range []requestParams{
			{method: "GET", path: "/admin/api/intake-catalog"},
			{method: "PATCH", path: "/admin/api/intake-catalog/datadog", body: []byte(`{"status":"ga"}`), contentType: "application/json"},
		} {
			params.authCookie = regularToken
			response := execRequest(server, params)
			assert.Equal(t, http.StatusNotFound, response.Code)
		}
	})

	t.Run("lists seeded intakes with implementation state", func(t *testing.T) {
		code, body := request("GET", "/admin/api/intake-catalog", nil)
		require.Equal(t, http.StatusOK, code)

		var response struct {
			Entries []adminIntakeCatalogEntry `json:"entries"`
		}
		require.NoError(t, json.Unmarshal(body, &response))
		byKey := map[string]adminIntakeCatalogEntry{}
		for _, entry := range response.Entries {
			byKey[entry.Key] = entry
		}

		assert.Equal(t, models.IntakeStatusGA, byKey["github-issues"].Status)
		assert.True(t, byKey["github-issues"].Implemented)
		assert.Equal(t, models.IntakeCategoryRepositoryProvider, byKey["gitlab"].Category)
		assert.False(t, byKey["gitlab"].Implemented)
	})

	t.Run("updates status and note", func(t *testing.T) {
		restoreDatadog(t)

		code, body := request("PATCH", "/admin/api/intake-catalog/datadog", map[string]any{
			"status":      models.IntakeStatusAlpha,
			"status_note": "Works with US1. EU1 is not tested.",
		})
		require.Equal(t, http.StatusOK, code, string(body))
		entry := decodeEntry(t, body)
		assert.Equal(t, models.IntakeStatusAlpha, entry.Status)
		assert.Equal(t, "Works with US1. EU1 is not tested.", entry.StatusNote)
		assert.Equal(t, r.Account.Name, entry.UpdatedByName)
	})

	t.Run("creates a planned intake that cannot leave planned or be deleted once used", func(t *testing.T) {
		code, body := request("POST", "/admin/api/intake-catalog", map[string]any{
			"key":      "acme-tracker",
			"name":     "Acme tracker",
			"category": models.IntakeCategoryIssueTracking,
		})
		require.Equal(t, http.StatusOK, code, string(body))
		entry := decodeEntry(t, body)
		assert.Equal(t, models.IntakeStatusPlanned, entry.Status)
		assert.True(t, entry.Deletable)

		code, _ = request("POST", "/admin/api/intake-catalog", map[string]any{
			"key":      "acme-tracker",
			"name":     "Acme tracker",
			"category": models.IntakeCategoryIssueTracking,
		})
		assert.Equal(t, http.StatusConflict, code)

		code, _ = request("PATCH", "/admin/api/intake-catalog/acme-tracker", map[string]any{"status": models.IntakeStatusBeta})
		assert.Equal(t, http.StatusBadRequest, code)

		code, _ = request("DELETE", "/admin/api/intake-catalog/acme-tracker", nil)
		assert.Equal(t, http.StatusNoContent, code)

		code, _ = request("DELETE", "/admin/api/intake-catalog/github-issues", nil)
		assert.Equal(t, http.StatusBadRequest, code)
	})
}

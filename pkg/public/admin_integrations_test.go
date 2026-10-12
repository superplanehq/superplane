package public

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
)

func TestAdminListOrgIntegrations(t *testing.T) {
	server, r, token := setupAdminTestServer(t)

	sentry, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "sentry", "referrizer-sentry", nil)
	require.NoError(t, err)
	require.NoError(t, database.Conn().Model(sentry).Updates(map[string]any{
		"state":             models.IntegrationStateError,
		"state_description": "Sentry is not sending issue events.",
		"metadata": datatypes.NewJSONType(map[string]any{
			"hostedApp":        true,
			"state":            "csrf-state-value",
			"installationUUID": "9eb2cdda-c7f4-42c4-9fd2-2f20a5d08215",
			"organization":     map[string]any{"slug": "referrizer"},
		}),
	}).Error)

	t.Run("returns connections with state and identifying details", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/organizations/" + r.Organization.ID.String() + "/integrations",
			authCookie: token,
		})
		require.Equal(t, http.StatusOK, response.Code)

		var body struct {
			Items  []adminIntegration `json:"items"`
			Total  int64              `json:"total"`
			Limit  int                `json:"limit"`
			Offset int                `json:"offset"`
		}
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))

		var found *adminIntegration
		for i := range body.Items {
			if body.Items[i].ID == sentry.ID.String() {
				found = &body.Items[i]
			}
		}
		require.NotNil(t, found)
		assert.Equal(t, "sentry", found.AppName)
		assert.Equal(t, "referrizer-sentry", found.InstallationName)
		assert.Equal(t, models.IntegrationStateError, found.State)
		assert.Equal(t, "Sentry is not sending issue events.", found.StateDescription)
		assert.Equal(t, "9eb2cdda-c7f4-42c4-9fd2-2f20a5d08215", found.Details["installation_uuid"])
		assert.Equal(t, "referrizer", found.Details["external_organization"])
		assert.Equal(t, "true", found.Details["hosted_app"])
		assert.Equal(t, 50, body.Limit)
		assert.Equal(t, 0, body.Offset)
		assert.GreaterOrEqual(t, body.Total, int64(1))
		assert.NotContains(t, response.Body.String(), "csrf-state-value")
	})

	t.Run("returns one page and keeps the rest for the next offset", func(t *testing.T) {
		name := "page-cap-" + uuid.NewString()[:8]
		_, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "sentry", name+"-b", nil)
		require.NoError(t, err)
		_, err = models.CreateIntegration(uuid.New(), r.Organization.ID, "github", name+"-a", nil)
		require.NoError(t, err)

		first := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/organizations/" + r.Organization.ID.String() + "/integrations?search=" + name + "&limit=1&offset=0",
			authCookie: token,
		})
		require.Equal(t, http.StatusOK, first.Code)

		var page struct {
			Items  []adminIntegration `json:"items"`
			Total  int64              `json:"total"`
			Limit  int                `json:"limit"`
			Offset int                `json:"offset"`
		}
		require.NoError(t, json.Unmarshal(first.Body.Bytes(), &page))
		require.Len(t, page.Items, 1)
		assert.Equal(t, int64(2), page.Total)
		assert.Equal(t, 1, page.Limit)
		assert.Equal(t, 0, page.Offset)
		assert.Equal(t, "github", page.Items[0].AppName)
		assert.Equal(t, name+"-a", page.Items[0].InstallationName)

		second := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/organizations/" + r.Organization.ID.String() + "/integrations?search=" + name + "&limit=1&offset=1",
			authCookie: token,
		})
		require.Equal(t, http.StatusOK, second.Code)
		require.NoError(t, json.Unmarshal(second.Body.Bytes(), &page))
		require.Len(t, page.Items, 1)
		assert.Equal(t, name+"-b", page.Items[0].InstallationName)
		assert.Equal(t, 1, page.Offset)
	})

	t.Run("treats percent and underscore as literal characters", func(t *testing.T) {
		name := "wild-" + uuid.NewString()[:8]
		_, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "github", name+"-a%_b", nil)
		require.NoError(t, err)
		_, err = models.CreateIntegration(uuid.New(), r.Organization.ID, "sentry", name+"-axb", nil)
		require.NoError(t, err)

		response := execRequest(server, requestParams{
			method: "GET",
			path: "/admin/api/organizations/" + r.Organization.ID.String() +
				"/integrations?search=" + url.QueryEscape(name+"-a%_b") + "&limit=50",
			authCookie: token,
		})
		require.Equal(t, http.StatusOK, response.Code)

		var page struct {
			Items []adminIntegration `json:"items"`
			Total int64              `json:"total"`
		}
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &page))
		require.Len(t, page.Items, 1)
		assert.Equal(t, int64(1), page.Total)
		assert.Equal(t, name+"-a%_b", page.Items[0].InstallationName)
	})

	t.Run("returns a Linear connection without tokens or actor details", func(t *testing.T) {
		linearIntegration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "linear", "Acme Linear", nil)
		require.NoError(t, err)
		require.NoError(t, database.Conn().Model(linearIntegration).Updates(map[string]any{
			"state": models.IntegrationStateReady,
			"metadata": datatypes.NewJSONType(map[string]any{
				"organization":         "Acme",
				"urlKey":               "acme",
				"hostedOAuth":          true,
				"state":                "csrf-state-value",
				"accessTokenExpiresAt": "2026-10-02T00:00:00Z",
				"user":                 map[string]any{"email": "ada@example.com", "name": "Ada Lovelace"},
			}),
		}).Error)

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/organizations/" + r.Organization.ID.String() + "/integrations?search=Acme%20Linear",
			authCookie: token,
		})
		require.Equal(t, http.StatusOK, response.Code)
		assert.NotContains(t, response.Body.String(), "csrf-state-value")
		assert.NotContains(t, response.Body.String(), "ada@example.com")
		assert.NotContains(t, response.Body.String(), "accessToken")

		var body struct {
			Items []adminIntegration `json:"items"`
		}
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		require.Len(t, body.Items, 1)
		assert.Equal(t, "linear", body.Items[0].AppName)
		assert.Equal(t, "Acme Linear", body.Items[0].InstallationName)
		assert.Equal(t, models.IntegrationStateReady, body.Items[0].State)
		assert.Equal(t, "Acme", body.Items[0].Details["external_organization"])
		assert.Equal(t, "acme", body.Items[0].Details["workspace_key"])
		assert.Equal(t, "true", body.Items[0].Details["hosted_app"])
	})

	t.Run("returns 404 for non-existent org", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/organizations/00000000-0000-0000-0000-000000000000/integrations",
			authCookie: token,
		})
		assert.Equal(t, http.StatusNotFound, response.Code)
	})
}

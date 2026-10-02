package public

import (
	"encoding/json"
	"net/http"
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
			Items []adminIntegration `json:"items"`
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
		assert.NotContains(t, response.Body.String(), "csrf-state-value")
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

package public

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/fleets"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestAdminPrepareFleetManager(t *testing.T) {
	server, r, token := setupAdminTestServer(t)
	server.BaseURL = "https://superplane.example"

	response := execRequest(server, requestParams{
		method:     http.MethodPost,
		path:       "/admin/api/installation/first-run/fleet-manager",
		authCookie: token,
	})
	require.Equal(t, http.StatusOK, response.Code)

	var body firstRunFleetManagerResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	assert.Equal(t, fleets.FirstRunFleetID(), body.FleetID)
	assert.NotEmpty(t, body.Token)
	assert.Contains(t, body.YAML, "superplaneUrl: https://superplane.example")
	assert.Contains(t, body.YAML, body.Token)

	fleet, err := models.FindInstallationRunnerFleet(database.DB(t.Context()), fleets.FirstRunFleetID())
	require.NoError(t, err)
	assert.Equal(t, fleets.FirstRunFleetID(), fleet.Slug)

	tokens, err := models.ListUserAPITokens(database.DB(t.Context()), r.User)
	require.NoError(t, err)
	require.NotEmpty(t, tokens)
}

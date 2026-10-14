package public

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestAdminGetFleetManagerSetup(t *testing.T) {
	t.Setenv("RUNNER_STABLE_VERSION", "v1.2.3")
	t.Setenv("FLEET_MANAGER_STABLE_VERSION", "v2.3.4")
	server, user, token := setupAdminTestServer(t)

	response := execRequest(server, requestParams{
		method:     http.MethodGet,
		path:       "/admin/api/installation/fleet-manager/setup",
		authCookie: token,
	})
	require.Equal(t, http.StatusOK, response.Code)

	var body fleetManagerSetupResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	assert.Equal(t, "v1.2.3", body.DefaultRunnerVersion)
	assert.Equal(t, "v2.3.4", body.DefaultFleetManagerVersion)
	assert.Equal(t, "2.3.4", body.DefaultChartVersion)
	assert.Equal(t, fleetManagerChartRepository, body.ChartRepository)
	assert.Equal(t, fleetManagerImageRepository, body.ImageRepository)
	assert.Equal(t, runnerReleaseURLTemplate, body.RunnerReleaseURLTemplate)
	require.Len(t, body.FleetPresets, 4)
	assert.Equal(t, models.RunnerFleetE1LargeAMD64, body.FleetPresets[0].ID)
	assert.Equal(t, "amd64", body.FleetPresets[0].Spec.Architecture)

	fleets, err := models.ListInstallationRunnerFleets(database.DB(t.Context()))
	require.NoError(t, err)
	assert.Empty(t, fleets)
	tokens, err := models.ListUserAPITokens(database.DB(t.Context()), user.User)
	require.NoError(t, err)
	assert.Empty(t, tokens)
}

func TestAdminGetFleetManagerSetupRequiresAdministrator(t *testing.T) {
	server, _, _ := setupAdminTestServer(t)
	response := execRequest(server, requestParams{
		method: http.MethodGet,
		path:   "/admin/api/installation/fleet-manager/setup",
	})
	assert.Equal(t, http.StatusUnauthorized, response.Code)
}

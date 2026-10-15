package public

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
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
	assert.Equal(t, "manual", body.InfrastructureDefaults.Source)
	assert.Nil(t, body.InfrastructureDefaults.Provider)
	assert.Equal(t, "superplane-fleet-manager-token", body.Credentials.ExistingSecret)
	assert.False(t, body.Credentials.AutoProvisioned)

	fleets, err := models.ListInstallationRunnerFleets(database.DB(t.Context()))
	require.NoError(t, err)
	assert.Empty(t, fleets)
	tokens, err := models.ListUserAPITokens(database.DB(t.Context()), user.User)
	require.NoError(t, err)
	assert.Empty(t, tokens)
}

func TestAdminGetFleetManagerSetupLoadsChartDefaults(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "fleet-manager-setup.json")
	config := `{
		"source":"terraform",
		"provider":"aws",
		"managerId":"test-cluster",
		"namespace":"superplane",
		"serviceAccountName":"superplane-fleet-manager",
		"infrastructure":{"region":"us-east-1","subnetIds":["subnet-a","subnet-b"]},
		"credentials":{"existingSecret":"generated-token","tokenKey":"TOKEN","provision":true}
	}`
	require.NoError(t, os.WriteFile(configPath, []byte(config), 0o600))
	t.Setenv("FLEET_MANAGER_SETUP_CONFIG_PATH", configPath)
	server, _, token := setupAdminTestServer(t)

	response := execRequest(server, requestParams{
		method:     http.MethodGet,
		path:       "/admin/api/installation/fleet-manager/setup",
		authCookie: token,
	})
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	var body fleetManagerSetupResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	require.NotNil(t, body.InfrastructureDefaults.Provider)
	assert.Equal(t, "aws", *body.InfrastructureDefaults.Provider)
	assert.Equal(t, "terraform", body.InfrastructureDefaults.Source)
	assert.Equal(t, "test-cluster", body.InfrastructureDefaults.ManagerID)
	assert.Equal(t, "us-east-1", body.InfrastructureDefaults.Fields["region"])
	assert.Equal(t, []any{"subnet-a", "subnet-b"}, body.InfrastructureDefaults.Fields["subnetIds"])
	assert.Equal(t, "generated-token", body.Credentials.ExistingSecret)
	assert.Equal(t, "TOKEN", body.Credentials.TokenKey)
	assert.True(t, body.Credentials.AutoProvisioned)
}

func TestAdminGetFleetManagerSetupRequiresAdministrator(t *testing.T) {
	server, _, _ := setupAdminTestServer(t)
	response := execRequest(server, requestParams{
		method: http.MethodGet,
		path:   "/admin/api/installation/fleet-manager/setup",
	})
	assert.Equal(t, http.StatusUnauthorized, response.Code)
}

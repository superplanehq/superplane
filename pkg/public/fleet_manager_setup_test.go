package public

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

func useRunnerReleases(t *testing.T, url string) {
	t.Helper()
	previous := runnerReleasesURL
	runnerReleasesURL = url
	t.Cleanup(func() { runnerReleasesURL = previous })
}

func TestAdminGetFleetManagerSetup(t *testing.T) {
	t.Setenv("RUNNER_STABLE_VERSION", "v1.2.3")
	t.Setenv("RUNNER_SUPPORTED_VERSIONS", "v1.1.0, v1.2.3, v1.0.0")
	t.Setenv("FLEET_MANAGER_STABLE_VERSION", "v2.3.4")
	useRunnerReleases(t, "http://127.0.0.1:1")
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
	assert.Equal(t, []string{"v1.2.3", "v1.1.0", "v1.0.0"}, body.RunnerVersions)
	require.Len(t, body.FleetPresets, 4)
	assert.Equal(t, models.RunnerFleetE1LargeAMD64, body.FleetPresets[0].ID)
	assert.Equal(t, "amd64", body.FleetPresets[0].Spec.Architecture)
	assert.Equal(t, "manual", body.InfrastructureDefaults.Source)
	assert.Nil(t, body.InfrastructureDefaults.Provider)

	fleets, err := models.ListInstallationRunnerFleets(database.DB(t.Context()))
	require.NoError(t, err)
	assert.Empty(t, fleets)
	tokens, err := models.ListUserAPITokens(database.DB(t.Context()), user.User)
	require.NoError(t, err)
	assert.Empty(t, tokens)
}

func TestAdminGetFleetManagerSetupMergesPublishedRunnerVersions(t *testing.T) {
	t.Setenv("RUNNER_STABLE_VERSION", "v1.2.3")
	t.Setenv("RUNNER_SUPPORTED_VERSIONS", "v1.0.0, v9.9.9, v1.4.0")

	var releases *httptest.Server
	releases = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("page") {
		case "", "1":
			w.Header().Set("Link", fmt.Sprintf(`<%s?page=2>; rel="next"`, releases.URL))
			_ = json.NewEncoder(w).Encode([]githubRunnerRelease{
				{TagName: "v3.0.0"},
				{TagName: "runner/v1.2.3"},
				{TagName: "runner/v2.0.0", Draft: true},
				{TagName: "runner/v2.1.0", Prerelease: true},
				{TagName: "runner/v1.2.3-rc.1"},
				{TagName: "runner/not-a-version"},
			})
		case "2":
			_ = json.NewEncoder(w).Encode([]githubRunnerRelease{
				{TagName: "runner/v1.4.0"},
				{TagName: "runner/v1.2.3"},
				{TagName: "runner/v0.9.0"},
			})
		default:
			http.Error(w, "unexpected page", http.StatusBadRequest)
		}
	}))
	t.Cleanup(releases.Close)
	useRunnerReleases(t, releases.URL)

	server, _, token := setupAdminTestServer(t)
	response := execRequest(server, requestParams{
		method:     http.MethodGet,
		path:       "/admin/api/installation/fleet-manager/setup",
		authCookie: token,
	})
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	var body fleetManagerSetupResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	assert.Equal(t, []string{"v9.9.9", "v1.4.0", "v1.2.3", "v1.0.0", "v0.9.0"}, body.RunnerVersions)
}

func TestAdminGetFleetManagerSetupKeepsConfiguredVersionsWhenReleaseLookupFails(t *testing.T) {
	releases := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusBadGateway)
	}))
	t.Cleanup(releases.Close)

	tests := []struct {
		name string
		url  string
	}{
		{name: "non-success response", url: releases.URL},
		{name: "network error", url: "http://127.0.0.1:1"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("RUNNER_STABLE_VERSION", "v1.2.3")
			t.Setenv("RUNNER_SUPPORTED_VERSIONS", "v1.1.0, v1.2.3, v1.0.0")
			useRunnerReleases(t, test.url)

			server, _, token := setupAdminTestServer(t)
			response := execRequest(server, requestParams{
				method:     http.MethodGet,
				path:       "/admin/api/installation/fleet-manager/setup",
				authCookie: token,
			})
			require.Equal(t, http.StatusOK, response.Code, response.Body.String())

			var body fleetManagerSetupResponse
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
			assert.Equal(t, []string{"v1.2.3", "v1.1.0", "v1.0.0"}, body.RunnerVersions)
		})
	}
}

func TestAdminGetFleetManagerSetupLoadsChartDefaults(t *testing.T) {
	useRunnerReleases(t, "http://127.0.0.1:1")
	configPath := filepath.Join(t.TempDir(), "fleet-manager-setup.json")
	config := `{
		"source":"terraform",
		"provider":"aws",
		"managerId":"test-cluster",
		"namespace":"superplane",
		"serviceAccountName":"superplane-fleet-manager",
		"infrastructure":{"region":"us-east-1","subnetIds":["subnet-a","subnet-b"]}
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
}

func TestAdminGetFleetManagerSetupRequiresAdministrator(t *testing.T) {
	server, _, _ := setupAdminTestServer(t)
	response := execRequest(server, requestParams{
		method: http.MethodGet,
		path:   "/admin/api/installation/fleet-manager/setup",
	})
	assert.Equal(t, http.StatusUnauthorized, response.Code)
}

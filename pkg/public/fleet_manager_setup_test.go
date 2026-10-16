package public

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

func useRunnerReleases(t *testing.T, base string) {
	t.Helper()
	previous := runnerReleaseAPIBase
	runnerReleaseAPIBase = base
	resetRunnerReleaseCache()
	resetFleetManagerReleaseCache()
	t.Cleanup(func() {
		runnerReleaseAPIBase = previous
		resetRunnerReleaseCache()
		resetFleetManagerReleaseCache()
	})
}

func runnerReleaseTagFromRequest(r *http.Request) string {
	const marker = "/releases/tags/"
	escaped := r.URL.EscapedPath()
	index := strings.Index(escaped, marker)
	if index < 0 {
		return ""
	}
	tag, err := url.PathUnescape(escaped[index+len(marker):])
	if err != nil {
		return ""
	}
	return tag
}

func expireRunnerReleaseCache(t *testing.T) {
	t.Helper()
	runnerReleaseCacheMu.Lock()
	defer runnerReleaseCacheMu.Unlock()
	require.True(t, cachedRunnerReleaseOK)
	cachedRunnerReleaseFetchedAt = time.Now().Add(-runnerReleaseCacheTTL - time.Second)
}

func expireRunnerReleaseFailure(t *testing.T) {
	t.Helper()
	runnerReleaseCacheMu.Lock()
	defer runnerReleaseCacheMu.Unlock()
	require.Error(t, cachedRunnerReleaseFailure)
	cachedRunnerReleaseFailedAt = time.Now().Add(-runnerReleaseFailureBackoff - time.Second)
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

func TestAdminGetFleetManagerSetupUsesNewestPublishedFleetManagerRelease(t *testing.T) {
	t.Setenv("RUNNER_STABLE_VERSION", "v1.2.3")
	t.Setenv("FLEET_MANAGER_STABLE_VERSION", "v0.1.0")

	published := map[string]githubRunnerRelease{
		"fleet-manager/v0.1.0": {TagName: "fleet-manager/v0.1.0"},
		"fleet-manager/v0.1.3": {TagName: "fleet-manager/v0.1.3", Draft: true},
		"fleet-manager/v0.1.4": {TagName: "fleet-manager/v0.1.4"},
	}
	releases := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "matching-refs") {
			if strings.Contains(r.URL.Path, "fleet-manager") {
				_ = json.NewEncoder(w).Encode([]githubMatchingRef{
					{Ref: "refs/tags/fleet-manager/v0.1.0"},
					{Ref: "refs/tags/fleet-manager/v0.1.3"},
					{Ref: "refs/tags/fleet-manager/v0.1.4"},
					{Ref: "refs/tags/fleet-manager/v0.1.4-rc.1"},
				})
				return
			}
			_ = json.NewEncoder(w).Encode([]githubMatchingRef{})
			return
		}

		release, ok := published[runnerReleaseTagFromRequest(r)]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(release)
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
	assert.Equal(t, "v0.1.4", body.DefaultFleetManagerVersion)
	assert.Equal(t, "0.1.4", body.DefaultChartVersion)
}

func TestPublishedFleetManagerVersionsSkipsRepeatLookupAfterFailure(t *testing.T) {
	var fail atomic.Bool
	var requests atomic.Int32
	releases := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if fail.Load() {
			http.Error(w, "unavailable", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "matching-refs") {
			_ = json.NewEncoder(w).Encode([]githubMatchingRef{{Ref: "refs/tags/fleet-manager/v0.1.4"}})
			return
		}
		if runnerReleaseTagFromRequest(r) != "fleet-manager/v0.1.4" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(githubRunnerRelease{TagName: "fleet-manager/v0.1.4"})
	}))
	t.Cleanup(releases.Close)
	useRunnerReleases(t, releases.URL)

	versions, err := publishedFleetManagerVersions()
	require.NoError(t, err)
	assert.Equal(t, []string{"v0.1.4"}, versions)
	downloaded := requests.Load()
	require.Positive(t, downloaded)

	fleetManagerReleaseCacheMu.Lock()
	cachedFleetManagerFetchedAt = time.Now().Add(-runnerReleaseCacheTTL - time.Second)
	fleetManagerReleaseCacheMu.Unlock()
	fail.Store(true)

	versions, err = publishedFleetManagerVersions()
	require.NoError(t, err)
	assert.Equal(t, []string{"v0.1.4"}, versions)
	failedRefresh := requests.Load()
	assert.Greater(t, failedRefresh, downloaded)

	versions, err = publishedFleetManagerVersions()
	require.NoError(t, err)
	assert.Equal(t, []string{"v0.1.4"}, versions)
	assert.Equal(t, failedRefresh, requests.Load())
}

func TestAdminGetFleetManagerSetupMergesPublishedRunnerVersions(t *testing.T) {
	t.Setenv("RUNNER_STABLE_VERSION", "v1.2.3")
	t.Setenv("RUNNER_SUPPORTED_VERSIONS", "v1.0.0, v9.9.9, v1.4.0")

	published := map[string]githubRunnerRelease{
		"runner/v1.2.3": {TagName: "runner/v1.2.3"},
		"runner/v2.0.0": {TagName: "runner/v2.0.0", Draft: true},
		"runner/v2.1.0": {TagName: "runner/v2.1.0", Prerelease: true},
		"runner/v1.4.0": {TagName: "runner/v1.4.0"},
		"runner/v0.9.0": {TagName: "runner/v0.9.0"},
	}
	var releases *httptest.Server
	releases = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "matching-refs") {
			switch r.URL.Query().Get("page") {
			case "", "1":
				w.Header().Set("Link", fmt.Sprintf(`<%s/git/matching-refs/tags/runner/?page=2>; rel="next"`, releases.URL))
				_ = json.NewEncoder(w).Encode([]githubMatchingRef{
					{Ref: "refs/tags/v3.0.0"},
					{Ref: "refs/tags/runner/v1.2.3"},
					{Ref: "refs/tags/runner/v2.0.0"},
					{Ref: "refs/tags/runner/v2.1.0"},
					{Ref: "refs/tags/runner/v1.2.3-rc.1"},
					{Ref: "refs/tags/runner/not-a-version"},
				})
			case "2":
				_ = json.NewEncoder(w).Encode([]githubMatchingRef{
					{Ref: "refs/tags/runner/v1.4.0"},
					{Ref: "refs/tags/runner/v1.2.3"},
					{Ref: "refs/tags/runner/v0.9.0"},
					{Ref: "refs/tags/runner/v3.1.0"},
				})
			default:
				http.Error(w, "unexpected page", http.StatusBadRequest)
			}
			return
		}

		release, ok := published[runnerReleaseTagFromRequest(r)]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(release)
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
	assert.Equal(t, "v1.4.0", body.DefaultRunnerVersion)
}

func TestAdminGetFleetManagerSetupKeepsCachedRunnerVersionsWhenLookupFails(t *testing.T) {
	t.Setenv("RUNNER_STABLE_VERSION", "v1.2.3")
	t.Setenv("RUNNER_SUPPORTED_VERSIONS", "v1.0.0")

	var fail atomic.Bool
	var requests atomic.Int32
	releases := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if fail.Load() {
			http.Error(w, "unavailable", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "matching-refs") {
			_ = json.NewEncoder(w).Encode([]githubMatchingRef{{Ref: "refs/tags/runner/v0.9.0"}})
			return
		}
		if runnerReleaseTagFromRequest(r) != "runner/v0.9.0" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(githubRunnerRelease{TagName: "runner/v0.9.0"})
	}))
	t.Cleanup(releases.Close)
	useRunnerReleases(t, releases.URL)

	server, _, token := setupAdminTestServer(t)
	load := func() fleetManagerSetupResponse {
		t.Helper()
		response := execRequest(server, requestParams{
			method:     http.MethodGet,
			path:       "/admin/api/installation/fleet-manager/setup",
			authCookie: token,
		})
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		var body fleetManagerSetupResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		return body
	}

	first := load()
	assert.Equal(t, []string{"v1.2.3", "v1.0.0", "v0.9.0"}, first.RunnerVersions)
	assert.Equal(t, "v1.2.3", first.DefaultRunnerVersion)
	downloaded := requests.Load()
	require.Positive(t, downloaded)

	assert.Equal(t, []string{"v1.2.3", "v1.0.0", "v0.9.0"}, load().RunnerVersions)
	assert.Equal(t, downloaded, requests.Load())

	expireRunnerReleaseCache(t)
	fail.Store(true)
	assert.Equal(t, []string{"v1.2.3", "v1.0.0", "v0.9.0"}, load().RunnerVersions)
	assert.Greater(t, requests.Load(), downloaded)

	failedRefresh := requests.Load()
	assert.Equal(t, []string{"v1.2.3", "v1.0.0", "v0.9.0"}, load().RunnerVersions)
	assert.Equal(t, failedRefresh, requests.Load())
}

func TestPublishedRunnerVersionsRemembersAFailedLookup(t *testing.T) {
	var requests atomic.Int32
	releases := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		http.Error(w, "unavailable", http.StatusBadGateway)
	}))
	t.Cleanup(releases.Close)
	useRunnerReleases(t, releases.URL)

	const callers = 4
	var wg sync.WaitGroup
	lookupErrors := make([]error, callers)
	wg.Add(callers)
	for i := range callers {
		go func() {
			defer wg.Done()
			_, lookupErrors[i] = publishedRunnerVersions()
		}()
	}
	wg.Wait()
	for _, err := range lookupErrors {
		require.Error(t, err)
	}
	assert.Equal(t, int32(1), requests.Load())

	_, err := publishedRunnerVersions()
	require.Error(t, err)
	assert.Equal(t, int32(1), requests.Load())

	expireRunnerReleaseFailure(t)
	_, err = publishedRunnerVersions()
	require.Error(t, err)
	assert.Equal(t, int32(2), requests.Load())
}

func TestAdminGetFleetManagerSetupOmitsVersionsFromIncompleteTagList(t *testing.T) {
	t.Setenv("RUNNER_STABLE_VERSION", "v1.2.3")
	t.Setenv("RUNNER_SUPPORTED_VERSIONS", "")

	var pages atomic.Int32
	var releases *httptest.Server
	releases = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "matching-refs") {
			page := pages.Add(1)
			w.Header().Set("Link", fmt.Sprintf(`<%s/git/matching-refs/tags/runner/?page=%d>; rel="next"`, releases.URL, page+1))
			_ = json.NewEncoder(w).Encode([]githubMatchingRef{{Ref: "refs/tags/runner/v8.0.0"}})
			return
		}
		_ = json.NewEncoder(w).Encode(githubRunnerRelease{TagName: "runner/v8.0.0"})
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
	assert.Equal(t, []string{"v1.2.3"}, body.RunnerVersions)
	assert.Equal(t, int32(maximumRunnerTagPages*2), pages.Load())
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

package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	gh "github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/test/support/contexts"
)

type githubManifest struct {
	DefaultPermissions map[string]string `json:"default_permissions"`
}

func TestGitHubSyncKeepsPrivateAppManifestFlow(t *testing.T) {
	integration := &contexts.IntegrationContext{}
	require.NoError(t, (&GitHub{}).Sync(core.SyncContext{
		Configuration: Configuration{Organization: "testhq", PrivateApp: true},
		Integration:   integration,
	}))

	require.NotNil(t, integration.BrowserAction)
	assert.Equal(t, "POST", integration.BrowserAction.Method)
	assert.Equal(t, "https://github.com/organizations/testhq/settings/apps/new", integration.BrowserAction.URL)
	assertManifestContainsChecksPermission(t, integration.BrowserAction.FormFields["manifest"])
}

func TestGitHubSyncDoesNotInstallTheHostedApp(t *testing.T) {
	setHostedAppEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)

	err := (&GitHub{}).Sync(core.SyncContext{
		OrganizationID: "11111111-1111-1111-1111-111111111111",
		Integration:    &contexts.IntegrationContext{},
	})
	require.EqualError(t, err, "select a repository from the global GitHub App catalog")
}

func TestGitHubHandleRequestRejectsHostedIntegrationCallbacks(t *testing.T) {
	for _, path := range []string{"/redirect", "/setup", "/webhook"} {
		t.Run(path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, path, nil)
			response := httptest.NewRecorder()

			(&GitHub{}).HandleRequest(core.HTTPRequestContext{
				Request:  request,
				Response: response,
				Integration: &contexts.IntegrationContext{
					Metadata: common.Metadata{HostedApp: true},
				},
			})

			assert.Equal(t, http.StatusNotFound, response.Code)
		})
	}
}

func TestOwnerFromRepositories(t *testing.T) {
	assert.Equal(t, "acme", ownerFromRepositories([]common.Repository{{URL: "https://github.com/acme/payments"}}))
	assert.Empty(t, ownerFromRepositories(nil))
}

func TestOwnerFromAppInstallation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/app/installations/42", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":42,"account":{"login":"acme","type":"Organization"}}`))
	}))
	t.Cleanup(server.Close)

	client := gh.NewClient(server.Client())
	baseURL, err := url.Parse(server.URL + "/")
	require.NoError(t, err)
	client.BaseURL = baseURL

	owner, err := ownerFromAppInstallation(context.Background(), client, "42")
	require.NoError(t, err)
	assert.Equal(t, "acme", owner)
}

func TestListInstallationRepositoriesPaginatesAllPages(t *testing.T) {
	type repository struct {
		ID      int64  `json:"id"`
		Name    string `json:"name"`
		HTMLURL string `json:"html_url"`
	}
	type response struct {
		Repositories []repository `json:"repositories"`
	}

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "" {
			w.Header().Set("Link", fmt.Sprintf(`<%s/installation/repositories?page=2&per_page=100>; rel="next"`, server.URL))
			_ = json.NewEncoder(w).Encode(response{Repositories: []repository{{ID: 1, Name: "one"}}})
			return
		}
		_ = json.NewEncoder(w).Encode(response{Repositories: []repository{{ID: 2, Name: "two"}}})
	}))
	t.Cleanup(server.Close)

	client := gh.NewClient(server.Client())
	baseURL, err := url.Parse(server.URL + "/")
	require.NoError(t, err)
	client.BaseURL = baseURL

	repositories, err := listInstallationRepositories(context.Background(), client)
	require.NoError(t, err)
	require.Len(t, repositories, 2)
	assert.Equal(t, int64(1), repositories[0].ID)
	assert.Equal(t, int64(2), repositories[1].ID)
}

func assertManifestContainsChecksPermission(t *testing.T, manifest string) {
	t.Helper()
	var parsed githubManifest
	require.NoError(t, json.Unmarshal([]byte(manifest), &parsed))
	assert.Equal(t, "read", parsed.DefaultPermissions["checks"])
	assert.Equal(t, "read", parsed.DefaultPermissions["vulnerability_alerts"])
}

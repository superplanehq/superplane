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

func Test__GitHub__Sync(t *testing.T) {
	g := &GitHub{}

	t.Run("personal scope", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		require.NoError(t, g.Sync(core.SyncContext{Integration: integrationCtx}))

		//
		// Browser action is created
		//
		require.NotNil(t, integrationCtx.BrowserAction)
		assert.Equal(t, integrationCtx.BrowserAction.Method, "POST")
		assert.NotEmpty(t, integrationCtx.BrowserAction.Description)
		assert.Equal(t, integrationCtx.BrowserAction.URL, "https://github.com/settings/apps/new")
		assertManifestContainsChecksPermission(t, integrationCtx.BrowserAction.FormFields["manifest"])

		//
		// Metadata is set
		//
		require.NotNil(t, integrationCtx.Metadata)
		metadata := integrationCtx.Metadata.(common.Metadata)
		assert.Empty(t, metadata.Owner)
		assert.NotEmpty(t, metadata.State)
	})

	t.Run("organization scope", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		require.NoError(t, g.Sync(core.SyncContext{
			Configuration: Configuration{Organization: "testhq"},
			Integration:   integrationCtx,
		}))

		//
		// Browser action is created
		//
		require.NotNil(t, integrationCtx.BrowserAction)
		assert.Equal(t, integrationCtx.BrowserAction.Method, "POST")
		assert.NotEmpty(t, integrationCtx.BrowserAction.Description)
		assert.Equal(t, integrationCtx.BrowserAction.URL, "https://github.com/organizations/testhq/settings/apps/new")
		assertManifestContainsChecksPermission(t, integrationCtx.BrowserAction.FormFields["manifest"])

		//
		// Metadata is set
		//
		require.NotNil(t, integrationCtx.Metadata)
		metadata := integrationCtx.Metadata.(common.Metadata)
		assert.Equal(t, metadata.Owner, "testhq")
		assert.NotEmpty(t, metadata.State)
	})

	t.Run("hosted public app", func(t *testing.T) {
		setHostedAppEnv(t)
		restore := withFactoriesEnabledForTest(func(string) bool { return true })
		t.Cleanup(restore)

		integrationCtx := &contexts.IntegrationContext{}
		require.NoError(t, g.Sync(core.SyncContext{
			OrganizationID: "11111111-1111-1111-1111-111111111111",
			ActorUserID:    "starter-user",
			Integration:    integrationCtx,
		}))

		require.NotNil(t, integrationCtx.BrowserAction)
		assert.Equal(t, "GET", integrationCtx.BrowserAction.Method)
		assert.Empty(t, integrationCtx.BrowserAction.FormFields)
		assert.Contains(t, integrationCtx.BrowserAction.URL, "https://github.com/apps/superplane/installations/new?state=")
		assert.NotContains(t, integrationCtx.BrowserAction.URL, "settings/apps/new")

		require.NotNil(t, integrationCtx.Metadata)
		metadata := integrationCtx.Metadata.(common.Metadata)
		assert.True(t, metadata.HostedApp)
		assert.Equal(t, int64(99), metadata.GitHubApp.ID)
		assert.Equal(t, "superplane", metadata.GitHubApp.Slug)
		assert.Equal(t, "starter-user", metadata.StartedByUserID)
		assert.NotEmpty(t, metadata.State)
		assert.Empty(t, metadata.InstallationID)
	})

	t.Run("hosted env with privateApp keeps manifest flow", func(t *testing.T) {
		setHostedAppEnv(t)
		restore := withFactoriesEnabledForTest(func(string) bool { return true })
		t.Cleanup(restore)

		integrationCtx := &contexts.IntegrationContext{}
		require.NoError(t, g.Sync(core.SyncContext{
			OrganizationID: "11111111-1111-1111-1111-111111111111",
			Configuration:  Configuration{PrivateApp: true},
			Integration:    integrationCtx,
		}))

		require.NotNil(t, integrationCtx.BrowserAction)
		assert.Equal(t, "POST", integrationCtx.BrowserAction.Method)
		assert.Equal(t, "https://github.com/settings/apps/new", integrationCtx.BrowserAction.URL)
		assert.NotContains(t, integrationCtx.BrowserAction.URL, "/apps/superplane/installations/new")

		require.NotNil(t, integrationCtx.Metadata)
		metadata := integrationCtx.Metadata.(common.Metadata)
		assert.False(t, metadata.HostedApp)
	})

	t.Run("hosted env without factories keeps manifest flow", func(t *testing.T) {
		setHostedAppEnv(t)
		restore := withFactoriesEnabledForTest(func(string) bool { return false })
		t.Cleanup(restore)

		integrationCtx := &contexts.IntegrationContext{}
		require.NoError(t, g.Sync(core.SyncContext{
			OrganizationID: "11111111-1111-1111-1111-111111111111",
			Integration:    integrationCtx,
		}))

		require.NotNil(t, integrationCtx.BrowserAction)
		assert.Equal(t, "POST", integrationCtx.BrowserAction.Method)
		assert.Equal(t, "https://github.com/settings/apps/new", integrationCtx.BrowserAction.URL)
	})
}

func Test__isPendingInstallationSetupAction(t *testing.T) {
	assert.True(t, isPendingInstallationSetupAction("install"))
	assert.True(t, isPendingInstallationSetupAction("update"))
	assert.False(t, isPendingInstallationSetupAction(""))
	assert.False(t, isPendingInstallationSetupAction("request"))
}

func Test__isInstallationRequestSetupAction(t *testing.T) {
	assert.True(t, isInstallationRequestSetupAction("request"))
	assert.False(t, isInstallationRequestSetupAction("install"))
	assert.False(t, isInstallationRequestSetupAction("update"))
	assert.False(t, isInstallationRequestSetupAction(""))
}

func Test__afterAppInstallation_installRequest(t *testing.T) {
	integration := &contexts.IntegrationContext{
		NewSetupFlow:  true,
		IntegrationID: "11111111-1111-1111-1111-111111111111",
		CurrentProperties: map[string]any{
			common.PropertyAppState: "csrf",
		},
	}
	ctx, rec := hostedRequestContext(
		integration,
		"/api/v1/integrations/11111111-1111-1111-1111-111111111111/setup?state=csrf&setup_action=request",
		nil,
	)

	(&GitHub{}).afterAppInstallation(ctx)

	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(
		t,
		"https://app.example/org-1/settings/integrations/11111111-1111-1111-1111-111111111111?githubSetup=request",
		rec.Header().Get("Location"),
	)
	require.NotNil(t, integration.Metadata)
	assert.True(t, integration.Metadata.(common.Metadata).InstallRequested)
	assert.Empty(t, integration.Metadata.(common.Metadata).InstallRequestedAccount)
}

func Test__afterAppInstallation_installRequest_persistsAccount(t *testing.T) {
	integration := &contexts.IntegrationContext{
		NewSetupFlow:  true,
		IntegrationID: "11111111-1111-1111-1111-111111111111",
		CurrentProperties: map[string]any{
			common.PropertyAppState: "csrf",
		},
	}
	ctx, rec := hostedRequestContext(
		integration,
		"/api/v1/integrations/11111111-1111-1111-1111-111111111111/setup?state=csrf&setup_action=request&account=acme",
		nil,
	)

	(&GitHub{}).afterAppInstallation(ctx)

	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(
		t,
		"https://app.example/org-1/settings/integrations/11111111-1111-1111-1111-111111111111?githubSetup=request&githubOrg=acme",
		rec.Header().Get("Location"),
	)
	require.NotNil(t, integration.Metadata)
	assert.Equal(t, "acme", integration.Metadata.(common.Metadata).InstallRequestedAccount)
}

func Test__isSafeIntegrationSetupReturnPath(t *testing.T) {
	assert.True(t, isSafeIntegrationSetupReturnPath("/org-1/workspaces/APP/setup?step=vcs"))
	assert.True(t, isSafeIntegrationSetupReturnPath("/onboarding?attempt=1&step=vcs"))
	assert.False(t, isSafeIntegrationSetupReturnPath("//evil.example/phishing"))
	assert.False(t, isSafeIntegrationSetupReturnPath("https://evil.example"))
	assert.False(t, isSafeIntegrationSetupReturnPath("/org-1"))
	assert.False(t, isSafeIntegrationSetupReturnPath(""))
}

func Test__ownerFromRepositories(t *testing.T) {
	assert.Equal(t, "acme", ownerFromRepositories([]common.Repository{
		{URL: "https://github.com/acme/payments"},
	}))
	assert.Empty(t, ownerFromRepositories(nil))
}

func Test__ownerFromInstallationAccount(t *testing.T) {
	assert.Empty(t, ownerFromInstallationAccount(nil))
	assert.Empty(t, ownerFromInstallationAccount(&gh.Installation{}))
	assert.Equal(t, "acme", ownerFromInstallationAccount(&gh.Installation{
		Account: &gh.User{Login: gh.Ptr("acme")},
	}))
}

func Test__ownerFromAppInstallation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/app/installations/42", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":42,"account":{"login":"acme","type":"Organization"}}`))
	}))
	t.Cleanup(srv.Close)

	client := gh.NewClient(srv.Client())
	baseURL, err := url.Parse(srv.URL + "/")
	require.NoError(t, err)
	client.BaseURL = baseURL

	owner, err := ownerFromAppInstallation(context.Background(), client, "42")
	require.NoError(t, err)
	assert.Equal(t, "acme", owner)
}

func Test__resolveInstallationOwner(t *testing.T) {
	t.Run("uses repository owner first", func(t *testing.T) {
		assert.Equal(
			t,
			"acme",
			resolveInstallationOwner(context.Background(), nil, "42", []common.Repository{
				{URL: "https://github.com/acme/payments"},
			}),
		)
	})

	t.Run("uses app installation when no repositories", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "/app/installations/42", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":42,"account":{"login":"acme"}}`))
		}))
		t.Cleanup(srv.Close)

		client := gh.NewClient(srv.Client())
		baseURL, err := url.Parse(srv.URL + "/")
		require.NoError(t, err)
		client.BaseURL = baseURL

		assert.Equal(t, "acme", resolveInstallationOwner(context.Background(), client, "42", nil))
	})

	t.Run("empty when installation lookup fails and no repositories", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "/app/installations/42", r.URL.Path)
			w.WriteHeader(http.StatusForbidden)
		}))
		t.Cleanup(srv.Close)

		client := gh.NewClient(srv.Client())
		baseURL, err := url.Parse(srv.URL + "/")
		require.NoError(t, err)
		client.BaseURL = baseURL

		assert.Empty(t, resolveInstallationOwner(context.Background(), client, "42", nil))
	})
}

func Test__listInstallationRepositories__paginates_all_pages(t *testing.T) {
	t.Parallel()

	type reposResponse struct {
		TotalCount   int `json:"total_count"`
		Repositories []struct {
			ID      int64  `json:"id"`
			Name    string `json:"name"`
			HTMLURL string `json:"html_url"`
		} `json:"repositories"`
	}

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/installation/repositories", r.URL.Path)

		page := r.URL.Query().Get("page")
		if page == "" {
			page = "1"
		}

		w.Header().Set("Content-Type", "application/json")

		switch page {
		case "1":
			// Provide Link header to instruct the client there is a next page.
			next := fmt.Sprintf(`<%s/installation/repositories?page=2&per_page=100>; rel="next", <%s/installation/repositories?page=2&per_page=100>; rel="last"`, srv.URL, srv.URL)
			w.Header().Set("Link", next)

			_ = json.NewEncoder(w).Encode(reposResponse{
				TotalCount: 2,
				Repositories: []struct {
					ID      int64  `json:"id"`
					Name    string `json:"name"`
					HTMLURL string `json:"html_url"`
				}{
					{ID: 1, Name: "repo1", HTMLURL: "https://github.com/test/repo1"},
				},
			})
		case "2":
			_ = json.NewEncoder(w).Encode(reposResponse{
				TotalCount: 2,
				Repositories: []struct {
					ID      int64  `json:"id"`
					Name    string `json:"name"`
					HTMLURL string `json:"html_url"`
				}{
					{ID: 2, Name: "repo2", HTMLURL: "https://github.com/test/repo2"},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	client := gh.NewClient(srv.Client())
	baseURL, err := url.Parse(srv.URL + "/")
	require.NoError(t, err)
	client.BaseURL = baseURL
	client.UploadURL = baseURL

	repos, err := listInstallationRepositories(context.Background(), client)
	require.NoError(t, err)
	require.Len(t, repos, 2)
	require.Equal(t, int64(1), repos[0].ID)
	require.Equal(t, "repo1", repos[0].Name)
	require.Equal(t, "https://github.com/test/repo1", repos[0].URL)
	require.Equal(t, int64(2), repos[1].ID)
	require.Equal(t, "repo2", repos[1].Name)
	require.Equal(t, "https://github.com/test/repo2", repos[1].URL)
}

func assertManifestContainsChecksPermission(t *testing.T, manifestJSON string) {
	t.Helper()

	require.NotEmpty(t, manifestJSON)

	var manifest githubManifest
	require.NoError(t, json.Unmarshal([]byte(manifestJSON), &manifest))
	require.Equal(t, "read", manifest.DefaultPermissions["checks"])
}

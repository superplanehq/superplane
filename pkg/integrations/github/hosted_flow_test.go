package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gh "github.com/google/go-github/v84/github"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestHostedSetupCallbackDoesNotBindInstallationID(t *testing.T) {
	setHostedAppEnv(t)
	integration := &contexts.IntegrationContext{
		State: "pending",
		Metadata: common.Metadata{
			State:                    "csrf",
			HostedApp:                true,
			StartedByUserID:          "11111111-1111-1111-1111-111111111111",
			InstallationsRefreshedAt: time.Now().UTC().Format(time.RFC3339Nano),
			GitHubApp:                common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
		},
	}
	ctx, rec := hostedRequestContext(
		integration,
		"/api/v1/github/app/setup?state=csrf&installation_id=999&setup_action=install",
		nil,
	)

	(&GitHub{}).afterAppInstallationLegacy(ctx)

	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "pending", integration.State)
	metadata := integration.Metadata.(common.Metadata)
	assert.Empty(t, metadata.InstallationID)
	assert.Empty(t, metadata.Repositories)
	assert.Empty(t, metadata.InstallationsRefreshedAt)
}

func TestHostedSetupCallbackRejectsInvalidState(t *testing.T) {
	integration := &contexts.IntegrationContext{
		State: "pending",
		Metadata: common.Metadata{
			State:     "csrf",
			HostedApp: true,
		},
	}
	ctx, rec := hostedRequestContext(
		integration,
		"/api/v1/github/app/setup?state=wrong&installation_id=11&setup_action=install",
		nil,
	)

	(&GitHub{}).afterAppInstallationLegacy(ctx)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Empty(t, integration.Metadata.(common.Metadata).InstallationID)
}

func TestHostedBindRequiresRepository(t *testing.T) {
	integration := &contexts.IntegrationContext{
		State: "pending",
		Metadata: common.Metadata{
			State:           "csrf",
			HostedApp:       true,
			StartedByUserID: "11111111-1111-1111-1111-111111111111",
		},
	}
	ctx, rec := hostedRequestContext(integration, "/api/v1/github/app/bind", nil)
	ctx.Request.Method = http.MethodPost
	ctx.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ctx.Request.Body = http.NoBody
	ctx.Request.URL.RawQuery = "state=csrf&installation_id=11"

	(&GitHub{}).afterHostedAppBind(ctx)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "pending", integration.State)
}

func TestHostedBindUsesLocalInstallationAccessWithoutIdentity(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	setHostedAppEnv(t)
	t.Cleanup(resetBindClientHooks)

	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	listAppInstallations = func(context.Context, *gh.Client) ([]common.PendingInstallation, error) {
		return []common.PendingInstallation{{ID: "11", AccountLogin: "acme", AccountType: "Organization"}}, nil
	}
	newInstallationClient = func(core.IntegrationContext, int64, string) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	listInstallationRepos = func(context.Context, *gh.Client) ([]common.Repository, error) {
		return []common.Repository{{ID: 101, Name: "api"}}, nil
	}
	integration := &contexts.IntegrationContext{
		State: "pending",
		Metadata: common.Metadata{
			State:           "csrf",
			HostedApp:       true,
			StartedByUserID: "missing-local-user",
			GitHubApp:       common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
		},
	}
	ctx, rec := hostedRequestContext(
		integration,
		"/api/v1/github/app/bind?state=csrf&installation_id=11&repository_id=101",
		nil,
	)
	ctx.Request.Method = http.MethodPost

	(&GitHub{}).afterHostedAppBind(ctx)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "ready", integration.State)
	metadata := integration.Metadata.(common.Metadata)
	assert.Equal(t, "11", metadata.InstallationID)
	assert.Equal(t, "acme", metadata.Owner)
	assert.Equal(t, []common.Repository{{ID: 101, Name: "acme/api"}}, metadata.Repositories)
}

func TestBindHostedInstallationRepositoriesScopesConnection(t *testing.T) {
	integration := &contexts.IntegrationContext{State: "pending"}
	ctx, _ := hostedRequestContext(integration, "/api/v1/github/app/bind", nil)
	metadata := common.Metadata{
		State:     "csrf",
		HostedApp: true,
		GitHubApp: common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
	}
	installation := common.PendingInstallation{ID: "11", AccountLogin: "acme"}
	repositories := []common.Repository{{ID: 101, Name: "acme/api"}}

	err := (&GitHub{}).bindHostedInstallationRepositories(ctx, metadata, installation, repositories)

	require.NoError(t, err)
	assert.Equal(t, "ready", integration.State)
	bound := integration.Metadata.(common.Metadata)
	assert.Equal(t, "11", bound.InstallationID)
	assert.Equal(t, "acme", bound.Owner)
	assert.Equal(t, repositories, bound.Repositories)
	assert.Equal(t, repositories, bound.SelectedRepositories)
	assert.True(t, bound.RepositoryScoped)
}

func TestSyncHostedAppKeepsVerifiedPickerMetadata(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	setHostedAppEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)

	integration := &contexts.IntegrationContext{
		Metadata: common.Metadata{
			State:           "csrf",
			HostedApp:       true,
			StartedByUserID: "missing-user",
			SetupReturnPath: "/onboarding?attempt=old&step=vcs",
			PendingInstallations: []common.PendingInstallation{
				{ID: "11", AccountLogin: "acme", Repositories: []common.Repository{{ID: 101, Name: "acme/api"}}},
			},
			GitHubApp: common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
		},
	}

	err := (&GitHub{}).Sync(core.SyncContext{
		OrganizationID: "11111111-1111-1111-1111-111111111111",
		ActorUserID:    "other-user",
		BaseURL:        "https://app.example",
		Configuration:  Configuration{SetupReturnPath: "/onboarding?attempt=new&step=vcs"},
		Integration:    integration,
	})

	require.NoError(t, err)
	metadata := integration.Metadata.(common.Metadata)
	assert.Equal(t, "csrf", metadata.State)
	assert.Equal(t, "missing-user", metadata.StartedByUserID)
	assert.Equal(t, "/onboarding?attempt=new&step=vcs", metadata.SetupReturnPath)
	require.Len(t, metadata.PendingInstallations, 1)
	assert.Nil(t, integration.BrowserAction)
}

func TestRequiresHostedInstallationDiscovery(t *testing.T) {
	now := time.Now().UTC()
	cached := common.Metadata{
		PendingInstallations: []common.PendingInstallation{
			{ID: "11", AccountLogin: "acme", Repositories: []common.Repository{{ID: 101, Name: "acme/api"}}},
		},
		InstallationsRefreshedAt: now.Format(time.RFC3339Nano),
	}

	assert.False(t, requiresHostedInstallationDiscovery(cached, now))
	assert.True(t, requiresHostedInstallationDiscovery(cached, now.Add(hostedInstallationDiscoveryInterval)))
	assert.False(t, requiresHostedInstallationDiscovery(common.Metadata{InstallationID: "11"}, now))
	assert.True(t, requiresHostedInstallationDiscovery(common.Metadata{}, now))
}

func TestMergeVerifiedInstallationsPreservesPriorResults(t *testing.T) {
	refreshed := []common.PendingInstallation{
		{ID: "11", AccountLogin: "renamed", Repositories: []common.Repository{{ID: 101, Name: "renamed/api"}}},
	}
	existing := []common.PendingInstallation{
		{ID: "11", AccountLogin: "acme", Repositories: []common.Repository{{ID: 101, Name: "acme/api"}}},
		{ID: "22", AccountLogin: "octo", Repositories: []common.Repository{{ID: 202, Name: "octo/web"}}},
	}

	merged := mergeVerifiedInstallations(refreshed, existing)

	assert.Equal(t, []common.PendingInstallation{refreshed[0], existing[1]}, merged)
}

func TestSyncHostedAppReconcilesInstallationRequests(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	setHostedAppEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)
	t.Cleanup(resetBindClientHooks)

	listAppInstallationRequests = func(context.Context, *gh.Client, string) ([]common.InstallRequest, error) {
		return []common.InstallRequest{{ID: "2", AccountLogin: "octo", RequesterLogin: "member"}}, nil
	}
	listAppInstallations = func(context.Context, *gh.Client) ([]common.PendingInstallation, error) {
		return []common.PendingInstallation{{ID: "11", AccountLogin: "acme", AccountType: "Organization"}}, nil
	}
	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) { return gh.NewClient(nil), nil }
	integration := &contexts.IntegrationContext{
		State: "pending",
		Metadata: common.Metadata{
			State:                "csrf",
			HostedApp:            true,
			StartedByGitHubLogin: "member",
			InstallRequests: []common.InstallRequest{
				{ID: "1", AccountLogin: "acme", RequesterLogin: "member"},
				{ID: "2", AccountLogin: "octo", RequesterLogin: "member"},
			},
			InstallRequested: true,
			GitHubApp:        common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
		},
	}

	err := (&GitHub{}).Sync(core.SyncContext{
		Logger:         logrus.NewEntry(logrus.New()),
		OrganizationID: "11111111-1111-1111-1111-111111111111",
		BaseURL:        "https://app.example",
		Integration:    integration,
	})

	require.NoError(t, err)
	metadata := integration.Metadata.(common.Metadata)
	assert.Equal(t, []common.InstallRequest{{ID: "2", AccountLogin: "octo", RequesterLogin: "member"}}, metadata.InstallRequests)
	assert.Equal(t, "octo", metadata.InstallRequestedAccount)
	require.Len(t, metadata.PendingInstallations, 1)
	assert.Equal(t, "acme", metadata.PendingInstallations[0].AccountLogin)
}

func TestListAppInstallationRequestsFiltersRequester(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/app/installation-requests", r.URL.Path)
		_, _ = w.Write([]byte(`[
			{"id":1,"account":{"login":"acme"},"requester":{"login":"member"}},
			{"id":2,"account":{"login":"other"},"requester":{"login":"someone-else"}}
		]`))
	}))
	defer server.Close()

	client := gh.NewClient(server.Client())
	client.BaseURL, _ = client.BaseURL.Parse(server.URL + "/")
	requests, err := listAppInstallationRequestsFromGitHub(context.Background(), client, "member")

	require.NoError(t, err)
	require.Len(t, requests, 1)
	assert.Equal(t, "acme", requests[0].AccountLogin)
}

func pendingHostedIntegration(state string) *contexts.IntegrationContext {
	return &contexts.IntegrationContext{
		IntegrationID: "11111111-1111-1111-1111-111111111111",
		State:         "pending",
		Metadata: common.Metadata{
			State:     state,
			HostedApp: true,
			GitHubApp: common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
		},
	}
}

func hostedRequestContext(
	integration *contexts.IntegrationContext,
	path string,
	httpCtx core.HTTPContext,
) (core.HTTPRequestContext, *httptest.ResponseRecorder) {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	recorder := httptest.NewRecorder()
	return core.HTTPRequestContext{
		Logger:         logrus.NewEntry(logrus.New()),
		Request:        request,
		Response:       recorder,
		OrganizationID: "org-1",
		BaseURL:        "https://app.example",
		HTTP:           httpCtx,
		Integration:    integration,
	}, recorder
}

func resetBindClientHooks() {
	newInstallationClient = newClientForAppInstallation
	newAppJWTClient = newClientForApp
	listInstallationRepos = listInstallationRepositories
	listAppInstallations = listAppInstallationsFromGitHub
	listAppInstallationRequests = listAppInstallationRequestsFromGitHub
}

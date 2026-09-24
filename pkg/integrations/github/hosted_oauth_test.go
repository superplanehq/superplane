package github

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	gh "github.com/google/go-github/v84/github"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Sync_hostedUserOAuth(t *testing.T) {
	setHostedAppOAuthEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)

	integrationCtx := &contexts.IntegrationContext{}
	require.NoError(t, (&GitHub{}).Sync(core.SyncContext{
		OrganizationID: "11111111-1111-1111-1111-111111111111",
		BaseURL:        "https://app.example",
		Integration:    integrationCtx,
	}))

	require.NotNil(t, integrationCtx.BrowserAction)
	assert.Equal(t, "GET", integrationCtx.BrowserAction.Method)
	assert.Contains(t, integrationCtx.BrowserAction.URL, "https://github.com/login/oauth/authorize?")
	assert.Contains(t, integrationCtx.BrowserAction.URL, "client_id=Iv1.abc")
	assert.Contains(t, integrationCtx.BrowserAction.URL, "redirect_uri=")
	assert.NotContains(t, integrationCtx.BrowserAction.URL, "/app/installations")
	assert.NotContains(t, integrationCtx.BrowserAction.URL, "installations/new")
	assert.Equal(t, integrationCtx.BrowserAction.URL, integrationCtx.Metadata.(common.Metadata).AuthorizeURL)
}

func Test__exchangeGitHubUserOAuthToken(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{jsonResponse(`{"access_token":"user-token"}`)},
	}

	token, err := exchangeGitHubUserOAuthToken(httpCtx, "Iv1.abc", "app-secret", "code-1", "https://app.example/api/v1/github/app/oauth/callback")
	require.NoError(t, err)
	assert.Equal(t, "user-token", token)
	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, "https://github.com/login/oauth/access_token", httpCtx.Requests[0].URL.String())
}

func Test__listUserAppInstallations(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{jsonResponse(`{
			"installations":[
				{"id":11,"account":{"login":"acme","type":"Organization"}},
				{"id":22,"account":{"login":"octo","type":"User"}}
			]
		}`)},
	}

	installations, err := listUserAppInstallations(httpCtx, "user-token", 99)
	require.NoError(t, err)
	require.Len(t, installations, 2)
	assert.Equal(t, "11", installations[0].ID)
	assert.Equal(t, "acme", installations[0].AccountLogin)
	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, "/user/installations", httpCtx.Requests[0].URL.Path)
	assert.NotEqual(t, "/app/installations", httpCtx.Requests[0].URL.Path)
}

func Test__listUserAppInstallations_filtersOtherApps(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{jsonResponse(`{
			"installations":[
				{"id":11,"app_id":99,"account":{"login":"acme","type":"Organization"}},
				{"id":22,"app_id":7,"account":{"login":"other","type":"User"}}
			]
		}`)},
	}

	installations, err := listUserAppInstallations(httpCtx, "user-token", 99)
	require.NoError(t, err)
	require.Len(t, installations, 1)
	assert.Equal(t, "11", installations[0].ID)
}

func Test__afterHostedAppOAuth(t *testing.T) {
	setHostedAppOAuthEnv(t)
	g := &GitHub{}

	t.Run("zero installs redirects to install URL", func(t *testing.T) {
		integration := pendingHostedIntegration("csrf")
		httpCtx := oauthHTTP(jsonResponse(`{"access_token":"user-token"}`), jsonResponse(`{"installations":[]}`))
		ctx, rec := hostedRequestContext(integration, "/api/v1/github/app/oauth/callback?state=csrf&code=abc", httpCtx)

		g.afterHostedAppOAuth(ctx)

		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Contains(t, rec.Header().Get("Location"), "https://github.com/apps/superplane/installations/new?state=csrf")
		assert.Equal(t, "pending", integration.State)
		assert.Empty(t, integration.CurrentSecrets)
		assertNoPlaintextSecrets(t, integration)
		require.NotNil(t, integration.BrowserAction)
		assert.Contains(t, integration.BrowserAction.URL, "installations/new")
	})

	t.Run("one install writes allowlist and stays pending", func(t *testing.T) {
		// A silent bind of the single installation would lock the connection
		// to that account (often the user's personal one) with no way to
		// install the App on an organization. The account picker must open.
		integration := pendingHostedIntegration("csrf")
		httpCtx := oauthHTTP(
			jsonResponse(`{"access_token":"user-token"}`),
			jsonResponse(`{"installations":[{"id":11,"account":{"login":"acme","type":"Organization"}}]}`),
		)
		ctx, rec := hostedRequestContext(integration, "/api/v1/github/app/oauth/callback?state=csrf&code=abc", httpCtx)

		g.afterHostedAppOAuth(ctx)

		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.NotEqual(t, "ready", integration.State)
		assert.Nil(t, integration.BrowserAction)
		metadata := integration.Metadata.(common.Metadata)
		assert.Empty(t, metadata.InstallationID)
		assert.Equal(t, "csrf", metadata.State)
		require.Len(t, metadata.PendingInstallations, 1)
		assert.Equal(t, "11", metadata.PendingInstallations[0].ID)
		assert.Equal(t, "acme", metadata.PendingInstallations[0].AccountLogin)
		assert.Empty(t, integration.CurrentSecrets)
		assertNoPlaintextSecrets(t, integration)
	})

	t.Run("many installs write allowlist and stay pending", func(t *testing.T) {
		integration := pendingHostedIntegration("csrf")
		integration.Metadata = common.Metadata{
			State:           "csrf",
			HostedApp:       true,
			SetupReturnPath: "/onboarding?attempt=1&step=vcs",
			GitHubApp:       common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
		}
		httpCtx := oauthHTTP(
			jsonResponse(`{"access_token":"user-token"}`),
			jsonResponse(`{"installations":[
				{"id":11,"account":{"login":"acme","type":"Organization"}},
				{"id":22,"account":{"login":"octo","type":"User"}}
			]}`),
			jsonResponse(`{"login":"member"}`),
		)
		ctx, rec := hostedRequestContext(integration, "/api/v1/github/app/oauth/callback?state=csrf&code=abc", httpCtx)

		g.afterHostedAppOAuth(ctx)

		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "https://app.example/onboarding?attempt=1&step=vcs", rec.Header().Get("Location"))
		assert.NotEqual(t, "ready", integration.State)
		assert.Nil(t, integration.BrowserAction)
		metadata := integration.Metadata.(common.Metadata)
		assert.Empty(t, metadata.InstallationID)
		assert.Equal(t, "csrf", metadata.State)
		assert.Equal(t, "/onboarding?attempt=1&step=vcs", metadata.SetupReturnPath)
		assert.Equal(t, "member", metadata.StartedByGitHubLogin)
		require.Len(t, metadata.PendingInstallations, 2)
		assert.Equal(t, "11", metadata.PendingInstallations[0].ID)
		assert.Empty(t, integration.CurrentSecrets)
		assertNoPlaintextSecrets(t, integration)
		require.Len(t, httpCtx.Requests, 3)
		assert.Equal(t, "/user/installations", httpCtx.Requests[1].URL.Path)
		assert.NotContains(t, httpCtx.Requests[1].URL.Path, "/app/installations")
		assert.Equal(t, "/user", httpCtx.Requests[2].URL.Path)
	})

	t.Run("approved install request clears the waiting state", func(t *testing.T) {
		integration := pendingHostedIntegration("csrf")
		integration.Metadata = common.Metadata{
			State:                   "csrf",
			HostedApp:               true,
			InstallRequested:        true,
			InstallRequestedAccount: "acme",
			GitHubApp:               common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
		}
		httpCtx := oauthHTTP(
			jsonResponse(`{"access_token":"user-token"}`),
			jsonResponse(`{"installations":[{"id":11,"account":{"login":"Acme","type":"Organization"}}]}`),
		)
		ctx, _ := hostedRequestContext(integration, "/api/v1/github/app/oauth/callback?state=csrf&code=abc", httpCtx)

		g.afterHostedAppOAuth(ctx)

		metadata := integration.Metadata.(common.Metadata)
		require.Len(t, metadata.PendingInstallations, 1)
		assert.False(t, metadata.InstallRequested)
		assert.Empty(t, metadata.InstallRequestedAccount)
	})

	t.Run("unapproved install request keeps the waiting state", func(t *testing.T) {
		integration := pendingHostedIntegration("csrf")
		integration.Metadata = common.Metadata{
			State:                   "csrf",
			HostedApp:               true,
			InstallRequested:        true,
			InstallRequestedAccount: "acme",
			GitHubApp:               common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
		}
		httpCtx := oauthHTTP(
			jsonResponse(`{"access_token":"user-token"}`),
			jsonResponse(`{"installations":[{"id":22,"account":{"login":"octo","type":"User"}}]}`),
		)
		ctx, _ := hostedRequestContext(integration, "/api/v1/github/app/oauth/callback?state=csrf&code=abc", httpCtx)

		g.afterHostedAppOAuth(ctx)

		metadata := integration.Metadata.(common.Metadata)
		require.Len(t, metadata.PendingInstallations, 1)
		assert.True(t, metadata.InstallRequested)
		assert.Equal(t, "acme", metadata.InstallRequestedAccount)
	})
}

func Test__afterHostedAppBind(t *testing.T) {
	g := &GitHub{}

	t.Run("rejects installation id outside the allowlist", func(t *testing.T) {
		integration := pendingHostedIntegration("csrf")
		integration.Metadata = common.Metadata{
			State:     "csrf",
			HostedApp: true,
			GitHubApp: common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
			PendingInstallations: []common.PendingInstallation{
				{ID: "11", AccountLogin: "acme"},
			},
		}
		ctx, rec := hostedRequestContext(integration, "/api/v1/github/app/bind?state=csrf&installation_id=99", nil)

		g.afterHostedAppBind(ctx)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.NotEqual(t, "ready", integration.State)
		metadata := integration.Metadata.(common.Metadata)
		assert.Empty(t, metadata.InstallationID)
	})

	t.Run("binds an allowlisted installation", func(t *testing.T) {
		t.Cleanup(resetBindClientHooks)
		listInstallationRepos = func(context.Context, *gh.Client) ([]common.Repository, error) {
			return []common.Repository{{ID: 1, Name: "repo", URL: "https://github.com/acme/repo"}}, nil
		}
		newInstallationClient = func(core.IntegrationContext, int64, string) (*gh.Client, error) {
			return gh.NewClient(nil), nil
		}

		integration := pendingHostedIntegration("csrf")
		integration.Metadata = common.Metadata{
			State:            "csrf",
			HostedApp:        true,
			InstallRequested: true,
			GitHubApp:        common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
			PendingInstallations: []common.PendingInstallation{
				{ID: "11", AccountLogin: "acme"},
			},
		}
		ctx, rec := hostedRequestContext(integration, "/api/v1/github/app/bind?state=csrf&installation_id=11", nil)

		g.afterHostedAppBind(ctx)

		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "ready", integration.State)
		metadata := integration.Metadata.(common.Metadata)
		assert.Equal(t, "11", metadata.InstallationID)
		// The picker state stays, so onboarding can move the connection to
		// another account later.
		assert.Equal(t, "csrf", metadata.State)
		assert.Len(t, metadata.PendingInstallations, 1)
		assert.False(t, metadata.InstallRequested)
		assertNoPlaintextSecrets(t, integration)
	})

	t.Run("rebinds a bound connection to another allowed installation", func(t *testing.T) {
		t.Cleanup(resetBindClientHooks)
		listInstallationRepos = func(context.Context, *gh.Client) ([]common.Repository, error) {
			return []common.Repository{{ID: 2, Name: "other", URL: "https://github.com/octo/other"}}, nil
		}
		newInstallationClient = func(core.IntegrationContext, int64, string) (*gh.Client, error) {
			return gh.NewClient(nil), nil
		}
		newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) {
			return gh.NewClient(nil), nil
		}
		listAppInstallations = func(context.Context, *gh.Client) ([]common.PendingInstallation, error) {
			return nil, nil
		}

		integration := pendingHostedIntegration("csrf")
		integration.State = "ready"
		integration.Metadata = common.Metadata{
			State:          "csrf",
			HostedApp:      true,
			InstallationID: "11",
			Owner:          "acme",
			GitHubApp:      common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
			PendingInstallations: []common.PendingInstallation{
				{ID: "11", AccountLogin: "acme"},
				{ID: "22", AccountLogin: "octo", AccountType: "Organization"},
			},
		}
		ctx, rec := hostedRequestContext(integration, "/api/v1/github/app/bind?state=csrf&installation_id=22", nil)

		g.afterHostedAppBind(ctx)

		assert.Equal(t, http.StatusSeeOther, rec.Code)
		metadata := integration.Metadata.(common.Metadata)
		assert.Equal(t, "22", metadata.InstallationID)
		assert.Equal(t, "octo", metadata.Owner)
	})

	t.Run("keeps requests for other accounts after binding", func(t *testing.T) {
		t.Cleanup(resetBindClientHooks)
		listInstallationRepos = func(context.Context, *gh.Client) ([]common.Repository, error) {
			return []common.Repository{{ID: 1, Name: "repo", URL: "https://github.com/acme/repo"}}, nil
		}
		newInstallationClient = func(core.IntegrationContext, int64, string) (*gh.Client, error) {
			return gh.NewClient(nil), nil
		}

		integration := pendingHostedIntegration("csrf")
		integration.Metadata = common.Metadata{
			State:     "csrf",
			HostedApp: true,
			GitHubApp: common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
			InstallRequests: []common.InstallRequest{
				{ID: "1", AccountLogin: "acme", RequesterLogin: "member"},
				{ID: "2", AccountLogin: "octo", RequesterLogin: "member"},
			},
			PendingInstallations: []common.PendingInstallation{{ID: "11", AccountLogin: "acme"}},
		}
		ctx, rec := hostedRequestContext(integration, "/api/v1/github/app/bind?state=csrf&installation_id=11", nil)

		g.afterHostedAppBind(ctx)

		assert.Equal(t, http.StatusSeeOther, rec.Code)
		metadata := integration.Metadata.(common.Metadata)
		assert.Equal(t, []common.InstallRequest{{ID: "2", AccountLogin: "octo", RequesterLogin: "member"}}, metadata.InstallRequests)
		assert.Equal(t, "octo", metadata.InstallRequestedAccount)
	})

	t.Run("redirects a bound connection when the state does not match", func(t *testing.T) {
		integration := pendingHostedIntegration("csrf")
		integration.State = "ready"
		integration.Metadata = common.Metadata{
			State:          "csrf",
			HostedApp:      true,
			InstallationID: "11",
			Owner:          "acme",
			GitHubApp:      common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
			PendingInstallations: []common.PendingInstallation{
				{ID: "11", AccountLogin: "acme"},
			},
		}
		ctx, rec := hostedRequestContext(integration, "/api/v1/github/app/bind?state=other&installation_id=11", nil)

		g.afterHostedAppBind(ctx)

		assert.Equal(t, http.StatusSeeOther, rec.Code)
		metadata := integration.Metadata.(common.Metadata)
		assert.Equal(t, "11", metadata.InstallationID)
		assert.Equal(t, "acme", metadata.Owner)
	})
}

func Test__afterAppInstallationLegacy_installRequest(t *testing.T) {
	t.Run("accepts request without installation id", func(t *testing.T) {
		integration := pendingHostedIntegration("csrf")
		ctx, rec := hostedRequestContext(
			integration,
			"/api/v1/github/app/setup?state=csrf&setup_action=request",
			nil,
		)

		(&GitHub{}).afterAppInstallationLegacy(ctx)

		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(
			t,
			"https://app.example/org-1/settings/integrations/11111111-1111-1111-1111-111111111111?githubSetup=request&githubIntegrationId=11111111-1111-1111-1111-111111111111",
			rec.Header().Get("Location"),
		)
		assert.Equal(t, "pending", integration.State)
		assert.Empty(t, integration.Metadata.(common.Metadata).InstallationID)
		assert.True(t, integration.Metadata.(common.Metadata).InstallRequested)
		assert.NotEmpty(t, integration.Metadata.(common.Metadata).InstallRequests[0].CreatedAt)
	})

	t.Run("persists the requested GitHub organization", func(t *testing.T) {
		integration := pendingHostedIntegration("csrf")
		ctx, rec := hostedRequestContext(
			integration,
			"/api/v1/github/app/setup?state=csrf&setup_action=request&account=acme",
			nil,
		)

		(&GitHub{}).afterAppInstallationLegacy(ctx)

		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(
			t,
			"https://app.example/org-1/settings/integrations/11111111-1111-1111-1111-111111111111?githubSetup=request&githubIntegrationId=11111111-1111-1111-1111-111111111111&githubOrg=acme",
			rec.Header().Get("Location"),
		)
		assert.Equal(t, "acme", integration.Metadata.(common.Metadata).InstallRequestedAccount)
	})

	t.Run("persists multiple requested GitHub organizations without choosing one", func(t *testing.T) {
		integration := pendingHostedIntegration("csrf")
		firstContext, _ := hostedRequestContext(
			integration,
			"/api/v1/github/app/setup?state=csrf&setup_action=request&account=acme",
			nil,
		)
		(&GitHub{}).afterAppInstallationLegacy(firstContext)

		secondContext, _ := hostedRequestContext(
			integration,
			"/api/v1/github/app/setup?state=csrf&setup_action=request&account=octo",
			nil,
		)
		(&GitHub{}).afterAppInstallationLegacy(secondContext)

		metadata := integration.Metadata.(common.Metadata)
		require.Len(t, metadata.InstallRequests, 2)
		assert.Empty(t, metadata.InstallRequestedAccount)
	})

	t.Run("returns to the stored onboarding path instead of settings", func(t *testing.T) {
		integration := pendingHostedIntegration("csrf")
		ctx, rec := hostedRequestContext(
			integration,
			"/api/v1/github/app/setup?state=csrf&setup_action=request&account=acme",
			nil,
		)
		ctx.Request.AddCookie(&http.Cookie{
			Name:  integrationSetupReturnCookie,
			Value: "/org-1/workspaces/APP/setup?step=vcs&pick=newest",
		})

		(&GitHub{}).afterAppInstallationLegacy(ctx)

		assert.Equal(t, http.StatusSeeOther, rec.Code)
		location, err := url.Parse(rec.Header().Get("Location"))
		require.NoError(t, err)
		assert.Equal(t, "/org-1/workspaces/APP/setup", location.Path)
		assert.Equal(t, "vcs", location.Query().Get("step"))
		assert.Equal(t, "newest", location.Query().Get("pick"))
		assert.Equal(t, "request", location.Query().Get("githubSetup"))
		assert.Equal(t, "acme", location.Query().Get("githubOrg"))
		assert.True(t, integration.Metadata.(common.Metadata).InstallRequested)
	})

	t.Run("rejects request with a mismatched state", func(t *testing.T) {
		integration := pendingHostedIntegration("csrf")
		ctx, rec := hostedRequestContext(
			integration,
			"/api/v1/github/app/setup?state=other&setup_action=request",
			nil,
		)

		(&GitHub{}).afterAppInstallationLegacy(ctx)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "pending", integration.State)
		assert.False(t, integration.Metadata.(common.Metadata).InstallRequested)
	})
}

func Test__afterAppInstallationLegacy_pendingAllowlist(t *testing.T) {
	t.Cleanup(resetBindClientHooks)
	listInstallationRepos = func(context.Context, *gh.Client) ([]common.Repository, error) {
		return []common.Repository{{ID: 1, Name: "repo", URL: "https://github.com/acme/repo"}}, nil
	}
	newInstallationClient = func(core.IntegrationContext, int64, string) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}

	t.Run("rejects installation id outside the allowlist", func(t *testing.T) {
		setHostedAppEnv(t)
		integration := pendingHostedIntegration("csrf")
		integration.Metadata = common.Metadata{
			State:     "csrf",
			HostedApp: true,
			GitHubApp: common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
			PendingInstallations: []common.PendingInstallation{
				{ID: "11", AccountLogin: "acme"},
			},
		}
		ctx, rec := hostedRequestContext(
			integration,
			"/api/v1/github/app/setup?state=csrf&installation_id=99&setup_action=install",
			nil,
		)

		(&GitHub{}).afterAppInstallationLegacy(ctx)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.NotEqual(t, "ready", integration.State)
	})

	t.Run("oauth off-list setup redirects to authorize", func(t *testing.T) {
		setHostedAppOAuthEnv(t)
		integration := pendingHostedIntegration("csrf")
		integration.Metadata = common.Metadata{
			State:     "csrf",
			HostedApp: true,
			GitHubApp: common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
			PendingInstallations: []common.PendingInstallation{
				{ID: "11", AccountLogin: "acme"},
			},
		}
		ctx, rec := hostedRequestContext(
			integration,
			"/api/v1/github/app/setup?state=csrf&installation_id=99&setup_action=install",
			nil,
		)

		(&GitHub{}).afterAppInstallationLegacy(ctx)

		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Contains(t, rec.Header().Get("Location"), "https://github.com/login/oauth/authorize?")
		assert.NotEqual(t, "ready", integration.State)
	})

	t.Run("oauth without allowlist redirects to authorize", func(t *testing.T) {
		setHostedAppOAuthEnv(t)
		integration := pendingHostedIntegration("csrf")
		ctx, rec := hostedRequestContext(
			integration,
			"/api/v1/github/app/setup?state=csrf&installation_id=99&setup_action=install",
			nil,
		)

		(&GitHub{}).afterAppInstallationLegacy(ctx)

		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Contains(t, rec.Header().Get("Location"), "https://github.com/login/oauth/authorize?")
		assert.NotEqual(t, "ready", integration.State)
		assert.Empty(t, integration.Metadata.(common.Metadata).InstallationID)
	})

	t.Run("binds an allowlisted installation", func(t *testing.T) {
		integration := pendingHostedIntegration("csrf")
		integration.Metadata = common.Metadata{
			State:     "csrf",
			HostedApp: true,
			GitHubApp: common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
			PendingInstallations: []common.PendingInstallation{
				{ID: "11", AccountLogin: "acme"},
			},
		}
		ctx, rec := hostedRequestContext(
			integration,
			"/api/v1/github/app/setup?state=csrf&installation_id=11&setup_action=install",
			nil,
		)

		(&GitHub{}).afterAppInstallationLegacy(ctx)

		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "ready", integration.State)
		metadata := integration.Metadata.(common.Metadata)
		assert.Equal(t, "11", metadata.InstallationID)
	})
}

func Test__Sync_hostedAppKeepsPendingMetadata(t *testing.T) {
	setHostedAppOAuthEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)

	integrationCtx := &contexts.IntegrationContext{
		Metadata: common.Metadata{
			State:           "csrf-keep",
			HostedApp:       true,
			StartedByUserID: "starter-user",
			SetupReturnPath: "/onboarding?attempt=old&step=vcs",
			PendingInstallations: []common.PendingInstallation{
				{ID: "11", AccountLogin: "acme"},
				{ID: "22", AccountLogin: "octo"},
			},
			GitHubApp: common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
		},
	}

	require.NoError(t, (&GitHub{}).Sync(core.SyncContext{
		OrganizationID: "11111111-1111-1111-1111-111111111111",
		ActorUserID:    "other-user",
		BaseURL:        "https://app.example",
		Configuration:  Configuration{SetupReturnPath: "/onboarding?attempt=new&step=vcs"},
		Integration:    integrationCtx,
	}))

	metadata := integrationCtx.Metadata.(common.Metadata)
	assert.Equal(t, "csrf-keep", metadata.State)
	assert.Equal(t, "starter-user", metadata.StartedByUserID)
	assert.Equal(t, "/onboarding?attempt=new&step=vcs", metadata.SetupReturnPath)
	require.Len(t, metadata.PendingInstallations, 2)
	assert.Nil(t, integrationCtx.BrowserAction)
	// The picker page has no browser action, so the connect screen re-asks
	// the GitHub account through the authorize URL kept in metadata.
	assert.Contains(t, metadata.AuthorizeURL, "https://github.com/login/oauth/authorize?")
	assert.Contains(t, metadata.AuthorizeURL, "state=csrf-keep")
}

func Test__Sync_hostedAppOffersApprovedInstallInPicker(t *testing.T) {
	setHostedAppOAuthEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)
	t.Cleanup(resetBindClientHooks)

	listAppInstallations = func(context.Context, *gh.Client) ([]common.PendingInstallation, error) {
		return []common.PendingInstallation{{ID: "11", AccountLogin: "acme", AccountType: "Organization"}}, nil
	}
	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}

	integrationCtx := &contexts.IntegrationContext{
		State: "pending",
		Metadata: common.Metadata{
			State:                   "csrf",
			HostedApp:               true,
			InstallRequested:        true,
			InstallRequestedAccount: "acme",
			GitHubApp:               common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
			PendingInstallations: []common.PendingInstallation{
				{ID: "22", AccountLogin: "octo", AccountType: "User"},
			},
		},
	}

	require.NoError(t, (&GitHub{}).Sync(core.SyncContext{
		Logger:         logrus.NewEntry(logrus.New()),
		OrganizationID: "11111111-1111-1111-1111-111111111111",
		BaseURL:        "https://app.example",
		Integration:    integrationCtx,
	}))

	// A silent bind must not happen: the approved installation joins the
	// picker and the member still picks the account.
	assert.NotEqual(t, "ready", integrationCtx.State)
	metadata := integrationCtx.Metadata.(common.Metadata)
	assert.Empty(t, metadata.InstallationID)
	assert.Equal(t, "csrf", metadata.State)
	assert.False(t, metadata.InstallRequested)
	assert.Empty(t, metadata.InstallRequestedAccount)
	require.Len(t, metadata.PendingInstallations, 2)
	assert.Equal(t, "11", metadata.PendingInstallations[1].ID)
	assert.Equal(t, "acme", metadata.PendingInstallations[1].AccountLogin)
}

func Test__Sync_hostedAppDoesNotDuplicatePickerEntry(t *testing.T) {
	setHostedAppOAuthEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)
	t.Cleanup(resetBindClientHooks)

	listAppInstallations = func(context.Context, *gh.Client) ([]common.PendingInstallation, error) {
		return []common.PendingInstallation{{ID: "11", AccountLogin: "acme", AccountType: "Organization"}}, nil
	}
	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}

	integrationCtx := &contexts.IntegrationContext{
		State: "pending",
		Metadata: common.Metadata{
			State:                   "csrf",
			HostedApp:               true,
			InstallRequested:        true,
			InstallRequestedAccount: "acme",
			GitHubApp:               common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
			PendingInstallations: []common.PendingInstallation{
				{ID: "11", AccountLogin: "acme", AccountType: "Organization"},
			},
		},
	}

	require.NoError(t, (&GitHub{}).Sync(core.SyncContext{
		Logger:         logrus.NewEntry(logrus.New()),
		OrganizationID: "11111111-1111-1111-1111-111111111111",
		BaseURL:        "https://app.example",
		Integration:    integrationCtx,
	}))

	metadata := integrationCtx.Metadata.(common.Metadata)
	require.Len(t, metadata.PendingInstallations, 1)
	assert.False(t, metadata.InstallRequested)
}

func Test__Sync_hostedAppReconcilesMultipleRequestsWithoutDependingOnOrder(t *testing.T) {
	setHostedAppOAuthEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)
	t.Cleanup(resetBindClientHooks)

	listAppInstallationRequests = func(context.Context, *gh.Client, string) ([]common.InstallRequest, error) {
		return []common.InstallRequest{
			{ID: "2", AccountLogin: "octo", RequesterLogin: "member"},
		}, nil
	}
	listAppInstallations = func(context.Context, *gh.Client) ([]common.PendingInstallation, error) {
		return []common.PendingInstallation{{ID: "11", AccountLogin: "acme", AccountType: "Organization"}}, nil
	}
	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) { return gh.NewClient(nil), nil }

	integrationCtx := &contexts.IntegrationContext{
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

	require.NoError(t, (&GitHub{}).Sync(core.SyncContext{
		Logger:         logrus.NewEntry(logrus.New()),
		OrganizationID: "11111111-1111-1111-1111-111111111111",
		BaseURL:        "https://app.example",
		Integration:    integrationCtx,
	}))

	metadata := integrationCtx.Metadata.(common.Metadata)
	require.Equal(t, []common.InstallRequest{{ID: "2", AccountLogin: "octo", RequesterLogin: "member"}}, metadata.InstallRequests)
	assert.Equal(t, "octo", metadata.InstallRequestedAccount)
	require.Len(t, metadata.PendingInstallations, 1)
	assert.Equal(t, "acme", metadata.PendingInstallations[0].AccountLogin)
}

// A member can request an installation without the request callback reaching
// this server, for example when the callback URL was unreachable. Sync must
// still find that member's open requests on GitHub.
func Test__Sync_hostedAppDiscoversInstallRequestWithoutCallback(t *testing.T) {
	setHostedAppOAuthEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)
	t.Cleanup(resetBindClientHooks)

	listAppInstallationRequests = func(context.Context, *gh.Client, string) ([]common.InstallRequest, error) {
		return []common.InstallRequest{
			{ID: "7", AccountLogin: "kittens-inc-1", RequesterLogin: "member"},
		}, nil
	}
	listAppInstallations = func(context.Context, *gh.Client) ([]common.PendingInstallation, error) {
		return []common.PendingInstallation{{ID: "11", AccountLogin: "acme", AccountType: "Organization"}}, nil
	}
	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) { return gh.NewClient(nil), nil }

	integrationCtx := &contexts.IntegrationContext{
		State: "pending",
		Metadata: common.Metadata{
			State:                "csrf",
			HostedApp:            true,
			StartedByGitHubLogin: "member",
			GitHubApp:            common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
			PendingInstallations: []common.PendingInstallation{
				{ID: "11", AccountLogin: "acme", AccountType: "Organization"},
			},
		},
	}

	require.NoError(t, (&GitHub{}).Sync(core.SyncContext{
		Logger:         logrus.NewEntry(logrus.New()),
		OrganizationID: "11111111-1111-1111-1111-111111111111",
		BaseURL:        "https://app.example",
		Integration:    integrationCtx,
	}))

	metadata := integrationCtx.Metadata.(common.Metadata)
	assert.True(t, metadata.InstallRequested)
	require.Equal(t, []common.InstallRequest{{ID: "7", AccountLogin: "kittens-inc-1", RequesterLogin: "member"}}, metadata.InstallRequests)
	require.Len(t, metadata.PendingInstallations, 1)
	assert.Equal(t, "acme", metadata.PendingInstallations[0].AccountLogin)
}

func Test__Sync_hostedReadyAppReconcilesApprovedRequest(t *testing.T) {
	setHostedAppOAuthEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)
	t.Cleanup(resetBindClientHooks)

	listAppInstallationRequests = func(context.Context, *gh.Client, string) ([]common.InstallRequest, error) {
		return nil, nil
	}
	listAppInstallations = func(context.Context, *gh.Client) ([]common.PendingInstallation, error) {
		return []common.PendingInstallation{{ID: "22", AccountLogin: "octo", AccountType: "Organization"}}, nil
	}
	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) { return gh.NewClient(nil), nil }

	integrationCtx := &contexts.IntegrationContext{
		State: "ready",
		Metadata: common.Metadata{
			State:                "csrf",
			HostedApp:            true,
			InstallationID:       "11",
			Owner:                "acme",
			StartedByGitHubLogin: "member",
			InstallRequested:     true,
			InstallRequests: []common.InstallRequest{
				{ID: "2", AccountLogin: "octo", RequesterLogin: "member"},
			},
			GitHubApp: common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
		},
	}

	require.NoError(t, (&GitHub{}).Sync(core.SyncContext{
		Logger:         logrus.NewEntry(logrus.New()),
		OrganizationID: "11111111-1111-1111-1111-111111111111",
		BaseURL:        "https://app.example",
		Integration:    integrationCtx,
	}))

	metadata := integrationCtx.Metadata.(common.Metadata)
	assert.Equal(t, "11", metadata.InstallationID)
	assert.False(t, metadata.InstallRequested)
	require.Len(t, metadata.PendingInstallations, 1)
	assert.Equal(t, "octo", metadata.PendingInstallations[0].AccountLogin)
}

func Test__listAppInstallationRequestsFromGitHubReturnsAllRequestsForRequester(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/app/installation-requests", request.URL.Path)
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`[
			{"id":2,"account":{"login":"octo"},"requester":{"login":"member"},"created_at":"2026-09-08T12:00:00Z"},
			{"id":1,"account":{"login":"acme"},"requester":{"login":"member"},"created_at":"2026-09-08T11:00:00Z"},
			{"id":3,"account":{"login":"other"},"requester":{"login":"another-user"}}
		]`))
	}))
	t.Cleanup(server.Close)

	client := gh.NewClient(server.Client())
	baseURL, err := url.Parse(server.URL + "/")
	require.NoError(t, err)
	client.BaseURL = baseURL

	requests, err := listAppInstallationRequestsFromGitHub(context.Background(), client, "member")
	require.NoError(t, err)
	require.Len(t, requests, 2)
	assert.Equal(t, "2", requests[0].ID)
	assert.Equal(t, "octo", requests[0].AccountLogin)
	assert.Equal(t, "1", requests[1].ID)
	assert.Equal(t, "acme", requests[1].AccountLogin)
}

func Test__Sync_hostedAppPreservesRequestsWhenGitHubLookupFails(t *testing.T) {
	setHostedAppOAuthEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)
	t.Cleanup(resetBindClientHooks)

	listAppInstallationRequests = func(context.Context, *gh.Client, string) ([]common.InstallRequest, error) {
		return nil, errors.New("GitHub unavailable")
	}
	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) { return gh.NewClient(nil), nil }

	existingRequest := common.InstallRequest{ID: "1", AccountLogin: "acme", RequesterLogin: "member"}
	integrationCtx := &contexts.IntegrationContext{
		State: "pending",
		Metadata: common.Metadata{
			State:                "csrf",
			HostedApp:            true,
			StartedByGitHubLogin: "member",
			InstallRequested:     true,
			InstallRequests:      []common.InstallRequest{existingRequest},
			GitHubApp:            common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
		},
	}

	require.NoError(t, (&GitHub{}).Sync(core.SyncContext{
		Logger:         logrus.NewEntry(logrus.New()),
		OrganizationID: "11111111-1111-1111-1111-111111111111",
		BaseURL:        "https://app.example",
		Integration:    integrationCtx,
	}))

	metadata := integrationCtx.Metadata.(common.Metadata)
	assert.Equal(t, []common.InstallRequest{existingRequest}, metadata.InstallRequests)
	assert.True(t, metadata.InstallRequested)
}

func Test__Sync_hostedAppFindsRequestedAccountOnGitHub(t *testing.T) {
	setHostedAppOAuthEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)
	t.Cleanup(resetBindClientHooks)

	// GitHub's request callback does not name the requested account, so the
	// connection stored only the requester's login. Sync finds the open
	// install request on GitHub and records the account.
	listAppInstallationRequests = func(_ context.Context, _ *gh.Client, requesterLogin string) ([]common.InstallRequest, error) {
		if requesterLogin == "member" {
			return []common.InstallRequest{{ID: "1", AccountLogin: "acme", RequesterLogin: "member"}}, nil
		}
		return nil, nil
	}
	listAppInstallations = func(context.Context, *gh.Client) ([]common.PendingInstallation, error) {
		return nil, nil // The request is still waiting for an approval.
	}
	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}

	integrationCtx := &contexts.IntegrationContext{
		State: "pending",
		Metadata: common.Metadata{
			State:                "csrf",
			HostedApp:            true,
			InstallRequested:     true,
			StartedByGitHubLogin: "member",
			GitHubApp:            common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
		},
	}

	require.NoError(t, (&GitHub{}).Sync(core.SyncContext{
		Logger:         logrus.NewEntry(logrus.New()),
		OrganizationID: "11111111-1111-1111-1111-111111111111",
		BaseURL:        "https://app.example",
		Integration:    integrationCtx,
	}))

	assert.NotEqual(t, "ready", integrationCtx.State)
	metadata := integrationCtx.Metadata.(common.Metadata)
	assert.Equal(t, "acme", metadata.InstallRequestedAccount)
	assert.True(t, metadata.InstallRequested)
	require.Len(t, metadata.InstallRequests, 1)
	assert.Equal(t, "1", metadata.InstallRequests[0].ID)
}

func Test__Sync_hostedAppOffersRequestWithoutStoredAccountInPicker(t *testing.T) {
	setHostedAppOAuthEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)
	t.Cleanup(resetBindClientHooks)

	listAppInstallationRequests = func(_ context.Context, _ *gh.Client, requesterLogin string) ([]common.InstallRequest, error) {
		if requesterLogin == "member" {
			return []common.InstallRequest{{ID: "1", AccountLogin: "acme", RequesterLogin: "member"}}, nil
		}
		return nil, nil
	}
	listAppInstallations = func(context.Context, *gh.Client) ([]common.PendingInstallation, error) {
		return []common.PendingInstallation{{ID: "11", AccountLogin: "acme", AccountType: "Organization"}}, nil
	}
	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}

	integrationCtx := &contexts.IntegrationContext{
		State: "pending",
		Metadata: common.Metadata{
			State:                "csrf",
			HostedApp:            true,
			InstallRequested:     true,
			StartedByGitHubLogin: "member",
			GitHubApp:            common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
		},
	}

	require.NoError(t, (&GitHub{}).Sync(core.SyncContext{
		Logger:         logrus.NewEntry(logrus.New()),
		OrganizationID: "11111111-1111-1111-1111-111111111111",
		BaseURL:        "https://app.example",
		Integration:    integrationCtx,
	}))

	assert.NotEqual(t, "ready", integrationCtx.State)
	metadata := integrationCtx.Metadata.(common.Metadata)
	assert.Empty(t, metadata.InstallationID)
	require.Len(t, metadata.PendingInstallations, 1)
	assert.Equal(t, "11", metadata.PendingInstallations[0].ID)
	assert.Equal(t, "acme", metadata.PendingInstallations[0].AccountLogin)
	assert.False(t, metadata.InstallRequested)
}

func Test__Sync_hostedAppKeepsWaitingWhenRequestNotApproved(t *testing.T) {
	setHostedAppOAuthEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)
	t.Cleanup(resetBindClientHooks)

	listAppInstallations = func(context.Context, *gh.Client) ([]common.PendingInstallation, error) {
		return nil, nil
	}
	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}

	integrationCtx := &contexts.IntegrationContext{
		State: "pending",
		Metadata: common.Metadata{
			State:                   "csrf",
			HostedApp:               true,
			InstallRequested:        true,
			InstallRequestedAccount: "acme",
			GitHubApp:               common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
		},
	}

	require.NoError(t, (&GitHub{}).Sync(core.SyncContext{
		Logger:         logrus.NewEntry(logrus.New()),
		OrganizationID: "11111111-1111-1111-1111-111111111111",
		BaseURL:        "https://app.example",
		Integration:    integrationCtx,
	}))

	assert.NotEqual(t, "ready", integrationCtx.State)
	metadata := integrationCtx.Metadata.(common.Metadata)
	assert.Empty(t, metadata.InstallationID)
	assert.True(t, metadata.InstallRequested)
	assert.Equal(t, "csrf", metadata.State)
}

func Test__Sync_hostedAppKeepsSinglePendingInstallation(t *testing.T) {
	setHostedAppOAuthEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)

	integrationCtx := &contexts.IntegrationContext{
		Metadata: common.Metadata{
			State:     "csrf-keep",
			HostedApp: true,
			PendingInstallations: []common.PendingInstallation{
				{ID: "11", AccountLogin: "acme"},
			},
			GitHubApp: common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
		},
	}

	require.NoError(t, (&GitHub{}).Sync(core.SyncContext{
		OrganizationID: "11111111-1111-1111-1111-111111111111",
		BaseURL:        "https://app.example",
		Integration:    integrationCtx,
	}))

	metadata := integrationCtx.Metadata.(common.Metadata)
	assert.Equal(t, "csrf-keep", metadata.State)
	require.Len(t, metadata.PendingInstallations, 1)
	assert.Nil(t, integrationCtx.BrowserAction)
}

func Test__Sync_hostedAppKeepsSetupReturnPathWhenConfigOmitsIt(t *testing.T) {
	setHostedAppOAuthEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)

	integrationCtx := &contexts.IntegrationContext{
		Metadata: common.Metadata{
			State:           "csrf-keep",
			HostedApp:       true,
			SetupReturnPath: "/onboarding?attempt=old&step=vcs",
			GitHubApp:       common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
		},
	}

	require.NoError(t, (&GitHub{}).Sync(core.SyncContext{
		OrganizationID: "11111111-1111-1111-1111-111111111111",
		BaseURL:        "https://app.example",
		Integration:    integrationCtx,
	}))

	assert.Equal(t, "/onboarding?attempt=old&step=vcs", integrationCtx.Metadata.(common.Metadata).SetupReturnPath)
}

func Test__afterAppInstallationLegacy_afterZeroInstallOAuth(t *testing.T) {
	setHostedAppOAuthEnv(t)
	integration := pendingHostedIntegration("csrf")
	ctx, rec := hostedRequestContext(
		integration,
		"/api/v1/github/app/setup?state=csrf&installation_id=11&setup_action=install",
		nil,
	)

	(&GitHub{}).afterAppInstallationLegacy(ctx)

	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Contains(t, rec.Header().Get("Location"), "https://github.com/login/oauth/authorize?")
	assert.NotEqual(t, "ready", integration.State)
	assert.Empty(t, integration.Metadata.(common.Metadata).InstallationID)
}

func Test__afterAppInstallationLegacy_afterZeroInstallWithoutOAuth(t *testing.T) {
	setHostedAppEnv(t)
	integration := pendingHostedIntegration("csrf")
	ctx, rec := hostedRequestContext(
		integration,
		"/api/v1/github/app/setup?state=csrf&installation_id=11&setup_action=install",
		nil,
	)

	(&GitHub{}).afterAppInstallationLegacy(ctx)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.NotEqual(t, "ready", integration.State)
	assert.Empty(t, integration.Metadata.(common.Metadata).InstallationID)
}

func setHostedAppOAuthEnv(t *testing.T) {
	t.Helper()
	setHostedAppEnv(t)
	t.Setenv(common.EnvGitHubAppClientID, "Iv1.abc")
	t.Setenv(common.EnvGitHubAppClientSecret, "app-secret")
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

func hostedRequestContext(integration *contexts.IntegrationContext, path string, httpCtx core.HTTPContext) (core.HTTPRequestContext, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	return core.HTTPRequestContext{
		Logger:         logrus.NewEntry(logrus.New()),
		Request:        httptest.NewRequest(http.MethodGet, path, nil),
		Response:       rec,
		OrganizationID: "org-1",
		BaseURL:        "https://app.example",
		HTTP:           httpCtx,
		Integration:    integration,
	}, rec
}

func oauthHTTP(responses ...*http.Response) *contexts.HTTPContext {
	return &contexts.HTTPContext{Responses: responses}
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func resetBindClientHooks() {
	newInstallationClient = newClientForAppInstallation
	newAppJWTClient = newClientForApp
	listInstallationRepos = listInstallationRepositories
	listAppInstallations = listAppInstallationsFromGitHub
	listAppInstallationRequests = listAppInstallationRequestsFromGitHub
}

func assertNoPlaintextSecrets(t *testing.T, integration *contexts.IntegrationContext) {
	t.Helper()
	raw, err := json.Marshal(integration.Metadata)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "user-token")
	assert.NotContains(t, string(raw), "app-secret")
	assert.NotContains(t, string(raw), "clientSecret")
	for _, secret := range integration.CurrentSecrets {
		assert.NotEqual(t, "user-token", string(secret.Value))
		assert.NotEqual(t, "app-secret", string(secret.Value))
	}
}

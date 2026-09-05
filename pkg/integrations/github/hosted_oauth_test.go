package github

import (
	"context"
	"encoding/json"
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
		require.Len(t, metadata.PendingInstallations, 2)
		assert.Equal(t, "11", metadata.PendingInstallations[0].ID)
		assert.Empty(t, integration.CurrentSecrets)
		assertNoPlaintextSecrets(t, integration)
		require.Len(t, httpCtx.Requests, 2)
		assert.Equal(t, "/user/installations", httpCtx.Requests[1].URL.Path)
		assert.NotContains(t, httpCtx.Requests[1].URL.Path, "/app/installations")
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
		assert.Empty(t, metadata.PendingInstallations)
		assert.False(t, metadata.InstallRequested)
		assertNoPlaintextSecrets(t, integration)
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
			"https://app.example/org-1/settings/integrations/11111111-1111-1111-1111-111111111111?githubSetup=request",
			rec.Header().Get("Location"),
		)
		assert.Equal(t, "pending", integration.State)
		assert.Empty(t, integration.Metadata.(common.Metadata).InstallationID)
		assert.True(t, integration.Metadata.(common.Metadata).InstallRequested)
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
			"https://app.example/org-1/settings/integrations/11111111-1111-1111-1111-111111111111?githubSetup=request&githubOrg=acme",
			rec.Header().Get("Location"),
		)
		assert.Equal(t, "acme", integration.Metadata.(common.Metadata).InstallRequestedAccount)
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

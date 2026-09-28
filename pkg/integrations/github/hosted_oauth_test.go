package github

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
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

func Test__Sync_hostedInstallURL(t *testing.T) {
	setHostedAppEnv(t)
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
	assert.Contains(t, integrationCtx.BrowserAction.URL, "https://github.com/apps/superplane/installations/new?state=")
	assert.NotContains(t, integrationCtx.BrowserAction.URL, "login/oauth/authorize")
	assert.Empty(t, integrationCtx.Metadata.(common.Metadata).AuthorizeURL)
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
		listAppInstallationsDetailed = func(core.IntegrationContext, int64) ([]hostedInstallationSnapshot, error) {
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

func Test__afterAppInstallationLegacy_firstClaim(t *testing.T) {
	setHostedAppEnv(t)

	t.Run("rejects an unknown installation", func(t *testing.T) {
		stubHostedClaim(t, hostedInstallationSnapshot{}, false)
		integration := pendingHostedIntegration("csrf")
		ctx, rec := hostedRequestContext(
			integration,
			"/api/v1/github/app/setup?state=csrf&installation_id=99&setup_action=install",
			nil,
		)

		(&GitHub{}).afterAppInstallationLegacy(ctx)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.NotEqual(t, "ready", integration.State)
		assert.Empty(t, integration.Metadata.(common.Metadata).PendingInstallations)
	})

	t.Run("rejects a stale webhook row", func(t *testing.T) {
		stubHostedClaim(t, hostedInstallationSnapshot{
			ID:           "99",
			AccountLogin: "acme",
			LastEventAt:  time.Now().Add(-30 * time.Minute),
		}, false)
		integration := pendingHostedIntegration("csrf")
		ctx, rec := hostedRequestContext(
			integration,
			"/api/v1/github/app/setup?state=csrf&installation_id=99&setup_action=install",
			nil,
		)

		(&GitHub{}).afterAppInstallationLegacy(ctx)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Empty(t, integration.Metadata.(common.Metadata).PendingInstallations)
	})

	t.Run("rejects a deleted installation", func(t *testing.T) {
		stubHostedClaim(t, hostedInstallationSnapshot{
			ID:           "99",
			AccountLogin: "acme",
			LastEventAt:  time.Now(),
			Deleted:      true,
		}, false)
		integration := pendingHostedIntegration("csrf")
		ctx, rec := hostedRequestContext(
			integration,
			"/api/v1/github/app/setup?state=csrf&installation_id=99&setup_action=install",
			nil,
		)

		(&GitHub{}).afterAppInstallationLegacy(ctx)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("rejects an installation used by another organization", func(t *testing.T) {
		stubHostedClaim(t, hostedInstallationSnapshot{
			ID:           "99",
			AccountLogin: "acme",
			LastEventAt:  time.Now(),
		}, true)
		integration := pendingHostedIntegration("csrf")
		ctx, rec := hostedRequestContext(
			integration,
			"/api/v1/github/app/setup?state=csrf&installation_id=99&setup_action=install",
			nil,
		)

		(&GitHub{}).afterAppInstallationLegacy(ctx)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("offers a fresh installation to the picker", func(t *testing.T) {
		stubHostedClaim(t, hostedInstallationSnapshot{
			ID:           "11",
			AccountLogin: "acme",
			AccountType:  "Organization",
			LastEventAt:  time.Now(),
		}, false)
		integration := pendingHostedIntegration("csrf")
		ctx, rec := hostedRequestContext(
			integration,
			"/api/v1/github/app/setup?state=csrf&installation_id=11&setup_action=install",
			nil,
		)

		(&GitHub{}).afterAppInstallationLegacy(ctx)

		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.NotEqual(t, "ready", integration.State)
		metadata := integration.Metadata.(common.Metadata)
		assert.Empty(t, metadata.InstallationID)
		require.Len(t, metadata.PendingInstallations, 1)
		assert.Equal(t, "11", metadata.PendingInstallations[0].ID)
		assert.Equal(t, "acme", metadata.PendingInstallations[0].AccountLogin)
	})

	t.Run("keeps an already offered installation on the picker", func(t *testing.T) {
		stubHostedClaim(t, hostedInstallationSnapshot{}, false)
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
		assert.NotEqual(t, "ready", integration.State)
		assert.Empty(t, integration.Metadata.(common.Metadata).InstallationID)
		require.Len(t, integration.Metadata.(common.Metadata).PendingInstallations, 1)
	})

	t.Run("uses the GitHub App JWT when the webhook row is missing", func(t *testing.T) {
		t.Cleanup(resetHostedClaimHooks)
		installationUsedByOtherOrg = func(string, string) (bool, error) { return false, nil }
		findHostedInstallation = func(core.HTTPRequestContext, string) (*hostedInstallationSnapshot, error) {
			return nil, nil
		}
		fetchAppInstallation = func(_ core.HTTPRequestContext, _ int64, installationID string) (hostedInstallationSnapshot, error) {
			return hostedInstallationSnapshot{
				ID:           installationID,
				AccountLogin: "acme",
				AccountType:  "Organization",
				CreatedAt:    time.Now(),
			}, nil
		}
		integration := pendingHostedIntegration("csrf")
		ctx, rec := hostedRequestContext(
			integration,
			"/api/v1/github/app/setup?state=csrf&installation_id=11&setup_action=install",
			nil,
		)

		(&GitHub{}).afterAppInstallationLegacy(ctx)

		assert.Equal(t, http.StatusSeeOther, rec.Code)
		require.Len(t, integration.Metadata.(common.Metadata).PendingInstallations, 1)
		assert.Equal(t, "acme", integration.Metadata.(common.Metadata).PendingInstallations[0].AccountLogin)
	})

	t.Run("rejects a stale GitHub App JWT fallback", func(t *testing.T) {
		t.Cleanup(resetHostedClaimHooks)
		installationUsedByOtherOrg = func(string, string) (bool, error) { return false, nil }
		findHostedInstallation = func(core.HTTPRequestContext, string) (*hostedInstallationSnapshot, error) {
			return nil, nil
		}
		fetchAppInstallation = func(_ core.HTTPRequestContext, _ int64, installationID string) (hostedInstallationSnapshot, error) {
			return hostedInstallationSnapshot{
				ID:           installationID,
				AccountLogin: "acme",
				CreatedAt:    time.Now().Add(-30 * time.Minute),
			}, nil
		}
		integration := pendingHostedIntegration("csrf")
		ctx, rec := hostedRequestContext(
			integration,
			"/api/v1/github/app/setup?state=csrf&installation_id=11&setup_action=install",
			nil,
		)

		(&GitHub{}).afterAppInstallationLegacy(ctx)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func Test__Sync_hostedAppKeepsPendingMetadata(t *testing.T) {
	setHostedAppEnv(t)
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
	assert.Empty(t, metadata.AuthorizeURL)
}

func Test__Sync_hostedAppOffersApprovedInstallInPicker(t *testing.T) {
	setHostedAppEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)
	t.Cleanup(resetBindClientHooks)
	stubHostedAdopt(t, "acme", "11", "Organization")

	listAppInstallationsDetailed = func(core.IntegrationContext, int64) ([]hostedInstallationSnapshot, error) {
		return []hostedInstallationSnapshot{{ID: "11", AccountLogin: "acme", AccountType: "Organization"}}, nil
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
	setHostedAppEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)
	t.Cleanup(resetBindClientHooks)
	stubHostedAdopt(t, "acme", "11", "Organization")

	listAppInstallationsDetailed = func(core.IntegrationContext, int64) ([]hostedInstallationSnapshot, error) {
		return []hostedInstallationSnapshot{{ID: "11", AccountLogin: "acme", AccountType: "Organization"}}, nil
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
	setHostedAppEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)
	t.Cleanup(resetBindClientHooks)
	stubHostedAdopt(t, "acme", "11", "Organization")

	listAppInstallationRequests = func(context.Context, *gh.Client, string) ([]common.InstallRequest, error) {
		return []common.InstallRequest{
			{ID: "2", AccountLogin: "octo", RequesterLogin: "member"},
		}, nil
	}
	listAppInstallationsDetailed = func(core.IntegrationContext, int64) ([]hostedInstallationSnapshot, error) {
		return []hostedInstallationSnapshot{{ID: "11", AccountLogin: "acme", AccountType: "Organization"}}, nil
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

func Test__Sync_hostedReadyAppReconcilesApprovedRequest(t *testing.T) {
	setHostedAppEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)
	t.Cleanup(resetBindClientHooks)
	stubHostedAdopt(t, "octo", "22", "Organization")

	listAppInstallationRequests = func(context.Context, *gh.Client, string) ([]common.InstallRequest, error) {
		return nil, nil
	}
	listAppInstallationsDetailed = func(core.IntegrationContext, int64) ([]hostedInstallationSnapshot, error) {
		return []hostedInstallationSnapshot{{ID: "22", AccountLogin: "octo", AccountType: "Organization"}}, nil
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
	setHostedAppEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)
	t.Cleanup(resetBindClientHooks)

	stubHostedAdoptNone(t)
	listAppInstallationsDetailed = func(core.IntegrationContext, int64) ([]hostedInstallationSnapshot, error) {
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

func Test__Sync_hostedAppKeepsWaitingWhenRequestNotApproved(t *testing.T) {
	stubHostedAdoptNone(t)

	setHostedAppEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)
	t.Cleanup(resetBindClientHooks)

	listAppInstallationsDetailed = func(core.IntegrationContext, int64) ([]hostedInstallationSnapshot, error) {
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
	setHostedAppEnv(t)
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
	setHostedAppEnv(t)
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

func stubHostedClaim(t *testing.T, record hostedInstallationSnapshot, usedByOtherOrg bool) {
	t.Helper()
	t.Cleanup(resetHostedClaimHooks)
	installationUsedByOtherOrg = func(string, string) (bool, error) { return usedByOtherOrg, nil }
	if record.ID == "" {
		findHostedInstallation = func(core.HTTPRequestContext, string) (*hostedInstallationSnapshot, error) {
			return nil, nil
		}
		fetchAppInstallation = func(core.HTTPRequestContext, int64, string) (hostedInstallationSnapshot, error) {
			return hostedInstallationSnapshot{}, nil
		}
		return
	}
	findHostedInstallation = func(_ core.HTTPRequestContext, installationID string) (*hostedInstallationSnapshot, error) {
		if installationID != record.ID {
			return nil, nil
		}
		copy := record
		return &copy, nil
	}
}

func stubHostedAdopt(t *testing.T, accountLogin, installationID, accountType string) {
	t.Helper()
	t.Cleanup(resetHostedClaimHooks)
	installationUsedByOtherOrg = func(string, string) (bool, error) { return false, nil }
	findHostedInstallationByAccount = func(_ context.Context, login string) (*hostedInstallationSnapshot, error) {
		if !strings.EqualFold(login, accountLogin) {
			return nil, nil
		}
		return &hostedInstallationSnapshot{
			ID:           installationID,
			AccountLogin: accountLogin,
			AccountType:  accountType,
		}, nil
	}
}

func stubHostedAdoptNone(t *testing.T) {
	t.Helper()
	t.Cleanup(resetHostedClaimHooks)
	installationUsedByOtherOrg = func(string, string) (bool, error) { return false, nil }
	findHostedInstallationByAccount = func(context.Context, string) (*hostedInstallationSnapshot, error) {
		return nil, nil
	}
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

func resetBindClientHooks() {
	newInstallationClient = newClientForAppInstallation
	newAppJWTClient = newClientForApp
	listInstallationRepos = listInstallationRepositories
	listAppInstallationsDetailed = listAppInstallationsDetailedFromGitHub
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

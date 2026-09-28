package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
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
		State:         "pending",
		IntegrationID: "11111111-1111-1111-1111-111111111111",
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
	assert.Equal(
		t,
		"https://app.example/org-1/settings/integrations/11111111-1111-1111-1111-111111111111?githubSetup=complete&githubIntegrationId=11111111-1111-1111-1111-111111111111",
		rec.Header().Get("Location"),
	)
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

func TestHostedBindUsesOptInLocalInstallationAccessWithoutIdentity(t *testing.T) {
	enableUnverifiedDevelopmentRepositories(t)
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

func TestHostedBindIgnoresUnrelatedInstallationFailure(t *testing.T) {
	enableUnverifiedDevelopmentRepositories(t)
	setHostedAppEnv(t)
	t.Cleanup(resetBindClientHooks)

	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	listAppInstallations = func(context.Context, *gh.Client) ([]common.PendingInstallation, error) {
		return []common.PendingInstallation{
			{ID: "11", AccountLogin: "acme", AccountType: "Organization"},
			{ID: "22", AccountLogin: "unavailable", AccountType: "Organization"},
		}, nil
	}
	newInstallationClient = func(_ core.IntegrationContext, _ int64, installationID string) (*gh.Client, error) {
		if installationID == "22" {
			return nil, assert.AnError
		}
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
	assert.Equal(t, []common.Repository{{ID: 101, Name: "acme/api"}}, metadata.Repositories)
}

func TestSyncHostedAppRetriesDiscoveryWithoutOpeningInstallPage(t *testing.T) {
	enableUnverifiedDevelopmentRepositories(t)
	setHostedAppEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)
	t.Cleanup(resetBindClientHooks)

	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	listAppInstallations = func(context.Context, *gh.Client) ([]common.PendingInstallation, error) {
		return []common.PendingInstallation{{ID: "11", AccountLogin: "acme"}}, nil
	}
	newInstallationClient = func(core.IntegrationContext, int64, string) (*gh.Client, error) {
		return nil, assert.AnError
	}
	integration := &contexts.IntegrationContext{State: "pending"}
	syncCtx := core.SyncContext{
		Logger:         logrus.NewEntry(logrus.New()),
		OrganizationID: "org-1",
		ActorUserID:    "user-1",
		BaseURL:        "https://app.example",
		Configuration:  Configuration{SetupReturnPath: "/onboarding?step=vcs"},
		Integration:    integration,
	}

	err := (&GitHub{}).Sync(syncCtx)
	require.ErrorContains(t, err, "failed to discover GitHub App installations")
	assert.Nil(t, integration.BrowserAction)
	metadata := integration.Metadata.(common.Metadata)
	assert.Empty(t, metadata.PendingInstallations)
	assert.Empty(t, metadata.InstallationsRefreshedAt)

	newInstallationClient = func(core.IntegrationContext, int64, string) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	listInstallationRepos = func(context.Context, *gh.Client) ([]common.Repository, error) {
		return []common.Repository{{ID: 101, Name: "api"}}, nil
	}
	listAppInstallationRequests = func(context.Context, *gh.Client, string) ([]common.InstallRequest, error) {
		return nil, nil
	}

	require.NoError(t, (&GitHub{}).Sync(syncCtx))
	assert.Nil(t, integration.BrowserAction)
	metadata = integration.Metadata.(common.Metadata)
	require.Len(t, metadata.PendingInstallations, 1)
	assert.Equal(t, "acme/api", metadata.PendingInstallations[0].Repositories[0].Name)
	assert.NotEmpty(t, metadata.InstallationsRefreshedAt)
}

func TestSyncHostedAppRecordsRequestBaselineBeforeOpeningGitHub(t *testing.T) {
	enableUnverifiedDevelopmentRepositories(t)
	setHostedAppEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)
	t.Cleanup(resetBindClientHooks)

	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	listAppInstallations = func(context.Context, *gh.Client) ([]common.PendingInstallation, error) {
		return nil, nil
	}
	listAppInstallationRequests = func(context.Context, *gh.Client, string) ([]common.InstallRequest, error) {
		return []common.InstallRequest{{ID: "existing-request"}}, nil
	}
	integration := &contexts.IntegrationContext{State: "pending"}

	err := (&GitHub{}).Sync(core.SyncContext{
		Logger:         logrus.NewEntry(logrus.New()),
		OrganizationID: "org-1",
		ActorUserID:    "user-1",
		BaseURL:        "https://app.example",
		Integration:    integration,
	})

	require.NoError(t, err)
	metadata := integration.Metadata.(common.Metadata)
	assert.Equal(t, []string{"existing-request"}, metadata.ObservedInstallRequestIDs)
	assert.True(t, metadata.InstallRequestBaselineCaptured)
	require.NotNil(t, integration.BrowserAction)
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
	assert.False(t, requiresHostedInstallationDiscovery(common.Metadata{
		InstallationID:           "11",
		InstallRequests:          []common.InstallRequest{{ID: "1"}},
		InstallationsRefreshedAt: now.Format(time.RFC3339Nano),
	}, now))
	assert.True(t, requiresHostedInstallationDiscovery(common.Metadata{
		InstallationID:           "11",
		InstallRequests:          []common.InstallRequest{{ID: "1"}},
		InstallationsRefreshedAt: now.Add(-hostedInstallationDiscoveryInterval).Format(time.RFC3339Nano),
	}, now))
	assert.True(t, requiresHostedInstallationDiscovery(common.Metadata{
		InstallationID:               "11",
		InstallRequestDiscoveryUntil: now.Add(time.Minute).Format(time.RFC3339Nano),
		InstallationsRefreshedAt:     now.Add(-hostedInstallationDiscoveryInterval).Format(time.RFC3339Nano),
	}, now))
	assert.False(t, requiresHostedInstallationDiscovery(common.Metadata{
		InstallationID:               "11",
		InstallRequestDiscoveryUntil: now.Add(-time.Second).Format(time.RFC3339Nano),
	}, now))
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
	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) { return gh.NewClient(nil), nil }
	integration := &contexts.IntegrationContext{
		State: "pending",
		Metadata: common.Metadata{
			State:                "csrf",
			HostedApp:            true,
			StartedByGitHubLogin: "member",
			PendingInstallations: []common.PendingInstallation{
				{
					ID:           "11",
					AccountLogin: "acme",
					Repositories: []common.Repository{{ID: 101, Name: "acme/api"}},
				},
			},
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

func TestSyncHostedAppKeepsInstallRequestUntilRepositoryAccessIsVerified(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	setHostedAppEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)
	t.Cleanup(resetBindClientHooks)

	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	listAppInstallationRequests = func(context.Context, *gh.Client, string) ([]common.InstallRequest, error) {
		return nil, nil
	}
	request := common.InstallRequest{
		ID:             "1",
		AccountLogin:   "acme",
		RequesterLogin: "member",
		CreatedAt:      time.Now().UTC().Format(time.RFC3339Nano),
	}
	integration := &contexts.IntegrationContext{
		State: "pending",
		Metadata: common.Metadata{
			State:                "csrf",
			HostedApp:            true,
			StartedByGitHubLogin: "member",
			PendingInstallations: []common.PendingInstallation{
				{ID: "11", AccountLogin: "acme", AccountType: "Organization"},
			},
			InstallRequests: []common.InstallRequest{request},
			GitHubApp:       common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
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
	require.Len(t, metadata.PendingInstallations, 1)
	assert.Empty(t, metadata.PendingInstallations[0].Repositories)
	assert.Equal(t, []common.InstallRequest{request}, metadata.InstallRequests)
	assert.True(t, metadata.InstallRequested)
	assert.Equal(t, "acme", metadata.InstallRequestedAccount)
}

func TestSyncHostedAppClearsClosedInstallRequestAfterGracePeriod(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	setHostedAppEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)
	t.Cleanup(resetBindClientHooks)

	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	listAppInstallationRequests = func(context.Context, *gh.Client, string) ([]common.InstallRequest, error) {
		return nil, nil
	}
	integration := &contexts.IntegrationContext{
		State: "pending",
		Metadata: common.Metadata{
			State:                "csrf",
			HostedApp:            true,
			StartedByGitHubLogin: "member",
			InstallRequests: []common.InstallRequest{{
				ID:             "1",
				AccountLogin:   "acme",
				RequesterLogin: "member",
				CreatedAt: time.Now().UTC().
					Add(-installRequestResolutionGracePeriod - time.Second).
					Format(time.RFC3339Nano),
			}},
			GitHubApp: common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
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
	assert.Empty(t, metadata.InstallRequests)
	assert.False(t, metadata.InstallRequested)
	assert.True(t, installRequestFollowUpDiscoveryActive(metadata, time.Now().UTC()))
}

func TestSyncHostedAppDiscoversLateApprovalAfterWaitingClears(t *testing.T) {
	enableUnverifiedDevelopmentRepositories(t)
	setHostedAppEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)
	t.Cleanup(resetBindClientHooks)

	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	listAppInstallationRequests = func(context.Context, *gh.Client, string) ([]common.InstallRequest, error) {
		return nil, nil
	}
	installations := []common.PendingInstallation{{ID: "11", AccountLogin: "existing", AccountType: "Organization"}}
	listAppInstallations = func(context.Context, *gh.Client) ([]common.PendingInstallation, error) {
		return slices.Clone(installations), nil
	}
	newInstallationClient = func(core.IntegrationContext, int64, string) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	listInstallationRepos = func(context.Context, *gh.Client) ([]common.Repository, error) {
		return []common.Repository{{ID: 101, Name: "api"}}, nil
	}
	integration := &contexts.IntegrationContext{
		State: "ready",
		Metadata: common.Metadata{
			State:                    "csrf",
			HostedApp:                true,
			InstallationID:           "11",
			StartedByGitHubLogin:     "development",
			InstallationsRefreshedAt: time.Now().UTC().Format(time.RFC3339Nano),
			PendingInstallations: []common.PendingInstallation{
				{ID: "11", AccountLogin: "existing", Repositories: []common.Repository{{ID: 101, Name: "existing/api"}}},
			},
			InstallRequests: []common.InstallRequest{{
				ID:             "1",
				AccountLogin:   "acme",
				RequesterLogin: "member",
				CreatedAt: time.Now().UTC().
					Add(-installRequestResolutionGracePeriod - time.Second).
					Format(time.RFC3339Nano),
			}},
			GitHubApp: common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
		},
	}
	ctx := core.SyncContext{
		Logger:         logrus.NewEntry(logrus.New()),
		OrganizationID: "11111111-1111-1111-1111-111111111111",
		BaseURL:        "https://app.example",
		Integration:    integration,
	}

	require.NoError(t, (&GitHub{}).Sync(ctx))
	metadata := integration.Metadata.(common.Metadata)
	assert.Empty(t, metadata.InstallRequests)
	assert.True(t, installRequestFollowUpDiscoveryActive(metadata, time.Now().UTC()))

	installations = append(installations, common.PendingInstallation{
		ID: "22", AccountLogin: "acme", AccountType: "Organization",
	})
	metadata.InstallationsRefreshedAt = time.Now().UTC().Add(-hostedInstallationDiscoveryInterval).Format(time.RFC3339Nano)
	integration.Metadata = metadata

	require.NoError(t, (&GitHub{}).Sync(ctx))
	metadata = integration.Metadata.(common.Metadata)
	assert.True(t, slices.ContainsFunc(metadata.PendingInstallations, func(installation common.PendingInstallation) bool {
		return installation.AccountLogin == "acme" && len(installation.Repositories) > 0
	}))
}

func TestSyncHostedAppDiscoversApprovedRequestOnBoundConnection(t *testing.T) {
	enableUnverifiedDevelopmentRepositories(t)
	setHostedAppEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)
	t.Cleanup(resetBindClientHooks)

	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	listAppInstallations = func(context.Context, *gh.Client) ([]common.PendingInstallation, error) {
		return []common.PendingInstallation{
			{ID: "11", AccountLogin: "existing", AccountType: "Organization"},
			{ID: "22", AccountLogin: "acme", AccountType: "Organization"},
		}, nil
	}
	newInstallationClient = func(core.IntegrationContext, int64, string) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	listInstallationRepos = func(context.Context, *gh.Client) ([]common.Repository, error) {
		return []common.Repository{{ID: 101, Name: "api"}}, nil
	}
	listAppInstallationRequests = func(context.Context, *gh.Client, string) ([]common.InstallRequest, error) {
		return nil, nil
	}
	integration := &contexts.IntegrationContext{
		State: "ready",
		Metadata: common.Metadata{
			State:                    "csrf",
			HostedApp:                true,
			InstallationID:           "11",
			StartedByGitHubLogin:     "development",
			InstallationsRefreshedAt: time.Now().UTC().Add(-hostedInstallationDiscoveryInterval).Format(time.RFC3339Nano),
			PendingInstallations: []common.PendingInstallation{
				{ID: "11", AccountLogin: "existing", Repositories: []common.Repository{{ID: 101, Name: "existing/api"}}},
			},
			InstallRequests: []common.InstallRequest{{
				ID:             "1",
				AccountLogin:   "acme",
				RequesterLogin: "member",
				CreatedAt:      time.Now().UTC().Format(time.RFC3339Nano),
			}},
			GitHubApp: common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
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
	assert.Empty(t, metadata.InstallRequests)
	assert.False(t, metadata.InstallRequested)
	assert.True(t, slices.ContainsFunc(metadata.PendingInstallations, func(installation common.PendingInstallation) bool {
		return installation.AccountLogin == "acme" && len(installation.Repositories) > 0
	}))
}

func TestSyncHostedAppDiscoversApprovedRequestWithFreshInstallationCache(t *testing.T) {
	enableUnverifiedDevelopmentRepositories(t)
	setHostedAppEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)
	t.Cleanup(resetBindClientHooks)

	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	listAppInstallations = func(context.Context, *gh.Client) ([]common.PendingInstallation, error) {
		return []common.PendingInstallation{
			{ID: "11", AccountLogin: "existing", AccountType: "Organization"},
			{ID: "22", AccountLogin: "approved", AccountType: "Organization"},
		}, nil
	}
	newInstallationClient = func(core.IntegrationContext, int64, string) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	listInstallationRepos = func(context.Context, *gh.Client) ([]common.Repository, error) {
		return []common.Repository{{ID: 101, Name: "api"}}, nil
	}
	listAppInstallationRequests = func(context.Context, *gh.Client, string) ([]common.InstallRequest, error) {
		return nil, nil
	}
	integration := &contexts.IntegrationContext{
		State: "ready",
		Metadata: common.Metadata{
			State:                    "csrf",
			HostedApp:                true,
			InstallationID:           "11",
			StartedByGitHubLogin:     "development",
			InstallationsRefreshedAt: time.Now().UTC().Format(time.RFC3339Nano),
			PendingInstallations: []common.PendingInstallation{
				{ID: "11", AccountLogin: "existing", Repositories: []common.Repository{{ID: 101, Name: "existing/api"}}},
			},
			InstallRequests: []common.InstallRequest{{
				ID:             "1",
				AccountLogin:   "approved",
				RequesterLogin: "member",
				CreatedAt:      time.Now().UTC().Format(time.RFC3339Nano),
			}},
			GitHubApp: common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
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
	assert.Empty(t, metadata.InstallRequests)
	assert.True(t, slices.ContainsFunc(metadata.PendingInstallations, func(installation common.PendingInstallation) bool {
		return installation.AccountLogin == "approved" && len(installation.Repositories) > 0
	}))
}

func TestSyncHostedAppKeepsOpenInstallRequestAfterRepositoryAccessIsVerified(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	setHostedAppEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)
	t.Cleanup(resetBindClientHooks)

	request := common.InstallRequest{ID: "1", AccountLogin: "acme", RequesterLogin: "member"}
	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	listAppInstallationRequests = func(context.Context, *gh.Client, string) ([]common.InstallRequest, error) {
		return []common.InstallRequest{request}, nil
	}
	integration := &contexts.IntegrationContext{
		State: "pending",
		Metadata: common.Metadata{
			State:                "csrf",
			HostedApp:            true,
			StartedByGitHubLogin: "member",
			PendingInstallations: []common.PendingInstallation{
				{
					ID:           "11",
					AccountLogin: "acme",
					Repositories: []common.Repository{{ID: 101, Name: "acme/api"}},
				},
			},
			InstallRequests: []common.InstallRequest{request},
			GitHubApp:       common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
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
	assert.Equal(t, []common.InstallRequest{request}, metadata.InstallRequests)
	assert.True(t, metadata.InstallRequested)
	assert.Equal(t, "acme", metadata.InstallRequestedAccount)
}

func TestSyncHostedAppClearsUnknownInstallRequestAfterNewInstallationIsVerified(t *testing.T) {
	enableUnverifiedDevelopmentRepositories(t)
	setHostedAppEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)
	t.Cleanup(resetBindClientHooks)

	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	installations := []common.PendingInstallation{{ID: "11", AccountLogin: "existing", AccountType: "User"}}
	listAppInstallations = func(context.Context, *gh.Client) ([]common.PendingInstallation, error) {
		return slices.Clone(installations), nil
	}
	newInstallationClient = func(core.IntegrationContext, int64, string) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	listInstallationRepos = func(context.Context, *gh.Client) ([]common.Repository, error) {
		return []common.Repository{{ID: 101, Name: "api"}}, nil
	}
	createdAt := time.Now().UTC()
	openRequests := []common.InstallRequest{
		{
			ID:             "existing-request",
			AccountLogin:   "other",
			RequesterLogin: "other-member",
			CreatedAt:      createdAt.Format(time.RFC3339Nano),
		},
		{
			ID:             "1",
			AccountLogin:   "approved",
			RequesterLogin: "member",
			CreatedAt:      createdAt.Format(time.RFC3339Nano),
		},
	}
	listAppInstallationRequests = func(context.Context, *gh.Client, string) ([]common.InstallRequest, error) {
		return slices.Clone(openRequests), nil
	}
	integration := &contexts.IntegrationContext{
		State: "pending",
		Metadata: common.Metadata{
			State:                "csrf",
			HostedApp:            true,
			StartedByGitHubLogin: "development",
			PendingInstallations: []common.PendingInstallation{
				{ID: "11", AccountLogin: "existing", Repositories: []common.Repository{{ID: 101, Name: "existing/api"}}},
			},
			InstallRequests: []common.InstallRequest{{
				RequesterLogin:     "development",
				CreatedAt:          createdAt.Format(time.RFC3339Nano),
				ExistingRequestIDs: []string{"existing-request"},
				BaselineCaptured:   true,
			}},
			GitHubApp: common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
		},
	}

	ctx := core.SyncContext{
		Logger:         logrus.NewEntry(logrus.New()),
		OrganizationID: "11111111-1111-1111-1111-111111111111",
		BaseURL:        "https://app.example",
		Integration:    integration,
	}

	require.NoError(t, (&GitHub{}).Sync(ctx))
	metadata := integration.Metadata.(common.Metadata)
	require.Len(t, metadata.InstallRequests, 1)
	assert.Equal(t, "approved", metadata.InstallRequests[0].AccountLogin)

	installations = append(installations, common.PendingInstallation{
		ID: "22", AccountLogin: "approved", AccountType: "Organization",
	})
	openRequests = nil
	metadata.InstallationsRefreshedAt = time.Now().UTC().Add(-hostedInstallationDiscoveryInterval).Format(time.RFC3339Nano)
	integration.Metadata = metadata
	require.NoError(t, (&GitHub{}).Sync(ctx))
	metadata = integration.Metadata.(common.Metadata)
	assert.Empty(t, metadata.InstallRequests)
	assert.False(t, metadata.InstallRequested)
	assert.True(t, slices.ContainsFunc(metadata.PendingInstallations, func(installation common.PendingInstallation) bool {
		return installation.ID == "22" && len(installation.Repositories) > 0
	}))
}

func TestTrackedOpenInstallRequestsDoesNotGuessBetweenConcurrentRequests(t *testing.T) {
	createdAt := time.Now().UTC().Format(time.RFC3339Nano)
	tracked := []common.InstallRequest{{CreatedAt: createdAt, BaselineCaptured: true}}
	open := []common.InstallRequest{
		{ID: "1", AccountLogin: "acme", CreatedAt: createdAt},
		{ID: "2", AccountLogin: "octo", CreatedAt: createdAt},
	}

	assert.Empty(t, trackedOpenInstallRequests(tracked, open))
}

func TestTrackedOpenInstallRequestsIgnoresRequestObservedBeforeRedirect(t *testing.T) {
	createdAt := time.Now().UTC().Format(time.RFC3339Nano)
	tracked := []common.InstallRequest{{
		CreatedAt:          createdAt,
		ExistingRequestIDs: []string{"1"},
		BaselineCaptured:   true,
	}}
	open := []common.InstallRequest{{ID: "1", AccountLogin: "other", CreatedAt: createdAt}}

	assert.Empty(t, trackedOpenInstallRequests(tracked, open))
}

func TestReconcileInstallRequestsRecordsDevelopmentBaseline(t *testing.T) {
	enableUnverifiedDevelopmentRepositories(t)
	t.Cleanup(resetBindClientHooks)

	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	listAppInstallationRequests = func(context.Context, *gh.Client, string) ([]common.InstallRequest, error) {
		return []common.InstallRequest{{ID: "1"}, {ID: "2"}}, nil
	}
	metadata := common.Metadata{StartedByGitHubLogin: "development"}

	_, err := (&GitHub{}).reconcileInstallRequests(
		core.SyncContext{Context: context.Background(), Integration: &contexts.IntegrationContext{}},
		common.HostedApp{ID: 99},
		&metadata,
	)

	require.NoError(t, err)
	assert.Equal(t, []string{"1", "2"}, metadata.ObservedInstallRequestIDs)
	assert.True(t, metadata.InstallRequestBaselineCaptured)
}

func TestSyncHostedAppKeepsDevelopmentInstallRequestWithUnverifiedRepositories(t *testing.T) {
	enableUnverifiedDevelopmentRepositories(t)
	setHostedAppEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)
	t.Cleanup(resetBindClientHooks)

	request := common.InstallRequest{ID: "1", AccountLogin: "acme", RequesterLogin: "member"}
	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	listAppInstallationRequests = func(_ context.Context, _ *gh.Client, requester string) ([]common.InstallRequest, error) {
		assert.Empty(t, requester)
		return []common.InstallRequest{
			request,
			{ID: "2", AccountLogin: "other", RequesterLogin: "another-member"},
		}, nil
	}
	integration := &contexts.IntegrationContext{
		State: "pending",
		Metadata: common.Metadata{
			State:                    "csrf",
			HostedApp:                true,
			StartedByGitHubLogin:     "development",
			InstallationsRefreshedAt: time.Now().UTC().Format(time.RFC3339Nano),
			PendingInstallations: []common.PendingInstallation{
				{
					ID:           "11",
					AccountLogin: "acme",
					Repositories: []common.Repository{{ID: 101, Name: "acme/api"}},
				},
			},
			InstallRequests: []common.InstallRequest{{AccountLogin: "acme"}},
			GitHubApp:       common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
		},
	}

	ctx := core.SyncContext{
		Logger:         logrus.NewEntry(logrus.New()),
		OrganizationID: "11111111-1111-1111-1111-111111111111",
		BaseURL:        "https://app.example",
		Integration:    integration,
	}

	require.NoError(t, (&GitHub{}).Sync(ctx))
	require.NoError(t, (&GitHub{}).Sync(ctx))
	metadata := integration.Metadata.(common.Metadata)
	assert.Equal(t, []common.InstallRequest{request}, metadata.InstallRequests)
	assert.True(t, metadata.InstallRequested)
	assert.Equal(t, "acme", metadata.InstallRequestedAccount)
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

func TestListAppInstallationRequestsWithoutRequesterReturnsAll(t *testing.T) {
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
	requests, err := listAppInstallationRequestsFromGitHub(context.Background(), client, "")

	require.NoError(t, err)
	assert.Len(t, requests, 2)
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

func stubEmptyHostedDiscovery(t *testing.T) {
	t.Helper()
	t.Cleanup(resetBindClientHooks)
	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	listAppInstallations = func(context.Context, *gh.Client) ([]common.PendingInstallation, error) {
		return nil, nil
	}
}

func enableUnverifiedDevelopmentRepositories(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", "development")
	t.Setenv(allowUnverifiedDevelopmentRepositoriesEnv, "yes")
}

package github

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
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
		BrowserAction: &core.BrowserAction{URL: "https://github.com/apps/superplane/installations/new?state=csrf"},
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
	assert.Equal(t, "999", metadata.SetupInstallationID)
	assert.NotEmpty(t, metadata.SetupInstallationReceivedAt)
	assert.Empty(t, metadata.InstallationsRefreshedAt)
	assert.Nil(t, integration.BrowserAction)
	assert.Equal(
		t,
		"https://app.example/org-1/settings/integrations/11111111-1111-1111-1111-111111111111?githubSetup=complete&githubIntegrationId=11111111-1111-1111-1111-111111111111",
		rec.Header().Get("Location"),
	)
}

func TestSyncHostedAppVerifiesOnlyCallbackInstallation(t *testing.T) {
	enableUnverifiedDevelopmentRepositories(t)
	setHostedAppEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)
	t.Cleanup(resetBindClientHooks)

	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	getAppInstallation = func(_ context.Context, _ *gh.Client, id int64) (*gh.Installation, error) {
		assert.Equal(t, int64(22), id)
		return &gh.Installation{
			ID:         gh.Ptr(id),
			TargetType: gh.Ptr("Organization"),
			Account:    &gh.User{Login: gh.Ptr("new-org"), Type: gh.Ptr("Organization")},
		}, nil
	}
	listAppInstallations = func(context.Context, *gh.Client) ([]common.PendingInstallation, error) {
		t.Fatal("callback verification must not list every App installation")
		return nil, nil
	}
	newInstallationClient = func(core.IntegrationContext, int64, string) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	listInstallationRepos = func(context.Context, *gh.Client) ([]common.Repository, error) {
		return []common.Repository{{ID: 202, Name: "api"}}, nil
	}
	listAppInstallationRequests = func(context.Context, *gh.Client, string) ([]common.InstallRequest, error) {
		return nil, nil
	}
	integration := &contexts.IntegrationContext{
		State: "pending",
		Metadata: common.Metadata{
			State:               "csrf",
			HostedApp:           true,
			SetupInstallationID: "22",
			StartedByUserID:     "user-1",
			PendingInstallations: []common.PendingInstallation{{
				ID:           "11",
				AccountLogin: "existing",
				Repositories: []common.Repository{{ID: 101, Name: "existing/web"}},
			}},
			GitHubApp: common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
		},
	}

	err := (&GitHub{}).Sync(core.SyncContext{
		Context:        context.Background(),
		Logger:         logrus.NewEntry(logrus.New()),
		OrganizationID: "org-1",
		ActorUserID:    "user-1",
		BaseURL:        "https://app.example",
		Integration:    integration,
	})

	require.NoError(t, err)
	metadata := integration.Metadata.(common.Metadata)
	assert.Empty(t, metadata.SetupInstallationID)
	require.Len(t, metadata.PendingInstallations, 2)
	assert.Equal(t, "new-org", metadata.PendingInstallations[0].AccountLogin)
	assert.Equal(t, "new-org/api", metadata.PendingInstallations[0].Repositories[0].Name)
	assert.Equal(t, "existing", metadata.PendingInstallations[1].AccountLogin)
}

func TestSyncHostedAppRetriesCallbackUntilInstallationIsVisible(t *testing.T) {
	enableUnverifiedDevelopmentRepositories(t)
	setHostedAppEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)
	t.Cleanup(resetBindClientHooks)

	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	attempts := 0
	getAppInstallation = func(_ context.Context, _ *gh.Client, id int64) (*gh.Installation, error) {
		attempts++
		if attempts == 1 {
			return nil, githubNotFoundError()
		}
		return &gh.Installation{
			ID:         gh.Ptr(id),
			TargetType: gh.Ptr("Organization"),
			Account:    &gh.User{Login: gh.Ptr("new-org"), Type: gh.Ptr("Organization")},
		}, nil
	}
	newInstallationClient = func(core.IntegrationContext, int64, string) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	repositoryAttempts := 0
	listInstallationRepos = func(context.Context, *gh.Client) ([]common.Repository, error) {
		repositoryAttempts++
		if repositoryAttempts == 1 {
			return nil, nil
		}
		return []common.Repository{{ID: 202, Name: "api"}}, nil
	}
	listAppInstallationRequests = func(context.Context, *gh.Client, string) ([]common.InstallRequest, error) {
		return nil, nil
	}
	integration := &contexts.IntegrationContext{
		State: "pending",
		Metadata: common.Metadata{
			State:                 "csrf",
			HostedApp:             true,
			SetupInstallationID:   "22",
			StartedByUserID:       "user-1",
			InstallationDiscovery: &common.InstallationDiscovery{Complete: true},
			GitHubApp:             common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
		},
	}
	ctx := core.SyncContext{
		Context:        context.Background(),
		Logger:         logrus.NewEntry(logrus.New()),
		OrganizationID: "org-1",
		ActorUserID:    "user-1",
		BaseURL:        "https://app.example",
		Integration:    integration,
	}

	require.NoError(t, (&GitHub{}).Sync(ctx))
	metadata := integration.Metadata.(common.Metadata)
	assert.Equal(t, "22", metadata.SetupInstallationID)
	assert.Nil(t, integration.BrowserAction)
	assert.Empty(t, metadata.PendingInstallations)

	require.NoError(t, (&GitHub{}).Sync(ctx))
	metadata = integration.Metadata.(common.Metadata)
	assert.Equal(t, "22", metadata.SetupInstallationID)
	assert.Nil(t, integration.BrowserAction)
	assert.Empty(t, metadata.PendingInstallations)

	require.NoError(t, (&GitHub{}).Sync(ctx))
	metadata = integration.Metadata.(common.Metadata)
	assert.Empty(t, metadata.SetupInstallationID)
	assert.Nil(t, integration.BrowserAction)
	require.Len(t, metadata.PendingInstallations, 1)
	assert.Equal(t, "new-org", metadata.PendingInstallations[0].AccountLogin)
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
	getAppInstallation = func(context.Context, *gh.Client, int64) (*gh.Installation, error) {
		return &gh.Installation{
			ID:         gh.Ptr(int64(11)),
			TargetType: gh.Ptr("Organization"),
			Account:    &gh.User{Login: gh.Ptr("acme"), Type: gh.Ptr("Organization")},
		}, nil
	}
	listAppInstallations = func(context.Context, *gh.Client) ([]common.PendingInstallation, error) {
		t.Fatal("binding must not list every App installation")
		return nil, nil
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

func TestHostedBindDoesNotCheckUnrelatedInstallation(t *testing.T) {
	enableUnverifiedDevelopmentRepositories(t)
	setHostedAppEnv(t)
	t.Cleanup(resetBindClientHooks)

	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	getAppInstallation = func(context.Context, *gh.Client, int64) (*gh.Installation, error) {
		return &gh.Installation{
			ID:         gh.Ptr(int64(11)),
			TargetType: gh.Ptr("Organization"),
			Account:    &gh.User{Login: gh.Ptr("acme"), Type: gh.Ptr("Organization")},
		}, nil
	}
	listAppInstallations = func(context.Context, *gh.Client) ([]common.PendingInstallation, error) {
		t.Fatal("binding must not list every App installation")
		return nil, nil
	}
	newInstallationClient = func(_ core.IntegrationContext, _ int64, installationID string) (*gh.Client, error) {
		if installationID == "22" {
			t.Fatal("binding checked an unrelated installation")
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

func TestSyncHostedAppDiscoversPersonalInstallationBeforeGitHubRedirect(t *testing.T) {
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
	findAppUserInstallation = func(_ context.Context, _ *gh.Client, login string) (*gh.Installation, error) {
		assert.Equal(t, "development", login)
		return &gh.Installation{
			ID:      gh.Ptr(int64(11)),
			Account: &gh.User{Login: gh.Ptr("my-account"), Type: gh.Ptr("User")},
		}, nil
	}
	newInstallationClient = func(core.IntegrationContext, int64, string) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	listInstallationRepos = func(context.Context, *gh.Client) ([]common.Repository, error) {
		return []common.Repository{{ID: 101, Name: "api"}}, nil
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

	require.NoError(t, (&GitHub{}).Sync(syncCtx))
	assert.Nil(t, integration.BrowserAction)
	metadata := integration.Metadata.(common.Metadata)
	require.Len(t, metadata.PendingInstallations, 1)
	assert.Equal(t, "my-account", metadata.PendingInstallations[0].AccountLogin)
	require.NotNil(t, metadata.InstallationDiscovery)
	assert.True(t, metadata.InstallationDiscovery.Active)
	assert.True(t, metadata.InstallationDiscovery.PersonalAccountChecked)
}

func TestSyncHostedAppMergesProgressiveInstallationPages(t *testing.T) {
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
	pages := []int{}
	listRecentAppInstallations = func(
		_ context.Context,
		_ *gh.Client,
		since time.Time,
		page int,
		perPage int,
	) ([]common.PendingInstallation, int, error) {
		assert.True(t, since.IsZero())
		assert.Equal(t, hostedInitialDiscoveryPageSize, perPage)
		pages = append(pages, page)
		if page == 1 {
			return []common.PendingInstallation{{ID: "11", AccountLogin: "my-account", AccountType: "User"}}, 2, nil
		}
		return []common.PendingInstallation{{ID: "22", AccountLogin: "my-org", AccountType: "Organization"}}, 0, nil
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
			State:                "csrf",
			HostedApp:            true,
			StartedByUserID:      "user-1",
			StartedByGitHubLogin: "development",
			SetupReturnPath:      "/onboarding?step=vcs",
			InstallationDiscovery: &common.InstallationDiscovery{
				Active:                 true,
				PersonalAccountChecked: true,
				NextPage:               1,
			},
			GitHubApp: common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
		},
	}
	ctx := core.SyncContext{
		Context:        context.Background(),
		Logger:         logrus.NewEntry(logrus.New()),
		OrganizationID: "org-1",
		BaseURL:        "https://app.example",
		Integration:    integration,
	}

	require.NoError(t, (&GitHub{}).Sync(ctx))
	metadata := integration.Metadata.(common.Metadata)
	require.Len(t, metadata.PendingInstallations, 1)
	assert.True(t, metadata.InstallationDiscovery.Active)
	assert.Nil(t, integration.BrowserAction)

	integration.Metadata = metadata
	require.NoError(t, (&GitHub{}).Sync(ctx))
	metadata = integration.Metadata.(common.Metadata)
	assert.Equal(t, []int{1, 2}, pages)
	assert.Len(t, metadata.PendingInstallations, 2)
	assert.False(t, metadata.InstallationDiscovery.Active)
	assert.True(t, metadata.InstallationDiscovery.Complete)
	assert.Nil(t, integration.BrowserAction)
}

func TestSyncHostedAppOpensInstallAfterCompleteEmptyDiscovery(t *testing.T) {
	enableUnverifiedDevelopmentRepositories(t)
	setHostedAppEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)
	t.Cleanup(resetBindClientHooks)

	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) { return gh.NewClient(nil), nil }
	listAppInstallationRequests = func(context.Context, *gh.Client, string) ([]common.InstallRequest, error) {
		return nil, nil
	}
	listRecentAppInstallations = func(context.Context, *gh.Client, time.Time, int, int) ([]common.PendingInstallation, int, error) {
		return nil, 0, nil
	}
	integration := &contexts.IntegrationContext{
		State: "pending",
		Metadata: common.Metadata{
			State:                "csrf",
			HostedApp:            true,
			StartedByUserID:      "user-1",
			StartedByGitHubLogin: "development",
			SetupReturnPath:      "/onboarding?step=vcs",
			InstallationDiscovery: &common.InstallationDiscovery{
				Active:                 true,
				PersonalAccountChecked: true,
				NextPage:               1,
			},
			GitHubApp: common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
		},
	}

	require.NoError(t, (&GitHub{}).Sync(core.SyncContext{
		Context: context.Background(), Logger: logrus.NewEntry(logrus.New()),
		BaseURL: "https://app.example", Integration: integration,
	}))
	require.NotNil(t, integration.BrowserAction)
	assert.Contains(t, integration.BrowserAction.URL, "github.com/apps/")
	metadata := integration.Metadata.(common.Metadata)
	assert.True(t, metadata.InstallationDiscovery.Complete)
}

func TestSyncHostedAppRetriesOnlyTransientInitialDiscoveryFailures(t *testing.T) {
	enableUnverifiedDevelopmentRepositories(t)
	setHostedAppEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)
	t.Cleanup(resetBindClientHooks)

	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) { return gh.NewClient(nil), nil }
	listAppInstallationRequests = func(context.Context, *gh.Client, string) ([]common.InstallRequest, error) {
		return nil, nil
	}
	listRecentAppInstallations = func(ctx context.Context, _ *gh.Client, _ time.Time, _, perPage int) ([]common.PendingInstallation, int, error) {
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		assert.LessOrEqual(t, time.Until(deadline), hostedInitialDiscoveryTimeout)
		assert.Equal(t, hostedInitialDiscoveryPageSize, perPage)
		return []common.PendingInstallation{
			{ID: "11", AccountLogin: "temporary"},
			{ID: "22", AccountLogin: "revoked"},
		}, 0, nil
	}
	temporaryAttempts := 0
	newInstallationClient = func(_ core.IntegrationContext, _ int64, installationID string) (*gh.Client, error) {
		if installationID == "11" {
			temporaryAttempts++
			if temporaryAttempts == 1 {
				return nil, context.DeadlineExceeded
			}
		}
		if installationID == "22" {
			return nil, errors.New("revoked")
		}
		return gh.NewClient(nil), nil
	}
	listInstallationRepos = func(context.Context, *gh.Client) ([]common.Repository, error) {
		return []common.Repository{{ID: 101, Name: "api"}}, nil
	}
	integration := &contexts.IntegrationContext{
		State: "pending",
		Metadata: common.Metadata{
			State:                "csrf",
			HostedApp:            true,
			StartedByUserID:      "user-1",
			StartedByGitHubLogin: "development",
			SetupReturnPath:      "/onboarding?step=vcs",
			InstallationDiscovery: &common.InstallationDiscovery{
				Active:                 true,
				PersonalAccountChecked: true,
				NextPage:               1,
			},
			GitHubApp: common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
		},
	}
	ctx := core.SyncContext{Context: context.Background(), Integration: integration}

	require.NoError(t, (&GitHub{}).Sync(ctx))
	metadata := integration.Metadata.(common.Metadata)
	require.Len(t, metadata.InstallationDiscovery.RetryCandidates, 1)
	assert.Equal(t, "11", metadata.InstallationDiscovery.RetryCandidates[0].ID)
	assert.True(t, metadata.InstallationDiscovery.Active)

	integration.Metadata = metadata
	require.NoError(t, (&GitHub{}).Sync(ctx))
	metadata = integration.Metadata.(common.Metadata)
	assert.True(t, metadata.InstallationDiscovery.Complete)
	require.Len(t, metadata.PendingInstallations, 1)
	assert.Equal(t, "temporary", metadata.PendingInstallations[0].AccountLogin)
	assert.Equal(t, 2, temporaryAttempts)
}

func TestSyncHostedAppChecksLaterPagesAfterOneBoundedRetry(t *testing.T) {
	enableUnverifiedDevelopmentRepositories(t)
	setHostedAppEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)
	t.Cleanup(resetBindClientHooks)

	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) { return gh.NewClient(nil), nil }
	listAppInstallationRequests = func(context.Context, *gh.Client, string) ([]common.InstallRequest, error) {
		return nil, nil
	}
	listRecentAppInstallations = func(_ context.Context, _ *gh.Client, _ time.Time, page, perPage int) ([]common.PendingInstallation, int, error) {
		assert.Equal(t, 2, page)
		assert.Equal(t, hostedInitialDiscoveryPageSize, perPage)
		return []common.PendingInstallation{{ID: "22", AccountLogin: "later-account"}}, 0, nil
	}
	retryAttempts := 0
	newInstallationClient = func(_ core.IntegrationContext, _ int64, installationID string) (*gh.Client, error) {
		if installationID == "11" {
			retryAttempts++
			return nil, context.DeadlineExceeded
		}
		return gh.NewClient(nil), nil
	}
	listInstallationRepos = func(context.Context, *gh.Client) ([]common.Repository, error) {
		return []common.Repository{{ID: 101, Name: "api"}}, nil
	}
	integration := &contexts.IntegrationContext{
		State: "pending",
		Metadata: common.Metadata{
			State:                "csrf",
			HostedApp:            true,
			StartedByUserID:      "user-1",
			StartedByGitHubLogin: "development",
			SetupReturnPath:      "/onboarding?step=vcs",
			InstallationDiscovery: &common.InstallationDiscovery{
				Active:                 true,
				PersonalAccountChecked: true,
				NextPage:               2,
				RetryCandidates: []common.PendingInstallation{{
					ID: "11", AccountLogin: "persistently-unavailable",
				}},
			},
			GitHubApp: common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
		},
	}
	ctx := core.SyncContext{Context: context.Background(), Integration: integration}

	require.NoError(t, (&GitHub{}).Sync(ctx))
	metadata := integration.Metadata.(common.Metadata)
	assert.Empty(t, metadata.PendingInstallations)
	assert.True(t, metadata.InstallationDiscovery.Active)
	assert.Empty(t, metadata.InstallationDiscovery.RetryCandidates)
	assert.Equal(t, 1, retryAttempts)

	integration.Metadata = metadata
	err := (&GitHub{}).Sync(ctx)
	require.EqualError(t, err, "GitHub account discovery is temporarily unavailable")
	metadata = integration.Metadata.(common.Metadata)
	require.Len(t, metadata.PendingInstallations, 1)
	assert.Equal(t, "later-account", metadata.PendingInstallations[0].AccountLogin)
	assert.True(t, metadata.InstallationDiscovery.Active)
	assert.False(t, metadata.InstallationDiscovery.Complete)
	assert.False(t, metadata.InstallationDiscovery.PersonalAccountChecked)
	assert.Equal(t, 1, metadata.InstallationDiscovery.NextPage)
	assert.Equal(t, 1, retryAttempts)
	assert.Nil(t, integration.BrowserAction)
}

func TestSyncHostedAppDoesNotCompleteAfterFinalTransientRetry(t *testing.T) {
	enableUnverifiedDevelopmentRepositories(t)
	setHostedAppEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)
	t.Cleanup(resetBindClientHooks)

	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) { return gh.NewClient(nil), nil }
	listAppInstallationRequests = func(context.Context, *gh.Client, string) ([]common.InstallRequest, error) {
		return nil, nil
	}
	newInstallationClient = func(core.IntegrationContext, int64, string) (*gh.Client, error) {
		return nil, context.DeadlineExceeded
	}
	integration := &contexts.IntegrationContext{
		State: "pending",
		Metadata: common.Metadata{
			State:                "csrf",
			HostedApp:            true,
			StartedByUserID:      "user-1",
			StartedByGitHubLogin: "development",
			SetupReturnPath:      "/onboarding?step=vcs",
			InstallationDiscovery: &common.InstallationDiscovery{
				Active:                 true,
				PersonalAccountChecked: true,
				RetryCandidates: []common.PendingInstallation{{
					ID: "11", AccountLogin: "temporarily-unavailable",
				}},
			},
			GitHubApp: common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
		},
	}

	err := (&GitHub{}).Sync(core.SyncContext{Context: context.Background(), Integration: integration})
	require.EqualError(t, err, "GitHub account discovery is temporarily unavailable")
	metadata := integration.Metadata.(common.Metadata)
	assert.True(t, metadata.InstallationDiscovery.Active)
	assert.False(t, metadata.InstallationDiscovery.Complete)
	assert.False(t, metadata.InstallationDiscovery.PersonalAccountChecked)
	assert.Equal(t, 1, metadata.InstallationDiscovery.NextPage)
	assert.Empty(t, metadata.InstallationDiscovery.RetryCandidates)
	assert.Nil(t, integration.BrowserAction)
}

func TestSyncHostedAppDiscardsCallbackInstallationWithoutRepositoriesAfterGracePeriod(t *testing.T) {
	enableUnverifiedDevelopmentRepositories(t)
	setHostedAppEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)
	t.Cleanup(resetBindClientHooks)

	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	getAppInstallation = func(context.Context, *gh.Client, int64) (*gh.Installation, error) {
		return &gh.Installation{
			ID:      gh.Ptr(int64(22)),
			Account: &gh.User{Login: gh.Ptr("no-access"), Type: gh.Ptr("Organization")},
		}, nil
	}
	newInstallationClient = func(core.IntegrationContext, int64, string) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	listInstallationRepos = func(context.Context, *gh.Client) ([]common.Repository, error) {
		return nil, nil
	}
	listAppInstallations = func(context.Context, *gh.Client) ([]common.PendingInstallation, error) {
		t.Fatal("an unusable callback must not fall back to full discovery")
		return nil, nil
	}
	listAppInstallationRequests = func(context.Context, *gh.Client, string) ([]common.InstallRequest, error) {
		return nil, nil
	}
	integration := &contexts.IntegrationContext{
		State: "pending",
		Metadata: common.Metadata{
			State:                       "csrf",
			HostedApp:                   true,
			StartedByUserID:             "user-1",
			SetupInstallationID:         "22",
			SetupInstallationReceivedAt: time.Now().UTC().Add(-setupInstallationVisibilityGracePeriod - time.Second).Format(time.RFC3339Nano),
			PendingInstallations: []common.PendingInstallation{{
				ID:           "11",
				AccountLogin: "existing",
				Repositories: []common.Repository{{ID: 101, Name: "existing/api"}},
			}},
			GitHubApp: common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
		},
	}

	require.NoError(t, (&GitHub{}).Sync(core.SyncContext{
		Context:        context.Background(),
		Logger:         logrus.NewEntry(logrus.New()),
		OrganizationID: "org-1",
		ActorUserID:    "user-1",
		BaseURL:        "https://app.example",
		Integration:    integration,
	}))

	assert.Nil(t, integration.BrowserAction)
	metadata := integration.Metadata.(common.Metadata)
	assert.Empty(t, metadata.SetupInstallationID)
	require.Len(t, metadata.PendingInstallations, 1)
	assert.Equal(t, "existing", metadata.PendingInstallations[0].AccountLogin)
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
	listRecentAppInstallations = func(context.Context, *gh.Client, time.Time, int, int) ([]common.PendingInstallation, int, error) {
		return nil, 0, nil
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
		t.Fatal("late approval must not list every App installation")
		return nil, nil
	}
	findAppOrganizationInstallation = func(_ context.Context, _ *gh.Client, account string) (*gh.Installation, error) {
		for _, installation := range installations {
			if installation.AccountLogin == account {
				id, err := strconv.ParseInt(installation.ID, 10, 64)
				require.NoError(t, err)
				return &gh.Installation{
					ID:      gh.Ptr(id),
					Account: &gh.User{Login: gh.Ptr(account), Type: gh.Ptr("Organization")},
				}, nil
			}
		}
		return nil, githubNotFoundError()
	}
	findAppUserInstallation = func(context.Context, *gh.Client, string) (*gh.Installation, error) {
		return nil, githubNotFoundError()
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
			State:                "csrf",
			HostedApp:            true,
			StartedByUserID:      "user-1",
			StartedByGitHubLogin: "development",
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
	assert.Empty(t, metadata.PendingInstallations)
	assert.True(t, installRequestFollowUpDiscoveryActive(metadata, time.Now().UTC()))
	assert.Equal(t, []string{"acme"}, metadata.InstallRequestDiscoveryAccounts)

	installations = append(installations, common.PendingInstallation{
		ID: "22", AccountLogin: "acme", AccountType: "Organization",
	})
	integration.Metadata = metadata

	require.NoError(t, (&GitHub{}).Sync(ctx))
	metadata = integration.Metadata.(common.Metadata)
	assert.True(t, slices.ContainsFunc(metadata.PendingInstallations, func(installation common.PendingInstallation) bool {
		return installation.AccountLogin == "acme" && len(installation.Repositories) > 0
	}))
	assert.Empty(t, metadata.InstallRequestDiscoveryAccounts)
	assert.Empty(t, metadata.InstallRequestDiscoveryUntil)
}

func TestSyncHostedAppDiscoversLegacyLateApprovalWithoutSavedAccount(t *testing.T) {
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
	listAppInstallations = func(context.Context, *gh.Client) ([]common.PendingInstallation, error) {
		t.Fatal("legacy recovery must not list every App installation")
		return nil, nil
	}
	discoveryPages := []int{}
	listRecentAppInstallations = func(
		_ context.Context,
		_ *gh.Client,
		since time.Time,
		page int,
		perPage int,
	) ([]common.PendingInstallation, int, error) {
		discoveryPages = append(discoveryPages, page)
		assert.False(t, since.IsZero())
		assert.Equal(t, hostedInstallRequestFallbackPageSize, perPage)
		if page == 2 {
			return nil, 0, nil
		}
		assert.Equal(t, 1, page)
		return []common.PendingInstallation{
			{ID: "11", AccountLogin: "unavailable", AccountType: "Organization"},
			{ID: "22", AccountLogin: "approved", AccountType: "Organization"},
			{ID: "33", AccountLogin: "revoked", AccountType: "Organization"},
		}, 2, nil
	}
	unavailableAttempts := 0
	revokedAttempts := 0
	newInstallationClient = func(_ core.IntegrationContext, _ int64, installationID string) (*gh.Client, error) {
		if installationID == "11" {
			unavailableAttempts++
			if unavailableAttempts < 3 {
				return nil, context.DeadlineExceeded
			}
		}
		if installationID == "33" {
			revokedAttempts++
			return nil, errors.New("revoked")
		}
		return gh.NewClient(nil), nil
	}
	listInstallationRepos = func(context.Context, *gh.Client) ([]common.Repository, error) {
		return []common.Repository{{ID: 101, Name: "api"}}, nil
	}
	integration := &contexts.IntegrationContext{
		State: "pending",
		Metadata: common.Metadata{
			State:                        "csrf",
			HostedApp:                    true,
			StartedByGitHubLogin:         "development",
			InstallRequestDiscoveryUntil: time.Now().UTC().Add(time.Minute).Format(time.RFC3339Nano),
			InstallationsRefreshedAt:     time.Now().UTC().Format(time.RFC3339Nano),
			GitHubApp:                    common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
		},
	}

	require.NoError(t, (&GitHub{}).Sync(core.SyncContext{
		Context:        context.Background(),
		Logger:         logrus.NewEntry(logrus.New()),
		OrganizationID: "11111111-1111-1111-1111-111111111111",
		BaseURL:        "https://app.example",
		Integration:    integration,
	}))

	metadata := integration.Metadata.(common.Metadata)
	assert.Equal(t, []int{1}, discoveryPages)
	assert.Equal(t, 1, unavailableAttempts)
	assert.Equal(t, 1, revokedAttempts)
	assert.NotEmpty(t, metadata.InstallationsRefreshedAt)
	assert.True(t, slices.ContainsFunc(metadata.PendingInstallations, func(installation common.PendingInstallation) bool {
		return installation.AccountLogin == "approved" && len(installation.Repositories) > 0
	}))

	metadata.InstallRequestDiscoveryUntil = time.Now().UTC().Add(-time.Second).Format(time.RFC3339Nano)
	integration.Metadata = metadata
	require.NoError(t, (&GitHub{}).Sync(core.SyncContext{
		Context:        context.Background(),
		Logger:         logrus.NewEntry(logrus.New()),
		OrganizationID: "11111111-1111-1111-1111-111111111111",
		BaseURL:        "https://app.example",
		Integration:    integration,
	}))
	assert.Equal(t, []int{1, 2}, discoveryPages)
	assert.Equal(t, 1, unavailableAttempts)
	assert.Equal(t, 1, revokedAttempts)

	require.NoError(t, (&GitHub{}).Sync(core.SyncContext{
		Context:        context.Background(),
		Logger:         logrus.NewEntry(logrus.New()),
		OrganizationID: "11111111-1111-1111-1111-111111111111",
		BaseURL:        "https://app.example",
		Integration:    integration,
	}))
	assert.Equal(t, []int{1, 2}, discoveryPages)
	assert.Equal(t, 2, unavailableAttempts)
	assert.Equal(t, 1, revokedAttempts)

	require.NoError(t, (&GitHub{}).Sync(core.SyncContext{
		Context:        context.Background(),
		Logger:         logrus.NewEntry(logrus.New()),
		OrganizationID: "11111111-1111-1111-1111-111111111111",
		BaseURL:        "https://app.example",
		Integration:    integration,
	}))
	require.NoError(t, (&GitHub{}).Sync(core.SyncContext{
		Context:        context.Background(),
		Logger:         logrus.NewEntry(logrus.New()),
		OrganizationID: "11111111-1111-1111-1111-111111111111",
		BaseURL:        "https://app.example",
		Integration:    integration,
	}))
	assert.Equal(t, []int{1, 2}, discoveryPages)
	assert.Equal(t, 3, unavailableAttempts)
	assert.Equal(t, 1, revokedAttempts)
}

func TestSyncHostedAppPreservesLegacyFallbackWithOverlappingRequest(t *testing.T) {
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
	listAppInstallations = func(context.Context, *gh.Client) ([]common.PendingInstallation, error) {
		t.Fatal("overlapping approval recovery must not list every App installation")
		return nil, nil
	}
	listRecentAppInstallations = func(
		context.Context,
		*gh.Client,
		time.Time,
		int,
		int,
	) ([]common.PendingInstallation, int, error) {
		return []common.PendingInstallation{{ID: "11", AccountLogin: "earlier", AccountType: "Organization"}}, 0, nil
	}
	findAppOrganizationInstallation = func(_ context.Context, _ *gh.Client, account string) (*gh.Installation, error) {
		assert.Equal(t, "later", account)
		return &gh.Installation{
			ID:      gh.Ptr(int64(22)),
			Account: &gh.User{Login: gh.Ptr(account), Type: gh.Ptr("Organization")},
		}, nil
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
			State:                        "csrf",
			HostedApp:                    true,
			StartedByGitHubLogin:         "development",
			InstallRequestDiscoveryUntil: time.Now().UTC().Add(time.Minute).Format(time.RFC3339Nano),
			InstallRequests: []common.InstallRequest{{
				ID:             "2",
				AccountLogin:   "later",
				RequesterLogin: "member",
				CreatedAt: time.Now().UTC().
					Add(-installRequestResolutionGracePeriod - time.Second).
					Format(time.RFC3339Nano),
			}},
			GitHubApp: common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
		},
	}

	require.NoError(t, (&GitHub{}).Sync(core.SyncContext{
		Context:        context.Background(),
		Logger:         logrus.NewEntry(logrus.New()),
		OrganizationID: "11111111-1111-1111-1111-111111111111",
		BaseURL:        "https://app.example",
		Integration:    integration,
	}))

	metadata := integration.Metadata.(common.Metadata)
	assert.True(t, slices.ContainsFunc(metadata.PendingInstallations, func(installation common.PendingInstallation) bool {
		return installation.AccountLogin == "earlier"
	}))
	assert.True(t, slices.ContainsFunc(metadata.PendingInstallations, func(installation common.PendingInstallation) bool {
		return installation.AccountLogin == "later"
	}))
}

func TestReconcileInstallRequestsPreservesOverlappingFollowUpAccounts(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Cleanup(resetBindClientHooks)

	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	listAppInstallationRequests = func(context.Context, *gh.Client, string) ([]common.InstallRequest, error) {
		return nil, nil
	}
	metadata := common.Metadata{
		StartedByGitHubLogin:            "member",
		InstallRequestDiscoveryUntil:    time.Now().UTC().Add(time.Minute).Format(time.RFC3339Nano),
		InstallRequestDiscoveryAccounts: []string{"earlier"},
		InstallRequests: []common.InstallRequest{{
			ID:             "2",
			AccountLogin:   "later",
			RequesterLogin: "member",
			CreatedAt: time.Now().UTC().
				Add(-installRequestResolutionGracePeriod - time.Second).
				Format(time.RFC3339Nano),
		}},
	}

	discovery, err := (&GitHub{}).reconcileInstallRequests(core.SyncContext{
		Context:     context.Background(),
		Integration: &contexts.IntegrationContext{},
	}, common.HostedApp{ID: 99}, &metadata)

	require.NoError(t, err)
	assert.Equal(t, []string{"earlier", "later"}, discovery.accounts)
	assert.Equal(t, discovery.accounts, metadata.InstallRequestDiscoveryAccounts)
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
	findAppOrganizationInstallation = func(context.Context, *gh.Client, string) (*gh.Installation, error) {
		return &gh.Installation{
			ID:      gh.Ptr(int64(22)),
			Account: &gh.User{Login: gh.Ptr("acme"), Type: gh.Ptr("Organization")},
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
	findAppOrganizationInstallation = func(_ context.Context, _ *gh.Client, account string) (*gh.Installation, error) {
		assert.Equal(t, "approved", account)
		return &gh.Installation{
			ID:         gh.Ptr(int64(22)),
			TargetType: gh.Ptr("Organization"),
			Account:    &gh.User{Login: gh.Ptr("approved"), Type: gh.Ptr("Organization")},
		}, nil
	}
	listAppInstallations = func(context.Context, *gh.Client) ([]common.PendingInstallation, error) {
		t.Fatal("approval reconciliation must not list every App installation")
		return nil, nil
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
	listRecentAppInstallations = func(
		context.Context,
		*gh.Client,
		time.Time,
		int,
		int,
	) ([]common.PendingInstallation, int, error) {
		return slices.Clone(installations), 0, nil
	}
	findAppOrganizationInstallation = func(_ context.Context, _ *gh.Client, account string) (*gh.Installation, error) {
		for _, installation := range installations {
			if installation.AccountLogin == account {
				id, err := strconv.ParseInt(installation.ID, 10, 64)
				require.NoError(t, err)
				return &gh.Installation{
					ID:      gh.Ptr(id),
					Account: &gh.User{Login: gh.Ptr(account), Type: gh.Ptr("Organization")},
				}, nil
			}
		}
		return nil, githubNotFoundError()
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

func TestListRecentAppInstallationsUsesBoundedPage(t *testing.T) {
	since := time.Now().UTC().Truncate(time.Second)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/app/installations", r.URL.Path)
		assert.Equal(t, "2", r.URL.Query().Get("page"))
		assert.Equal(t, "8", r.URL.Query().Get("per_page"))
		assert.Equal(t, since.Format(time.RFC3339), r.URL.Query().Get("since"))
		w.Header().Set("Link", `<https://api.github.com/app/installations?page=3>; rel="next"`)
		_, _ = w.Write([]byte(`[{
			"id": 22,
			"account": {"login": "approved", "type": "Organization"}
		}]`))
	}))
	defer server.Close()

	client := gh.NewClient(server.Client())
	client.BaseURL, _ = client.BaseURL.Parse(server.URL + "/")
	installations, nextPage, err := listRecentAppInstallationsFromGitHub(
		context.Background(),
		client,
		since,
		2,
		hostedInstallRequestFallbackPageSize,
	)

	require.NoError(t, err)
	assert.Equal(t, 3, nextPage)
	assert.Equal(t, []common.PendingInstallation{{
		ID:           "22",
		AccountLogin: "approved",
		AccountType:  "Organization",
	}}, installations)
}

func resetBindClientHooks() {
	newInstallationClient = newClientForAppInstallation
	newAppJWTClient = newClientForApp
	listInstallationRepos = listInstallationRepositories
	listAppInstallations = listAppInstallationsFromGitHub
	listRecentAppInstallations = listRecentAppInstallationsFromGitHub
	listAppInstallationRequests = listAppInstallationRequestsFromGitHub
	getAppInstallation = getAppInstallationFromGitHub
	findAppOrganizationInstallation = findAppOrganizationInstallationFromGitHub
	findAppUserInstallation = findAppUserInstallationFromGitHub
	getRepositoryPermission = getRepositoryPermissionFromGitHub
}

func githubNotFoundError() error {
	return &gh.ErrorResponse{Response: &http.Response{StatusCode: http.StatusNotFound}}
}

func stubEmptyHostedDiscovery(t *testing.T) {
	t.Helper()
	t.Cleanup(resetBindClientHooks)
	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	findAppUserInstallation = func(context.Context, *gh.Client, string) (*gh.Installation, error) {
		return nil, githubNotFoundError()
	}
	listRecentAppInstallations = func(context.Context, *gh.Client, time.Time, int, int) ([]common.PendingInstallation, int, error) {
		return nil, 0, nil
	}
}

func enableUnverifiedDevelopmentRepositories(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", "development")
	t.Setenv(allowUnverifiedDevelopmentRepositoriesEnv, "yes")
	findAppUserInstallation = func(context.Context, *gh.Client, string) (*gh.Installation, error) {
		return nil, githubNotFoundError()
	}
}

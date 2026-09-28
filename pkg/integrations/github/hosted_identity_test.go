package github

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	gh "github.com/google/go-github/v84/github"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support/contexts"
)

type hostedIdentityStub struct {
	login       string
	rows        []hostedInstallationSnapshot
	memberOf    map[string]bool
	memberErr   error
	cache       map[string]*models.HostedAppInstallationMember
	savedChecks []string
	reconciled  []hostedInstallationSnapshot
	pruned      [][]string
}

func stubHostedIdentity(t *testing.T, stub *hostedIdentityStub) {
	t.Helper()
	t.Cleanup(resetHostedIdentityHooks)

	findGitHubLoginForUser = func(string) (string, error) { return stub.login, nil }
	listHostedInstallationRows = func() ([]hostedInstallationSnapshot, error) { return stub.rows, nil }
	findCachedInstallationMember = func(installationID, login string) (*models.HostedAppInstallationMember, error) {
		if stub.cache == nil {
			return nil, nil
		}
		return stub.cache[installationID+"/"+login], nil
	}
	saveCachedInstallationMember = func(installationID, login string, allowed, errored bool, _ time.Time) error {
		saved := installationID + "/" + login
		if errored {
			saved += ":errored"
		}
		stub.savedChecks = append(stub.savedChecks, saved)
		return nil
	}
	checkInstallationMembership = func(_ core.IntegrationContext, _ int64, _, accountLogin, _ string) (bool, error) {
		if stub.memberErr != nil {
			return false, stub.memberErr
		}
		return stub.memberOf[accountLogin], nil
	}
	listAppInstallationsDetailed = func(core.IntegrationContext, int64) ([]hostedInstallationSnapshot, error) {
		return nil, nil
	}
	saveReconciledInstallation = func(snapshot hostedInstallationSnapshot) error {
		stub.reconciled = append(stub.reconciled, snapshot)
		return nil
	}
	pruneReconciledInstallations = func(liveInstallationIDs []string) error {
		stub.pruned = append(stub.pruned, liveInstallationIDs)
		return nil
	}
}

func Test__Sync_hostedIdentityPrepopulatesPicker(t *testing.T) {
	setHostedAppEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)

	stubHostedIdentity(t, &hostedIdentityStub{
		login: "dev-user",
		rows: []hostedInstallationSnapshot{
			{ID: "31", AccountLogin: "dev-user", AccountType: "User"},
			{ID: "32", AccountLogin: "acme", AccountType: "Organization"},
			{ID: "33", AccountLogin: "other-org", AccountType: "Organization"},
			{ID: "34", AccountLogin: "gone", AccountType: "User", Deleted: true},
		},
		memberOf: map[string]bool{"acme": true},
	})

	integrationCtx := &contexts.IntegrationContext{}
	require.NoError(t, (&GitHub{}).Sync(core.SyncContext{
		Logger:         logrus.NewEntry(logrus.New()),
		OrganizationID: "11111111-1111-1111-1111-111111111111",
		ActorUserID:    "22222222-2222-2222-2222-222222222222",
		BaseURL:        "https://app.example",
		Integration:    integrationCtx,
	}))

	metadata := integrationCtx.Metadata.(common.Metadata)
	assert.Equal(t, "dev-user", metadata.StartedByGitHubLogin)
	require.Len(t, metadata.PendingInstallations, 2)
	assert.Equal(t, "31", metadata.PendingInstallations[0].ID)
	assert.Equal(t, "32", metadata.PendingInstallations[1].ID)
	// The picker has options, so the connect stays in the app: no browser
	// action to GitHub.
	assert.Nil(t, integrationCtx.BrowserAction)
	assert.NotEmpty(t, metadata.State)
}

func Test__Sync_hostedIdentityKeepsInstallFlowWithoutIdentity(t *testing.T) {
	setHostedAppEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)

	stubHostedIdentity(t, &hostedIdentityStub{login: ""})

	integrationCtx := &contexts.IntegrationContext{}
	require.NoError(t, (&GitHub{}).Sync(core.SyncContext{
		OrganizationID: "11111111-1111-1111-1111-111111111111",
		BaseURL:        "https://app.example",
		Integration:    integrationCtx,
	}))

	metadata := integrationCtx.Metadata.(common.Metadata)
	assert.Empty(t, metadata.PendingInstallations)
	require.NotNil(t, integrationCtx.BrowserAction)
	assert.Contains(t, integrationCtx.BrowserAction.URL, "installations/new?state=")
}

func Test__Sync_hostedIdentityClearsMatchingInstallRequest(t *testing.T) {
	setHostedAppEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)

	stubHostedIdentity(t, &hostedIdentityStub{
		login: "member",
		rows: []hostedInstallationSnapshot{
			{ID: "41", AccountLogin: "acme", AccountType: "Organization"},
		},
		memberOf: map[string]bool{"acme": true},
	})

	integrationCtx := &contexts.IntegrationContext{
		State: "pending",
		Metadata: common.Metadata{
			State:     "csrf",
			HostedApp: true,
			InstallRequests: []common.InstallRequest{
				{ID: "1", AccountLogin: "acme", RequesterLogin: "member"},
			},
			InstallRequested: true,
			GitHubApp:        common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
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

	metadata := integrationCtx.Metadata.(common.Metadata)
	assert.False(t, metadata.InstallRequested)
	assert.Empty(t, metadata.InstallRequests)
	// The merge appends: the existing picker entry stays first.
	require.Len(t, metadata.PendingInstallations, 2)
	assert.Equal(t, "22", metadata.PendingInstallations[0].ID)
	assert.Equal(t, "41", metadata.PendingInstallations[1].ID)
}

func Test__Sync_hostedIdentityLeavesBoundConnectionAlone(t *testing.T) {
	setHostedAppEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)

	stub := &hostedIdentityStub{
		login: "member",
		rows: []hostedInstallationSnapshot{
			{ID: "51", AccountLogin: "member", AccountType: "User"},
		},
	}
	stubHostedIdentity(t, stub)

	integrationCtx := &contexts.IntegrationContext{
		State: "ready",
		Metadata: common.Metadata{
			State:          "csrf",
			HostedApp:      true,
			InstallationID: "11",
			Owner:          "acme",
			GitHubApp:      common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
			PendingInstallations: []common.PendingInstallation{
				{ID: "11", AccountLogin: "acme"},
			},
		},
	}

	require.NoError(t, (&GitHub{}).Sync(core.SyncContext{
		OrganizationID: "11111111-1111-1111-1111-111111111111",
		BaseURL:        "https://app.example",
		Integration:    integrationCtx,
	}))

	metadata := integrationCtx.Metadata.(common.Metadata)
	assert.Equal(t, "11", metadata.InstallationID)
	require.Len(t, metadata.PendingInstallations, 1)
	assert.Empty(t, stub.reconciled)
}

func Test__afterAppInstallationLegacy_identityClaim(t *testing.T) {
	setHostedAppEnv(t)

	t.Run("identity allows a stale installation", func(t *testing.T) {
		stubHostedClaim(t, hostedInstallationSnapshot{
			ID:           "61",
			AccountLogin: "dev-user",
			AccountType:  "User",
			LastEventAt:  time.Now().Add(-2 * time.Hour),
		}, false)
		stubHostedIdentity(t, &hostedIdentityStub{login: "dev-user"})

		integration := pendingHostedIntegration("csrf")
		ctx, rec := hostedRequestContext(
			integration,
			"/api/v1/github/app/setup?state=csrf&installation_id=61&setup_action=install",
			nil,
		)

		(&GitHub{}).afterAppInstallationLegacy(ctx)

		assert.Equal(t, http.StatusSeeOther, rec.Code)
		metadata := integration.Metadata.(common.Metadata)
		require.Len(t, metadata.PendingInstallations, 1)
		assert.Equal(t, "61", metadata.PendingInstallations[0].ID)
	})

	t.Run("identity allows an installation another organization uses", func(t *testing.T) {
		stubHostedClaim(t, hostedInstallationSnapshot{
			ID:           "62",
			AccountLogin: "acme",
			AccountType:  "Organization",
			LastEventAt:  time.Now().Add(-2 * time.Hour),
		}, true)
		stubHostedIdentity(t, &hostedIdentityStub{
			login:    "member",
			memberOf: map[string]bool{"acme": true},
		})

		integration := pendingHostedIntegration("csrf")
		ctx, rec := hostedRequestContext(
			integration,
			"/api/v1/github/app/setup?state=csrf&installation_id=62&setup_action=install",
			nil,
		)

		(&GitHub{}).afterAppInstallationLegacy(ctx)

		assert.Equal(t, http.StatusSeeOther, rec.Code)
		metadata := integration.Metadata.(common.Metadata)
		require.Len(t, metadata.PendingInstallations, 1)
		assert.Equal(t, "acme", metadata.PendingInstallations[0].AccountLogin)
	})

	t.Run("identity miss falls through to the first-claim rules", func(t *testing.T) {
		stubHostedClaim(t, hostedInstallationSnapshot{
			ID:           "63",
			AccountLogin: "acme",
			AccountType:  "Organization",
			LastEventAt:  time.Now().Add(-2 * time.Hour),
		}, false)
		stubHostedIdentity(t, &hostedIdentityStub{
			login:    "member",
			memberOf: map[string]bool{},
		})

		integration := pendingHostedIntegration("csrf")
		ctx, rec := hostedRequestContext(
			integration,
			"/api/v1/github/app/setup?state=csrf&installation_id=63&setup_action=install",
			nil,
		)

		(&GitHub{}).afterAppInstallationLegacy(ctx)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("identity error falls through instead of rejecting a fresh claim", func(t *testing.T) {
		stubHostedClaim(t, hostedInstallationSnapshot{
			ID:           "64",
			AccountLogin: "acme",
			AccountType:  "Organization",
			LastEventAt:  time.Now(),
		}, false)
		stubHostedIdentity(t, &hostedIdentityStub{
			login:     "member",
			memberErr: errors.New("GitHub unavailable"),
		})

		integration := pendingHostedIntegration("csrf")
		ctx, rec := hostedRequestContext(
			integration,
			"/api/v1/github/app/setup?state=csrf&installation_id=64&setup_action=install",
			nil,
		)

		(&GitHub{}).afterAppInstallationLegacy(ctx)

		assert.Equal(t, http.StatusSeeOther, rec.Code)
		require.Len(t, integration.Metadata.(common.Metadata).PendingInstallations, 1)
	})
}

func Test__Sync_hostedIdentityAdoptsCrossOrgApprovedRequest(t *testing.T) {
	setHostedAppEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)
	t.Cleanup(resetBindClientHooks)

	stubHostedAdopt(t, "acme", "71", "Organization")
	installationUsedByOtherOrg = func(string, string) (bool, error) { return true, nil }
	stubHostedIdentity(t, &hostedIdentityStub{
		login:    "member",
		memberOf: map[string]bool{"acme": true},
	})

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

	metadata := integrationCtx.Metadata.(common.Metadata)
	assert.False(t, metadata.InstallRequested)
	require.Len(t, metadata.PendingInstallations, 1)
	assert.Equal(t, "71", metadata.PendingInstallations[0].ID)
}

func Test__Sync_hostedAdoptRequiresInstallationAfterRequest(t *testing.T) {
	setHostedAppEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)

	now := time.Now().UTC()

	syncWithRequest := func(t *testing.T, installedAt time.Time, requestedAt time.Time) common.Metadata {
		t.Helper()
		t.Cleanup(resetHostedClaimHooks)
		stubHostedIdentity(t, &hostedIdentityStub{login: ""})
		installationUsedByOtherOrg = func(string, string) (bool, error) { return false, nil }
		findHostedInstallationByAccount = func(_ context.Context, login string) (*hostedInstallationSnapshot, error) {
			if !strings.EqualFold(login, "victim-org") {
				return nil, nil
			}
			return &hostedInstallationSnapshot{
				ID:           "95",
				AccountLogin: "victim-org",
				AccountType:  "Organization",
				CreatedAt:    installedAt,
				LastEventAt:  installedAt,
			}, nil
		}

		integrationCtx := &contexts.IntegrationContext{
			State: "pending",
			Metadata: common.Metadata{
				State:     "csrf",
				HostedApp: true,
				InstallRequests: []common.InstallRequest{
					{AccountLogin: "victim-org", CreatedAt: requestedAt.Format(time.RFC3339Nano)},
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
		return integrationCtx.Metadata.(common.Metadata)
	}

	t.Run("a forged request cannot adopt an installation that already existed", func(t *testing.T) {
		metadata := syncWithRequest(t, now.Add(-2*time.Hour), now)

		assert.Empty(t, metadata.PendingInstallations)
		require.Len(t, metadata.InstallRequests, 1)
		assert.Equal(t, "victim-org", metadata.InstallRequests[0].AccountLogin)
	})

	t.Run("an installation created after the request is adopted", func(t *testing.T) {
		metadata := syncWithRequest(t, now, now.Add(-time.Hour))

		require.Len(t, metadata.PendingInstallations, 1)
		assert.Equal(t, "95", metadata.PendingInstallations[0].ID)
		assert.Empty(t, metadata.InstallRequests)
	})
}

func Test__afterHostedAppBind_rechecksExclusivity(t *testing.T) {
	setHostedAppEnv(t)
	g := &GitHub{}

	pickerMetadata := func(installationID string) common.Metadata {
		return common.Metadata{
			State:     "csrf",
			HostedApp: true,
			GitHubApp: common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
			PendingInstallations: []common.PendingInstallation{
				{ID: installationID, AccountLogin: "acme", AccountType: "Organization"},
			},
		}
	}

	t.Run("blocks an unverified bind when another organization uses the installation", func(t *testing.T) {
		stubHostedClaim(t, hostedInstallationSnapshot{
			ID:           "91",
			AccountLogin: "acme",
			AccountType:  "Organization",
		}, true)
		stubHostedIdentity(t, &hostedIdentityStub{login: "member", memberOf: map[string]bool{}})

		integration := pendingHostedIntegration("csrf")
		integration.Metadata = pickerMetadata("91")
		ctx, rec := hostedRequestContext(integration, "/api/v1/github/app/bind?state=csrf&installation_id=91", nil)

		g.afterHostedAppBind(ctx)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.NotEqual(t, "ready", integration.State)
	})

	t.Run("identity-verified bind proceeds when another organization uses the installation", func(t *testing.T) {
		t.Cleanup(resetBindClientHooks)
		stubHostedClaim(t, hostedInstallationSnapshot{
			ID:           "92",
			AccountLogin: "acme",
			AccountType:  "Organization",
		}, true)
		stubHostedIdentity(t, &hostedIdentityStub{login: "member", memberOf: map[string]bool{"acme": true}})
		listInstallationRepos = func(context.Context, *gh.Client) ([]common.Repository, error) {
			return []common.Repository{{ID: 1, Name: "repo", URL: "https://github.com/acme/repo"}}, nil
		}
		newInstallationClient = func(core.IntegrationContext, int64, string) (*gh.Client, error) {
			return gh.NewClient(nil), nil
		}

		integration := pendingHostedIntegration("csrf")
		integration.Metadata = pickerMetadata("92")
		ctx, rec := hostedRequestContext(integration, "/api/v1/github/app/bind?state=csrf&installation_id=92", nil)

		g.afterHostedAppBind(ctx)

		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "ready", integration.State)
		assert.Equal(t, "92", integration.Metadata.(common.Metadata).InstallationID)
	})
}

func Test__userCanAccessInstallation(t *testing.T) {
	g := &GitHub{}
	integration := &contexts.IntegrationContext{}

	t.Run("own account matches without a lookup", func(t *testing.T) {
		stub := &hostedIdentityStub{login: "dev-user", memberErr: errors.New("must not be called")}
		stubHostedIdentity(t, stub)

		allowed, err := g.userCanAccessInstallation(integration, 99, "Dev-User", hostedInstallationSnapshot{
			ID:           "1",
			AccountLogin: "dev-user",
			AccountType:  "User",
		})
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("fresh cache answers without a lookup", func(t *testing.T) {
		stub := &hostedIdentityStub{
			memberErr: errors.New("must not be called"),
			cache: map[string]*models.HostedAppInstallationMember{
				"2/member": {Allowed: true, CheckedAt: time.Now().UTC()},
			},
		}
		stubHostedIdentity(t, stub)

		allowed, err := g.userCanAccessInstallation(integration, 99, "member", hostedInstallationSnapshot{
			ID:           "2",
			AccountLogin: "acme",
			AccountType:  "Organization",
		})
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("stale cache re-checks GitHub and stores the result", func(t *testing.T) {
		stub := &hostedIdentityStub{
			memberOf: map[string]bool{"acme": true},
			cache: map[string]*models.HostedAppInstallationMember{
				"3/member": {Allowed: false, CheckedAt: time.Now().UTC().Add(-time.Hour)},
			},
		}
		stubHostedIdentity(t, stub)

		allowed, err := g.userCanAccessInstallation(integration, 99, "member", hostedInstallationSnapshot{
			ID:           "3",
			AccountLogin: "acme",
			AccountType:  "Organization",
		})
		require.NoError(t, err)
		assert.True(t, allowed)
		assert.Equal(t, []string{"3/member"}, stub.savedChecks)
	})

	t.Run("live check ignores a fresh cached answer", func(t *testing.T) {
		stub := &hostedIdentityStub{
			memberOf: map[string]bool{},
			cache: map[string]*models.HostedAppInstallationMember{
				"5/member": {Allowed: true, CheckedAt: time.Now().UTC()},
			},
		}
		stubHostedIdentity(t, stub)

		allowed, err := g.userCanAccessInstallationLive(integration, 99, "member", hostedInstallationSnapshot{
			ID:           "5",
			AccountLogin: "acme",
			AccountType:  "Organization",
		})
		require.NoError(t, err)
		assert.False(t, allowed, "a revoked membership must not ride the cache into a claim")
		assert.Equal(t, []string{"5/member"}, stub.savedChecks)
	})

	t.Run("failed lookup is cached so retries do not burn the budget", func(t *testing.T) {
		stub := &hostedIdentityStub{memberErr: errors.New("GitHub unavailable")}
		stubHostedIdentity(t, stub)

		allowed, err := g.userCanAccessInstallation(integration, 99, "member", hostedInstallationSnapshot{
			ID:           "6",
			AccountLogin: "acme",
			AccountType:  "Organization",
		})
		require.Error(t, err)
		assert.False(t, allowed)
		assert.Equal(t, []string{"6/member:errored"}, stub.savedChecks)
	})

	t.Run("fresh errored cache blocks a retry inside the short window", func(t *testing.T) {
		stub := &hostedIdentityStub{
			memberErr: errors.New("must not be called"),
			cache: map[string]*models.HostedAppInstallationMember{
				"7/member": {Allowed: false, Errored: true, CheckedAt: time.Now().UTC().Add(-30 * time.Second)},
			},
		}
		stubHostedIdentity(t, stub)

		allowed, err := g.userCanAccessInstallation(integration, 99, "member", hostedInstallationSnapshot{
			ID:           "7",
			AccountLogin: "acme",
			AccountType:  "Organization",
		})
		require.NoError(t, err)
		assert.False(t, allowed)
		assert.Empty(t, stub.savedChecks)
	})

	t.Run("errored cache retries before the full check TTL", func(t *testing.T) {
		stub := &hostedIdentityStub{
			memberOf: map[string]bool{"acme": true},
			cache: map[string]*models.HostedAppInstallationMember{
				"8/member": {Allowed: false, Errored: true, CheckedAt: time.Now().UTC().Add(-2 * time.Minute)},
			},
		}
		stubHostedIdentity(t, stub)

		allowed, err := g.userCanAccessInstallation(integration, 99, "member", hostedInstallationSnapshot{
			ID:           "8",
			AccountLogin: "acme",
			AccountType:  "Organization",
		})
		require.NoError(t, err)
		assert.True(t, allowed, "a fixed app permission must show without waiting the full check TTL")
		assert.Equal(t, []string{"8/member"}, stub.savedChecks)
	})

	t.Run("clean not-a-member answer holds for the full check TTL", func(t *testing.T) {
		stub := &hostedIdentityStub{
			memberErr: errors.New("must not be called"),
			cache: map[string]*models.HostedAppInstallationMember{
				"9/member": {Allowed: false, Errored: false, CheckedAt: time.Now().UTC().Add(-2 * time.Minute)},
			},
		}
		stubHostedIdentity(t, stub)

		allowed, err := g.userCanAccessInstallation(integration, 99, "member", hostedInstallationSnapshot{
			ID:           "9",
			AccountLogin: "acme",
			AccountType:  "Organization",
		})
		require.NoError(t, err)
		assert.False(t, allowed)
		assert.Empty(t, stub.savedChecks)
	})

	t.Run("deleted installation is never accessible", func(t *testing.T) {
		stubHostedIdentity(t, &hostedIdentityStub{login: "dev-user"})

		allowed, err := g.userCanAccessInstallation(integration, 99, "dev-user", hostedInstallationSnapshot{
			ID:           "4",
			AccountLogin: "dev-user",
			AccountType:  "User",
			Deleted:      true,
		})
		require.NoError(t, err)
		assert.False(t, allowed)
	})
}

func Test__reconcileHostedInstallations(t *testing.T) {
	setHostedAppEnv(t)

	stub := &hostedIdentityStub{}
	stubHostedIdentity(t, stub)
	listAppInstallationsDetailed = func(core.IntegrationContext, int64) ([]hostedInstallationSnapshot, error) {
		return []hostedInstallationSnapshot{
			{ID: "81", AccountLogin: "acme", AccountType: "Organization", CreatedAt: time.Now().Add(-time.Hour)},
		}, nil
	}

	g := &GitHub{}
	ctx := core.SyncContext{
		Logger:      logrus.NewEntry(logrus.New()),
		Integration: &contexts.IntegrationContext{},
	}
	app := common.HostedApp{ID: 99, Slug: "superplane"}

	g.reconcileHostedInstallations(ctx, app)
	require.Len(t, stub.reconciled, 1)
	assert.Equal(t, "81", stub.reconciled[0].ID)

	// Rows GitHub no longer lists are pruned, so a dead installation stops
	// wasting discovery lookups.
	require.Len(t, stub.pruned, 1)
	assert.Equal(t, []string{"81"}, stub.pruned[0])

	// A second run inside the TTL window does not ask GitHub again.
	g.reconcileHostedInstallations(ctx, app)
	assert.Len(t, stub.reconciled, 1)
	assert.Len(t, stub.pruned, 1)
}

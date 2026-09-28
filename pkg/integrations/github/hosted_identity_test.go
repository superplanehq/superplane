package github

import (
	"errors"
	"net/http"
	"testing"
	"time"

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
	saveCachedInstallationMember = func(installationID, login string, allowed bool, _ time.Time) error {
		stub.savedChecks = append(stub.savedChecks, installationID+"/"+login)
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

	// A second run inside the TTL window does not ask GitHub again.
	g.reconcileHostedInstallations(ctx, app)
	assert.Len(t, stub.reconciled, 1)
}

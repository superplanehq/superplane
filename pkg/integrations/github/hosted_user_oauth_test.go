package github

import (
	"net/http"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func setHostedAppUserOAuthEnv(t *testing.T) {
	t.Helper()
	setHostedAppEnv(t)
	t.Setenv(common.EnvGitHubAppClientID, "Iv23liClient")
	t.Setenv(common.EnvGitHubAppClientSecret, "app-secret")
}

type hostedUserOAuthStub struct {
	login         string
	installations []hostedInstallationSnapshot
	listErr       error
	exchangeErr   error
	exchangedCode string
	recorded      []hostedInstallationSnapshot
	memberSaves   []string
	linked        []string
}

func stubHostedUserOAuth(t *testing.T, stub *hostedUserOAuthStub) {
	t.Helper()
	t.Cleanup(resetHostedUserOAuthHooks)
	t.Cleanup(resetHostedIdentityHooks)

	exchangeHostedUserOAuthCode = func(_ common.HostedApp, code string) (string, error) {
		if stub.exchangeErr != nil {
			return "", stub.exchangeErr
		}
		stub.exchangedCode = code
		return "user-token", nil
	}
	listUserInstallations = func(token string) (hostedUserIdentity, []hostedInstallationSnapshot, error) {
		require.Equal(t, "user-token", token)
		if stub.listErr != nil {
			return hostedUserIdentity{}, nil, stub.listErr
		}
		return hostedUserIdentity{Login: stub.login, ProviderID: "9001"}, stub.installations, nil
	}
	linkGitHubAccountForUser = func(userID string, identity hostedUserIdentity) error {
		stub.linked = append(stub.linked, userID+"/"+identity.Login+"/"+identity.ProviderID)
		return nil
	}
	saveReconciledInstallation = func(snapshot hostedInstallationSnapshot) error {
		stub.recorded = append(stub.recorded, snapshot)
		return nil
	}
	saveCachedInstallationMember = func(installationID, login string, allowed, errored bool, _ time.Time) error {
		if allowed && !errored {
			stub.memberSaves = append(stub.memberSaves, installationID+"/"+login)
		}
		return nil
	}
}

func Test__afterHostedAppUserOAuth(t *testing.T) {
	g := &GitHub{}

	t.Run("fills the picker from the user's installations and returns to setup", func(t *testing.T) {
		setHostedAppUserOAuthEnv(t)
		stub := &hostedUserOAuthStub{
			login: "member",
			installations: []hostedInstallationSnapshot{
				{ID: "11", AccountLogin: "member", AccountType: "User"},
				{ID: "12", AccountLogin: "acme", AccountType: "Organization"},
			},
		}
		stubHostedUserOAuth(t, stub)

		integration := pendingHostedIntegration("csrf")
		integration.Metadata = common.Metadata{
			State:           "csrf",
			HostedApp:       true,
			GitHubApp:       common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
			StartedByUserID: "22222222-2222-2222-2222-222222222222",
			SetupReturnPath: "/org-1/workspaces/PAY/setup?step=vcs",
		}
		ctx, rec := hostedRequestContext(
			integration,
			"/api/v1/github/app/oauth/callback?state=csrf&code=abc",
			nil,
		)

		g.HandleRequest(ctx)

		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "https://app.example/org-1/workspaces/PAY/setup?step=vcs", rec.Header().Get("Location"))
		assert.Equal(t, "abc", stub.exchangedCode)

		metadata := integration.Metadata.(common.Metadata)
		require.Len(t, metadata.PendingInstallations, 2)
		assert.Equal(t, "member", metadata.PendingInstallations[0].AccountLogin)
		assert.Equal(t, "acme", metadata.PendingInstallations[1].AccountLogin)
		assert.Equal(t, "member", metadata.StartedByGitHubLogin)
		assert.Len(t, stub.recorded, 2)
		assert.Equal(t, []string{"11/member", "12/member"}, stub.memberSaves)
		assert.Equal(t, []string{"22222222-2222-2222-2222-222222222222/member/9001"}, stub.linked)
		assertNoPlaintextSecrets(t, integration)
	})

	t.Run("resolves a waiting install request the user token proves", func(t *testing.T) {
		setHostedAppUserOAuthEnv(t)
		stub := &hostedUserOAuthStub{
			login: "member",
			installations: []hostedInstallationSnapshot{
				{ID: "12", AccountLogin: "acme", AccountType: "Organization"},
			},
		}
		stubHostedUserOAuth(t, stub)

		integration := pendingHostedIntegration("csrf")
		integration.Metadata = common.Metadata{
			State:     "csrf",
			HostedApp: true,
			GitHubApp: common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
			InstallRequests: []common.InstallRequest{
				{AccountLogin: "acme", RequesterLogin: "member"},
			},
			InstallRequested: true,
		}
		ctx, rec := hostedRequestContext(
			integration,
			"/api/v1/github/app/oauth/callback?state=csrf&code=abc",
			nil,
		)

		g.HandleRequest(ctx)

		assert.Equal(t, http.StatusSeeOther, rec.Code)
		metadata := integration.Metadata.(common.Metadata)
		assert.Empty(t, metadata.InstallRequests)
		require.Len(t, metadata.PendingInstallations, 1)
	})

	t.Run("a user without installations continues to the install page", func(t *testing.T) {
		setHostedAppUserOAuthEnv(t)
		stubHostedUserOAuth(t, &hostedUserOAuthStub{login: "member"})

		integration := pendingHostedIntegration("csrf")
		ctx, rec := hostedRequestContext(
			integration,
			"/api/v1/github/app/oauth/callback?state=csrf&code=abc",
			nil,
		)

		g.HandleRequest(ctx)

		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(
			t,
			"https://github.com/apps/superplane/installations/new?state=csrf",
			rec.Header().Get("Location"),
		)
	})

	t.Run("rejects a state mismatch", func(t *testing.T) {
		setHostedAppUserOAuthEnv(t)
		stubHostedUserOAuth(t, &hostedUserOAuthStub{login: "member"})

		integration := pendingHostedIntegration("csrf")
		ctx, rec := hostedRequestContext(
			integration,
			"/api/v1/github/app/oauth/callback?state=forged&code=abc",
			nil,
		)

		g.HandleRequest(ctx)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("a token exchange error returns to setup without options", func(t *testing.T) {
		setHostedAppUserOAuthEnv(t)
		stubHostedUserOAuth(t, &hostedUserOAuthStub{exchangeErr: assert.AnError})

		integration := pendingHostedIntegration("csrf")
		ctx, rec := hostedRequestContext(
			integration,
			"/api/v1/github/app/oauth/callback?state=csrf&code=abc",
			nil,
		)

		g.HandleRequest(ctx)

		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Empty(t, integration.Metadata.(common.Metadata).PendingInstallations)
	})
}

func Test__Sync_hostedConnectActionUsesUserOAuth(t *testing.T) {
	setHostedAppUserOAuthEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)

	stubHostedIdentity(t, &hostedIdentityStub{login: ""})

	integrationCtx := &contexts.IntegrationContext{}
	require.NoError(t, (&GitHub{}).Sync(core.SyncContext{
		Logger:         logrus.NewEntry(logrus.New()),
		OrganizationID: "11111111-1111-1111-1111-111111111111",
		BaseURL:        "https://app.example",
		Integration:    integrationCtx,
	}))

	require.NotNil(t, integrationCtx.BrowserAction)
	assert.Contains(t, integrationCtx.BrowserAction.URL, "https://github.com/login/oauth/authorize")
	assert.Contains(t, integrationCtx.BrowserAction.URL, "client_id=Iv23liClient")
	assert.Contains(
		t,
		integrationCtx.BrowserAction.URL,
		"redirect_uri=https%3A%2F%2Fapp.example%2Fapi%2Fv1%2Fgithub%2Fapp%2Foauth%2Fcallback",
	)
}

func Test__Sync_hostedPickerClearsStaleBrowserAction(t *testing.T) {
	setHostedAppEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)

	stubHostedIdentity(t, &hostedIdentityStub{login: ""})

	integrationCtx := &contexts.IntegrationContext{
		State: "pending",
		Metadata: common.Metadata{
			State:     "csrf",
			HostedApp: true,
			GitHubApp: common.GitHubAppMetadata{ID: 99, Slug: "superplane"},
			PendingInstallations: []common.PendingInstallation{
				{ID: "11", AccountLogin: "acme", AccountType: "Organization"},
			},
		},
	}
	integrationCtx.NewBrowserAction(core.BrowserAction{URL: "https://github.com/apps/superplane/installations/new?state=csrf"})

	require.NoError(t, (&GitHub{}).Sync(core.SyncContext{
		Logger:         logrus.NewEntry(logrus.New()),
		OrganizationID: "11111111-1111-1111-1111-111111111111",
		BaseURL:        "https://app.example",
		Integration:    integrationCtx,
	}))

	// The picker has options, so the stale install action must not send the
	// browser to GitHub again.
	assert.Nil(t, integrationCtx.BrowserAction)
}

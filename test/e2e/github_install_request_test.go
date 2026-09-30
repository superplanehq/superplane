package e2e

import (
	"testing"
	"time"

	pw "github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/support"
)

func TestGitHubInstallRequest(t *testing.T) {
	t.Run("pending catalog request appears in repository selection", func(t *testing.T) {
		steps := &githubInstallRequestSteps{t: t}
		steps.start()
		factory := steps.givenAnIncompleteWorkspaceExists()
		steps.givenALinkedGitHubIdentityWithPendingApproval()
		steps.visitWorkspaceRepositorySelection(factory)
		steps.assertThePendingRequestIsExplained()
	})

	t.Run("owner approval without state opens the approved page", func(t *testing.T) {
		steps := &githubInstallRequestSteps{t: t}
		steps.start()
		steps.whenGitHubReturnsAnOwnerApproval()
		steps.assertTheOwnerApprovalIsExplained()
	})
}

type githubInstallRequestSteps struct {
	t       *testing.T
	session *session.TestSession
}

func (s *githubInstallRequestSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
	require.NoError(s.t, models.EnableExperimentalFeature(s.session.OrgID, features.FeatureFactories))
}

func (s *githubInstallRequestSteps) givenAnIncompleteWorkspaceExists() *models.Factory {
	factory, err := models.CreateFactory(database.DB(s.t.Context()), s.session.OrgID, support.RandomName("workspace"), "", "")
	require.NoError(s.t, err)
	return factory
}

func (s *githubInstallRequestSteps) givenALinkedGitHubIdentityWithPendingApproval() {
	const providerUserID = int64(42)
	require.NoError(s.t, models.SaveAccountLinkedAccount(
		database.DB(s.t.Context()),
		models.NewAccountLinkedAccount(s.session.Account.ID, models.ProviderGitHub, "42", "octocat", "", ""),
	))
	require.NoError(s.t, models.ReplaceVCSProviderInstallRequests(
		database.DB(s.t.Context()),
		models.ProviderGitHub,
		[]models.VCSProviderInstallRequest{{
			RequestID:    101,
			AccountLogin: "acme",
			AccountType:  "Organization",
			RequesterID:  providerUserID,
			RequestedAt:  time.Now(),
		}},
	))
}

func (s *githubInstallRequestSteps) visitWorkspaceRepositorySelection(factory *models.Factory) {
	s.session.Visit("/" + s.session.OrgSlug + "/workspaces/" + factory.Key + "/setup?step=repo")
	s.session.AssertVisible(q.TestID("workspace-setup"))
}

func (s *githubInstallRequestSteps) whenGitHubReturnsAnOwnerApproval() {
	const installationID = int64(159131070)
	accountID := int64(301)
	require.NoError(s.t, models.ReplaceVCSProviderInstallRequests(
		database.DB(s.t.Context()),
		models.ProviderGitHub,
		[]models.VCSProviderInstallRequest{{
			RequestID:    101,
			AccountID:    &accountID,
			AccountLogin: "acme",
			RequesterID:  42,
			RequestedAt:  time.Now(),
		}},
	))
	require.NoError(s.t, models.UpsertVCSProviderInstallation(
		database.DB(s.t.Context()),
		&models.VCSProviderInstallation{
			Provider:       models.ProviderGitHub,
			InstallationID: installationID,
			AccountID:      &accountID,
			AccountLogin:   "acme",
			AccountType:    "Organization",
		},
	))
	s.session.Visit("/api/v1/github/app/setup?installation_id=159131070&setup_action=install")
}

func (s *githubInstallRequestSteps) assertThePendingRequestIsExplained() {
	require.NoError(s.t, s.session.Page().Locator(
		`[data-testid="first-run-github-install-requested"]`,
	).First().WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible, Timeout: pw.Float(15000)}))
	s.assertPendingRequestCopy()
}

func (s *githubInstallRequestSteps) assertPendingRequestCopy() {
	s.session.AssertText("Waiting for approval")
}

func (s *githubInstallRequestSteps) assertTheOwnerApprovalIsExplained() {
	require.Contains(s.t, s.session.Page().URL(), "/github/approved")
	require.NotContains(s.t, s.session.Page().URL(), "/workspaces/")
	require.NotContains(s.t, s.session.Page().URL(), "missing state")
	s.session.AssertVisible(q.TestID("github-install-approved"))
	s.session.AssertText("Request approved")
	s.session.AssertText("The SuperPlane GitHub App is approved.")
	s.session.AssertVisible(q.TestID("github-install-approved-open"))
}

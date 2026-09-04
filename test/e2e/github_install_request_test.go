package e2e

import (
	"encoding/json"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	pw "github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestGitHubInstallRequest(t *testing.T) {
	t.Run("setup callback without installation_id returns to onboarding with approval copy", func(t *testing.T) {
		steps := &githubInstallRequestSteps{t: t}
		steps.start()
		factory := steps.givenAnIncompleteWorkspaceExists()
		steps.givenAPendingHostedGitHubConnectionExists(factory)
		steps.rememberReturnToWorkspaceSetup(factory)
		steps.whenGitHubReturnsAnInstallRequest()
		steps.assertThePendingRequestIsExplained()
	})

	t.Run("connect screen explains a pending GitHub request from the return query", func(t *testing.T) {
		steps := &githubInstallRequestSteps{t: t}
		steps.start()
		factory := steps.givenAnIncompleteWorkspaceExists()
		steps.visitWorkspaceSetupWithInstallRequest(factory)
		steps.assertTheConnectScreenExplainsThePendingRequest()
	})

	t.Run("owner approval without state opens the approved page", func(t *testing.T) {
		steps := &githubInstallRequestSteps{t: t}
		steps.start()
		factory := steps.givenAnIncompleteWorkspaceExists()
		steps.rememberReturnToWorkspaceSetup(factory)
		steps.whenGitHubReturnsAnOwnerApproval()
		steps.assertTheOwnerApprovalIsExplained()
	})
}

type githubInstallRequestSteps struct {
	t           *testing.T
	session     *session.TestSession
	state       string
	integration *models.Integration
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

func (s *githubInstallRequestSteps) givenAPendingHostedGitHubConnectionExists(factory *models.Factory) {
	user, err := models.FindActiveHumanUserByAccountAndOrganization(
		database.DB(s.t.Context()),
		s.session.OrgID,
		s.session.Account.ID,
	)
	require.NoError(s.t, err)

	s.state = "e2e-github-request-" + uuid.NewString()
	integration, err := models.CreateIntegration(
		uuid.New(),
		s.session.OrgID,
		"github",
		support.RandomName("github"),
		nil,
	)
	require.NoError(s.t, err)

	integration.Metadata = datatypes.NewJSONType(map[string]any{
		"state":           s.state,
		"hostedApp":       true,
		"startedByUserID": user.ID.String(),
		"setupReturnPath": "/" + s.session.OrgSlug + "/workspaces/" + factory.Key + "/setup?step=vcs",
	})
	require.NoError(s.t, database.Conn().Save(integration).Error)
	s.integration = integration
}

func (s *githubInstallRequestSteps) rememberReturnToWorkspaceSetup(factory *models.Factory) {
	s.session.Visit("/" + s.session.OrgSlug + "/workspaces/" + factory.Key + "/setup?step=vcs")
	s.session.AssertVisible(q.TestID("workspace-setup"))
	s.openConnectScreenIfNeeded()

	now := time.Now().UnixMilli()
	slugPath := "/" + s.session.OrgSlug + "/workspaces/" + factory.Key + "/setup?step=vcs"
	returnPaths := map[string]string{
		s.session.OrgID.String(): "/" + s.session.OrgID.String() + "/workspaces/" + factory.Key + "/setup?step=vcs",
		s.session.OrgSlug:        slugPath,
	}
	for organizationID, path := range returnPaths {
		payload, err := json.Marshal(map[string]any{
			"path":      path,
			"createdAt": now,
		})
		require.NoError(s.t, err)
		_, err = s.session.Page().Evaluate(`([key, value]) => { window.localStorage.setItem(key, value); }`, []string{
			"integration-setup-return:" + organizationID,
			string(payload),
		})
		require.NoError(s.t, err)
	}
	_, err := s.session.Page().Evaluate(`(cookie) => { document.cookie = cookie; }`,
		"sp_integration_setup_return="+url.QueryEscape(slugPath)+"; Path=/; Max-Age=900; SameSite=Lax")
	require.NoError(s.t, err)
}

func (s *githubInstallRequestSteps) openConnectScreenIfNeeded() {
	welcome, err := s.session.Page().GetByTestId("first-run-welcome").Count()
	require.NoError(s.t, err)
	if welcome == 0 {
		s.session.AssertVisible(q.TestID("first-run-connect"))
		return
	}
	s.session.Click(q.TestID("first-run-get-started"))
	s.session.AssertVisible(q.TestID("first-run-connect"))
}

func (s *githubInstallRequestSteps) whenGitHubReturnsAnInstallRequest() {
	s.session.Visit("/api/v1/github/app/setup?state=" + s.state + "&setup_action=request")
}

func (s *githubInstallRequestSteps) whenGitHubReturnsAnOwnerApproval() {
	s.session.Visit("/api/v1/github/app/setup?installation_id=159131070&setup_action=install")
}

func (s *githubInstallRequestSteps) visitWorkspaceSetupWithInstallRequest(factory *models.Factory) {
	s.session.Visit("/" + s.session.OrgSlug + "/workspaces/" + factory.Key + "/setup?step=vcs&githubSetup=request")
}

func (s *githubInstallRequestSteps) assertTheConnectScreenExplainsThePendingRequest() {
	s.session.AssertVisible(q.TestID("first-run-connect"))
	s.session.AssertVisible(q.TestID("first-run-github-install-requested"))
	s.session.AssertVisible(q.TestID("first-run-connect-github"))
	s.assertPendingRequestCopy()
}

func (s *githubInstallRequestSteps) assertThePendingRequestIsExplained() {
	require.Contains(s.t, s.session.Page().URL(), "/setup")
	require.NotContains(s.t, s.session.Page().URL(), "/settings/integrations/")
	require.NotContains(s.t, s.session.Page().URL(), "invalid installation ID")
	s.openConnectScreenIfNeeded()
	require.NoError(s.t, s.session.Page().Locator(
		`[data-testid="first-run-github-install-requested"], [data-testid="github-install-requested"]`,
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
	s.session.AssertText("The SuperPlane GitHub App is approved. The person who asked can click Connect GitHub again.")
}

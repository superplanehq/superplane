package e2e

import (
	"encoding/json"
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
		steps.givenAPendingHostedGitHubConnectionExists()
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

func (s *githubInstallRequestSteps) givenAPendingHostedGitHubConnectionExists() {
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
	})
	require.NoError(s.t, database.Conn().Save(integration).Error)
	s.integration = integration
}

func (s *githubInstallRequestSteps) rememberReturnToWorkspaceSetup(factory *models.Factory) {
	s.session.Visit("/" + s.session.OrgSlug + "/workspaces/" + factory.Key + "/setup?step=vcs")
	s.session.AssertVisible(q.TestID("workspace-setup"))
	s.openConnectScreenIfNeeded()

	now := time.Now().UnixMilli()
	returnPaths := map[string]string{
		s.session.OrgID.String(): "/" + s.session.OrgID.String() + "/workspaces/" + factory.Key + "/setup?step=vcs",
		s.session.OrgSlug:        "/" + s.session.OrgSlug + "/workspaces/" + factory.Key + "/setup?step=vcs",
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
	require.NotContains(s.t, s.session.Page().URL(), "invalid installation ID")
	require.NoError(s.t, s.session.Page().Locator(
		`[data-testid="first-run-github-install-requested"], [data-testid="github-install-requested"]`,
	).First().WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible, Timeout: pw.Float(15000)}))
	s.assertPendingRequestCopy()
}

func (s *githubInstallRequestSteps) assertPendingRequestCopy() {
	s.session.AssertText("Ask a GitHub organization admin to approve the SuperPlane GitHub App.")
	s.session.AssertText("After they approve, click Connect GitHub again.")
}

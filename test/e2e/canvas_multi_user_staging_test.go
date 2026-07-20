package e2e

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
	"github.com/superplanehq/superplane/test/support"
)

func TestCanvasMultiUserStaging(t *testing.T) {
	t.Run("staging is isolated per user on the live canvas", func(t *testing.T) {
		steps := &canvasMultiUserStagingSteps{t: t}
		steps.start()

		steps.givenACanvas()
		ownerAccount := steps.session.Account
		ownerUserID := steps.userIDForAccount(ownerAccount.Email)

		steps.whenUserStagesNode(ownerAccount, "AlphaNode")
		steps.whenUserExitsEditMode()

		collaboratorAccount, collaboratorUserID := steps.createCollaborator()
		steps.whenUserStagesNode(collaboratorAccount, "BetaNode")
		steps.whenUserExitsEditMode()

		steps.canvas.AssertHasStagingForUser(ownerUserID)
		steps.canvas.AssertHasStagingForUser(collaboratorUserID)
		require.True(t, steps.canvas.StagingContainsNodeForUser(ownerUserID, "AlphaNode"))
		require.True(t, steps.canvas.StagingContainsNodeForUser(collaboratorUserID, "BetaNode"))

		steps.whenUserEntersEditMode(ownerAccount)
		steps.thenNodeIsVisible("AlphaNode")
		steps.thenNodeIsHidden("BetaNode")
		steps.whenUserExitsEditMode()

		steps.whenUserEntersEditMode(collaboratorAccount)
		steps.thenNodeIsVisible("BetaNode")
		steps.thenNodeIsHidden("AlphaNode")
		steps.whenUserExitsEditMode()

		steps.whenUserCommitsStaging(ownerAccount)
		steps.canvas.AssertNoStagingForUser(ownerUserID)
		steps.canvas.AssertLiveCanvasHasNode("AlphaNode")
		steps.canvas.AssertHasStagingForUser(collaboratorUserID)
		steps.canvas.AssertStagingStaleForUser(collaboratorUserID)
	})
}

type canvasMultiUserStagingSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps
}

func (s *canvasMultiUserStagingSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *canvasMultiUserStagingSteps) givenACanvas() {
	s.canvas = shared.NewCanvasSteps("E2E Multi User Staging", s.t, s.session)
	s.canvas.Create()
	s.canvas.Visit()
}

func (s *canvasMultiUserStagingSteps) createCollaborator() (*models.Account, uuid.UUID) {
	email := support.RandomName("collaborator") + "@superplane.local"
	account, err := models.CreateAccount("Collaborator User", email)
	require.NoError(s.t, err)

	user, err := models.CreateUser(s.session.OrgID, account.ID, email, "Collaborator User")
	require.NoError(s.t, err)

	authService, err := authorization.NewAuthService()
	require.NoError(s.t, err)
	err = authService.AssignRole(user.ID.String(), models.RoleOrgAdmin, s.session.OrgID.String(), models.DomainTypeOrganization)
	require.NoError(s.t, err)

	return account, user.ID
}

func (s *canvasMultiUserStagingSteps) userIDForAccount(email string) uuid.UUID {
	return s.canvas.UserIDForEmail(email)
}

func (s *canvasMultiUserStagingSteps) whenUserStagesNode(account *models.Account, nodeName string) {
	s.canvas.LoginAs(account)
	s.canvas.Visit()
	s.canvas.EnterEditMode()
	s.canvas.AddNoop(nodeName, models.Position{X: 500, Y: 200})
	s.session.AssertText(nodeName)
	s.canvas.WaitForStagingOnCurrentDraft()
	s.canvas.ClickOnEmptyCanvasArea()
}

func (s *canvasMultiUserStagingSteps) whenUserEntersEditMode(account *models.Account) {
	s.canvas.LoginAs(account)
	s.canvas.Visit()
	s.canvas.EnterEditMode()
}

func (s *canvasMultiUserStagingSteps) whenUserExitsEditMode() {
	s.canvas.ExitEditMode()
}

func (s *canvasMultiUserStagingSteps) whenUserCommitsStaging(account *models.Account) {
	s.canvas.LoginAs(account)
	s.canvas.Visit()
	s.canvas.EnterEditMode()
	s.canvas.ClickOnEmptyCanvasArea()
	s.canvas.CommitStaging()
}

func (s *canvasMultiUserStagingSteps) thenNodeIsVisible(nodeName string) {
	s.session.AssertText(nodeName)
}

func (s *canvasMultiUserStagingSteps) thenNodeIsHidden(nodeName string) {
	s.session.AssertHidden(q.TestID("node", nodeName, "header"))
}

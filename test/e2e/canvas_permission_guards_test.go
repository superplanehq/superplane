package e2e

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
	"github.com/superplanehq/superplane/test/support"
)

func TestCanvasPermissionGuards(t *testing.T) {
	t.Run("viewer can read canvas but cannot enter edit mode", func(t *testing.T) {
		steps := &canvasPermissionGuardSteps{t: t}
		steps.start()
		steps.givenACanvasExists("Viewer Read Only Canvas")
		steps.loginAsViewer()
		steps.visitCanvas()
		steps.assertEditDisabled()
		steps.assertNoStagingActions()
	})

	t.Run("viewer cannot open agent without agent permissions", func(t *testing.T) {
		steps := &canvasPermissionGuardSteps{t: t}
		steps.start()
		steps.enableAgentFeature()
		steps.givenACanvasExists("Viewer Agent Guard Canvas")
		steps.loginAsViewer()
		steps.visitCanvas()
		steps.assertAgentHidden()
	})

	t.Run("update can stage, reset, commit, and edit version history", func(t *testing.T) {
		steps := &canvasPermissionGuardSteps{t: t}
		steps.start()
		steps.givenACanvasExists("Canvas Editor Canvas")
		steps.loginWithCanvasPermissions("canvas-editor", canvasPermission("update"))
		steps.visitCanvas()
		steps.enterEditMode()
		steps.stageNoopNode("Canvas Update Node")
		steps.assertStagingActionsEnabled()
		steps.assertVersionHistoryAllowsEditing()
	})
}

type canvasPermissionGuardSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps
}

func (s *canvasPermissionGuardSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *canvasPermissionGuardSteps) givenACanvasExists(name string) {
	s.canvas = shared.NewCanvasSteps(name, s.t, s.session)
	s.canvas.Create()
}

func (s *canvasPermissionGuardSteps) enableAgentFeature() {
	require.NoError(s.t, models.EnableExperimentalFeature(s.session.OrgID, features.FeatureClaudeManagedAgents))
}

func (s *canvasPermissionGuardSteps) loginAsViewer() {
	loginAsViewer(s.t, s.session)
}

func (s *canvasPermissionGuardSteps) loginWithCanvasPermissions(roleLabel string, permissions ...*permissionSpec) {
	loginWithCanvasPermissions(s.t, s.session, roleLabel, permissions...)
}

func (s *canvasPermissionGuardSteps) visitCanvas() {
	s.canvas.Visit()
}

func (s *canvasPermissionGuardSteps) enterEditMode() {
	s.canvas.EnterEditModeWithoutStagingActionAssertions()
}

func (s *canvasPermissionGuardSteps) stageNoopNode(name string) {
	s.canvas.AddNoop(name, models.Position{X: 500, Y: 200})
	s.canvas.WaitForStaging(uuid.Nil)
	s.canvas.ClickOnEmptyCanvasArea()
}

func (s *canvasPermissionGuardSteps) assertEditDisabled() {
	s.session.AssertDisabled(q.TestID("canvas-edit-button"))
}

func (s *canvasPermissionGuardSteps) assertNoStagingActions() {
	s.session.AssertHidden(q.TestID("canvas-reset-staging-button"))
	s.session.AssertHidden(q.TestID("canvas-commit-staging-button"))
}

func (s *canvasPermissionGuardSteps) assertAgentHidden() {
	s.session.AssertHidden(q.TestID("canvas-tool-sidebar-toggle"))
	s.session.AssertHidden(q.TestID("canvas-tool-sidebar"))
}

func (s *canvasPermissionGuardSteps) assertStagingActionsEnabled() {
	s.canvas.AssertStagingActionsVisibleAndEnabled()
}

func (s *canvasPermissionGuardSteps) assertVersionHistoryAllowsEditing() {
	s.session.Click(q.TestID("canvas-versions-sidebar-toggle"))
	s.session.AssertHidden(q.Text("You do not have permission to edit this canvas."))
}

type permissionSpec struct {
	resource string
	action   string
}

func canvasPermission(action string) *permissionSpec {
	return organizationPermission("canvases", action)
}

func organizationPermission(resource, action string) *permissionSpec {
	return &permissionSpec{resource: resource, action: action}
}

func loginAsViewer(t *testing.T, sess *session.TestSession) {
	account := createAccountForRole(t, sess, "viewer", models.RoleOrgViewer)
	sess.Account = account
	sess.Login()
}

func loginWithCanvasPermissions(t *testing.T, sess *session.TestSession, roleLabel string, permissions ...*permissionSpec) {
	loginWithOrganizationPermissions(t, sess, roleLabel, permissions...)
}

func loginWithOrganizationPermissions(t *testing.T, sess *session.TestSession, roleLabel string, permissions ...*permissionSpec) {
	roleName := support.RandomName(roleLabel)
	rolePermissions := make([]*authorization.Permission, 0, len(permissions))
	for _, permission := range permissions {
		rolePermissions = append(rolePermissions, &authorization.Permission{
			Resource:   permission.resource,
			Action:     permission.action,
			DomainType: models.DomainTypeOrganization,
		})
	}

	authService, err := authorization.NewAuthService()
	require.NoError(t, err)

	err = authService.CreateCustomRole(sess.OrgID.String(), &authorization.RoleDefinition{
		Name:        roleName,
		DisplayName: roleLabel,
		DomainType:  models.DomainTypeOrganization,
		Description: "E2E permission guard role",
		Permissions: rolePermissions,
		InheritsFrom: &authorization.RoleDefinition{
			Name:       models.RoleOrgViewer,
			DomainType: models.DomainTypeOrganization,
		},
	})
	require.NoError(t, err)

	account := createAccountForRole(t, sess, roleLabel, roleName)
	sess.Account = account
	sess.Login()
}

func createAccountForRole(t *testing.T, sess *session.TestSession, label, roleName string) *models.Account {
	email := support.RandomName(label) + "@superplane.local"
	account, err := models.CreateAccount(label, email)
	require.NoError(t, err)

	user, err := models.CreateUser(sess.OrgID, account.ID, email, label)
	require.NoError(t, err)

	authService, err := authorization.NewAuthService()
	require.NoError(t, err)

	err = authService.AssignRole(user.ID.String(), roleName, sess.OrgID.String(), models.DomainTypeOrganization)
	require.NoError(t, err)

	return account
}

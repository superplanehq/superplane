package e2e

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
)

func TestRoles(t *testing.T) {
	steps := &RolesSteps{t: t}

	t.Run("creating a new role", func(t *testing.T) {
		steps.start()
		steps.visitCreateRolePage()
		steps.fillInCreateRoleForm("E2E Example Role")
		steps.selectAllOrganizationPermissions()
		steps.submitRoleForm()
		steps.assertRoleSavedInDB("E2E Example Role")
	})
}

type RolesSteps struct {
	t       *testing.T
	session *session.TestSession
}

func (s *RolesSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *RolesSteps) visitCreateRolePage() {
	s.session.Visit("/" + s.session.OrgID.String() + "/settings/create-role")
}

func (s *RolesSteps) fillInCreateRoleForm(name string) {
	nameInput := q.Locator(`input[placeholder="Enter role name"]`)

	s.session.FillIn(nameInput, name)
	s.session.Sleep(300)
}

func (s *RolesSteps) selectAllOrganizationPermissions() {
	selectAllButtons := q.Locator(`button:has-text("Select all")`)
	s.session.Click(selectAllButtons)
	s.session.Sleep(300)
}

func (s *RolesSteps) submitRoleForm() {
	createButton := q.Locator("button:has-text('Create Role')")

	s.session.ScrollToTheBottomOfPage()
	s.session.Click(createButton)
	s.session.Sleep(1500)
}

func (s *RolesSteps) assertRoleSavedInDB(displayName string) {
	var metadata []models.RoleMetadata
	err := database.Conn().Where("domain_type = ? AND domain_id = ?", models.DomainTypeOrganization, s.session.OrgID.String()).Find(&metadata).Error
	require.NoError(s.t, err)

	for _, m := range metadata {
		if m.DisplayName == displayName {
			return
		}
	}

	require.Fail(s.t, "role metadata not found for display name %q", displayName)
}

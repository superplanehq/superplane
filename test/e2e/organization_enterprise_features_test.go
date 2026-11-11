package e2e

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/support"
)

func TestOrganizationGrantControlsCustomRoles(t *testing.T) {
	steps := &organizationGrantSteps{t: t}
	steps.start()
	steps.whenTheGrantedOrganizationCreatesARole("Release manager")
	steps.thenTheRoleIsListed("Release manager")

	steps.givenAnOrganizationWithoutTheGrant()
	steps.whenThatOrganizationTriesToCreateARole("Blocked role")
	steps.thenTheRoleIsNotCreated("Blocked role")
}

type organizationGrantSteps struct {
	t       *testing.T
	session *session.TestSession
	other   *models.Organization
}

func (s *organizationGrantSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	// Organization settings stay behind the factories feature.
	require.NoError(s.t, models.EnableExperimentalFeature(s.session.OrgID, features.FeatureFactories))
	s.session.Login()
}

func (s *organizationGrantSteps) whenTheGrantedOrganizationCreatesARole(name string) {
	s.createRole(s.session.OrgID.String(), name)
	s.session.WaitUntilURLContains("/settings/roles")
}

func (s *organizationGrantSteps) thenTheRoleIsListed(name string) {
	s.session.AssertText(name)
}

func (s *organizationGrantSteps) givenAnOrganizationWithoutTheGrant() {
	account, err := models.CreateAccount("Ungranted owner", support.RandomName("ungranted")+"@superplane.local")
	require.NoError(s.t, err)

	organization, err := models.CreateOrganization(support.RandomName("ungranted-org"), "")
	require.NoError(s.t, err)
	user, err := models.CreateUser(organization.ID, account.ID, account.Email, account.Name)
	require.NoError(s.t, err)
	authService, err := authorization.NewAuthService()
	require.NoError(s.t, err)
	tx := database.Conn().Begin()
	if err := authService.SetupOrganization(tx, organization.ID.String(), user.ID.String()); err != nil {
		tx.Rollback()
		require.NoError(s.t, err)
	}
	require.NoError(s.t, tx.Commit().Error)

	s.other = organization
	s.session.Account = account
	s.session.OrgID = organization.ID
	s.session.Login()
}

func (s *organizationGrantSteps) whenThatOrganizationTriesToCreateARole(name string) {
	s.createRole(s.other.ID.String(), name)
	s.session.AssertText("Failed to create role")
}

func (s *organizationGrantSteps) thenTheRoleIsNotCreated(name string) {
	s.session.Visit("/" + s.other.ID.String() + "/settings/roles")
	s.session.AssertHidden(q.Text(name))
}

func (s *organizationGrantSteps) createRole(orgID, name string) {
	s.session.Visit("/" + orgID + "/settings/create-role")
	s.session.FillIn(q.Locator("#role-name-input"), name)
	s.session.Click(q.Locator(`button:has-text("Select all")`))
	s.session.Click(q.Locator(`button:has-text("Create Role")`))
}

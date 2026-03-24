package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
)

func TestAdminDashboard(t *testing.T) {
	t.Run("non-admin user is redirected away from admin page", func(t *testing.T) {
		steps := &adminSteps{t: t}
		steps.start()
		steps.session.Login()
		steps.session.Visit("/admin")
		// The frontend AdminLayout checks installation_admin and redirects to /
		steps.session.Sleep(1000)
		steps.assertNotOnAdminPage()
	})

	t.Run("admin user can access admin dashboard and see organizations", func(t *testing.T) {
		steps := &adminSteps{t: t}
		steps.start()
		steps.promoteToAdmin()
		steps.session.Login()
		steps.session.Visit("/admin")
		steps.assertOnAdminPage()
		steps.assertOrganizationVisible("e2e-org")
	})

	t.Run("admin user can view organization details", func(t *testing.T) {
		steps := &adminSteps{t: t}
		steps.start()
		steps.promoteToAdmin()
		steps.session.Login()
		steps.session.Visit("/admin")
		steps.clickOrganization("e2e-org")
		steps.assertUserVisible("E2E User")
	})

	t.Run("admin user can start and end impersonation", func(t *testing.T) {
		steps := &adminSteps{t: t}
		steps.start()
		steps.createImpersonationTarget()
		steps.promoteToAdmin()
		steps.session.Login()

		// Navigate to admin accounts page
		steps.session.Visit("/admin/accounts")
		steps.session.Sleep(500)

		// Start impersonation on the other user
		steps.clickImpersonate()
		steps.session.Sleep(2000)

		// Should land on org selector with the impersonation banner
		steps.assertImpersonationBannerVisible()

		// End impersonation by clicking Exit
		steps.clickExitImpersonation()
		steps.session.Sleep(2000)

		// Should be back on admin page
		steps.assertOnAdminPage()
	})
}

func TestAdminOwnerSetupPromotion(t *testing.T) {
	t.Run("owner setup promotes first account to installation admin", func(t *testing.T) {
		steps := &adminSetupSteps{t: t}
		steps.start()
		steps.visitSetupPage()
		steps.fillInOwnerDetailsAndSubmit()
		steps.assertAccountIsInstallationAdmin()
	})
}

// -- Admin dashboard step helpers --

type adminSteps struct {
	t       *testing.T
	session *session.TestSession
}

func (s *adminSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
}

func (s *adminSteps) createImpersonationTarget() {
	account, err := models.CreateAccount("Other User", "other@superplane.local")
	require.NoError(s.t, err)
	_, err = models.CreateUser(s.session.OrgID, account.ID, account.Email, account.Name)
	require.NoError(s.t, err)
}

func (s *adminSteps) promoteToAdmin() {
	err := models.PromoteToInstallationAdmin(s.session.Account.ID.String())
	require.NoError(s.t, err)
}

func (s *adminSteps) assertOnAdminPage() {
	s.session.AssertText("Installation Admin")
	s.session.AssertText("All Organizations")
}

func (s *adminSteps) assertNotOnAdminPage() {
	s.session.AssertURLContains("/")
	// Should NOT be on /admin - the admin layout redirects non-admins
	url := s.session.Page().URL()
	assert.NotContains(s.t, url, "/admin")
}

func (s *adminSteps) assertOrganizationVisible(name string) {
	s.session.AssertText(name)
}

func (s *adminSteps) clickOrganization(name string) {
	s.session.Click(q.Text(name))
	s.session.Sleep(500)
}

func (s *adminSteps) assertUserVisible(name string) {
	s.session.AssertText(name)
}

func (s *adminSteps) clickImpersonate() {
	s.session.Click(q.Text("Impersonate"))
}

func (s *adminSteps) assertImpersonationBannerVisible() {
	s.session.AssertText("You are viewing as")
	s.session.AssertText("Exit Impersonation")
}

func (s *adminSteps) clickExitImpersonation() {
	s.session.Click(q.Text("Exit Impersonation"))
}

// -- Owner setup + admin promotion step helpers --

type adminSetupSteps struct {
	t       *testing.T
	session *session.TestSession
}

func (s *adminSetupSteps) start() {
	middleware.ResetOwnerSetupStateForTests()
	s.session = ctx.NewSession(s.t)
	s.session.StartWithoutUser()
}

func (s *adminSetupSteps) visitSetupPage() {
	s.session.Visit("/setup")
}

func (s *adminSetupSteps) fillInOwnerDetailsAndSubmit() {
	s.session.FillIn(q.Locator(`input[type="email"]`), "admin-test@superplane.local")
	s.session.FillIn(q.Locator(`input[placeholder="First name"]`), "Admin")
	s.session.FillIn(q.Locator(`input[placeholder="Last name"]`), "User")
	s.session.FillIn(q.Locator(`input[placeholder="Password"]`), "Password1")
	s.session.FillIn(q.Locator(`input[placeholder="Confirm password"]`), "Password1")
	s.session.Click(q.Text("Next"))
	s.session.Click(q.Text("Do this later"))
	s.waitForSetupToComplete()
}

func (s *adminSetupSteps) waitForSetupToComplete() {
	for i := 0; i < 50; i++ {
		var count int64
		err := database.Conn().Model(&models.Organization{}).Count(&count).Error
		if err == nil && count > 0 {
			s.session.Sleep(500)
			return
		}
		s.session.Sleep(200)
	}
	s.t.Fatal("timed out waiting for owner setup to complete")
}

func (s *adminSetupSteps) assertAccountIsInstallationAdmin() {
	account, err := models.FindAccountByEmail("admin-test@superplane.local")
	require.NoError(s.t, err)
	assert.True(s.t, account.IsInstallationAdmin(), "owner account should be an installation admin")
}

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
)

func TestOwnerSetupFlow(t *testing.T) {
	steps := &ownerSetupSteps{t: t}

	t.Run("completing owner setup via UI creates owner and redirects to home", func(t *testing.T) {
		steps.start()
		steps.visitSetupPage()
		steps.fillInOwnerDetailsAndSubmit("owner@example.com", "Owner", "User", "Password1")
		steps.assertOwnerAndOrganizationCreated()
		steps.assertRedirectedToOrganizationHome()
		steps.assertOwnerSetupIsNoLongerRequired()
	})

	t.Run("can login with email and password after owner setup", func(t *testing.T) {
		steps.start()
		steps.visitSetupPage()
		steps.fillInOwnerDetailsAndSubmit("owner@example.com", "Owner", "User", "Password1")
		steps.assertOwnerAndOrganizationCreated()
		steps.assertRedirectedToOrganizationHome()
		steps.clearCookies()
		steps.visitLoginPage()
		steps.clickEmailPasswordLogin()
		steps.fillInEmailAndPassword("owner@example.com", "Password1")
		steps.submitLoginForm()
		steps.assertRedirectedToOrganizationHome()
	})
}

type ownerSetupSteps struct {
	t       *testing.T
	session *session.TestSession
	orgID   string
}

func (s *ownerSetupSteps) start() {
	middleware.ResetOwnerSetupStateForTests()

	s.session = ctx.NewSession(s.t)
	s.session.StartWithoutUser()
}

func (s *ownerSetupSteps) visitSetupPage() {
	s.session.Visit("/")
}

func (s *ownerSetupSteps) fillInOwnerDetailsAndSubmit(email, firstName, lastName, password string) {
	s.session.FillIn(q.Locator(`input[type="email"]`), email)
	s.session.FillIn(q.Locator(`input[placeholder="First name"]`), firstName)
	s.session.FillIn(q.Locator(`input[placeholder="Last name"]`), lastName)
	s.session.FillIn(q.Locator(`input[placeholder="Password"]`), password)
	s.session.Click(q.Text("Next"))
	s.session.Sleep(500) // wait for redirect
}

func (s *ownerSetupSteps) assertOwnerAndOrganizationCreated() {
	var userCount int64
	var orgCount int64
	var accountsCount int64

	err := database.Conn().Model(&models.User{}).Count(&userCount).Error
	assert.NoError(s.t, err, "count users")
	assert.Equal(s.t, int64(1), userCount, "expected exactly one user to be created")

	err = database.Conn().Model(&models.Organization{}).Count(&orgCount).Error
	assert.NoError(s.t, err, "count organizations")
	assert.Equal(s.t, int64(1), orgCount, "expected exactly one organization to be created")

	err = database.Conn().Model(&models.Account{}).Count(&accountsCount).Error
	assert.NoError(s.t, err, "count organizations")
	assert.Equal(s.t, int64(1), accountsCount, "expected exactly one account to be created")

	org, err := models.FindOrganizationByName("Demo")
	assert.NoError(s.t, err, "find organization Demo")

	s.orgID = org.ID.String()
}

func (s *ownerSetupSteps) assertRedirectedToOrganizationHome() {
	currentURL := s.session.Page().URL()
	assert.Contains(s.t, currentURL, "/"+s.orgID, "expected to be redirected to organization home")
}

func (s *ownerSetupSteps) assertOwnerSetupIsNoLongerRequired() {
	required := middleware.IsOwnerSetupRequired()
	assert.False(s.t, required, "owner setup should no longer be required after completion")
}

func (s *ownerSetupSteps) clearCookies() {
	err := s.session.Page().Context().ClearCookies()
	assert.NoError(s.t, err, "clear cookies")
}

func (s *ownerSetupSteps) visitLoginPage() {
	s.session.Visit("/login")
	s.session.Sleep(500) // wait for page load
}

func (s *ownerSetupSteps) clickEmailPasswordLogin() {
	s.session.Click(q.Text("Email & Password"))
	s.session.Sleep(500) // wait for navigation
}

func (s *ownerSetupSteps) fillInEmailAndPassword(email, password string) {
	s.session.FillIn(q.Locator(`input[type="email"]`), email)
	s.session.FillIn(q.Locator(`input[type="password"]`), password)
}

func (s *ownerSetupSteps) submitLoginForm() {
	s.session.Click(q.Text("Sign in"))
	s.session.Sleep(1000) // wait for redirect
}

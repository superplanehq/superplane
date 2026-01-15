package e2e

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
)

func TestOwnerSetupFlow(t *testing.T) {
	t.Run("completing owner setup via UI creates owner and redirects to home", func(t *testing.T) {
		steps := &ownerSetupSteps{t: t}
		steps.start()
		steps.visitRootPage()
		steps.assertRedirectedToSetup()
		steps.visitSetupPage()
		steps.fillInOwnerDetailsAndSubmit("owner@example.com", "Owner", "User", "Password1")
		steps.assertOwnerAndOrganizationCreated()
		steps.assertRedirectedToOrganizationHome()
		steps.assertOwnerSetupIsNoLongerRequired()
	})

	t.Run("can complete owner setup with SMTP configuration", func(t *testing.T) {
		steps := &ownerSetupSteps{t: t}
		steps.start()
		steps.visitRootPage()
		steps.assertRedirectedToSetup()
		steps.visitSetupPage()
		steps.fillInOwnerDetails("smtp-owner@example.com", "SMTP", "Owner", "Password1")
		steps.chooseSMTPSetup()
		steps.fillInSMTPDetails("smtp.example.com", "587", "smtp-user", "smtp-pass", "SuperPlane", "noreply@example.com", true)
		steps.submitSMTPSetup()
		steps.assertOwnerAndOrganizationCreated()
		steps.assertRedirectedToOrganizationHome()
		steps.assertOwnerSetupIsNoLongerRequired()
	})

	t.Run("can login with email and password after owner setup", func(t *testing.T) {
		steps := &ownerSetupSteps{t: t}
		steps.start()
		steps.visitRootPage()
		steps.assertRedirectedToSetup()
		steps.visitSetupPage()
		steps.fillInOwnerDetailsAndSubmit("owner@example.com", "Owner", "User", "Password1")
		steps.assertOwnerAndOrganizationCreated()
		steps.assertRedirectedToOrganizationHome()
		steps.clearCookies()
		steps.visitLoginPage()
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
	s.session.Visit("/setup")
}

func (s *ownerSetupSteps) visitRootPage() {
	s.session.Visit("/")
}

func (s *ownerSetupSteps) assertRedirectedToSetup() {
	// Give the router a moment to handle the redirect.
	for i := 0; i < 10; i++ {
		currentURL := s.session.Page().URL()
		if strings.Contains(currentURL, "/setup") {
			return
		}
		s.session.Sleep(200)
	}

	currentURL := s.session.Page().URL()
	assert.Contains(s.t, currentURL, "/setup", "expected to be redirected to owner setup")
}

func (s *ownerSetupSteps) fillInOwnerDetailsAndSubmit(email, firstName, lastName, password string) {
	s.fillInOwnerDetails(email, firstName, lastName, password)
	s.session.Click(q.Text("Next"))
	s.session.Click(q.Text("Skip for now"))
	// Poll for setup to complete - wait for organization to be created in database
	s.waitForSetupToComplete()
}

func (s *ownerSetupSteps) fillInOwnerDetails(email, firstName, lastName, password string) {
	s.session.FillIn(q.Locator(`input[type="email"]`), email)
	s.session.FillIn(q.Locator(`input[placeholder="First name"]`), firstName)
	s.session.FillIn(q.Locator(`input[placeholder="Last name"]`), lastName)
	s.session.FillIn(q.Locator(`input[placeholder="Password"]`), password)
	s.session.FillIn(q.Locator(`input[placeholder="Confirm password"]`), password)
}

func (s *ownerSetupSteps) chooseSMTPSetup() {
	s.session.Click(q.Text("Next"))
	s.session.Click(q.Text("Set up SMTP"))
}

func (s *ownerSetupSteps) fillInSMTPDetails(host, port, username, password, fromName, fromEmail string, useTLS bool) {
	s.session.FillIn(q.Locator(`input[placeholder="smtp.example.com"]`), host)
	s.session.FillIn(q.Locator(`input[placeholder="587"]`), port)
	if username != "" {
		s.session.FillIn(q.Locator(`input[placeholder="smtp-user"]`), username)
	}
	if password != "" {
		s.session.FillIn(q.Locator(`input[placeholder="SMTP password"]`), password)
	}
	if fromName != "" {
		s.session.FillIn(q.Locator(`input[placeholder="SuperPlane"]`), fromName)
	}
	s.session.FillIn(q.Locator(`input[placeholder="noreply@example.com"]`), fromEmail)

	if !useTLS {
		s.session.Click(q.Locator(`input[type="checkbox"]`))
	}
}

func (s *ownerSetupSteps) submitSMTPSetup() {
	s.session.Click(q.Text("Finish setup"))
	s.waitForSetupToComplete()
}

func (s *ownerSetupSteps) waitForSetupToComplete() {
	// Poll for up to 10 seconds, checking every 200ms
	for i := 0; i < 50; i++ {
		var orgCount int64
		err := database.Conn().Model(&models.Organization{}).Count(&orgCount).Error
		if err == nil && orgCount > 0 {
			// Setup completed - give it a moment for redirect
			s.session.Sleep(500)
			return
		}
		s.session.Sleep(200)
	}
	// If we get here, setup didn't complete in time
	s.t.Log("Warning: Setup may not have completed - proceeding with assertions")
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

func (s *ownerSetupSteps) fillInEmailAndPassword(email, password string) {
	s.session.FillIn(q.Locator(`input[type="email"]`), email)
	s.session.FillIn(q.Locator(`input[type="password"]`), password)
}

func (s *ownerSetupSteps) submitLoginForm() {
	s.session.Click(q.Text("Login"))
	s.session.Sleep(1000) // wait for redirect
}

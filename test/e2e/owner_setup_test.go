package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
)

func TestOwnerSetupFlow(t *testing.T) {
	t.Cleanup(func() {
		middleware.MarkOwnerSetupCompleted()
	})
	t.Run("completing owner setup via UI creates owner and redirects to home", func(t *testing.T) {
		steps := &ownerSetupSteps{t: t}
		steps.start()
		steps.visitRootPage()
		steps.assertRedirectedToSetup()
		steps.visitSetupPage()
		steps.fillInOwnerDetailsAndSubmit("owner@example.com", "Owner", "User", "Password1")
		steps.assertOwnerAndOrganizationCreated()
		steps.assertRedirectedToOrganization()
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
		steps.assertRedirectedToOrganization()
		steps.clearCookies()
		steps.visitLoginPage()
		steps.fillInEmailAndPassword("owner@example.com", "Password1")
		steps.submitLoginForm()
		steps.assertRedirectedToOrganization()
	})
}

type ownerSetupSteps struct {
	t       *testing.T
	session *session.TestSession
	orgSlug string
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
	s.session.WaitUntilURLContains("/setup")
}

func (s *ownerSetupSteps) fillInOwnerDetailsAndSubmit(email, firstName, lastName, password string) {
	s.fillInOwnerDetails(email, firstName, lastName, password)
	s.session.Click(q.Text("Finish setup"))
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

func (s *ownerSetupSteps) waitForSetupToComplete() {
	require.Eventually(s.t, func() bool {
		var orgCount int64
		err := database.Conn().Model(&models.Organization{}).Count(&orgCount).Error
		return err == nil && orgCount > 0
	}, 10*time.Second, 200*time.Millisecond, "owner setup did not create an organization")
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

	s.orgSlug = org.Slug
}

func (s *ownerSetupSteps) assertRedirectedToOrganization() {
	// The app lands on "/" first and then resolves the organization redirect
	// after several async requests, so poll instead of checking once.
	s.session.WaitUntilURLContains("/" + s.orgSlug)
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
	s.session.AssertVisible(q.Text("Welcome to SuperPlane"))
}

func (s *ownerSetupSteps) fillInEmailAndPassword(email, password string) {
	// With magic code enabled, toggle to password form first,
	// then fill in the fields that belong to it.
	s.session.Click(q.Text("Sign in with password instead"))
	s.session.AssertVisible(q.Locator(`input[type="password"]`))
	s.session.FillIn(q.Locator(`input[type="email"]`), email)
	s.session.FillIn(q.Locator(`input[type="password"]`), password)
}

func (s *ownerSetupSteps) submitLoginForm() {
	s.session.Click(q.Text("Login"))
	s.session.WaitUntilURLDoesNotContain("/login")
}

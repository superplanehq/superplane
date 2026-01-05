package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/utils"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
)

func TestSignupWithInvitation(t *testing.T) {
	steps := &signupWithInvitationSteps{t: t}

	t.Run("signup with invitation succeeds when BLOCK_SIGNUP is enabled", func(t *testing.T) {
		invitedEmail := "invited-user@example.com"
		steps.start()
		steps.createInvitation(invitedEmail)
		steps.visitSignupPage()
		steps.fillInSignupForm("Invited User", invitedEmail, "Password123")
		steps.submitSignupForm()
		steps.assertAccountCreated(invitedEmail)
		steps.assertInvitationAccepted(invitedEmail)
		steps.assertUserCreated(invitedEmail)
		steps.assertRedirectedToOrganization()
	})

	t.Run("signup without invitation fails when BLOCK_SIGNUP is enabled", func(t *testing.T) {
		nonInvitedEmail := "non-invited-user@example.com"
		steps.start()
		steps.visitSignupPage()
		steps.fillInSignupForm("Non Invited User", nonInvitedEmail, "Password123")
		steps.submitSignupForm()
		steps.assertSignupBlocked()
		steps.assertAccountNotCreated(nonInvitedEmail)
	})

	t.Run("signup with invitation accepts pending invitations", func(t *testing.T) {
		invitedEmail := "multi-invite-user@example.com"
		steps.start()
		steps.createInvitation(invitedEmail)
		steps.createSecondOrganizationAndInvitation(invitedEmail)
		steps.visitSignupPage()
		steps.fillInSignupForm("Multi Invite User", invitedEmail, "Password123")
		steps.submitSignupForm()
		steps.assertAccountCreated(invitedEmail)
		steps.assertAllInvitationsAccepted(invitedEmail)
		steps.assertUserCreatedInAllOrganizations(invitedEmail)
	})
}

type signupWithInvitationSteps struct {
	t       *testing.T
	session *session.TestSession
	orgID   string
}

func (s *signupWithInvitationSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
	s.orgID = s.session.OrgID.String()
}

func (s *signupWithInvitationSteps) createInvitation(email string) {
	invitation, err := models.CreateInvitation(
		s.session.OrgID,
		s.session.Account.ID,
		email,
		models.InvitationStatePending,
	)
	require.NoError(s.t, err, "create invitation")
	require.NotNil(s.t, invitation, "invitation should be created")
}

func (s *signupWithInvitationSteps) createSecondOrganizationAndInvitation(email string) {
	orgName := "e2e-org-2"
	organization, err := models.CreateOrganization(orgName, "")
	require.NoError(s.t, err, "create second organization")

	invitation, err := models.CreateInvitation(
		organization.ID,
		s.session.Account.ID,
		email,
		models.InvitationStatePending,
	)
	require.NoError(s.t, err, "create second invitation")
	require.NotNil(s.t, invitation, "second invitation should be created")
}

func (s *signupWithInvitationSteps) visitSignupPage() {
	s.session.ClearCookies()
	s.session.Visit("/signup/email")
	s.session.Sleep(500) // wait for page load
}

func (s *signupWithInvitationSteps) fillInSignupForm(name, email, password string) {
	nameInput := q.Locator(`input[name="name"]`)
	emailInput := q.Locator(`input[type="email"]`)
	passwordInput := q.Locator(`input[type="password"]`)

	s.session.FillIn(nameInput, name)
	s.session.FillIn(emailInput, email)
	s.session.FillIn(passwordInput, password)
	s.session.Sleep(300)
}

func (s *signupWithInvitationSteps) submitSignupForm() {
	submitButton := q.Text("Sign up")
	s.session.Click(submitButton)
	s.session.Sleep(2000) // wait for signup to complete
}

func (s *signupWithInvitationSteps) assertAccountCreated(email string) {
	normalizedEmail := utils.NormalizeEmail(email)
	account, err := models.FindAccountByEmail(normalizedEmail)
	require.NoError(s.t, err, "account should be created")
	assert.Equal(s.t, normalizedEmail, account.Email, "account email should match")

	// Verify password auth was created
	passwordAuth, err := models.FindAccountPasswordAuthByAccountID(account.ID)
	require.NoError(s.t, err, "password auth should be created")
	assert.NotEmpty(s.t, passwordAuth.PasswordHash, "password hash should be set")
}

func (s *signupWithInvitationSteps) assertInvitationAccepted(email string) {
	normalizedEmail := utils.NormalizeEmail(email)
	var invitation models.OrganizationInvitation
	err := database.Conn().
		Where("email = ? AND organization_id = ?", normalizedEmail, s.orgID).
		First(&invitation).
		Error
	require.NoError(s.t, err, "invitation should exist")
	assert.Equal(s.t, models.InvitationStateAccepted, invitation.State, "invitation should be accepted")
}

func (s *signupWithInvitationSteps) assertAllInvitationsAccepted(email string) {
	normalizedEmail := utils.NormalizeEmail(email)
	var invitations []models.OrganizationInvitation
	err := database.Conn().
		Where("email = ?", normalizedEmail).
		Find(&invitations).
		Error
	require.NoError(s.t, err, "should find invitations")
	require.Greater(s.t, len(invitations), 0, "should have at least one invitation")

	for _, invitation := range invitations {
		assert.Equal(s.t, models.InvitationStateAccepted, invitation.State, "all invitations should be accepted")
	}
}

func (s *signupWithInvitationSteps) assertUserCreated(email string) {
	normalizedEmail := utils.NormalizeEmail(email)
	account, err := models.FindAccountByEmail(normalizedEmail)
	require.NoError(s.t, err, "account should exist")

	var user models.User
	err = database.Conn().
		Where("account_id = ? AND organization_id = ?", account.ID, s.orgID).
		First(&user).
		Error
	require.NoError(s.t, err, "user should be created in organization")
	assert.Equal(s.t, normalizedEmail, user.Email, "user email should match")
}

func (s *signupWithInvitationSteps) assertUserCreatedInAllOrganizations(email string) {
	normalizedEmail := utils.NormalizeEmail(email)
	account, err := models.FindAccountByEmail(normalizedEmail)
	require.NoError(s.t, err, "account should exist")

	var userCount int64
	err = database.Conn().
		Model(&models.User{}).
		Where("account_id = ?", account.ID).
		Count(&userCount).
		Error
	require.NoError(s.t, err, "should count users")
	assert.GreaterOrEqual(s.t, userCount, int64(2), "user should be created in multiple organizations")
}

func (s *signupWithInvitationSteps) assertRedirectedToOrganization() {
	currentURL := s.session.Page().URL()
	assert.Contains(s.t, currentURL, "/", "should be redirected after signup")
	// Should be redirected to organization home or organization select page
	assert.NotContains(s.t, currentURL, "/signup", "should not be on signup page")
	assert.NotContains(s.t, currentURL, "/login", "should not be on login page")
}

func (s *signupWithInvitationSteps) assertSignupBlocked() {
	// Check for error message indicating signup is blocked
	s.session.AssertText("signup is currently disabled")
}

func (s *signupWithInvitationSteps) assertAccountNotCreated(email string) {
	normalizedEmail := utils.NormalizeEmail(email)
	_, err := models.FindAccountByEmail(normalizedEmail)
	require.Error(s.t, err, "account should not be created")
}

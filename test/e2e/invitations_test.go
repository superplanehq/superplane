package e2e

import (
	"errors"
	"fmt"
	"testing"

	pw "github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func TestInvitations(t *testing.T) {
	steps := &invitationSteps{t: t}

	t.Run("accepting invite link assigns viewer role", func(t *testing.T) {
		steps.startLoggedIn()
		token := steps.createInviteLink()
		invitee := steps.createInviteeAccount()
		steps.loginAs(invitee)
		steps.acceptInvite(token)
		steps.assertInviteeViewerRole(invitee.Email)
	})

	t.Run("following invite link and creating password account", func(t *testing.T) {
		steps.startLoggedOut()
		token := steps.createInviteLink()

		steps.followInviteLinkToLogin(token)
		steps.openSignupForm()

		firstName := "Invite"
		lastName := "User"
		email := support.RandomName("invitee") + "@superplane.local"
		password := "TestPassword123!"

		steps.fillSignupForm(firstName, lastName, email, password)
		steps.submitSignup()
		steps.waitForOrganizationRedirect()
		steps.assertInviteeViewerRole(email)
	})

	t.Run("disabled invite link no longer works", func(t *testing.T) {
		steps.startLoggedIn()
		token := steps.createInviteLink()
		steps.disableInviteLink(token)
		steps.visitInviteLink(token)
		steps.assertInviteLinkDisabled()
	})

	t.Run("viewer sees invite link access message", func(t *testing.T) {
		steps.startLoggedIn()
		token := steps.createInviteLink()
		invitee := steps.createInviteeAccount()
		steps.loginAs(invitee)
		steps.acceptInvite(token)
		steps.visitMembersSettings()
		steps.assertViewerInviteLinkMessage()
	})
}

type invitationSteps struct {
	t       *testing.T
	session *session.TestSession
}

func (s *invitationSteps) startLoggedIn() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *invitationSteps) startLoggedOut() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
}

func (s *invitationSteps) createInviteLink() string {
	inviteLink, err := models.FindInviteLinkByOrganizationID(s.session.OrgID.String())
	if err == nil {
		return inviteLink.Token.String()
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		require.NoError(s.t, err)
	}

	inviteLink, err = models.CreateInviteLink(s.session.OrgID)
	require.NoError(s.t, err)
	return inviteLink.Token.String()
}

func (s *invitationSteps) createInviteeAccount() *models.Account {
	account, err := models.CreateAccount("Invitee User", support.RandomName("invitee")+"@superplane.local")
	require.NoError(s.t, err)
	return account
}

func (s *invitationSteps) loginAs(account *models.Account) {
	s.session.Account = account
	s.session.Login()
}

func (s *invitationSteps) acceptInvite(token string) {
	s.session.Visit("/invite/" + token)
	s.waitForOrganizationRedirect()
}

func (s *invitationSteps) waitForOrganizationRedirect() {
	waitErr := s.session.Page().WaitForURL("**/"+s.session.OrgID.String()+"*", pw.PageWaitForURLOptions{
		Timeout: pw.Float(10000),
	})
	require.NoError(s.t, waitErr)
}

func (s *invitationSteps) visitInviteLink(token string) {
	s.session.Visit("/invite/" + token)
}

func (s *invitationSteps) visitMembersSettings() {
	s.session.Visit(fmt.Sprintf("/%s/settings/members", s.session.OrgID.String()))
}

func (s *invitationSteps) followInviteLinkToLogin(token string) {
	s.session.Visit("/invite/" + token)
	waitErr := s.session.Page().WaitForURL("**/login?redirect=**", pw.PageWaitForURLOptions{
		Timeout: pw.Float(10000),
	})
	require.NoError(s.t, waitErr)
}

func (s *invitationSteps) openSignupForm() {
	button := s.session.Page().Locator("text=Create an account").First()
	require.NoError(s.t, button.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible}))
	require.NoError(s.t, button.Click())
}

func (s *invitationSteps) fillSignupForm(firstName, lastName, email, password string) {
	page := s.session.Page()

	require.NoError(s.t, page.Locator(`input[placeholder="First name"]`).Fill(firstName))
	require.NoError(s.t, page.Locator(`input[placeholder="Last name"]`).Fill(lastName))
	require.NoError(s.t, page.Locator(`input[placeholder="Email"]`).Fill(email))
	require.NoError(s.t, page.Locator(`input[placeholder="Password"]`).Fill(password))
	require.NoError(s.t, page.Locator(`input[placeholder="Repeat password"]`).Fill(password))
}

func (s *invitationSteps) submitSignup() {
	button := s.session.Page().Locator("text=Create account").First()
	require.NoError(s.t, button.Click())
}

func (s *invitationSteps) disableInviteLink(token string) {
	inviteLink, err := models.FindInviteLinkByToken(token)
	require.NoError(s.t, err)
	inviteLink.Enabled = false
	require.NoError(s.t, models.SaveInviteLink(inviteLink))
}

func (s *invitationSteps) assertInviteLinkDisabled() {
	s.session.AssertText("Invite link not available")
}

func (s *invitationSteps) assertViewerInviteLinkMessage() {
	s.session.AssertText("Invite link to add members")
	s.session.AssertText("Reach out to an organization owner or admin to invite new members.")

	copyLinkVisible, err := s.session.Page().Locator("text=Copy link").IsVisible()
	require.NoError(s.t, err)
	require.False(s.t, copyLinkVisible)
}

func (s *invitationSteps) assertInviteeViewerRole(email string) {
	user, err := models.FindActiveUserByEmail(s.session.OrgID.String(), email)
	require.NoError(s.t, err)

	authService, err := authorization.NewAuthService()
	require.NoError(s.t, err)

	roles, err := authService.GetUserRolesForOrg(user.ID.String(), s.session.OrgID.String())
	require.NoError(s.t, err)
	require.NotEmpty(s.t, roles)

	roleNames := make([]string, len(roles))
	for i, role := range roles {
		roleNames[i] = role.Name
	}
	assert.Contains(s.t, roleNames, models.RoleOrgViewer)
}

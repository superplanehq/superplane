package e2e

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	pw "github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func TestInvitations(t *testing.T) {
	t.Run("accepting invite link assigns viewer role", func(t *testing.T) {
		steps := &invitationSteps{t: t}
		steps.startLoggedIn()
		token := steps.createInviteLink()
		invitee := steps.createInviteeAccount()
		steps.loginAs(invitee)
		steps.acceptInvite(token)
		steps.assertInviteeViewerRole(invitee.Email)
	})

	t.Run("following invite link and creating password account", func(t *testing.T) {
		steps := &invitationSteps{t: t}
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
		steps := &invitationSteps{t: t}
		steps.startLoggedIn()
		token := steps.createInviteLink()
		steps.disableInviteLink(token)
		steps.visitInviteLink(token)
		steps.assertInviteLinkDisabled()
	})

	t.Run("viewer sees invite link access message", func(t *testing.T) {
		steps := &invitationSteps{t: t}
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
	inviteLink, err := models.FindInviteLinkByOrganizationID(database.DB(s.t.Context()), s.session.OrgID.String())
	if err == nil {
		return inviteLink.Token.String()
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		require.NoError(s.t, err)
	}

	inviteLink, err = models.CreateInviteLink(database.DB(s.t.Context()), s.session.OrgID)
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

// openSignupForm navigates from the login page to the signup form.
//
// Depending on which login method is primary, "Create an account" may
// already be visible, or a "Sign in with password instead" toggle may
// need to be clicked first. This waits for whichever appears first and
// clicks the toggle only if needed.
func (s *invitationSteps) openSignupForm() {
	page := s.session.Page()
	const timeout = 15000 * time.Millisecond

	toggle := page.Locator("text=Sign in with password instead").First()
	createAccount := page.Locator("text=Create an account").First()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if visible, err := toggle.IsVisible(); err == nil && visible {
			require.NoError(s.t, toggle.Click())
			break
		}
		if visible, err := createAccount.IsVisible(); err == nil && visible {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if err := createAccount.WaitFor(pw.LocatorWaitForOptions{
		State:   pw.WaitForSelectorStateVisible,
		Timeout: pw.Float(float64(timeout / time.Millisecond)),
	}); err != nil {
		s.t.Fatalf("neither the password-login toggle nor \"Create an account\" appeared within %s: %v", timeout, err)
	}
	require.NoError(s.t, createAccount.Click())
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
	tx := database.DB(s.t.Context())
	inviteLink, err := models.FindInviteLinkByToken(tx, token)
	require.NoError(s.t, err)
	inviteLink.Enabled = false
	require.NoError(s.t, models.SaveInviteLink(tx, inviteLink))
}

func (s *invitationSteps) assertInviteLinkDisabled() {
	s.session.AssertText("Invite link not available")
}

func (s *invitationSteps) assertViewerInviteLinkMessage() {
	s.session.AssertText("Invite link to add members")
	s.session.AssertText("You don't have permission to manage invite links.")

	copyLinkVisible, err := s.session.Page().Locator("text=Copy link").IsVisible()
	require.NoError(s.t, err)
	require.False(s.t, copyLinkVisible)
}

func (s *invitationSteps) assertInviteeViewerRole(email string) {
	user, err := models.FindActiveUserByEmail(s.session.OrgID.String(), email)
	require.NoError(s.t, err)

	authService, err := authorization.NewAuthService()
	require.NoError(s.t, err)

	roles, err := authService.GetUserRolesForOrg(context.Background(), user.ID.String(), s.session.OrgID.String())
	require.NoError(s.t, err)
	require.NotEmpty(s.t, roles)

	require.Equal(s.t, len(roles), 1)

	role := roles[0]
	assert.Equal(s.t, role.Name, models.RoleOrgViewer)
}

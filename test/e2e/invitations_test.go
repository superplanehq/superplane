package e2e

import (
	"fmt"
	"testing"

	pw "github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func TestInvitations(t *testing.T) {
	t.Run("existing member can open an enabled invite link", func(t *testing.T) {
		steps := &invitationSteps{t: t}
		steps.startLoggedIn()
		token := steps.seedInviteLink(true)
		steps.acceptInvite(token)
	})

	t.Run("new account cannot join through an invite link", func(t *testing.T) {
		steps := &invitationSteps{t: t}
		steps.startLoggedIn()
		token := steps.seedInviteLink(true)
		invitee := steps.createInviteeAccount()
		steps.loginAs(invitee)
		steps.visitInviteLink(token)
		steps.assertEmailInvitationRequired()
		steps.assertNotOrganizationMember(invitee.Email)
	})

	t.Run("password signup from an invite link does not join the organization", func(t *testing.T) {
		steps := &invitationSteps{t: t}
		steps.startLoggedOut()
		token := steps.seedInviteLink(true)

		steps.followInviteLinkToLogin(token)
		steps.openSignupForm()

		firstName := "Invite"
		lastName := "User"
		email := support.RandomName("invitee") + "@superplane.local"
		password := "TestPassword123!"

		steps.fillSignupForm(firstName, lastName, email, password)
		steps.submitSignup()
		steps.assertEmailInvitationRequired()
		steps.assertNotOrganizationMember(email)
	})

	t.Run("disabled invite link no longer works", func(t *testing.T) {
		steps := &invitationSteps{t: t}
		steps.startLoggedIn()
		token := steps.seedInviteLink(false)
		steps.visitInviteLink(token)
		steps.assertInviteLinkDisabled()
	})

	t.Run("operator cannot invite members", func(t *testing.T) {
		steps := &invitationSteps{t: t}
		steps.startLoggedIn()
		loginAsOperator(steps.t, steps.session)
		steps.visitMembersSettings()
		steps.assertMemberVisible(steps.session.Account.Email)
		steps.assertInviteMemberHidden()
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

func (s *invitationSteps) seedInviteLink(enabled bool) string {
	return support.SeedHistoricalInviteLink(s.t, s.session.OrgID, enabled).Token.String()
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
	waitErr := s.session.Page().WaitForURL("**/"+s.session.OrgSlug+"*", pw.PageWaitForURLOptions{
		Timeout: pw.Float(30000),
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
		Timeout: pw.Float(30000),
	})
	require.NoError(s.t, waitErr)
}

func (s *invitationSteps) openSignupForm() {
	s.session.AssertVisible(q.Text("Continue with email"))
	s.session.Click(q.Text("Sign in with password instead"))
	s.session.Click(q.Text("Create an account"))
}

func (s *invitationSteps) fillSignupForm(firstName, lastName, email, password string) {
	s.session.FillIn(q.Locator(`input[placeholder="First name"]`), firstName)
	s.session.FillIn(q.Locator(`input[placeholder="Last name"]`), lastName)
	s.session.FillIn(q.Locator(`input[placeholder="Email"]`), email)
	s.session.FillIn(q.Locator(`input[placeholder="Password"]`), password)
	s.session.FillIn(q.Locator(`input[placeholder="Repeat password"]`), password)
}

func (s *invitationSteps) submitSignup() {
	s.session.Click(q.Text("Create account"))
}

func (s *invitationSteps) assertInviteLinkDisabled() {
	s.session.AssertText("Invitation unavailable")
	s.session.AssertText("invite link disabled")
}

func (s *invitationSteps) assertEmailInvitationRequired() {
	s.session.AssertText("Invitation unavailable")
	s.session.AssertText("This organization uses email invitations. Ask an organization admin for an invitation.")
}

func (s *invitationSteps) assertInviteMemberHidden() {
	s.session.AssertHidden(q.Text("Invite member"))
}

func (s *invitationSteps) assertMemberVisible(email string) {
	s.session.AssertText(email)
}

func (s *invitationSteps) assertNotOrganizationMember(email string) {
	_, err := models.FindActiveUserByEmail(s.session.OrgID.String(), email)
	require.ErrorIs(s.t, err, gorm.ErrRecordNotFound)
}

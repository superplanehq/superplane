package e2e

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/support"
)

func TestMagicCodeLogin(t *testing.T) {
	steps := &magicCodeSteps{t: t}

	t.Run("magic code email form is shown as primary login method", func(t *testing.T) {
		steps.start()
		steps.visitLoginPage()
		steps.assertMagicCodeFormVisible()
	})

	t.Run("requesting code transitions to code input step", func(t *testing.T) {
		steps.start()
		steps.visitLoginPage()
		steps.enterEmailAndRequestCode("e2e@superplane.local")
		steps.assertCodeStepVisible()
	})

	t.Run("back button returns to email step", func(t *testing.T) {
		steps.start()
		steps.visitLoginPage()
		steps.enterEmailAndRequestCode("e2e@superplane.local")
		steps.assertCodeStepVisible()
		steps.clickBackToEmail()
		steps.assertMagicCodeFormVisible()
	})

	t.Run("existing user can login with magic code", func(t *testing.T) {
		steps.start()
		steps.visitLoginPage()
		email := "e2e@superplane.local"
		steps.enterEmailAndRequestCode(email)
		steps.insertKnownMagicCode(email, "123456")
		steps.enterCodeAndSubmit("123456")
		steps.assertRedirectedToOrganization()
	})

	t.Run("new user can sign up with magic code via invite link", func(t *testing.T) {
		steps.start()
		inviteToken := steps.createInviteLink()
		email := support.RandomName("magic") + "@superplane.local"
		steps.followInviteLinkToLogin(inviteToken)
		steps.enterEmailAndRequestCode(email)
		steps.insertKnownMagicCode(email, "654321")
		steps.enterCodeAndSubmit("654321")
		steps.assertAccountCreated(email)
	})

	t.Run("invalid code shows error message", func(t *testing.T) {
		steps.start()
		steps.visitLoginPage()
		email := "e2e@superplane.local"
		steps.enterEmailAndRequestCode(email)
		steps.insertKnownMagicCode(email, "111111")
		steps.enterCodeAndSubmit("999999")
		steps.assertInvalidCodeError()
	})

	t.Run("can toggle to password login and back", func(t *testing.T) {
		steps.start()
		steps.visitLoginPage()
		steps.assertMagicCodeFormVisible()
		steps.clickPasswordLoginToggle()
		steps.assertPasswordFormVisible()
		steps.clickMagicCodeToggle()
		steps.assertMagicCodeFormVisible()
	})
}

type magicCodeSteps struct {
	t       *testing.T
	session *session.TestSession
}

func (s *magicCodeSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
}

func (s *magicCodeSteps) visitLoginPage() {
	s.session.Visit("/login")
	s.session.Sleep(500)
}

func (s *magicCodeSteps) assertMagicCodeFormVisible() {
	s.session.AssertVisible(q.Text("Continue with email"))
	s.session.AssertVisible(q.Locator(`input[type="email"]`))
}

func (s *magicCodeSteps) enterEmailAndRequestCode(email string) {
	s.session.FillIn(q.Locator(`input[type="email"]`), email)
	s.session.Click(q.Text("Continue with email"))
	s.session.Sleep(500)
}

func (s *magicCodeSteps) assertCodeStepVisible() {
	s.session.AssertVisible(q.Text("Check your email"))
	s.session.AssertVisible(q.Locator(`input[name="code"]`))
	s.session.AssertVisible(q.Text("Use a different email"))
}

func (s *magicCodeSteps) clickBackToEmail() {
	s.session.Click(q.Text("Use a different email"))
	s.session.Sleep(300)
}

// insertKnownMagicCode inserts a magic code with a known plaintext value
// directly into the database. This bypasses the email delivery flow,
// allowing the e2e test to enter the code in the UI.
func (s *magicCodeSteps) insertKnownMagicCode(email, code string) {
	codeHash := crypto.HashToken(code)
	expiresAt := time.Now().Add(10 * time.Minute)
	_, err := models.CreateAccountMagicCode(strings.ToLower(strings.TrimSpace(email)), codeHash, expiresAt)
	if err != nil {
		s.t.Fatalf("insert known magic code: %v", err)
	}
}

func (s *magicCodeSteps) enterCodeAndSubmit(code string) {
	s.session.FillIn(q.Locator(`input[name="code"]`), code)
	s.session.Click(q.Text("Sign in"))
	s.session.Sleep(1500)
}

func (s *magicCodeSteps) assertRedirectedToOrganization() {
	currentURL := s.session.Page().URL()
	assert.Contains(s.t, currentURL, "/"+s.session.OrgID.String(),
		"expected redirect to organization home, got %s", currentURL)
}

func (s *magicCodeSteps) assertAccountCreated(email string) {
	var count int64
	err := database.Conn().Model(&models.Account{}).Where("email = ?", email).Count(&count).Error
	assert.NoError(s.t, err)
	assert.Equal(s.t, int64(1), count, "expected account to be created for %s", email)
}

func (s *magicCodeSteps) assertInvalidCodeError() {
	s.session.AssertVisible(q.Text("Invalid or expired code"))
}

func (s *magicCodeSteps) clickPasswordLoginToggle() {
	s.session.Click(q.Text("Sign in with password instead"))
	s.session.Sleep(300)
}

func (s *magicCodeSteps) assertPasswordFormVisible() {
	s.session.AssertVisible(q.Locator(`input[type="password"]`))
	s.session.AssertVisible(q.Text("Login"))
}

func (s *magicCodeSteps) clickMagicCodeToggle() {
	s.session.Click(q.Text("Sign in with email code instead"))
	s.session.Sleep(300)
}

func (s *magicCodeSteps) createInviteLink() string {
	inviteLink, err := models.FindInviteLinkByOrganizationID(s.session.OrgID.String())
	if err == nil {
		return inviteLink.Token.String()
	}
	inviteLink, err = models.CreateInviteLink(s.session.OrgID)
	require.NoError(s.t, err)
	return inviteLink.Token.String()
}

func (s *magicCodeSteps) followInviteLinkToLogin(token string) {
	s.session.Visit("/login?redirect=" + url.QueryEscape("/invite/"+token))
	s.session.Sleep(500)
}

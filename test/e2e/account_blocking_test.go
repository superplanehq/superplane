package e2e

import (
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

func TestAccountBlocking(t *testing.T) {
	t.Run("admin can block and unblock an account", func(t *testing.T) {
		steps := &accountBlockingSteps{t: t}
		steps.start()
		steps.createTargetAccountWithPassword()
		steps.promoteAdminAndLogin()
		steps.visitAccountsPage()
		steps.blockTargetAccount()
		steps.assertTargetAccountIsBlocked()
		steps.unblockTargetAccount()
		steps.assertTargetAccountIsUnblocked()
	})

	t.Run("blocked account cannot sign in with password", func(t *testing.T) {
		steps := &accountBlockingSteps{t: t}
		steps.start()
		steps.createTargetAccountWithPassword()
		steps.blockTargetInDatabase()
		steps.clearCookies()
		steps.visitLoginPage()
		steps.signInWithPassword(steps.targetEmail, steps.targetPassword)
		steps.assertAccountBlockedMessageVisible()
	})

	t.Run("blocked account does not receive a magic code", func(t *testing.T) {
		steps := &accountBlockingSteps{t: t}
		steps.start()
		steps.createTargetAccountWithPassword()
		steps.blockTargetInDatabase()
		steps.clearCookies()
		steps.visitLoginPage()
		steps.requestMagicCode(steps.targetEmail)
		steps.assertMagicCodeStepVisible()
		steps.assertNoMagicCodeStored(steps.targetEmail)
	})

	t.Run("blocked account cannot verify a magic code", func(t *testing.T) {
		steps := &accountBlockingSteps{t: t}
		steps.start()
		steps.createTargetAccountWithPassword()
		steps.blockTargetInDatabase()
		steps.clearCookies()
		steps.visitLoginPage()
		steps.requestMagicCode(steps.targetEmail)
		steps.insertKnownMagicCode(steps.targetEmail, "123456")
		steps.enterMagicCodeAndSubmit("123456")
		steps.assertAccountBlockedMessageVisible()
	})

	t.Run("unblocked account can sign in with password again", func(t *testing.T) {
		steps := &accountBlockingSteps{t: t}
		steps.start()
		steps.createTargetAccountWithPassword()
		steps.blockTargetInDatabase()
		steps.unblockTargetInDatabase()
		steps.clearCookies()
		steps.visitLoginPage()
		steps.signInWithPassword(steps.targetEmail, steps.targetPassword)
		steps.assertSignedInToOrganization()
	})
}

type accountBlockingSteps struct {
	t              *testing.T
	session        *session.TestSession
	targetAccount  *models.Account
	targetEmail    string
	targetPassword string
}

func (s *accountBlockingSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
}

func (s *accountBlockingSteps) createTargetAccountWithPassword() {
	s.targetEmail = support.RandomName("blocked") + "@superplane.local"
	s.targetPassword = "TestPassword123!"

	account, err := models.CreateAccount("Blocked Target", s.targetEmail)
	require.NoError(s.t, err)

	_, err = models.CreateUser(s.session.OrgID, account.ID, account.Email, account.Name)
	require.NoError(s.t, err)

	hash, err := crypto.HashPassword(s.targetPassword)
	require.NoError(s.t, err)
	_, err = models.CreateAccountPasswordAuth(account.ID, hash)
	require.NoError(s.t, err)

	s.targetAccount = account
}

func (s *accountBlockingSteps) promoteAdminAndLogin() {
	err := models.PromoteToInstallationAdmin(s.session.Account.ID.String())
	require.NoError(s.t, err)
	s.session.Login()
}

func (s *accountBlockingSteps) visitAccountsPage() {
	s.session.Visit("/admin/accounts")
	s.session.Sleep(500)
	s.session.AssertText(s.targetEmail)
}

func (s *accountBlockingSteps) blockTargetAccount() {
	s.clickAccountAction("Block")
	s.session.AssertText("Block Account")
	s.session.Click(q.Locator(`button:has-text("Block Account")`))
	s.session.Sleep(1000)
}

func (s *accountBlockingSteps) unblockTargetAccount() {
	s.clickAccountAction("Unblock")
	s.session.AssertText("Unblock Account")
	s.session.Click(q.Locator(`button:has-text("Unblock Account")`))
	s.session.Sleep(1000)
}

func (s *accountBlockingSteps) clickAccountAction(label string) {
	selector := `tr:has-text("` + s.targetEmail + `") button:has-text("` + label + `")`
	s.session.Click(q.Locator(selector))
	s.session.Sleep(300)
}

func (s *accountBlockingSteps) assertTargetAccountIsBlocked() {
	s.session.AssertVisible(q.Locator(`tr:has-text("` + s.targetEmail + `"):has-text("Blocked")`))

	account, err := models.FindAccountByEmail(s.targetEmail)
	require.NoError(s.t, err)
	assert.True(s.t, account.IsBlocked(), "account should be blocked after admin confirms")
}

func (s *accountBlockingSteps) assertTargetAccountIsUnblocked() {
	s.session.AssertVisible(q.Locator(`tr:has-text("` + s.targetEmail + `") button:has-text("Block")`))

	account, err := models.FindAccountByEmail(s.targetEmail)
	require.NoError(s.t, err)
	assert.False(s.t, account.IsBlocked(), "account should be unblocked after admin confirms")
}

func (s *accountBlockingSteps) blockTargetInDatabase() {
	require.NoError(s.t, s.targetAccount.Block(database.Conn(), time.Now()))
}

func (s *accountBlockingSteps) unblockTargetInDatabase() {
	require.NoError(s.t, s.targetAccount.Unblock(database.Conn()))
}

func (s *accountBlockingSteps) clearCookies() {
	err := s.session.Page().Context().ClearCookies()
	require.NoError(s.t, err)
}

func (s *accountBlockingSteps) visitLoginPage() {
	s.session.Visit("/login")
	s.session.Sleep(500)
}

func (s *accountBlockingSteps) signInWithPassword(email, password string) {
	s.session.Click(q.Text("Sign in with password instead"))
	s.session.Sleep(300)
	s.session.FillIn(q.Locator(`input[type="email"]`), email)
	s.session.FillIn(q.Locator(`input[type="password"]`), password)
	s.session.Click(q.Text("Login"))
	s.session.Sleep(1000)
}

func (s *accountBlockingSteps) requestMagicCode(email string) {
	s.session.FillIn(q.Locator(`input[type="email"]`), email)
	s.session.Click(q.Text("Continue with email"))
	s.session.Sleep(500)
}

func (s *accountBlockingSteps) assertMagicCodeStepVisible() {
	s.session.AssertVisible(q.Text("Check your email"))
}

func (s *accountBlockingSteps) assertNoMagicCodeStored(email string) {
	count, err := models.CountRecentMagicCodes(strings.ToLower(strings.TrimSpace(email)), time.Now().Add(-time.Hour))
	require.NoError(s.t, err)
	assert.Zero(s.t, count, "blocked accounts must not receive magic codes")
}

func (s *accountBlockingSteps) insertKnownMagicCode(email, code string) {
	codeHash := crypto.HashToken(code)
	expiresAt := time.Now().Add(10 * time.Minute)
	_, err := models.CreateAccountMagicCode(strings.ToLower(strings.TrimSpace(email)), codeHash, expiresAt)
	require.NoError(s.t, err)
}

func (s *accountBlockingSteps) enterMagicCodeAndSubmit(code string) {
	s.session.FillIn(q.Locator(`input[name="code"]`), code)
	s.session.Click(q.Text("Sign in"))
	s.session.Sleep(1000)
}

func (s *accountBlockingSteps) assertAccountBlockedMessageVisible() {
	s.session.AssertText(models.AccountBlockedMessage)
}

func (s *accountBlockingSteps) assertSignedInToOrganization() {
	currentURL := s.session.Page().URL()
	assert.Contains(s.t, currentURL, "/"+s.session.OrgSlug,
		"expected redirect to organization home, got %s", currentURL)
}

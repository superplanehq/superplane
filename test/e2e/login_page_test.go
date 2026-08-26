package e2e

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	pw "github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
)

func TestLoginPage(t *testing.T) {
	t.Run("login page should include redirect URL in auth links", func(t *testing.T) {
		steps := &TestLoginPageSteps{t: t}
		steps.Start()
		steps.VisitProtectedRandomURL()
		steps.AssertRedirectedToLoginWithRedirectParam()
		steps.AssertAuthLinksHaveRedirectParam()
	})

	t.Run("after login user should be redirected back original URL", func(t *testing.T) {
		steps := &TestLoginPageSteps{t: t}
		steps.Start()
		steps.VisitProtectedRandomURL()
		steps.AssertRedirectedToLoginWithRedirectParam()
		steps.LoginAndReturnToRedirectedURL()
		steps.session.AssertURLContains(steps.protectedURLPath)
	})

	t.Run("unauthenticated user sees login page", func(t *testing.T) {
		steps := &TestLoginPageSteps{t: t}
		steps.Start()
		steps.VisitLoginPage()
		steps.AssertLoginPageVisible()
	})

	t.Run("authenticated user gets redirected from login page", func(t *testing.T) {
		steps := &TestLoginPageSteps{t: t}
		steps.Start()
		steps.session.Login()
		steps.VisitLoginPage()
		steps.AssertRedirectedFromLoginPage()
	})

	t.Run("user with invalid token sees login page", func(t *testing.T) {
		steps := &TestLoginPageSteps{t: t}
		steps.Start()
		steps.SetInvalidAuthCookie()
		steps.VisitLoginPage()
		steps.AssertLoginPageVisible()
		steps.session.AssertURLContains("/login")
	})
}

type TestLoginPageSteps struct {
	t                *testing.T
	session          *session.TestSession
	protectedURLPath string
}

func (steps *TestLoginPageSteps) Start() {
	steps.session = ctx.NewSession(steps.t)
	steps.session.Start()
}

func (steps *TestLoginPageSteps) VisitLoginPage() {
	steps.session.Visit("/login")
	steps.session.Sleep(500)
}

func (steps *TestLoginPageSteps) VisitProtectedRandomURL() {
	steps.protectedURLPath = "/" + steps.session.OrgID.String() + "/apps/redirect-test"
	steps.session.Visit(steps.protectedURLPath)
}

func (steps *TestLoginPageSteps) AssertLoginPageVisible() {
	steps.session.AssertVisible(q.Text("Welcome to SuperPlane"))
}

func (steps *TestLoginPageSteps) AssertRedirectedToLoginWithRedirectParam() {
	steps.session.Sleep(500)
	steps.session.AssertURLContains("/login")
	steps.session.AssertURLContains("redirect=")

	encodedPath := url.QueryEscape(steps.protectedURLPath)
	steps.session.AssertURLContains(encodedPath)
}

func (steps *TestLoginPageSteps) AssertRedirectedFromLoginPage() {
	steps.session.Sleep(1000)
	currentURL := steps.session.Page().URL()
	assert.False(steps.t, strings.Contains(currentURL, "/login"), "expected to redirect away from login, got %s", currentURL)
}

func (steps *TestLoginPageSteps) AssertAuthLinksHaveRedirectParam() {
	links := steps.session.Page().Locator(`a[href^="/auth/"]`)
	count, err := links.Count()
	assert.NoError(steps.t, err)
	if count == 0 {
		return
	}

	for i := 0; i < count; i++ {
		href, hrefErr := links.Nth(i).GetAttribute("href")
		assert.NoError(steps.t, hrefErr)
		assert.NotEmpty(steps.t, href)
		assert.Contains(steps.t, href, "redirect=")
	}
}

func (steps *TestLoginPageSteps) LoginAndReturnToRedirectedURL() {
	steps.session.Login()

	currentURL := steps.session.Page().URL()
	parsedURL, err := url.Parse(currentURL)
	if err != nil {
		steps.t.Fatalf("failed to parse URL: %v", err)
	}

	redirectParam := parsedURL.Query().Get("redirect")
	if redirectParam == "" {
		steps.t.Fatal("redirect parameter not found in login URL")
	}

	steps.session.Visit("/login?redirect=" + url.QueryEscape(redirectParam))
	// Authenticated /login triggers a client-side redirect via Login.tsx; wait instead of a fixed sleep (flaky on CI).
	waitErr := steps.session.Page().WaitForURL("**"+steps.protectedURLPath+"**", pw.PageWaitForURLOptions{
		Timeout: pw.Float(30000),
	})
	require.NoError(steps.t, waitErr)
}

func (steps *TestLoginPageSteps) SetInvalidAuthCookie() {
	err := steps.session.Page().Context().AddCookies([]pw.OptionalCookie{{
		Name:     "account_token",
		Value:    "invalid-token",
		URL:      pw.String(steps.session.BaseURL + "/"),
		HttpOnly: pw.Bool(true),
	}})
	assert.NoError(steps.t, err)
}

const googleDevAccountEmail = "dev@superplane.local"

func TestGoogleSSONoAccountSignup(t *testing.T) {
	t.Run("google sign-in without an account asks to create one", func(t *testing.T) {
		steps := &googleSSONoAccountSteps{t: t}
		steps.start()
		steps.visitLoginPage()
		steps.capture("01-login")
		steps.clickContinueWithGoogle()
		steps.assertNoAccountPromptVisible()
		steps.capture("02-prompt")
		steps.clickCreateAccount()
		steps.assertAccountCreatedAndSignedIn()
		steps.capture("03-after-create")
	})

	t.Run("use a different account returns to sign in", func(t *testing.T) {
		steps := &googleSSONoAccountSteps{t: t}
		steps.start()
		steps.visitLoginPage()
		steps.clickContinueWithGoogle()
		steps.assertNoAccountPromptVisible()
		steps.clickUseADifferentAccount()
		steps.assertLoginPageVisible()
	})

	t.Run("existing google user signs in from the login page", func(t *testing.T) {
		steps := &googleSSONoAccountSteps{t: t}
		steps.start()
		steps.givenTheGoogleDevAccountExists()
		steps.visitLoginPage()
		steps.clickContinueWithGoogle()
		steps.assertLeftLoginPage()
	})
}

type googleSSONoAccountSteps struct {
	t       *testing.T
	session *session.TestSession
}

func (s *googleSSONoAccountSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
}

func (s *googleSSONoAccountSteps) visitLoginPage() {
	s.session.Visit("/login")
	s.session.AssertVisible(q.Text("Continue with Google"))
}

func (s *googleSSONoAccountSteps) clickContinueWithGoogle() {
	s.session.Click(q.Text("Continue with Google"))
}

func (s *googleSSONoAccountSteps) assertNoAccountPromptVisible() {
	s.session.AssertVisible(q.Text("No account found"))
	s.session.AssertVisible(q.Text("This Google account does not have a SuperPlane account."))
	s.session.AssertVisible(q.Text("Create an account to continue."))
	s.session.AssertVisible(q.Text("Create account"))
	s.session.AssertVisible(q.Text("Use a different account"))
	s.session.AssertURLContains("auth_error=signup_required")
	s.session.AssertURLContains("provider=google")
}

func (s *googleSSONoAccountSteps) clickCreateAccount() {
	s.session.Click(q.Text("Create account"))
}

func (s *googleSSONoAccountSteps) clickUseADifferentAccount() {
	s.session.Click(q.Text("Use a different account"))
}

func (s *googleSSONoAccountSteps) assertAccountCreatedAndSignedIn() {
	waitErr := s.session.Page().WaitForURL("**/welcome**", pw.PageWaitForURLOptions{
		Timeout: pw.Float(s.sessionTimeout()),
	})
	require.NoError(s.t, waitErr)

	account, err := models.FindAccountByEmail(googleDevAccountEmail)
	require.NoError(s.t, err)
	assert.Equal(s.t, googleDevAccountEmail, account.Email)
}

func (s *googleSSONoAccountSteps) assertLoginPageVisible() {
	s.session.AssertVisible(q.Text("Welcome to SuperPlane"))
	s.session.AssertVisible(q.Text("Continue with Google"))
	assert.NotContains(s.t, s.session.Page().URL(), "auth_error=signup_required")
}

func (s *googleSSONoAccountSteps) givenTheGoogleDevAccountExists() {
	_, err := models.CreateAccount("Dev User", googleDevAccountEmail)
	require.NoError(s.t, err)
}

func (s *googleSSONoAccountSteps) assertLeftLoginPage() {
	s.session.WaitUntilURLDoesNotContain("/login")
	currentURL := s.session.Page().URL()
	assert.NotContains(s.t, currentURL, "auth_error=signup_required")
}

func (s *googleSSONoAccountSteps) capture(name string) {
	s.session.Sleep(300)
	path := fmt.Sprintf("/app/tmp/screenshots/sso-no-account-%s.png", name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		s.t.Fatalf("screenshot dir %s: %v", path, err)
	}

	if _, err := s.session.Page().Screenshot(pw.PageScreenshotOptions{
		Path:     pw.String(path),
		FullPage: pw.Bool(true),
		Type:     pw.ScreenshotTypePng,
	}); err != nil {
		s.t.Fatalf("screenshot %s: %v", name, err)
	}
}

func (s *googleSSONoAccountSteps) sessionTimeout() float64 {
	return 15000
}

package e2e

import (
	"net/url"
	"strings"
	"testing"

	pw "github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/assert"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
)

func TestLoginPage(t *testing.T) {
	steps := &TestLoginPageSteps{t: t}

	t.Run("login page should include redirect URL in auth links", func(t *testing.T) {
		steps.Start()
		steps.VisitProtectedRandomURL()
		steps.AssertRedirectedToLoginWithRedirectParam()
		steps.AssertAuthLinksHaveRedirectParam()
	})

	t.Run("after login user should be redirected back original URL", func(t *testing.T) {
		steps.Start()
		steps.VisitProtectedRandomURL()
		steps.AssertRedirectedToLoginWithRedirectParam()
		steps.LoginAndReturnToRedirectedURL()
		steps.session.AssertURLContains(steps.protectedURLPath)
	})

	t.Run("unauthenticated user sees login page", func(t *testing.T) {
		steps.Start()
		steps.VisitLoginPage()
		steps.AssertLoginPageVisible()
	})

	t.Run("authenticated user gets redirected from login page", func(t *testing.T) {
		steps.Start()
		steps.session.Login()
		steps.VisitLoginPage()
		steps.AssertRedirectedFromLoginPage()
	})

	t.Run("user with invalid token sees login page", func(t *testing.T) {
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
	steps.protectedURLPath = "/" + steps.session.OrgID.String() + "/workflows/redirect-test"
	steps.session.Visit(steps.protectedURLPath)
}

func (steps *TestLoginPageSteps) AssertLoginPageVisible() {
	steps.session.AssertVisible(q.Text("Login"))
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
	steps.session.Sleep(500)
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

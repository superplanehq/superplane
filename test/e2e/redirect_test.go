package e2e

import (
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/test/e2e/session"
)

func TestRedirectAfterLogin(t *testing.T) {
	steps := &TestRedirectSteps{t: t}

	t.Run("login page should include redirect URL in auth provider links", func(t *testing.T) {
		steps.StartWithoutLogin()
		steps.VisitProtectedRandomURL()
		steps.AssertAuthProvidersHaveRedirectParam()
	})

	t.Run("after login user should be redirected back original URL", func(t *testing.T) {
		steps.StartWithoutLogin()
		steps.VisitProtectedRandomURL()
		steps.LoginAndVisitAuthCallback()
		steps.session.AssertURLContains(steps.protectedURLPath)
	})
}

type TestRedirectSteps struct {
	t                *testing.T
	session          *session.TestSession
	randomUUID       string
	protectedURLPath string
}

func (steps *TestRedirectSteps) StartWithoutLogin() {
	steps.session = ctx.NewSession(steps.t)
	steps.session.Start()

	steps.randomUUID = uuid.New().String()
	steps.protectedURLPath = "/" + steps.session.OrgID.String() + "/workflows/" + steps.randomUUID
}

func (steps *TestRedirectSteps) VisitProtectedRandomURL() {

	steps.session.Visit(steps.protectedURLPath)
}

func (steps *TestRedirectSteps) AssertRedirectedToLoginWithRedirectParam() {

	steps.session.Sleep(1000)

	steps.session.AssertURLContains("/login")
	steps.session.AssertURLContains("redirect=")

	encodedPath := url.QueryEscape(steps.protectedURLPath)
	steps.session.AssertURLContains(encodedPath)
}

func (steps *TestRedirectSteps) LoginAndVisitAuthCallback() {
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

	authCallbackURL := "/auth/github?redirect=" + url.QueryEscape(redirectParam)
	steps.session.Visit(authCallbackURL)
}

func (steps *TestRedirectSteps) AssertAuthProvidersHaveRedirectParam() {

	steps.session.Sleep(500)

	steps.session.AssertURLContains("/login")
	steps.session.AssertURLContains("redirect=")

	pageContent, err := steps.session.Page().Content()
	if err != nil {
		steps.t.Fatalf("failed to get page content: %v", err)
	}
	assert.Contains(steps.t, pageContent, "/auth/github")
	assert.Contains(steps.t, pageContent, "redirect=")
}

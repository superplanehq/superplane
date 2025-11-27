package e2e

import (
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/test/e2e/session"
)

func TestRedirectAfterLogin(t *testing.T) {
	steps := &TestRedirectSteps{t: t}

	t.Run("visiting protected url without auth should redirect to login with redirect param", func(t *testing.T) {
		steps.StartWithoutLogin()
		steps.VisitProtectedRandomURL()
		steps.AssertRedirectedToLoginWithRedirectParam()
	})

	t.Run("after login user should be redirected back to original URL", func(t *testing.T) {
		steps.StartWithoutLogin()
		steps.VisitProtectedRandomURL()
		steps.LoginAndVisitAuthCallback()
		steps.AssertRedirectedBackToOriginalURL()
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
	// Note: we don't call session.Login() here to test unauthenticated access

	// Generate a random UUID for the test URL
	steps.randomUUID = uuid.New().String()
	steps.protectedURLPath = "/" + steps.session.OrgID.String() + "/workflows/" + steps.randomUUID
}

func (steps *TestRedirectSteps) VisitProtectedRandomURL() {
	// Try to visit a protected URL with a random workflow UUID
	// This should trigger the middleware redirect since we're not authenticated
	steps.session.Visit(steps.protectedURLPath)
}

func (steps *TestRedirectSteps) AssertRedirectedToLoginWithRedirectParam() {
	// Wait a bit for the redirect to happen
	steps.session.Sleep(1000)

	// Check that we're on the login page with redirect parameter
	steps.session.AssertURLContains("/login")
	steps.session.AssertURLContains("redirect=")

	// The redirect parameter should contain the encoded original URL
	encodedPath := url.QueryEscape(steps.protectedURLPath)
	steps.session.AssertURLContains(encodedPath)
}

func (steps *TestRedirectSteps) LoginAndVisitAuthCallback() {
	// Add the auth cookie to simulate successful login
	steps.session.Login()

	// Extract the redirect parameter from the current URL for use in auth callback
	currentURL := steps.session.Page().URL()
	parsedURL, err := url.Parse(currentURL)
	if err != nil {
		steps.t.Fatalf("failed to parse URL: %v", err)
	}

	redirectParam := parsedURL.Query().Get("redirect")
	if redirectParam == "" {
		steps.t.Fatal("redirect parameter not found in login URL")
	}

	// Simulate visiting the auth callback with the redirect parameter
	// This should redirect back to the original URL
	authCallbackURL := "/auth/github?redirect=" + url.QueryEscape(redirectParam)
	steps.session.Visit(authCallbackURL)
}

func (steps *TestRedirectSteps) AssertRedirectedBackToOriginalURL() {
	// Wait for redirect back to original URL
	steps.session.Sleep(1000)

	// Verify we're back at the original protected URL
	steps.session.AssertURLContains(steps.session.OrgID.String())
	steps.session.AssertURLContains("workflows")
	steps.session.AssertURLContains(steps.randomUUID)
}
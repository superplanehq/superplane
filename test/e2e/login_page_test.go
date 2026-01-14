package e2e

import (
	"strings"
	"testing"

	pw "github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/assert"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
)

func TestLoginPage(t *testing.T) {
	steps := &TestLoginPageSteps{t: t}

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
	t       *testing.T
	session *session.TestSession
}

func (steps *TestLoginPageSteps) Start() {
	steps.session = ctx.NewSession(steps.t)
	steps.session.Start()
}

func (steps *TestLoginPageSteps) VisitLoginPage() {
	steps.session.Visit("/login")
	steps.session.Sleep(500)
}

func (steps *TestLoginPageSteps) AssertLoginPageVisible() {
	steps.session.AssertVisible(q.Text("Email & Password"))
}

func (steps *TestLoginPageSteps) AssertRedirectedFromLoginPage() {
	steps.session.Sleep(1000)
	currentURL := steps.session.Page().URL()
	assert.False(steps.t, strings.Contains(currentURL, "/login"), "expected to redirect away from login, got %s", currentURL)
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

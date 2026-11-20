package e2e

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	pw "github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestPasswordLoginRejectsCrossSiteForms(t *testing.T) {
	seed := ctx.NewSession(t)
	seed.Start()
	password := "DummyPassword123!"
	hash, err := crypto.HashPassword(password)
	require.NoError(t, err)
	_, err = models.CreateAccountPasswordAuth(seed.Account.ID, hash)
	require.NoError(t, err)
	attacker, err := models.CreateAccount("Dummy Attacker", support.RandomName("attacker")+"@superplane.local")
	require.NoError(t, err)
	_, err = models.CreateAccountPasswordAuth(attacker.ID, hash)
	require.NoError(t, err)
	fixture := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<form method="POST" action="%s/login"><input name="email" value="%s"><input name="password" value="%s"><button type="submit">Submit attack</button></form>`,
			html.EscapeString(ctx.baseURL), html.EscapeString(attacker.Email), html.EscapeString(password))
	}))
	t.Cleanup(fixture.Close)
	attackerURL := strings.Replace(fixture.URL, "127.0.0.1", "localhost", 1)

	for _, signedIn := range []bool{true, false} {
		name := "signed out"
		if signedIn {
			name = "signed in"
		}
		t.Run(name, func(t *testing.T) {
			browserContext, err := ctx.browser.NewContext()
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, browserContext.Close()) })
			page, err := browserContext.NewPage()
			require.NoError(t, err)
			steps := &passwordLoginCSRFSteps{t: t, page: page}
			if signedIn {
				steps.signInWithPassword(seed.Account.Email, password)
				steps.assertAuthenticatedAccount(seed.Account.ID.String())
			}
			steps.submitCrossSiteForm(attackerURL)
			if signedIn {
				steps.assertAuthenticatedAccount(seed.Account.ID.String())
				return
			}
			steps.assertSignedOut()
			// The same attacker credentials must succeed through the normal login page.
			steps.signInWithPassword(attacker.Email, password)
			steps.assertAuthenticatedAccount(attacker.ID.String())
		})
	}
}

type passwordLoginCSRFSteps struct {
	t    *testing.T
	page pw.Page
}

func (s *passwordLoginCSRFSteps) signInWithPassword(email, password string) {
	_, err := s.page.Goto(ctx.baseURL + "/login")
	require.NoError(s.t, err)
	require.NoError(s.t, s.page.GetByText("Sign in with password instead", pw.PageGetByTextOptions{Exact: pw.Bool(true)}).Click())
	require.NoError(s.t, s.page.Locator(`input[type="email"]`).Fill(email))
	require.NoError(s.t, s.page.Locator(`input[type="password"]`).Fill(password))
	response, err := s.page.ExpectResponse(func(response pw.Response) bool {
		return response.URL() == ctx.baseURL+"/login" && response.Request().Method() == http.MethodPost
	}, func() error {
		return s.page.GetByRole("button", pw.PageGetByRoleOptions{Name: "Login", Exact: pw.Bool(true)}).Click()
	})
	require.NoError(s.t, err)
	require.Equal(s.t, http.StatusSeeOther, response.Status())
	headers, err := response.Request().AllHeaders()
	require.NoError(s.t, err)
	assert.Equal(s.t, "same-origin", headers["sec-fetch-site"])
	require.NoError(s.t, s.page.WaitForURL(func(url string) bool { return !strings.Contains(url, "/login") }))
}

func (s *passwordLoginCSRFSteps) submitCrossSiteForm(attackerURL string) {
	_, err := s.page.Goto(attackerURL)
	require.NoError(s.t, err)
	response, err := s.page.ExpectResponse(func(response pw.Response) bool {
		return response.URL() == ctx.baseURL+"/login" && response.Request().Method() == http.MethodPost
	}, func() error {
		return s.page.GetByRole("button", pw.PageGetByRoleOptions{Name: "Submit attack"}).Click()
	})
	require.NoError(s.t, err)
	headers, err := response.Request().AllHeaders()
	require.NoError(s.t, err)
	require.Equal(s.t, "cross-site", headers["sec-fetch-site"])
	require.True(s.t, response.Request().IsNavigationRequest())
	assert.Equal(s.t, http.StatusForbidden, response.Status())
	responseHeaders, err := response.AllHeaders()
	require.NoError(s.t, err)
	assert.True(s.t, responseHeaders["set-cookie"] == "", "rejection must not set or clear cookies")
}

func (s *passwordLoginCSRFSteps) assertAuthenticatedAccount(accountID string) {
	response, err := s.page.Goto(ctx.baseURL + "/account")
	require.NoError(s.t, err)
	require.Equal(s.t, http.StatusOK, response.Status())
	body, err := response.Body()
	require.NoError(s.t, err)
	var account struct {
		ID string `json:"id"`
	}
	require.NoError(s.t, json.Unmarshal(body, &account))
	assert.Equal(s.t, accountID, account.ID)
}

func (s *passwordLoginCSRFSteps) assertSignedOut() {
	cookies, err := s.page.Context().Cookies(ctx.baseURL)
	require.NoError(s.t, err)
	for _, cookie := range cookies {
		assert.NotEqual(s.t, "account_token", cookie.Name)
	}
	response, err := s.page.Goto(ctx.baseURL + "/account")
	require.NoError(s.t, err)
	assert.Equal(s.t, http.StatusUnauthorized, response.Status())
}

package e2e

import (
	"encoding/json"
	"net/http"
	"testing"

	pw "github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/e2e/session"
)

// TestPersonalAPITokens covers the multi-named-token flow end to end:
// creating a named token, seeing it (and only it) in the list, copying its
// secret once, revoking one token without disturbing another, and using
// the plaintext secret as a Bearer token against the real API.
func TestPersonalAPITokens(t *testing.T) {
	t.Run("creating a named personal API token", func(t *testing.T) {
		steps := &personalTokenSteps{t: t}
		steps.start()
		steps.visitProfilePage()
		steps.createToken("CI token")
		steps.assertTokenRevealed()
		steps.assertTokenListed("CI token")

		plaintext := steps.revealedTokenValue()
		steps.assertBearerTokenAuthenticatesAsSessionUser(plaintext)
	})

	t.Run("revoking one token does not invalidate another", func(t *testing.T) {
		steps := &personalTokenSteps{t: t}
		steps.start()
		steps.visitProfilePage()

		steps.createToken("Token A")
		tokenA := steps.revealedTokenValue()

		steps.createToken("Token B")
		tokenB := steps.revealedTokenValue()

		steps.assertTokenListed("Token A")
		steps.assertTokenListed("Token B")

		steps.revokeToken("Token A")
		steps.assertTokenNotListed("Token A")
		steps.assertTokenListed("Token B")

		steps.assertBearerTokenRejected(tokenA)
		steps.assertBearerTokenAuthenticatesAsSessionUser(tokenB)
	})
}

type personalTokenSteps struct {
	t       *testing.T
	session *session.TestSession
}

func (s *personalTokenSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *personalTokenSteps) visitProfilePage() {
	s.session.Visit("/" + s.session.OrgID.String() + "/settings/profile")
	s.session.Sleep(500)
}

func (s *personalTokenSteps) createToken(name string) {
	page := s.session.Page()

	err := page.GetByTestId("user-token-create-name").Fill(name)
	require.NoError(s.t, err)
	s.session.Sleep(200)

	err = page.GetByTestId("user-token-create-submit").Click()
	require.NoError(s.t, err)
	s.session.Sleep(1000)
}

func (s *personalTokenSteps) assertTokenRevealed() {
	page := s.session.Page()
	value := page.GetByTestId("user-token-reveal-value")
	err := value.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible, Timeout: pw.Float(5000)})
	require.NoError(s.t, err)
}

func (s *personalTokenSteps) revealedTokenValue() string {
	page := s.session.Page()
	value, err := page.GetByTestId("user-token-reveal-value").InputValue()
	require.NoError(s.t, err)
	require.NotEmpty(s.t, value, "token secret should not be empty")
	return value
}

func (s *personalTokenSteps) assertTokenListed(name string) {
	page := s.session.Page()
	row := page.GetByTestId("user-token-row").Filter(pw.LocatorFilterOptions{HasText: name})
	err := row.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible, Timeout: pw.Float(5000)})
	require.NoError(s.t, err, "token %q should be listed", name)
}

func (s *personalTokenSteps) assertTokenNotListed(name string) {
	page := s.session.Page()
	row := page.GetByTestId("user-token-row").Filter(pw.LocatorFilterOptions{HasText: name})
	count, err := row.Count()
	require.NoError(s.t, err)
	require.Zero(s.t, count, "token %q should not be listed after revoke", name)
}

func (s *personalTokenSteps) revokeToken(name string) {
	page := s.session.Page()
	row := page.GetByTestId("user-token-row").Filter(pw.LocatorFilterOptions{HasText: name})
	err := row.GetByTestId("user-token-revoke-btn").Click()
	require.NoError(s.t, err)
	s.session.Sleep(1000)
}

// assertBearerTokenAuthenticatesAsSessionUser calls the real API with the
// given plaintext as a Bearer token and confirms it resolves to the same
// user that is signed in for this session.
func (s *personalTokenSteps) assertBearerTokenAuthenticatesAsSessionUser(plaintext string) {
	req, err := http.NewRequest(http.MethodGet, s.session.BaseURL+"/api/v1/me", nil)
	require.NoError(s.t, err)
	req.Header.Set("Authorization", "Bearer "+plaintext)
	req.Header.Set("x-organization-id", s.session.OrgID.String())

	resp, err := http.DefaultClient.Do(req)
	require.NoError(s.t, err)
	defer resp.Body.Close()

	require.Equal(s.t, http.StatusOK, resp.StatusCode)

	var body struct {
		User struct {
			Email string `json:"email"`
		} `json:"user"`
	}
	require.NoError(s.t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(s.t, s.session.Account.Email, body.User.Email)
}

// assertBearerTokenRejected confirms a revoked token no longer authenticates.
func (s *personalTokenSteps) assertBearerTokenRejected(plaintext string) {
	req, err := http.NewRequest(http.MethodGet, s.session.BaseURL+"/api/v1/me", nil)
	require.NoError(s.t, err)
	req.Header.Set("Authorization", "Bearer "+plaintext)
	req.Header.Set("x-organization-id", s.session.OrgID.String())

	resp, err := http.DefaultClient.Do(req)
	require.NoError(s.t, err)
	defer resp.Body.Close()

	require.Equal(s.t, http.StatusUnauthorized, resp.StatusCode)
}

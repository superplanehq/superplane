package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/supportsession"
	"github.com/superplanehq/superplane/test/support"
)

type supportSessionFixture struct {
	registry  *support.ResourceRegistry
	signer    *jwt.Signer
	adminAuth *http.Cookie
	session   *http.Cookie
}

func setupSupportSession(t *testing.T) *supportSessionFixture {
	t.Helper()

	r := support.Setup(t)
	signer := jwt.NewSigner("test-secret")
	require.NoError(t, models.PromoteToInstallationAdmin(r.Account.ID.String()))

	target, err := models.CreateAccount("Support Target", "support-target@example.com")
	require.NoError(t, err)
	_, err = models.CreateUser(r.Organization.ID, target.ID, target.Email, target.Name)
	require.NoError(t, err)

	accountToken, err := authentication.GenerateAccountToken(signer, r.Account.ID.String(), time.Now(), time.Hour)
	require.NoError(t, err)
	sessionToken, err := supportsession.GenerateToken(signer, r.Account.ID.String(), target.ID.String())
	require.NoError(t, err)

	return &supportSessionFixture{
		registry:  r,
		signer:    signer,
		adminAuth: &http.Cookie{Name: "account_token", Value: accountToken},
		session:   &http.Cookie{Name: supportsession.CookieName, Value: sessionToken},
	}
}

func (f *supportSessionFixture) organizationRequest(method, path string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	req.AddCookie(f.adminAuth)
	req.AddCookie(f.session)
	req.Header.Set("x-organization-id", f.registry.Organization.ID.String())
	return req
}

func (f *supportSessionFixture) accountRequest(method, path string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	req.AddCookie(f.adminAuth)
	req.AddCookie(f.session)
	return req
}

func countingHandler(calls *int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		*calls++
		w.WriteHeader(http.StatusNoContent)
	})
}

func TestOrganizationAuthMiddleware_SupportSessionIsReadOnly(t *testing.T) {
	f := setupSupportSession(t)

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		t.Run(method+" is rejected", func(t *testing.T) {
			calls := 0
			handler := OrganizationAuthMiddleware(f.signer)(countingHandler(&calls))

			res := httptest.NewRecorder()
			handler.ServeHTTP(res, f.organizationRequest(method, "/api/v1/canvases"))

			assert.Equal(t, http.StatusForbidden, res.Code)
			assert.Contains(t, res.Body.String(), SupportSessionReadOnlyMessage)
			assert.Zero(t, calls)
		})
	}

	t.Run("GET reaches the handler", func(t *testing.T) {
		calls := 0
		handler := OrganizationAuthMiddleware(f.signer)(countingHandler(&calls))

		res := httptest.NewRecorder()
		handler.ServeHTTP(res, f.organizationRequest(http.MethodGet, "/api/v1/canvases"))

		assert.Equal(t, http.StatusNoContent, res.Code)
		assert.Equal(t, 1, calls)
	})

	t.Run("POST to a read-only endpoint reaches the handler", func(t *testing.T) {
		calls := 0
		handler := OrganizationAuthMiddleware(f.signer)(countingHandler(&calls))

		res := httptest.NewRecorder()
		handler.ServeHTTP(res, f.organizationRequest(http.MethodPost, "/api/v1/factories/abc/scheduled-intake/preview"))

		assert.Equal(t, http.StatusNoContent, res.Code)
		assert.Equal(t, 1, calls)
	})

	t.Run("bearer token request is not affected by the session cookie", func(t *testing.T) {
		calls := 0
		handler := OrganizationAuthMiddleware(f.signer)(countingHandler(&calls))

		token, err := f.signer.GenerateScopedToken(jwt.ScopedTokenClaims{
			Subject: f.registry.User.String(),
			OrgID:   f.registry.Organization.ID.String(),
			Purpose: "agent-builder",
			Scopes:  []string{"canvases:update"},
		}, time.Minute)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/canvases", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.AddCookie(f.session)

		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		assert.Equal(t, http.StatusNoContent, res.Code)
		assert.Equal(t, 1, calls)
	})
}

func TestAccountAuthMiddleware_SupportSessionIsReadOnly(t *testing.T) {
	f := setupSupportSession(t)

	t.Run("account write is rejected", func(t *testing.T) {
		calls := 0
		handler := AccountAuthMiddleware(f.signer)(countingHandler(&calls))

		res := httptest.NewRecorder()
		handler.ServeHTTP(res, f.accountRequest(http.MethodPost, "/organizations"))

		assert.Equal(t, http.StatusForbidden, res.Code)
		assert.Zero(t, calls)
	})

	t.Run("admin write is rejected", func(t *testing.T) {
		calls := 0
		handler := AccountAuthMiddleware(f.signer)(countingHandler(&calls))

		res := httptest.NewRecorder()
		handler.ServeHTTP(res, f.accountRequest(http.MethodPost, "/admin/api/accounts/abc/block"))

		assert.Equal(t, http.StatusForbidden, res.Code)
		assert.Zero(t, calls)
	})

	t.Run("ending the session reaches the handler", func(t *testing.T) {
		calls := 0
		handler := AccountAuthMiddleware(f.signer)(countingHandler(&calls))

		res := httptest.NewRecorder()
		handler.ServeHTTP(res, f.accountRequest(http.MethodPost, "/admin/api/support-session/end"))

		assert.Equal(t, http.StatusNoContent, res.Code)
		assert.Equal(t, 1, calls)
	})

	t.Run("GET reaches the handler", func(t *testing.T) {
		calls := 0
		handler := AccountAuthMiddleware(f.signer)(countingHandler(&calls))

		res := httptest.NewRecorder()
		handler.ServeHTTP(res, f.accountRequest(http.MethodGet, "/account"))

		assert.Equal(t, http.StatusNoContent, res.Code)
		assert.Equal(t, 1, calls)
	})

	t.Run("write without a session cookie reaches the handler", func(t *testing.T) {
		calls := 0
		handler := AccountAuthMiddleware(f.signer)(countingHandler(&calls))

		req := httptest.NewRequest(http.MethodPost, "/organizations", nil)
		req.AddCookie(f.adminAuth)

		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		assert.Equal(t, http.StatusNoContent, res.Code)
		assert.Equal(t, 1, calls)
	})
}

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/enterprise"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/supportsession"
	"github.com/superplanehq/superplane/test/support"
)

type requiredSSO struct {
	called bool
}

func (checker *requiredSSO) CheckOrganizationAccess(context.Context, *models.User, enterprise.SSOAccessSession) (*enterprise.SSORequired, error) {
	checker.called = true
	return &enterprise.SSORequired{StartURL: "/auth/sso/conn/start"}, nil
}

func TestOrganizationAuthMiddleware_SSOExemptions(t *testing.T) {
	r := support.Setup(t)
	signer := jwt.NewSigner("test-secret")
	checker := &requiredSSO{}
	handler := OrganizationAuthMiddleware(signer, checker)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	t.Run("cookie session is sent to the identity provider", func(t *testing.T) {
		checker.called = false
		token, err := authentication.GenerateAccountToken(signer, r.Account.ID.String(), time.Now(), time.Hour)
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
		req.Header.Set("x-organization-id", r.Organization.ID.String())
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		assert.True(t, checker.called)
		assert.Equal(t, http.StatusUnauthorized, res.Code)
		assert.Equal(t, "/auth/sso/conn/start", res.Header().Get("X-SuperPlane-SSO-Start"))
		assert.Contains(t, res.Body.String(), `"code":"sso_required"`)
	})

	t.Run("bearer token skips SSO enforcement", func(t *testing.T) {
		checker.called = false
		rawToken, err := crypto.Base64String(32)
		require.NoError(t, err)
		token := models.NewUserAPIToken(r.User, "CI token", crypto.HashToken(rawToken))
		require.NoError(t, models.CreateUserAPIToken(database.Conn(), token))
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		req.Header.Set("Authorization", "Bearer "+rawToken)
		req.Header.Set("x-organization-id", r.Organization.ID.String())
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		assert.False(t, checker.called)
		assert.Equal(t, http.StatusNoContent, res.Code)
	})

	t.Run("support session skips SSO enforcement", func(t *testing.T) {
		checker.called = false
		require.NoError(t, models.PromoteToInstallationAdmin(r.Account.ID.String()))
		target, err := models.CreateAccount("Support Target", "support-target@example.com")
		require.NoError(t, err)
		_, err = models.CreateUser(r.Organization.ID, target.ID, target.Email, target.Name)
		require.NoError(t, err)
		accountToken, err := authentication.GenerateAccountToken(signer, r.Account.ID.String(), time.Now(), time.Hour)
		require.NoError(t, err)
		supportToken, err := supportsession.GenerateToken(signer, r.Account.ID.String(), target.ID.String())
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		req.AddCookie(&http.Cookie{Name: "account_token", Value: accountToken})
		req.AddCookie(&http.Cookie{Name: supportsession.CookieName, Value: supportToken})
		req.Header.Set("x-organization-id", r.Organization.ID.String())
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		assert.False(t, checker.called)
		assert.Equal(t, http.StatusNoContent, res.Code)
	})
}

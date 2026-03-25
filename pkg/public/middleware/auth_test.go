package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/test/support"
)

func TestOrganizationAuthMiddleware_CookieAuthErrors(t *testing.T) {
	r := support.Setup(t)
	signer := jwt.NewSigner("test-secret")

	token, err := signer.Generate(r.Account.ID.String(), time.Hour)
	require.NoError(t, err)

	handler := OrganizationAuthMiddleware(signer)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	t.Run("missing account cookie returns unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		req.Header.Set("x-organization-id", r.Organization.ID.String())

		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		assert.Equal(t, http.StatusUnauthorized, res.Code)
	})

	t.Run("missing organization id returns not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		req.AddCookie(&http.Cookie{Name: "account_token", Value: token})

		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		assert.Equal(t, http.StatusNotFound, res.Code)
	})

	t.Run("organization without matching user returns not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
		req.Header.Set("x-organization-id", uuid.NewString())

		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		assert.Equal(t, http.StatusNotFound, res.Code)
	})

	t.Run("valid cookie and organization reaches next handler", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
		req.Header.Set("x-organization-id", r.Organization.ID.String())

		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		assert.Equal(t, http.StatusNoContent, res.Code)
	})
}

func TestOrganizationAuthMiddleware_BearerAuth(t *testing.T) {
	r := support.Setup(t)
	signer := jwt.NewSigner("test-secret")

	handler := OrganizationAuthMiddleware(signer)(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		user, ok := GetUserFromContext(req.Context())
		require.True(t, ok)
		assert.Equal(t, r.User, user.ID)

		w.WriteHeader(http.StatusNoContent)
	}))

	t.Run("scoped token reaches next handler", func(t *testing.T) {
		token, err := signer.GenerateScopedToken(jwt.ScopedTokenClaims{
			Subject: r.User.String(),
			OrgID:   r.Organization.ID.String(),
			Purpose: "agent-builder",
			Scopes:  []string{"canvases:read"},
		}, time.Minute)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		assert.Equal(t, http.StatusNoContent, res.Code)
	})

	t.Run("scoped token with wrong org returns unauthorized", func(t *testing.T) {
		token, err := signer.GenerateScopedToken(jwt.ScopedTokenClaims{
			Subject: r.User.String(),
			OrgID:   uuid.NewString(),
			Purpose: "agent-builder",
			Scopes:  []string{"canvases:read"},
		}, time.Minute)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		assert.Equal(t, http.StatusUnauthorized, res.Code)
	})

	t.Run("existing api token still reaches next handler", func(t *testing.T) {
		rawToken, err := crypto.Base64String(32)
		require.NoError(t, err)
		require.NoError(t, r.UserModel.UpdateTokenHash(crypto.HashToken(rawToken)))

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		req.Header.Set("Authorization", "Bearer "+rawToken)

		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		assert.Equal(t, http.StatusNoContent, res.Code)
	})
}

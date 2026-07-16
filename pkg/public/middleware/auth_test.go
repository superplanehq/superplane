package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jwtLib "github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestOrganizationAuthMiddleware_CookieAuthErrors(t *testing.T) {
	r := support.Setup(t)
	signer := jwt.NewSigner("test-secret")

	token, err := authentication.GenerateAccountToken(signer, r.Account.ID.String(), time.Now(), time.Hour)
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

	t.Run("scoped token issued before password change is rejected", func(t *testing.T) {
		token, err := signer.GenerateScopedToken(jwt.ScopedTokenClaims{
			Subject: r.User.String(),
			OrgID:   r.Organization.ID.String(),
			Purpose: "agent-builder",
			Scopes:  []string{"canvases:read"},
		}, time.Minute)
		require.NoError(t, err)

		// Bump the account's password_changed_at to a moment after the
		// token was issued, simulating a password rotation.
		require.NoError(t, r.Account.MarkPasswordChangedInTransaction(database.Conn(), time.Now().Add(time.Minute)))

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		assert.Equal(t, http.StatusUnauthorized, res.Code)
	})

	t.Run("expired API key api token is rejected", func(t *testing.T) {
		rawToken, err := crypto.Base64String(32)
		require.NoError(t, err)
		expiredAt := time.Now().Add(-time.Minute)
		apiKey := &models.User{
			ID:              uuid.New(),
			OrganizationID:  r.Organization.ID,
			Name:            "expired-bot",
			Type:            models.UserTypeAPIKey,
			TokenHash:       crypto.HashToken(rawToken),
			APIKeyExpiresAt: &expiredAt,
			APIKeyCanvasIDs: datatypes.NewJSONSlice([]string{}),
		}
		require.NoError(t, database.Conn().Create(apiKey).Error)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		req.Header.Set("Authorization", "Bearer "+rawToken)

		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		assert.Equal(t, http.StatusUnauthorized, res.Code)
	})
}

func TestAccountAuthMiddleware_FreshnessCheck(t *testing.T) {
	r := support.Setup(t)
	signer := jwt.NewSigner("test-secret")

	handler := AccountAuthMiddleware(signer)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	t.Run("token without ses claim is rejected", func(t *testing.T) {
		legacyToken, err := signer.Generate(r.Account.ID.String(), time.Hour)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/account", nil)
		req.AddCookie(&http.Cookie{Name: "account_token", Value: legacyToken})

		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		assert.Equal(t, http.StatusUnauthorized, res.Code)
	})

	t.Run("cookie issued before password change is rejected", func(t *testing.T) {
		oldToken, err := authentication.GenerateAccountToken(signer, r.Account.ID.String(), time.Now(), time.Hour)
		require.NoError(t, err)

		// Bump password_changed_at so the cookie's iat is now stale.
		require.NoError(t, r.Account.MarkPasswordChangedInTransaction(database.Conn(), time.Now().Add(time.Minute)))

		req := httptest.NewRequest(http.MethodGet, "/account", nil)
		req.AddCookie(&http.Cookie{Name: "account_token", Value: oldToken})

		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		assert.Equal(t, http.StatusUnauthorized, res.Code)
	})

	t.Run("cookie issued after password change is accepted", func(t *testing.T) {
		// Roll password_changed_at into the past so a freshly-issued
		// cookie's iat is newer than it.
		require.NoError(t, r.Account.MarkPasswordChangedInTransaction(database.Conn(), time.Now().Add(-time.Hour)))

		freshToken, err := authentication.GenerateAccountToken(signer, r.Account.ID.String(), time.Now(), time.Hour)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/account", nil)
		req.AddCookie(&http.Cookie{Name: "account_token", Value: freshToken})

		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		assert.Equal(t, http.StatusNoContent, res.Code)
	})
}

func TestOrganizationAuthMiddleware_RefreshesSessionOnActivity(t *testing.T) {
	r := support.Setup(t)
	signer := jwt.NewSigner("test-secret")

	issuedAt := time.Now().Add(-2 * time.Hour)
	sessionStart := issuedAt
	oldToken := mintTestAccountToken(t, signer, r.Account.ID.String(), issuedAt, sessionStart, 24*time.Hour)

	handler := OrganizationAuthMiddleware(signer)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/canvases", nil)
	req.AddCookie(&http.Cookie{Name: "account_token", Value: oldToken})
	req.Header.Set("x-organization-id", r.Organization.ID.String())

	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	require.Equal(t, http.StatusNoContent, res.Code)

	cookies := res.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, "account_token", cookies[0].Name)
	assert.NotEqual(t, oldToken, cookies[0].Value)
}

func mintTestAccountToken(t *testing.T, signer *jwt.Signer, accountID string, issuedAt, sessionStart time.Time, ttl time.Duration) string {
	t.Helper()

	token := jwtLib.NewWithClaims(jwtLib.SigningMethodHS256, jwtLib.MapClaims{
		"sub": accountID,
		"iat": issuedAt.Unix(),
		"nbf": issuedAt.Unix(),
		"exp": issuedAt.Add(ttl).Unix(),
		"ses": fmt.Sprintf("%d", sessionStart.Unix()),
	})

	tokenString, err := token.SignedString([]byte(signer.Secret))
	require.NoError(t, err)

	return tokenString
}

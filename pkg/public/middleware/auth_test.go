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
	"github.com/superplanehq/superplane/pkg/supportsession"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestOrganizationAuthMiddleware_CookieAuthErrors(t *testing.T) {
	r := support.Setup(t)
	signer := jwt.NewSigner("test-secret")

	token, err := authentication.GenerateAccountToken(signer, r.Account.ID.String(), time.Now(), time.Hour, r.Account.SessionGeneration)
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

	t.Run("organization slug in header resolves to the matching user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
		req.Header.Set("x-organization-id", r.Organization.Slug)

		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		assert.Equal(t, http.StatusNoContent, res.Code)
	})

	t.Run("unknown organization slug returns not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
		req.Header.Set("x-organization-id", "does-not-exist-slug")

		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		assert.Equal(t, http.StatusNotFound, res.Code)
	})

	t.Run("blocked account cookie returns contact-support message", func(t *testing.T) {
		require.NoError(t, r.Account.Block(database.Conn(), time.Now()))

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
		req.Header.Set("x-organization-id", r.Organization.ID.String())

		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		assert.Equal(t, http.StatusForbidden, res.Code)
		assert.Contains(t, res.Body.String(), models.AccountBlockedMessage)
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
			SessionGeneration: &r.Account.SessionGeneration,
			Subject:           r.User.String(),
			OrgID:             r.Organization.ID.String(),
			Purpose:           "test-scope",
			Scopes:            []string{"canvases:read"},
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
			SessionGeneration: &r.Account.SessionGeneration,
			Subject:           r.User.String(),
			OrgID:             uuid.NewString(),
			Purpose:           "test-scope",
			Scopes:            []string{"canvases:read"},
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
			SessionGeneration: &r.Account.SessionGeneration,
			Subject:           r.User.String(),
			OrgID:             r.Organization.ID.String(),
			Purpose:           "test-scope",
			Scopes:            []string{"canvases:read"},
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

	t.Run("scoped token for blocked account returns contact-support message", func(t *testing.T) {
		token, err := signer.GenerateScopedToken(jwt.ScopedTokenClaims{
			SessionGeneration: &r.Account.SessionGeneration,
			Subject:           r.User.String(),
			OrgID:             r.Organization.ID.String(),
			Purpose:           "test-scope",
			Scopes:            []string{"canvases:read"},
		}, time.Minute)
		require.NoError(t, err)
		require.NoError(t, r.Account.Block(database.Conn(), time.Now()))

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		assert.Equal(t, http.StatusForbidden, res.Code)
		assert.Contains(t, res.Body.String(), models.AccountBlockedMessage)
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

	t.Run("personal token for blocked account is rejected", func(t *testing.T) {
		rawToken, err := crypto.Base64String(32)
		require.NoError(t, err)
		require.NoError(t, r.UserModel.UpdateTokenHash(crypto.HashToken(rawToken)))
		require.NoError(t, r.Account.Block(database.Conn(), time.Now()))

		// Re-set hash after block cleared it so we exercise the blocked-account check.
		require.NoError(t, r.UserModel.UpdateTokenHash(crypto.HashToken(rawToken)))

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		req.Header.Set("Authorization", "Bearer "+rawToken)

		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		assert.Equal(t, http.StatusForbidden, res.Code)
		assert.Contains(t, res.Body.String(), models.AccountBlockedMessage)
	})
}

func TestOrganizationAuthMiddleware_PersonalAPIToken(t *testing.T) {
	r := support.Setup(t)
	signer := jwt.NewSigner("test-secret")

	handler := OrganizationAuthMiddleware(signer)(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		user, ok := GetUserFromContext(req.Context())
		require.True(t, ok)
		assert.Equal(t, r.User, user.ID)

		w.WriteHeader(http.StatusNoContent)
	}))

	t.Run("valid personal token authenticates as the owning user", func(t *testing.T) {
		rawToken, err := crypto.Base64String(32)
		require.NoError(t, err)
		token := models.NewUserAPIToken(r.User, "CI token", crypto.HashToken(rawToken))
		require.NoError(t, models.CreateUserAPIToken(database.Conn(), token))

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		req.Header.Set("Authorization", "Bearer "+rawToken)

		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		assert.Equal(t, http.StatusNoContent, res.Code)

		found, err := models.FindUserAPITokenByHash(database.Conn(), crypto.HashToken(rawToken))
		require.NoError(t, err)
		assert.NotNil(t, found.LastUsedAt, "last_used_at should advance on successful auth")
	})

	t.Run("revoked personal token is rejected", func(t *testing.T) {
		rawToken, err := crypto.Base64String(32)
		require.NoError(t, err)
		token := models.NewUserAPIToken(r.User, "Revoked token", crypto.HashToken(rawToken))
		require.NoError(t, models.CreateUserAPIToken(database.Conn(), token))
		require.NoError(t, token.HardDelete(database.Conn()))

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		req.Header.Set("Authorization", "Bearer "+rawToken)

		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		assert.Equal(t, http.StatusUnauthorized, res.Code)
	})

	t.Run("revoking one personal token does not invalidate the others", func(t *testing.T) {
		rawA, err := crypto.Base64String(32)
		require.NoError(t, err)
		tokenA := models.NewUserAPIToken(r.User, "Token A", crypto.HashToken(rawA))
		require.NoError(t, models.CreateUserAPIToken(database.Conn(), tokenA))

		rawB, err := crypto.Base64String(32)
		require.NoError(t, err)
		tokenB := models.NewUserAPIToken(r.User, "Token B", crypto.HashToken(rawB))
		require.NoError(t, models.CreateUserAPIToken(database.Conn(), tokenB))

		require.NoError(t, tokenA.HardDelete(database.Conn()))

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		req.Header.Set("Authorization", "Bearer "+rawB)

		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		assert.Equal(t, http.StatusNoContent, res.Code)
	})

	t.Run("blocking the account revokes the personal token", func(t *testing.T) {
		rawToken, err := crypto.Base64String(32)
		require.NoError(t, err)
		token := models.NewUserAPIToken(r.User, "Blocked account token", crypto.HashToken(rawToken))
		require.NoError(t, models.CreateUserAPIToken(database.Conn(), token))

		// Account.Block hard-deletes every personal token for the account,
		// so the token stops authenticating immediately.
		require.NoError(t, r.Account.Block(database.Conn(), time.Now()))

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		req.Header.Set("Authorization", "Bearer "+rawToken)

		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		assert.Equal(t, http.StatusUnauthorized, res.Code)
	})

	t.Run("personal token is rejected as a defense-in-depth check for a blocked account", func(t *testing.T) {
		// Simulate a row that somehow survived a block (defense in depth):
		// block the account directly, without going through Account.Block,
		// so the token row is not deleted, then confirm auth still rejects it.
		rawToken, err := crypto.Base64String(32)
		require.NoError(t, err)
		token := models.NewUserAPIToken(r.User, "Surviving token", crypto.HashToken(rawToken))
		require.NoError(t, models.CreateUserAPIToken(database.Conn(), token))

		now := time.Now()
		require.NoError(t, database.Conn().Model(&models.Account{}).
			Where("id = ?", r.Account.ID).
			Update("blocked_at", now).Error)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		req.Header.Set("Authorization", "Bearer "+rawToken)

		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		assert.Equal(t, http.StatusForbidden, res.Code)
		assert.Contains(t, res.Body.String(), models.AccountBlockedMessage)
	})
}

func TestOrganizationAuthMiddleware_APIKeyWithDeletedCreator(t *testing.T) {
	r := support.Setup(t)
	signer := jwt.NewSigner("test-secret")
	rawToken, err := crypto.Base64String(32)
	require.NoError(t, err)

	description := "key owned by a removed member"
	apiKey, err := models.CreateAPIKey(
		database.Conn(),
		r.Organization.ID,
		"removed-member-key",
		&description,
		r.User,
		nil,
		nil,
	)
	require.NoError(t, err)
	require.NoError(t, apiKey.UpdateTokenHash(crypto.HashToken(rawToken)))
	require.NoError(t, r.UserModel.Delete())

	handler := OrganizationAuthMiddleware(signer)(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		user, ok := GetUserFromContext(req.Context())
		require.True(t, ok)
		assert.Equal(t, apiKey.ID, user.ID)
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+rawToken)

	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	assert.Equal(t, http.StatusNoContent, res.Code)
}

func TestOrganizationAuthMiddleware_BlockedSupportSessionTargetFallsBackToAdmin(t *testing.T) {
	r := support.Setup(t)
	signer := jwt.NewSigner("test-secret")
	require.NoError(t, models.PromoteToInstallationAdmin(r.Account.ID.String()))

	targetAccount, err := models.CreateAccount("Blocked Target", "blocked-target@example.com")
	require.NoError(t, err)
	_, err = models.CreateUser(r.Organization.ID, targetAccount.ID, targetAccount.Email, targetAccount.Name)
	require.NoError(t, err)

	accountToken, err := authentication.GenerateAccountToken(signer, r.Account.ID.String(), time.Now(), time.Hour, r.Account.SessionGeneration)
	require.NoError(t, err)
	supportSessionToken, err := supportsession.GenerateToken(signer, r.Account.ID.String(), targetAccount.ID.String())
	require.NoError(t, err)
	require.NoError(t, targetAccount.Block(database.Conn(), time.Now()))

	handler := OrganizationAuthMiddleware(signer)(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		user, ok := GetUserFromContext(req.Context())
		require.True(t, ok)
		assert.Equal(t, r.User, user.ID)
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.AddCookie(&http.Cookie{Name: "account_token", Value: accountToken})
	req.AddCookie(&http.Cookie{Name: "support_session_token", Value: supportSessionToken})
	req.Header.Set("x-organization-id", r.Organization.ID.String())

	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	assert.Equal(t, http.StatusNoContent, res.Code)
	assertSupportSessionCookieCleared(t, res)
}

func TestAccountAuthMiddleware_BlockedSupportSessionTargetClearsCookie(t *testing.T) {
	r := support.Setup(t)
	signer := jwt.NewSigner("test-secret")
	require.NoError(t, models.PromoteToInstallationAdmin(r.Account.ID.String()))

	targetAccount, err := models.CreateAccount("Blocked Target", "blocked-account-target@example.com")
	require.NoError(t, err)

	accountToken, err := authentication.GenerateAccountToken(signer, r.Account.ID.String(), time.Now(), time.Hour, r.Account.SessionGeneration)
	require.NoError(t, err)
	supportSessionToken, err := supportsession.GenerateToken(signer, r.Account.ID.String(), targetAccount.ID.String())
	require.NoError(t, err)
	require.NoError(t, targetAccount.Block(database.Conn(), time.Now()))

	handler := AccountAuthMiddleware(signer)(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		account, ok := GetEffectiveAccountFromContext(req.Context())
		require.True(t, ok)
		assert.Equal(t, r.Account.ID, account.ID)
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/account", nil)
	req.AddCookie(&http.Cookie{Name: "account_token", Value: accountToken})
	req.AddCookie(&http.Cookie{Name: "support_session_token", Value: supportSessionToken})

	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	assert.Equal(t, http.StatusNoContent, res.Code)
	assertSupportSessionCookieCleared(t, res)
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
		oldToken, err := authentication.GenerateAccountToken(signer, r.Account.ID.String(), time.Now(), time.Hour, r.Account.SessionGeneration)
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

		freshToken, err := authentication.GenerateAccountToken(signer, r.Account.ID.String(), time.Now(), time.Hour, r.Account.SessionGeneration)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/account", nil)
		req.AddCookie(&http.Cookie{Name: "account_token", Value: freshToken})

		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		assert.Equal(t, http.StatusNoContent, res.Code)
	})

	t.Run("blocked account cookie is rejected with contact-support message", func(t *testing.T) {
		token, err := authentication.GenerateAccountToken(signer, r.Account.ID.String(), time.Now(), time.Hour, r.Account.SessionGeneration)
		require.NoError(t, err)
		require.NoError(t, r.Account.Block(database.Conn(), time.Now()))

		for _, path := range []string{"/account", "/account/experimental-features", "/admin/api/accounts"} {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.AddCookie(&http.Cookie{Name: "account_token", Value: token})

			res := httptest.NewRecorder()
			handler.ServeHTTP(res, req)

			assert.Equal(t, http.StatusForbidden, res.Code, path)
			assert.Contains(t, res.Body.String(), models.AccountBlockedMessage, path)
		}
	})
}

func TestSessionRevocationWithinIssuanceSecond(t *testing.T) {
	for _, transport := range []string{"cookie", "scoped"} {
		for _, offset := range []time.Duration{500 * time.Millisecond, 2 * time.Second} {
			t.Run(fmt.Sprintf("%s/%s", transport, offset), func(t *testing.T) {
				r := support.Setup(t)
				signer := jwt.NewSigner("test-secret")
				mint := func() string {
					if transport == "cookie" {
						res := httptest.NewRecorder()
						req := httptest.NewRequest(http.MethodGet, "/account", nil)
						require.NoError(t, authentication.IssueAccountSession(res, req, signer, r.Account))
						return res.Result().Cookies()[0].Value
					}
					token, err := signer.GenerateScopedToken(jwt.ScopedTokenClaims{
						SessionGeneration: &r.Account.SessionGeneration,
						Subject:           r.User.String(), OrgID: r.Organization.ID.String(), Purpose: "test", Scopes: []string{"canvases:read"},
					}, time.Hour)
					require.NoError(t, err)
					return token
				}
				calls := 0
				next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					calls++
					w.WriteHeader(http.StatusNoContent)
				})
				handler := AccountAuthMiddleware(signer)(next)
				if transport == "scoped" {
					handler = OrganizationAuthMiddleware(signer)(next)
				}
				check := func(token string, status int) {
					req := httptest.NewRequest(http.MethodGet, "/account", nil)
					if transport == "cookie" {
						req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
					} else {
						req.Header.Set("Authorization", "Bearer "+token)
					}
					res := httptest.NewRecorder()
					handler.ServeHTTP(res, req)
					require.Equal(t, status, res.Code)
				}
				oldToken := mint()
				check(oldToken, http.StatusNoContent)
				claims, err := signer.ValidateAndGetClaims(oldToken)
				require.NoError(t, err)
				changedAt := time.Unix(int64(claims["iat"].(float64)), 0).Add(offset)
				err = database.Conn().Transaction(func(tx *gorm.DB) error {
					require.NoError(t, r.Account.MarkPasswordChangedInTransaction(tx, changedAt))
					return fmt.Errorf("rollback rotation")
				})
				require.Error(t, err)
				check(oldToken, http.StatusNoContent)
				require.NoError(t, r.Account.MarkPasswordChangedInTransaction(database.Conn(), changedAt))
				check(oldToken, http.StatusUnauthorized)
				require.Equal(t, 2, calls)
				freshClaims, err := signer.ValidateAndGetClaims(mint())
				require.NoError(t, err)
				// Keep replacement issuance in the same second without scheduler assumptions.
				freshClaims["iat"] = claims["iat"]
				freshToken, err := jwtLib.NewWithClaims(jwtLib.SigningMethodHS256, freshClaims).SignedString([]byte(signer.Secret))
				require.NoError(t, err)
				check(freshToken, http.StatusNoContent)
			})
		}
	}
}

func TestAuthenticationSessionGenerationCompatibility(t *testing.T) {
	for _, transport := range []string{"cookie", "scoped"} {
		for _, scenario := range []struct {
			name                 string
			generation           int64
			claim                any
			present              bool
			passwordChangeOffset *time.Duration
			status               int
		}{
			{name: "legacy initial account", status: http.StatusNoContent},
			{name: "explicit zero", present: true, claim: "0", status: http.StatusNoContent},
			{name: "large generation", generation: 9007199254740993, present: true, claim: "9007199254740993", status: http.StatusNoContent},
			{name: "legacy after rotation", generation: 1, status: http.StatusUnauthorized},
			{name: "legacy same second change", passwordChangeOffset: durationPointer(500 * time.Millisecond), status: http.StatusUnauthorized},
			{name: "legacy earlier second change", passwordChangeOffset: durationPointer(-time.Second), status: http.StatusNoContent},
			{name: "missing legacy issuance", claim: "missing iat", passwordChangeOffset: durationPointer(-time.Second), status: http.StatusUnauthorized},
			{name: "negative", present: true, claim: "-1", status: http.StatusUnauthorized},
			{name: "fraction", present: true, claim: "0.5", status: http.StatusUnauthorized},
			{name: "number", present: true, claim: 0, status: http.StatusUnauthorized},
			{name: "null", present: true, claim: nil, status: http.StatusUnauthorized},
			{name: "overflow", present: true, claim: "9223372036854775808", status: http.StatusUnauthorized},
		} {
			t.Run(transport+"/"+scenario.name, func(t *testing.T) {
				r := support.Setup(t)
				signer := jwt.NewSigner("test-secret")
				now := time.Now().Truncate(time.Second)
				updates := map[string]any{"session_generation": scenario.generation}
				if scenario.passwordChangeOffset != nil {
					updates["password_changed_at"] = now.Add(*scenario.passwordChangeOffset)
				}
				require.NoError(t, database.Conn().Model(r.Account).Updates(updates).Error)
				claims := jwtLib.MapClaims{
					"sub": r.Account.ID.String(), "iat": now.Unix(), "nbf": now.Unix(), "exp": now.Add(time.Hour).Unix(), "ses": fmt.Sprint(now.Unix()),
				}
				if scenario.present {
					claims[jwt.SessionGenerationClaim] = scenario.claim
				}
				if scenario.claim == "missing iat" {
					delete(claims, "iat")
				}
				calls := 0
				next := countingHandler(&calls)
				handler := AccountAuthMiddleware(signer)(next)
				if transport == "scoped" {
					claims["sub"] = r.User.String()
					claims["aud"] = jwt.ScopedTokenAudience
					claims["token_type"] = jwt.ScopedTokenType
					claims["org_id"] = r.Organization.ID.String()
					claims["purpose"] = "test"
					claims["scopes"] = []string{"canvases:read"}
					handler = OrganizationAuthMiddleware(signer)(next)
				}
				token, err := jwtLib.NewWithClaims(jwtLib.SigningMethodHS256, claims).SignedString([]byte(signer.Secret))
				require.NoError(t, err)
				req := httptest.NewRequest(http.MethodGet, "/account", nil)
				if transport == "cookie" {
					req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
				} else {
					req.Header.Set("Authorization", "Bearer "+token)
				}
				res := httptest.NewRecorder()
				handler.ServeHTTP(res, req)
				require.Equal(t, scenario.status, res.Code)
				if scenario.status == http.StatusUnauthorized {
					require.Zero(t, calls)
				} else {
					require.Equal(t, 1, calls)
				}
			})
		}
	}
}

func durationPointer(duration time.Duration) *time.Duration { return &duration }

func TestOrganizationAuthMiddleware_AccountlessScopedToken(t *testing.T) {
	r := support.Setup(t)
	signer := jwt.NewSigner("test-secret")
	apiKey := &models.User{ID: uuid.New(), OrganizationID: r.Organization.ID, Name: "bot", Type: models.UserTypeAPIKey}
	require.NoError(t, database.Conn().Create(apiKey).Error)
	token, err := signer.GenerateScopedToken(jwt.ScopedTokenClaims{
		Subject: apiKey.ID.String(), OrgID: r.Organization.ID.String(), Purpose: "test", Scopes: []string{"canvases:read"},
	}, time.Hour)
	require.NoError(t, err)
	require.NoError(t, r.Account.MarkPasswordChangedInTransaction(database.Conn(), time.Now()))
	calls := 0
	handler := OrganizationAuthMiddleware(signer)(countingHandler(&calls))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/canvases", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	require.Equal(t, http.StatusNoContent, res.Code)
	require.Equal(t, 1, calls)
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

func assertSupportSessionCookieCleared(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()

	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == "support_session_token" {
			assert.Less(t, cookie.MaxAge, 0)
			return
		}
	}

	assert.Fail(t, "support session cookie was not cleared")
}

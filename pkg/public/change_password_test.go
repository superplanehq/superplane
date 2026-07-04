package public

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/impersonation"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

const testCurrentPassword = "current-pass-123"
const testNewPassword = "new-secure-pass-456"

// setupChangePasswordTestServer constructs a server with password login
// enabled and seeds a password auth row for the test account so the
// change-password handler can exercise the success path.
func setupChangePasswordTestServer(t *testing.T) (*Server, *support.ResourceRegistry, string) {
	t.Setenv("ENABLE_PASSWORD_LOGIN", "yes")

	r := support.Setup(t)

	signer := jwt.NewSigner("test-client-secret")
	server, err := NewServer(r.Encryptor, r.Registry, signer, support.NewOIDCProvider(), r.GitProvider, "", "", "", "test", "/app/templates", r.AuthService, nil, false)
	require.NoError(t, err)
	server.RegisterWebRoutes("")

	hash, err := crypto.HashPassword(testCurrentPassword)
	require.NoError(t, err)

	_, err = models.CreateAccountPasswordAuth(r.Account.ID, hash)
	require.NoError(t, err)

	token, err := authentication.GenerateAccountToken(signer, r.Account.ID.String(), time.Now(), time.Hour)
	require.NoError(t, err)

	return server, r, token
}

func setupChangePasswordTestServerNoPassword(t *testing.T) (*Server, *support.ResourceRegistry, string) {
	t.Setenv("ENABLE_PASSWORD_LOGIN", "yes")

	r := support.Setup(t)

	signer := jwt.NewSigner("test-client-secret")
	server, err := NewServer(r.Encryptor, r.Registry, signer, support.NewOIDCProvider(), r.GitProvider, "", "", "", "test", "/app/templates", r.AuthService, nil, false)
	require.NoError(t, err)
	server.RegisterWebRoutes("")

	token, err := authentication.GenerateAccountToken(signer, r.Account.ID.String(), time.Now(), time.Hour)
	require.NoError(t, err)

	return server, r, token
}

func setupChangePasswordTestServerLoginDisabled(t *testing.T) (*Server, *support.ResourceRegistry, string) {
	t.Setenv("ENABLE_PASSWORD_LOGIN", "no")

	r := support.Setup(t)

	signer := jwt.NewSigner("test-client-secret")
	server, err := NewServer(r.Encryptor, r.Registry, signer, support.NewOIDCProvider(), r.GitProvider, "", "", "", "test", "/app/templates", r.AuthService, nil, false)
	require.NoError(t, err)
	server.RegisterWebRoutes("")

	hash, err := crypto.HashPassword(testCurrentPassword)
	require.NoError(t, err)

	_, err = models.CreateAccountPasswordAuth(r.Account.ID, hash)
	require.NoError(t, err)

	token, err := authentication.GenerateAccountToken(signer, r.Account.ID.String(), time.Now(), time.Hour)
	require.NoError(t, err)

	return server, r, token
}

func sendChangePasswordRequest(server *Server, body []byte, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(http.MethodPost, "/account/password", bytes.NewReader(body))
	req.Header.Add("Content-Type", "application/json")
	for _, c := range cookies {
		req.AddCookie(c)
	}

	res := httptest.NewRecorder()
	server.Router.ServeHTTP(res, req)
	return res
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

func TestChangePassword_Unauthenticated(t *testing.T) {
	server, _, _ := setupChangePasswordTestServer(t)

	body := mustMarshal(t, map[string]string{
		"currentPassword": testCurrentPassword,
		"newPassword":     testNewPassword,
	})

	res := sendChangePasswordRequest(server, body)
	assert.Equal(t, http.StatusUnauthorized, res.Code)
}

func TestChangePassword_PasswordLoginDisabled(t *testing.T) {
	server, _, token := setupChangePasswordTestServerLoginDisabled(t)

	body := mustMarshal(t, map[string]string{
		"currentPassword": testCurrentPassword,
		"newPassword":     testNewPassword,
	})

	res := sendChangePasswordRequest(server, body, &http.Cookie{Name: "account_token", Value: token})
	assert.Equal(t, http.StatusForbidden, res.Code)
}

func TestChangePassword_AccountWithoutPasswordAuth(t *testing.T) {
	server, _, token := setupChangePasswordTestServerNoPassword(t)

	body := mustMarshal(t, map[string]string{
		"currentPassword": testCurrentPassword,
		"newPassword":     testNewPassword,
	})

	res := sendChangePasswordRequest(server, body, &http.Cookie{Name: "account_token", Value: token})
	assert.Equal(t, http.StatusForbidden, res.Code)
}

func TestChangePassword_RejectsImpersonation(t *testing.T) {
	server, r, token := setupChangePasswordTestServer(t)

	require.NoError(t, models.PromoteToInstallationAdmin(r.Account.ID.String()))

	target, err := models.CreateAccount("Target User", "target-impersonate@example.com")
	require.NoError(t, err)

	signer := jwt.NewSigner("test-client-secret")
	impToken, err := impersonation.GenerateToken(signer, r.Account.ID.String(), target.ID.String())
	require.NoError(t, err)

	body := mustMarshal(t, map[string]string{
		"currentPassword": testCurrentPassword,
		"newPassword":     testNewPassword,
	})

	res := sendChangePasswordRequest(server, body,
		&http.Cookie{Name: "account_token", Value: token},
		&http.Cookie{Name: impersonation.CookieName, Value: impToken},
	)
	assert.Equal(t, http.StatusForbidden, res.Code)
}

func TestChangePassword_ValidationErrors(t *testing.T) {
	server, _, token := setupChangePasswordTestServer(t)

	cases := []struct {
		name string
		body any
	}{
		{
			name: "missing current password",
			body: map[string]string{"newPassword": testNewPassword},
		},
		{
			name: "missing new password",
			body: map[string]string{"currentPassword": testCurrentPassword},
		},
		{
			name: "new password too short",
			body: map[string]string{"currentPassword": testCurrentPassword, "newPassword": "short"},
		},
		{
			name: "new password equals current",
			body: map[string]string{"currentPassword": testCurrentPassword, "newPassword": testCurrentPassword},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res := sendChangePasswordRequest(server, mustMarshal(t, c.body),
				&http.Cookie{Name: "account_token", Value: token})
			assert.Equal(t, http.StatusBadRequest, res.Code)
		})
	}
}

func TestChangePassword_WrongCurrentPassword(t *testing.T) {
	server, _, token := setupChangePasswordTestServer(t)

	body := mustMarshal(t, map[string]string{
		"currentPassword": "wrong-password",
		"newPassword":     testNewPassword,
	})

	res := sendChangePasswordRequest(server, body, &http.Cookie{Name: "account_token", Value: token})
	assert.Equal(t, http.StatusUnauthorized, res.Code)
}

func TestChangePassword_Success(t *testing.T) {
	server, r, token := setupChangePasswordTestServer(t)

	// Seed an existing API token on the user so we can verify it gets cleared.
	originalTokenHash := crypto.HashToken("some-existing-api-token")
	require.NoError(t, r.UserModel.UpdateTokenHash(originalTokenHash))

	// Drop a stale impersonation cookie on the request to verify it is cleared.
	staleImpersonation := &http.Cookie{Name: impersonation.CookieName, Value: "stale-token-value"}

	body := mustMarshal(t, map[string]string{
		"currentPassword": testCurrentPassword,
		"newPassword":     testNewPassword,
	})

	res := sendChangePasswordRequest(server, body,
		&http.Cookie{Name: "account_token", Value: token},
		staleImpersonation,
	)
	require.Equal(t, http.StatusNoContent, res.Code)

	// Password hash is rotated.
	passwordAuth, err := models.FindAccountPasswordAuthByAccountID(r.Account.ID)
	require.NoError(t, err)
	assert.True(t, crypto.VerifyPassword(passwordAuth.PasswordHash, testNewPassword))
	assert.False(t, crypto.VerifyPassword(passwordAuth.PasswordHash, testCurrentPassword))

	// password_changed_at is set.
	updatedAccount, err := models.FindAccountByID(r.Account.ID.String())
	require.NoError(t, err)
	require.NotNil(t, updatedAccount.PasswordChangedAt)
	assert.WithinDuration(t, time.Now(), *updatedAccount.PasswordChangedAt, 5*time.Second)

	// Every user under the account had token_hash cleared.
	var users []models.User
	require.NoError(t, database.Conn().Where("account_id = ?", r.Account.ID).Find(&users).Error)
	require.NotEmpty(t, users)
	for _, u := range users {
		assert.Empty(t, u.TokenHash, "token_hash should have been cleared for user %s", u.ID)
	}

	// Cookies were set: a refreshed account_token AND a cleared impersonation_token.
	cookies := res.Result().Cookies()
	var newAuthCookie, clearedImpCookie *http.Cookie
	for _, c := range cookies {
		switch c.Name {
		case "account_token":
			newAuthCookie = c
		case impersonation.CookieName:
			clearedImpCookie = c
		}
	}

	require.NotNil(t, newAuthCookie, "a fresh account_token cookie should be issued")
	assert.NotEqual(t, token, newAuthCookie.Value, "the reissued account_token must not equal the old one")
	assert.True(t, newAuthCookie.HttpOnly)
	assert.NotZero(t, newAuthCookie.MaxAge)

	require.NotNil(t, clearedImpCookie, "the impersonation cookie should be cleared")
	assert.Equal(t, "", clearedImpCookie.Value)
	assert.Equal(t, -1, clearedImpCookie.MaxAge)
}

func TestGetAccount_HasPassword(t *testing.T) {
	t.Run("returns false when account has no password auth", func(t *testing.T) {
		os.Unsetenv("ENABLE_PASSWORD_LOGIN")
		server, _, token := setupTestServer(support.Setup(t), t)

		req, _ := http.NewRequest(http.MethodGet, "/account", nil)
		req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
		res := httptest.NewRecorder()
		server.Router.ServeHTTP(res, req)

		require.Equal(t, http.StatusOK, res.Code)
		var resp AccountResponse
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &resp))
		assert.False(t, resp.HasPassword)
	})

	t.Run("returns true when account has password auth", func(t *testing.T) {
		server, r, token := setupChangePasswordTestServer(t)

		req, _ := http.NewRequest(http.MethodGet, "/account", nil)
		req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
		res := httptest.NewRecorder()
		server.Router.ServeHTTP(res, req)

		require.Equal(t, http.StatusOK, res.Code)
		var resp AccountResponse
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &resp))
		assert.True(t, resp.HasPassword)
		assert.Equal(t, r.Account.ID.String(), resp.ID)
	})
}

// TestChangePassword_InvalidatesOldCookie verifies the freshness check
// in the middleware: an account_token issued before password_changed_at
// is rejected, while a fresh one issued after is accepted.
func TestChangePassword_InvalidatesOldCookie(t *testing.T) {
	server, r, oldToken := setupChangePasswordTestServer(t)

	// Confirm the cookie works before any password change.
	check := func(token string) int {
		req, _ := http.NewRequest(http.MethodGet, "/account", nil)
		req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
		res := httptest.NewRecorder()
		server.Router.ServeHTTP(res, req)
		return res.Code
	}

	require.Equal(t, http.StatusOK, check(oldToken))

	// Wait a second so the new cookie's iat is strictly newer than the
	// old cookie's, since JWT iat resolution is whole seconds.
	time.Sleep(1100 * time.Millisecond)

	body := mustMarshal(t, map[string]string{
		"currentPassword": testCurrentPassword,
		"newPassword":     testNewPassword,
	})
	res := sendChangePasswordRequest(server, body, &http.Cookie{Name: "account_token", Value: oldToken})
	require.Equal(t, http.StatusNoContent, res.Code)

	// The old cookie is now stale and rejected.
	assert.Equal(t, http.StatusUnauthorized, check(oldToken))

	// A freshly minted cookie with iat after password_changed_at is accepted.
	signer := jwt.NewSigner("test-client-secret")
	freshToken, err := authentication.GenerateAccountToken(signer, r.Account.ID.String(), time.Now(), time.Hour)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, check(freshToken))
}

// TestChangePassword_ReissuedCookieValid verifies the cookie returned
// by the change-password handler itself is treated as fresh.
func TestChangePassword_ReissuedCookieValid(t *testing.T) {
	server, _, oldToken := setupChangePasswordTestServer(t)

	time.Sleep(1100 * time.Millisecond)

	body := mustMarshal(t, map[string]string{
		"currentPassword": testCurrentPassword,
		"newPassword":     testNewPassword,
	})
	res := sendChangePasswordRequest(server, body, &http.Cookie{Name: "account_token", Value: oldToken})
	require.Equal(t, http.StatusNoContent, res.Code)

	var newAuthCookie *http.Cookie
	for _, c := range res.Result().Cookies() {
		if c.Name == "account_token" {
			newAuthCookie = c
			break
		}
	}
	require.NotNil(t, newAuthCookie)

	req, _ := http.NewRequest(http.MethodGet, "/account", nil)
	req.AddCookie(newAuthCookie)
	check := httptest.NewRecorder()
	server.Router.ServeHTTP(check, req)
	assert.Equal(t, http.StatusOK, check.Code)
}

// TestSetAccountCookie_Helper sanity-checks the new cookie helper so any
// future drift in cookie attributes is caught explicitly.
func TestSetAccountCookie_Helper(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/account", nil)

	authentication.SetAccountCookie(w, r, "abc", time.Hour)

	cookies := w.Result().Cookies()
	require.Len(t, cookies, 1)
	c := cookies[0]
	assert.Equal(t, "account_token", c.Name)
	assert.Equal(t, "abc", c.Value)
	assert.Equal(t, "/", c.Path)
	assert.True(t, c.HttpOnly)
	assert.Equal(t, http.SameSiteLaxMode, c.SameSite)
	assert.Equal(t, int(time.Hour.Seconds()), c.MaxAge)
}

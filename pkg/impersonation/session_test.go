package impersonation

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/jwt"
)

func TestGenerateToken(t *testing.T) {
	signer := jwt.NewSigner("test-secret")

	t.Run("generates a valid token with all claims", func(t *testing.T) {
		token, err := GenerateToken(signer, "admin-123", "user-456", "org-789")
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		claims, err := ValidateToken(signer, token)
		require.NoError(t, err)
		assert.Equal(t, "admin-123", claims.AdminAccountID)
		assert.Equal(t, "user-456", claims.ImpersonatedUserID)
		assert.Equal(t, "org-789", claims.ImpersonatedOrgID)
	})
}

func TestValidateToken(t *testing.T) {
	signer := jwt.NewSigner("test-secret")

	t.Run("rejects token signed with different secret", func(t *testing.T) {
		otherSigner := jwt.NewSigner("other-secret")
		token, err := GenerateToken(otherSigner, "admin-1", "user-2", "org-3")
		require.NoError(t, err)

		claims, err := ValidateToken(signer, token)
		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("rejects expired token", func(t *testing.T) {
		// Generate a token with a very short TTL using GenerateWithClaims directly
		token, err := signer.GenerateWithClaims(0*time.Second, map[string]string{
			"type":                 TokenType,
			"admin_account_id":     "admin-1",
			"impersonated_user_id": "user-2",
			"impersonated_org_id":  "org-3",
			"sub":                  "admin-1",
		})
		require.NoError(t, err)

		// Wait for expiry
		time.Sleep(1100 * time.Millisecond)

		claims, err := ValidateToken(signer, token)
		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("rejects token with wrong type claim", func(t *testing.T) {
		// Generate a regular account token (no "type" claim)
		token, err := signer.Generate("account-123", time.Hour)
		require.NoError(t, err)

		claims, err := ValidateToken(signer, token)
		assert.Error(t, err)
		assert.Nil(t, claims)
		assert.Contains(t, err.Error(), "not an impersonation token")
	})

	t.Run("rejects token with missing claims", func(t *testing.T) {
		token, err := signer.GenerateWithClaims(time.Hour, map[string]string{
			"type":             TokenType,
			"admin_account_id": "admin-1",
			// missing user_id and org_id
		})
		require.NoError(t, err)

		claims, err := ValidateToken(signer, token)
		assert.Error(t, err)
		assert.Nil(t, claims)
		assert.Contains(t, err.Error(), "incomplete impersonation claims")
	})

	t.Run("rejects completely invalid token string", func(t *testing.T) {
		claims, err := ValidateToken(signer, "not-a-valid-jwt")
		assert.Error(t, err)
		assert.Nil(t, claims)
	})
}

func TestSetCookie(t *testing.T) {
	t.Run("sets impersonation cookie with correct attributes", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/admin/api/impersonate/start", nil)

		SetCookie(w, r, "test-token-value")

		cookies := w.Result().Cookies()
		require.Len(t, cookies, 1)

		cookie := cookies[0]
		assert.Equal(t, CookieName, cookie.Name)
		assert.Equal(t, "test-token-value", cookie.Value)
		assert.Equal(t, "/", cookie.Path)
		assert.Equal(t, int(TTL.Seconds()), cookie.MaxAge)
		assert.True(t, cookie.HttpOnly)
	})
}

func TestClearCookie(t *testing.T) {
	t.Run("deletes impersonation cookie", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/admin/api/impersonate/end", nil)

		ClearCookie(w, r)

		cookies := w.Result().Cookies()
		require.Len(t, cookies, 1)

		cookie := cookies[0]
		assert.Equal(t, CookieName, cookie.Name)
		assert.Equal(t, "", cookie.Value)
		assert.Equal(t, -1, cookie.MaxAge)
	})
}

func TestReadCookie(t *testing.T) {
	t.Run("reads impersonation cookie from request", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.AddCookie(&http.Cookie{Name: CookieName, Value: "my-token"})

		token, err := ReadCookie(r)
		require.NoError(t, err)
		assert.Equal(t, "my-token", token)
	})

	t.Run("returns error when cookie is missing", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		token, err := ReadCookie(r)
		assert.Error(t, err)
		assert.Empty(t, token)
	})
}

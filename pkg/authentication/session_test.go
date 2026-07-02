package authentication

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jwtLib "github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/test/support"
)

func TestAccountSessionTTL(t *testing.T) {
	t.Run("defaults to 24 hours", func(t *testing.T) {
		ResetAccountSessionTTLForTests()
		t.Cleanup(ResetAccountSessionTTLForTests)

		assert.Equal(t, 24*time.Hour, AccountSessionTTL())
	})

	t.Run("reads ACCOUNT_SESSION_TTL from environment", func(t *testing.T) {
		ResetAccountSessionTTLForTests()
		t.Cleanup(ResetAccountSessionTTLForTests)
		t.Setenv("ACCOUNT_SESSION_TTL", "48h")

		assert.Equal(t, 48*time.Hour, AccountSessionTTL())
	})
}

func TestAccountSessionMaxAge(t *testing.T) {
	t.Run("defaults to 7 days", func(t *testing.T) {
		ResetAccountSessionTTLForTests()
		t.Cleanup(ResetAccountSessionTTLForTests)

		assert.Equal(t, 7*24*time.Hour, AccountSessionMaxAge())
	})

	t.Run("reads ACCOUNT_SESSION_MAX_AGE from environment", func(t *testing.T) {
		ResetAccountSessionTTLForTests()
		t.Cleanup(ResetAccountSessionTTLForTests)
		t.Setenv("ACCOUNT_SESSION_MAX_AGE", "168h")

		assert.Equal(t, 168*time.Hour, AccountSessionMaxAge())
	})
}

func mintAccountToken(t *testing.T, signer *jwt.Signer, accountID string, issuedAt time.Time, sessionStart time.Time, ttl time.Duration) string {
	t.Helper()

	token := jwtLib.NewWithClaims(jwtLib.SigningMethodHS256, jwtLib.MapClaims{
		"sub":             accountID,
		"iat":             issuedAt.Unix(),
		"nbf":             issuedAt.Unix(),
		"exp":             issuedAt.Add(ttl).Unix(),
		sessionStartClaim: fmt.Sprintf("%d", sessionStart.Unix()),
	})

	tokenString, err := token.SignedString([]byte(signer.Secret))
	require.NoError(t, err)

	return tokenString
}

func TestMaybeRefreshAccountSession(t *testing.T) {
	r := support.Setup(t)
	signer := jwt.NewSigner("test-secret")

	t.Run("refreshes on activity when token is older than one minute", func(t *testing.T) {
		ResetAccountSessionTTLForTests()
		t.Cleanup(ResetAccountSessionTTLForTests)

		issuedAt := time.Now().Add(-2 * time.Minute)
		oldToken := mintAccountToken(t, signer, r.Account.ID.String(), issuedAt, issuedAt, 20*time.Hour)

		req := httptest.NewRequest(http.MethodGet, "/account", nil)
		req.AddCookie(&http.Cookie{Name: "account_token", Value: oldToken})
		res := httptest.NewRecorder()

		MaybeRefreshAccountSession(res, req, signer, r.Account)

		cookies := res.Result().Cookies()
		require.Len(t, cookies, 1)
		assert.NotEqual(t, oldToken, cookies[0].Value)
		assert.Equal(t, int(24*time.Hour.Seconds()), cookies[0].MaxAge)

		newClaims, err := signer.ValidateAndGetClaims(cookies[0].Value)
		require.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("%d", issuedAt.Unix()), newClaims[sessionStartClaim])
	})

	t.Run("does not refresh a token that was just issued", func(t *testing.T) {
		ResetAccountSessionTTLForTests()
		t.Cleanup(ResetAccountSessionTTLForTests)

		token, err := signer.GenerateWithClaims(24*time.Hour, map[string]string{
			"sub":             r.Account.ID.String(),
			sessionStartClaim: fmt.Sprintf("%d", time.Now().Unix()),
		})
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/account", nil)
		req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
		res := httptest.NewRecorder()

		MaybeRefreshAccountSession(res, req, signer, r.Account)

		assert.Empty(t, res.Result().Cookies())
	})

	t.Run("extends session for daily morning usage pattern", func(t *testing.T) {
		ResetAccountSessionTTLForTests()
		t.Cleanup(ResetAccountSessionTTLForTests)

		sessionStart := time.Now().Add(-24*time.Hour + time.Minute)
		issuedAt := sessionStart
		oldToken := mintAccountToken(t, signer, r.Account.ID.String(), issuedAt, sessionStart, 24*time.Hour)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/canvases", nil)
		req.AddCookie(&http.Cookie{Name: "account_token", Value: oldToken})
		res := httptest.NewRecorder()

		MaybeRefreshAccountSession(res, req, signer, r.Account)

		cookies := res.Result().Cookies()
		require.Len(t, cookies, 1)

		newClaims, err := signer.ValidateAndGetClaims(cookies[0].Value)
		require.NoError(t, err)

		exp, ok := newClaims["exp"].(float64)
		require.True(t, ok)
		assert.Greater(t, int64(exp)-time.Now().Unix(), int64(23*time.Hour.Seconds()))
	})

	t.Run("rejects tokens without ses claim", func(t *testing.T) {
		token, err := signer.Generate(r.Account.ID.String(), time.Hour)
		require.NoError(t, err)

		assert.False(t, IsAccountSessionWithinMaxAge(mustClaims(t, signer, token)))
	})

	t.Run("does not refresh once absolute max age is reached", func(t *testing.T) {
		ResetAccountSessionTTLForTests()
		t.Cleanup(ResetAccountSessionTTLForTests)
		t.Setenv("ACCOUNT_SESSION_MAX_AGE", "48h")

		sessionStart := time.Now().Add(-49 * time.Hour)
		issuedAt := time.Now().Add(-2 * time.Minute)
		oldToken := mintAccountToken(t, signer, r.Account.ID.String(), issuedAt, sessionStart, 24*time.Hour)

		req := httptest.NewRequest(http.MethodGet, "/account", nil)
		req.AddCookie(&http.Cookie{Name: "account_token", Value: oldToken})
		res := httptest.NewRecorder()

		MaybeRefreshAccountSession(res, req, signer, r.Account)

		assert.Empty(t, res.Result().Cookies())
		assert.False(t, IsAccountSessionWithinMaxAge(mustClaims(t, signer, oldToken)))
	})
}

func mustClaims(t *testing.T, signer *jwt.Signer, token string) jwtLib.MapClaims {
	t.Helper()

	claims, err := signer.ValidateAndGetClaims(token)
	require.NoError(t, err)

	return claims
}

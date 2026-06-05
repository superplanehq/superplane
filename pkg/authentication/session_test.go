package authentication

import (
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

func mintAccountToken(t *testing.T, signer *jwt.Signer, accountID string, issuedAt time.Time, ttl time.Duration) string {
	t.Helper()

	token := jwtLib.NewWithClaims(jwtLib.SigningMethodHS256, jwtLib.MapClaims{
		"sub": accountID,
		"iat": issuedAt.Unix(),
		"nbf": issuedAt.Unix(),
		"exp": issuedAt.Add(ttl).Unix(),
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

		oldToken := mintAccountToken(t, signer, r.Account.ID.String(), time.Now().Add(-2*time.Minute), 20*time.Hour)

		req := httptest.NewRequest(http.MethodGet, "/account", nil)
		req.AddCookie(&http.Cookie{Name: "account_token", Value: oldToken})
		res := httptest.NewRecorder()

		MaybeRefreshAccountSession(res, req, signer, r.Account)

		cookies := res.Result().Cookies()
		require.Len(t, cookies, 1)
		assert.NotEqual(t, oldToken, cookies[0].Value)
		assert.Equal(t, int(24*time.Hour.Seconds()), cookies[0].MaxAge)
	})

	t.Run("does not refresh a token that was just issued", func(t *testing.T) {
		ResetAccountSessionTTLForTests()
		t.Cleanup(ResetAccountSessionTTLForTests)

		token, err := signer.Generate(r.Account.ID.String(), 24*time.Hour)
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

		// Token from yesterday morning with only a minute of lifetime left.
		issuedAt := time.Now().Add(-24*time.Hour + time.Minute)
		oldToken := mintAccountToken(t, signer, r.Account.ID.String(), issuedAt, 24*time.Hour)

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
}

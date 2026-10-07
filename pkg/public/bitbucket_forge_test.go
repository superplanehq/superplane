package public

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

const forgeTestAppID = "ari:cloud:ecosystem::app/11111111-1111-1111-1111-111111111111"

func TestHandleBitbucketForgeUninstallClearsTheCachedToken(t *testing.T) {
	r := support.Setup(t)
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	useForgeTestApp(t, privateKey)

	installationID := uuid.NewString()
	_, err = models.SaveBitbucketForgeDelivery(database.Conn(), models.BitbucketForgeDelivery{
		InstallationID: installationID,
		SystemToken:    []byte("cached-token"),
		TokenExpiresAt: time.Now().Add(2 * time.Hour),
		DeliveredAt:    time.Now(),
	})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/bitbucket/forge/uninstall", nil)
	request.Header.Set("Authorization", "Bearer "+signForgeInvocation(t, privateKey, installationID))
	request.Header.Set("x-forge-oauth-system", signForgeSystemToken(t, time.Now().Add(time.Hour)))
	response := httptest.NewRecorder()

	(&Server{encryptor: r.Encryptor}).HandleBitbucketForgeUninstall(response, request)

	require.Equal(t, http.StatusNoContent, response.Code)
	installation, err := models.FindBitbucketForgeInstallation(database.Conn(), installationID)
	require.NoError(t, err)
	assert.NotNil(t, installation.UninstalledAt)
	assert.Empty(t, installation.SystemToken)
}

func useForgeTestApp(t *testing.T, privateKey *rsa.PrivateKey) {
	t.Helper()
	jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]string{{
				"kid": "forge-test-key",
				"kty": "RSA",
				"n":   base64.RawURLEncoding.EncodeToString(privateKey.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.E)).Bytes()),
			}},
		})
	}))
	t.Cleanup(jwks.Close)
	t.Setenv(config.EnvBitbucketForgeAppID, forgeTestAppID)
	t.Setenv(config.EnvBitbucketForgeInstallURL, "https://bitbucket.example/install")
	t.Setenv(config.EnvBitbucketForgeJWKSURL, jwks.URL)
}

func signForgeInvocation(t *testing.T, privateKey *rsa.PrivateKey, installationID string) string {
	t.Helper()
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, jwtlib.MapClaims{
		"aud": forgeTestAppID,
		"exp": time.Now().Add(time.Minute).Unix(),
		"app": map[string]any{
			"id":             forgeTestAppID,
			"installationId": installationID,
		},
		"context": map[string]any{
			"installContext": "ari:cloud:bitbucket::workspace/" + uuid.NewString(),
		},
	})
	token.Header["kid"] = "forge-test-key"
	signed, err := token.SignedString(privateKey)
	require.NoError(t, err)
	return signed
}

func signForgeSystemToken(t *testing.T, expires time.Time) string {
	t.Helper()
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, jwtlib.RegisteredClaims{
		ExpiresAt: jwtlib.NewNumericDate(expires),
	})
	signed, err := token.SignedString([]byte("system-token-test-secret"))
	require.NoError(t, err)
	return signed
}

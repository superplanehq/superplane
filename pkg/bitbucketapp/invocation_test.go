package bitbucketapp

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"math/big"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseInvocationAcceptsSignedFitAndReadsSystemTokenExpiry(t *testing.T) {
	privateKey := testRSAKey(t)
	const appID = "ari:cloud:ecosystem::app/11111111-1111-1111-1111-111111111111"
	const installationID = "install-1"
	const workspaceID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	expires := time.Now().Add(2 * time.Hour).UTC().Truncate(time.Second)

	fit := signFit(t, privateKey, appID, installationID, "ari:cloud:bitbucket::workspace/"+workspaceID)
	systemToken := signSystemToken(t, expires)

	invocation, err := ParseInvocation(fit, systemToken, appID, rsaKeyfunc(privateKey))
	require.NoError(t, err)
	assert.Equal(t, installationID, invocation.InstallationID)
	assert.Equal(t, workspaceID, invocation.WorkspaceUUID)
	assert.Equal(t, systemToken, invocation.SystemToken)
	assert.WithinDuration(t, expires, invocation.SystemTokenExpires, time.Second)
}

func TestRSAPublicKeyAcceptsPaddedForgeModulus(t *testing.T) {
	key := testRSAKey(t)
	modulus := base64.URLEncoding.EncodeToString(key.N.Bytes())
	require.Contains(t, modulus, "=")
	exponent := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())

	parsed, err := RSAPublicKey(modulus, exponent)
	require.NoError(t, err)
	assert.Equal(t, 0, key.N.Cmp(parsed.N))
	assert.Equal(t, key.E, parsed.E)
}

func TestParseInvocationRejectsADifferentApp(t *testing.T) {
	privateKey := testRSAKey(t)
	const appID = "ari:cloud:ecosystem::app/11111111-1111-1111-1111-111111111111"
	fit := signFit(t, privateKey, appID, "install-1", "ari:cloud:bitbucket::workspace/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	systemToken := signSystemToken(t, time.Now().Add(time.Hour))

	_, err := ParseInvocation(fit, systemToken, "ari:cloud:ecosystem::app/other", rsaKeyfunc(privateKey))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "different app")
}

func testRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return key
}

func rsaKeyfunc(key *rsa.PrivateKey) jwtlib.Keyfunc {
	return func(token *jwtlib.Token) (any, error) {
		return &key.PublicKey, nil
	}
}

func signFit(t *testing.T, key *rsa.PrivateKey, appID, installationID, installContext string) string {
	t.Helper()
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, jwtlib.MapClaims{
		"aud": appID,
		"app": map[string]any{
			"id":             appID,
			"installationId": installationID,
			"apiBaseUrl":     "https://api.atlassian.com",
		},
		"context": map[string]any{
			"installContext": installContext,
		},
	})
	signed, err := token.SignedString(key)
	require.NoError(t, err)
	return signed
}

func signSystemToken(t *testing.T, expires time.Time) string {
	t.Helper()
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, jwtlib.RegisteredClaims{
		ExpiresAt: jwtlib.NewNumericDate(expires),
	})
	signed, err := token.SignedString([]byte("system-token-test-secret"))
	require.NoError(t, err)
	return signed
}

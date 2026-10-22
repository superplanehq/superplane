package jwt

import (
	"testing"
	"time"

	gojwt "github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateWithJSONClaimsPreservesValuesAndLifetime(t *testing.T) {
	signer := NewSigner("test-secret")
	before := time.Now().Unix()
	token, err := signer.GenerateWithJSONClaims(time.Hour, map[string]any{
		"sub":         "user-123",
		"factory_ids": []string{"workspace-123", "workspace-456"},
		"enabled":     true,
	})
	require.NoError(t, err)
	after := time.Now().Unix()

	parsed, err := gojwt.Parse(token, func(token *gojwt.Token) (any, error) {
		require.Equal(t, gojwt.SigningMethodHS256, token.Method)
		return []byte("test-secret"), nil
	})
	require.NoError(t, err)
	require.True(t, parsed.Valid)
	claims, ok := parsed.Claims.(gojwt.MapClaims)
	require.True(t, ok)
	assert.Equal(t, "user-123", claims["sub"])
	assert.Equal(t, []any{"workspace-123", "workspace-456"}, claims["factory_ids"])
	assert.Equal(t, true, claims["enabled"])
	for _, name := range []string{"iat", "nbf", "exp"} {
		value, ok := claims[name].(float64)
		require.True(t, ok, "claim %s must be numeric", name)
		if name == "exp" {
			value -= time.Hour.Seconds()
		}
		assert.GreaterOrEqual(t, value, float64(before), "claim %s", name)
		assert.LessOrEqual(t, value, float64(after), "claim %s", name)
	}
}

func TestGenerateWithClaimsRejectsReservedTimeClaims(t *testing.T) {
	signer := NewSigner("test-secret")
	for _, name := range []string{"iat", "nbf", "exp"} {
		t.Run(name, func(t *testing.T) {
			t.Run("JSON claims", func(t *testing.T) {
				token, err := signer.GenerateWithJSONClaims(time.Hour, map[string]any{name: time.Now().Unix()})
				require.ErrorContains(t, err, "cannot override reserved claim")
				assert.Empty(t, token)
			})
			t.Run("string claims", func(t *testing.T) {
				token, err := signer.GenerateWithClaims(time.Hour, map[string]string{name: "override"})
				require.ErrorContains(t, err, "cannot override reserved claim")
				assert.Empty(t, token)
			})
		})
	}
}

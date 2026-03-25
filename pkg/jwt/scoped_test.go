package jwt

import (
	"testing"
	"time"

	gojwt "github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAndValidateScopedToken(t *testing.T) {
	signer := NewSigner("test-secret")

	token, err := signer.GenerateScopedToken(ScopedTokenClaims{
		Subject: "user-123",
		OrgID:   "org-123",
		Purpose: "agent-builder",
		Scopes: []string{
			"canvases:read:canvas-123",
			"canvases:read:canvas-123",
			"  ",
			" org:read ",
		},
	}, time.Minute)
	require.NoError(t, err)

	parsedToken, err := gojwt.Parse(token, func(token *gojwt.Token) (interface{}, error) {
		return []byte(signer.Secret), nil
	})
	require.NoError(t, err)
	rawClaims, ok := parsedToken.Claims.(gojwt.MapClaims)
	require.True(t, ok)
	assert.Equal(t, ScopedTokenAudience, rawClaims["aud"])
	assert.Equal(t, []any{"canvases:read:canvas-123", "org:read"}, rawClaims["scopes"])
	assert.NotContains(t, rawClaims, "permissions")

	claims, err := signer.ValidateScopedToken(token)
	require.NoError(t, err)

	assert.Equal(t, ScopedTokenType, claims.TokenType)
	assert.Equal(t, "user-123", claims.Subject)
	assert.Equal(t, "org-123", claims.OrgID)
	assert.Equal(t, "agent-builder", claims.Purpose)
	assert.Equal(
		t,
		[]string{"canvases:read:canvas-123", "org:read"},
		claims.Scopes,
	)
	assert.True(t, claims.VerifyAudience(ScopedTokenAudience, true))
	assert.NotNil(t, claims.ExpiresAt)
}

func TestGenerateScopedTokenRequiresScopes(t *testing.T) {
	signer := NewSigner("test-secret")

	_, err := signer.GenerateScopedToken(ScopedTokenClaims{
		Subject: "user-123",
		OrgID:   "org-123",
		Purpose: "agent-builder",
	}, time.Minute)
	require.Error(t, err)
	assert.Equal(t, "at least one scope is required", err.Error())
}

func TestValidateScopedTokenRejectsWrongTokenType(t *testing.T) {
	signer := NewSigner("test-secret")
	now := time.Now()
	token := gojwt.NewWithClaims(gojwt.SigningMethodHS256, ScopedTokenClaims{
		Subject:   "user-123",
		Audience:  ScopedTokenAudience,
		ExpiresAt: gojwt.NewNumericDate(now.Add(time.Minute)),
		NotBefore: gojwt.NewNumericDate(now),
		IssuedAt:  gojwt.NewNumericDate(now),
		TokenType: "session",
		OrgID:     "org-123",
		Purpose:   "agent-builder",
		Scopes:    []string{"canvases:read:canvas-123"},
	})

	tokenString, err := token.SignedString([]byte(signer.Secret))
	require.NoError(t, err)

	_, err = signer.ValidateScopedToken(tokenString)
	require.Error(t, err)
	assert.Equal(t, "invalid token_type", err.Error())
}

func TestPermissionsFromScopes(t *testing.T) {
	assert.Equal(
		t,
		[]Permission{
			{ResourceType: "org", Action: "read"},
			{ResourceType: "canvases", Action: "read", Resources: []string{"canvas-123", "canvas-456"}},
			{ResourceType: "canvases", Action: "update", Resources: []string{"canvas-123"}},
		},
		PermissionsFromScopes([]string{
			"org:read",
			"canvases:read:canvas-123",
			"canvases:read:canvas-456",
			"canvases:read:canvas-123",
			"canvases:update:canvas-123",
		}),
	)
}

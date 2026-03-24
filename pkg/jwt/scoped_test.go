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
		OrgID:   "org-123",
		Purpose: "agent_chat",
		Permissions: []Permission{
			{
				ResourceType: "canvases",
				Action:       "read",
				Resources:    []string{"canvas-123", "canvas-123", "  "},
			},
			{
				ResourceType: " org ",
				Action:       " read ",
			},
			{
				ResourceType: "canvases",
				Action:       "read",
				Resources:    []string{"canvas-123"},
			},
		},
		RegisteredClaims: gojwt.RegisteredClaims{
			Subject: "user-123",
		},
	}, time.Minute)
	require.NoError(t, err)

	claims, err := signer.ValidateScopedToken(token)
	require.NoError(t, err)

	assert.Equal(t, ScopedTokenType, claims.TokenType)
	assert.Equal(t, "user-123", claims.Subject)
	assert.Equal(t, "org-123", claims.OrgID)
	assert.Equal(t, "agent_chat", claims.Purpose)
	assert.Equal(
		t,
		[]Permission{
			{
				ResourceType: "canvases",
				Action:       "read",
				Resources:    []string{"canvas-123"},
			},
			{
				ResourceType: "org",
				Action:       "read",
			},
		},
		claims.Permissions,
	)
	assert.True(t, claims.VerifyAudience(ScopedTokenAudience, true))
	assert.NotNil(t, claims.ExpiresAt)
}

func TestGenerateScopedTokenRequiresPermissions(t *testing.T) {
	signer := NewSigner("test-secret")

	_, err := signer.GenerateScopedToken(ScopedTokenClaims{
		OrgID:   "org-123",
		Purpose: "agent_chat",
		RegisteredClaims: gojwt.RegisteredClaims{
			Subject: "user-123",
		},
	}, time.Minute)
	require.Error(t, err)
	assert.Equal(t, "at least one permission is required", err.Error())
}

func TestValidateScopedTokenRejectsWrongTokenType(t *testing.T) {
	signer := NewSigner("test-secret")
	now := time.Now()
	token := gojwt.NewWithClaims(gojwt.SigningMethodHS256, ScopedTokenClaims{
		TokenType: "session",
		OrgID:     "org-123",
		Purpose:   "agent_chat",
		Permissions: []Permission{
			{
				ResourceType: "canvases",
				Action:       "read",
				Resources:    []string{"canvas-123"},
			},
		},
		RegisteredClaims: gojwt.RegisteredClaims{
			Subject:   "user-123",
			Audience:  gojwt.ClaimStrings{ScopedTokenAudience},
			IssuedAt:  gojwt.NewNumericDate(now),
			NotBefore: gojwt.NewNumericDate(now),
			ExpiresAt: gojwt.NewNumericDate(now.Add(time.Minute)),
		},
	})

	tokenString, err := token.SignedString([]byte(signer.Secret))
	require.NoError(t, err)

	_, err = signer.ValidateScopedToken(tokenString)
	require.Error(t, err)
	assert.Equal(t, "invalid token_type", err.Error())
}

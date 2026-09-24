package livelogstoken

import (
	"testing"
	"time"

	gojwt "github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"
)

func TestValidateLiveLogToken(t *testing.T) {
	secret := "live-log-secret"
	now := time.Now()
	claims := Claims{
		TaskID:  "task-123",
		Purpose: Purpose,
		RegisteredClaims: gojwt.RegisteredClaims{
			Audience:  gojwt.ClaimStrings{Audience},
			ExpiresAt: gojwt.NewNumericDate(now.Add(time.Minute)),
			IssuedAt:  gojwt.NewNumericDate(now),
			NotBefore: gojwt.NewNumericDate(now),
		},
	}
	tokenString, err := gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	require.NoError(t, err)

	require.NoError(t, Validate(tokenString, "task-123", secret))
	require.Error(t, Validate(tokenString, "other-task", secret))
}

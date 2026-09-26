package runnerregistrationtoken

import (
	"testing"
	"time"

	gojwt "github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"
)

func TestMintAndValidate(t *testing.T) {
	secret := "registration-secret"
	token, err := Mint("fleet-a", secret, time.Now().Add(time.Minute))
	require.NoError(t, err)

	claims, err := Validate(token, "fleet-a", secret)
	require.NoError(t, err)
	require.NotEmpty(t, claims.ID)
	require.Equal(t, "fleet-a", claims.FleetID)

	_, err = Validate(token, "other-fleet", secret)
	require.Error(t, err)

	other, err := Mint("fleet-a", secret, time.Now().Add(time.Minute))
	require.NoError(t, err)
	a, err := Validate(token, "fleet-a", secret)
	require.NoError(t, err)
	b, err := Validate(other, "fleet-a", secret)
	require.NoError(t, err)
	require.NotEqual(t, a.ID, b.ID)
}

func TestValidateRejectsWrongPurpose(t *testing.T) {
	secret := "registration-secret"
	now := time.Now()
	claims := Claims{
		FleetID: "fleet-a",
		Purpose: "other",
		RegisteredClaims: gojwt.RegisteredClaims{
			ID:        "jti-1",
			Audience:  gojwt.ClaimStrings{Audience},
			ExpiresAt: gojwt.NewNumericDate(now.Add(time.Minute)),
			IssuedAt:  gojwt.NewNumericDate(now),
		},
	}
	token, err := gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	require.NoError(t, err)
	_, err = Validate(token, "fleet-a", secret)
	require.Error(t, err)
}

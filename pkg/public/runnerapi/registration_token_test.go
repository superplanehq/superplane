package runnerapi

import (
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestMintRegistrationTokenUsesPersistedRegistrationClaims(t *testing.T) {
	signer := jwt.NewSigner("secret")
	createdAt := time.Now().Add(-time.Second).Truncate(time.Second)
	runner := &models.Runner{
		ID:            uuid.New(),
		FleetID:       uuid.New(),
		State:         models.RunnerStatePending,
		RunnerVersion: "0.1.0",
		CreatedAt:     createdAt,
		UpdatedAt:     createdAt,
	}
	registration := &models.RunnerRegistration{
		JTI:       uuid.New(),
		RunnerID:  runner.ID,
		ExpiresAt: createdAt.Add(10 * time.Minute),
		CreatedAt: createdAt,
	}
	fleetID := "e1-large-amd64"
	taskID := uuid.New()

	tokenValue, err := MintRegistrationToken(signer, runner, registration, fleetID, &taskID)
	require.NoError(t, err)

	token, err := jwtlib.ParseWithClaims(tokenValue, &RegistrationClaims{}, func(token *jwtlib.Token) (any, error) {
		return []byte(signer.Secret), nil
	})
	require.NoError(t, err)
	require.True(t, token.Valid)

	claims, ok := token.Claims.(*RegistrationClaims)
	require.True(t, ok)
	assert.Equal(t, runner.ID.String(), claims.Subject)
	assert.Equal(t, fleetID, claims.FleetID)
	assert.Equal(t, taskID.String(), claims.TaskID)
	assert.Equal(t, registration.JTI.String(), claims.ID)
	assert.Equal(t, RegistrationAudience, claims.Audience[0])
	assert.Equal(t, registration.ExpiresAt.Unix(), claims.ExpiresAt.Unix())
}

func TestMintRegistrationTokenIsStableForAnIdempotentRetry(t *testing.T) {
	signer := jwt.NewSigner("secret")
	createdAt := time.Now().Add(-time.Second).Truncate(time.Second)
	runner := &models.Runner{
		ID:            uuid.New(),
		FleetID:       uuid.New(),
		State:         models.RunnerStatePending,
		RunnerVersion: "0.1.0",
		CreatedAt:     createdAt,
		UpdatedAt:     createdAt,
	}
	registration := &models.RunnerRegistration{
		JTI:       uuid.New(),
		RunnerID:  runner.ID,
		ExpiresAt: createdAt.Add(10 * time.Minute),
		CreatedAt: createdAt,
	}
	fleetID := "e1-large-amd64"

	first, err := MintRegistrationToken(signer, runner, registration, fleetID, nil)
	require.NoError(t, err)
	second, err := MintRegistrationToken(signer, runner, registration, fleetID, nil)
	require.NoError(t, err)

	assert.Equal(t, first, second)
}

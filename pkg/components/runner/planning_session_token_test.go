package runner

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/jwt"
)

func TestMintAndParsePlanningSessionToken(t *testing.T) {
	t.Parallel()

	signer := jwt.NewSigner("planning-secret")
	scope := PlanningSessionScope{
		OrganizationID: uuid.New(),
		FactoryID:      uuid.New(),
		SessionID:      uuid.New(),
		CanvasRunID:    uuid.New(),
	}

	token, err := MintPlanningSessionToken(signer, scope, time.Hour)
	require.NoError(t, err)

	parsed, err := ParsePlanningSessionToken(signer, token)
	require.NoError(t, err)
	assert.Equal(t, scope, *parsed)
}

func TestParsePlanningSessionTokenRejectsWrongPurpose(t *testing.T) {
	t.Parallel()

	signer := jwt.NewSigner("planning-secret")
	token, err := signer.GenerateWithClaims(time.Hour, map[string]string{
		"purpose":       "other",
		"org_id":        uuid.New().String(),
		"factory_id":    uuid.New().String(),
		"session_id":    uuid.New().String(),
		"canvas_run_id": uuid.New().String(),
	})
	require.NoError(t, err)

	_, err = ParsePlanningSessionToken(signer, token)
	require.Error(t, err)
}

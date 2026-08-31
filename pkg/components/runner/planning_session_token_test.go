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

func TestParsePlanningSessionTokenRejectsWorkOrderPurpose(t *testing.T) {
	t.Parallel()

	signer := jwt.NewSigner("planning-secret")
	token, err := MintWorkOrderSurveyToken(signer, WorkOrderSurveyScope{
		OrganizationID: uuid.New(),
		FactoryID:      uuid.New(),
		WorkOrderID:    uuid.New(),
		CanvasRunID:    uuid.New(),
	}, time.Hour)
	require.NoError(t, err)

	_, err = ParsePlanningSessionToken(signer, token)
	require.Error(t, err)
}

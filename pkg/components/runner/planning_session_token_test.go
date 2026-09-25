package runner

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
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

func TestRunnerSuperplaneBaseURLUsesDockerHostForLocalBroker(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "http://host.docker.internal:8091")
	t.Setenv("BASE_URL", "https://dead.trycloudflare.com")
	t.Setenv("WEBHOOKS_BASE_URL", "https://dead.trycloudflare.com")
	t.Setenv("PUBLIC_API_PORT", "8000")

	assert.Equal(t, "http://host.docker.internal:8000", RunnerSuperplaneBaseURL(""))
}

func TestRunnerSuperplaneBaseURLRewritesLoopbackForLocalBroker(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "http://host.docker.internal:8091")
	t.Setenv("BASE_URL", "http://localhost:8000")
	t.Setenv("WEBHOOKS_BASE_URL", "http://host.docker.internal:8000")

	assert.Equal(t, "http://host.docker.internal:8000", RunnerSuperplaneBaseURL(""))
}

func TestRunnerSuperplaneBaseURLUsesPublicURLForRemoteBroker(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("BASE_URL", "https://app.example")
	t.Setenv("WEBHOOKS_BASE_URL", "https://hooks.example")

	assert.Equal(t, "https://hooks.example", RunnerSuperplaneBaseURL("https://app.example"))
}

func TestPlanningSessionKindDeterminesAnalysisMode(t *testing.T) {
	t.Parallel()

	analysis := &models.FactoryPlanningSession{Kind: models.PlanningSessionKindWorkOrderAnalysis}
	assert.True(t, analysis.IsAnalysisSession())
	assert.False(t, (&models.FactoryPlanningSession{Kind: models.PlanningSessionKindTaskCreation}).IsAnalysisSession())
}

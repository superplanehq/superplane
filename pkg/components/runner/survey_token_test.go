package runner

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/jwt"
)

func TestMintAndParseWorkOrderSurveyToken(t *testing.T) {
	t.Parallel()

	signer := jwt.NewSigner("survey-secret")
	scope := WorkOrderSurveyScope{
		OrganizationID: uuid.New(),
		FactoryID:      uuid.New(),
		WorkOrderID:    uuid.New(),
		CanvasRunID:    uuid.New(),
		ExecutionID:    uuid.New(),
	}

	token, err := MintWorkOrderSurveyToken(signer, scope, time.Hour)
	require.NoError(t, err)

	parsed, err := ParseWorkOrderSurveyToken(signer, token)
	require.NoError(t, err)
	assert.Equal(t, scope, *parsed)
}

func TestParseWorkOrderSurveyTokenRejectsWrongPurpose(t *testing.T) {
	t.Parallel()

	signer := jwt.NewSigner("survey-secret")
	token, err := signer.GenerateWithClaims(time.Hour, map[string]string{
		"purpose":       "runner_live_logs",
		"org_id":        uuid.New().String(),
		"factory_id":    uuid.New().String(),
		"work_order_id": uuid.New().String(),
		"canvas_run_id": uuid.New().String(),
		"execution_id":  uuid.New().String(),
	})
	require.NoError(t, err)

	_, err = ParseWorkOrderSurveyToken(signer, token)
	require.Error(t, err)
}

func TestParseWorkOrderSurveyTokenRejectsOtherSigner(t *testing.T) {
	t.Parallel()

	token, err := MintWorkOrderSurveyToken(jwt.NewSigner("a"), WorkOrderSurveyScope{
		OrganizationID: uuid.New(),
		FactoryID:      uuid.New(),
		WorkOrderID:    uuid.New(),
		CanvasRunID:    uuid.New(),
	}, time.Hour)
	require.NoError(t, err)

	_, err = ParseWorkOrderSurveyToken(jwt.NewSigner("b"), token)
	require.Error(t, err)
}

func TestWorkOrderSurveyEnvVars(t *testing.T) {
	t.Parallel()

	env := WorkOrderSurveyEnvVars("https://app.example.test", "tok-1")
	require.Len(t, env, 2)
	assert.Equal(t, EnvSuperplaneBaseURL, env[0].Name)
	assert.Equal(t, "https://app.example.test", env[0].Value)
	assert.Equal(t, EnvSuperplaneRunToken, env[1].Name)
	assert.Equal(t, "tok-1", env[1].Value)
}

func TestPublicSuperplaneBaseURLPrefersWebhooksURL(t *testing.T) {
	t.Setenv("WEBHOOKS_BASE_URL", "https://tunnel.example/")
	t.Setenv("BASE_URL", "http://localhost:8000")

	assert.Equal(t, "https://tunnel.example", PublicSuperplaneBaseURL("http://localhost:8000"))
}

func TestPublicSuperplaneBaseURLRejectsLoopback(t *testing.T) {
	t.Setenv("WEBHOOKS_BASE_URL", "")
	t.Setenv("BASE_URL", "")

	assert.Empty(t, PublicSuperplaneBaseURL("http://127.0.0.1:8000"))
	assert.Empty(t, PublicSuperplaneBaseURL("http://localhost:8000"))
}

package runner_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
)

func TestAttachWorkOrderSurveyEnvUsesContextScopeWithoutDatabaseLookup(t *testing.T) {
	t.Setenv("JWT_SECRET", "survey-secret")
	t.Setenv("WEBHOOKS_BASE_URL", "https://public.example.test")

	orgID := uuid.New()
	env := runner.AttachWorkOrderSurveyEnv(core.ExecutionContext{
		ID:             uuid.New(),
		RunID:          uuid.New(),
		FactoryID:      uuid.New(),
		WorkOrderID:    uuid.New(),
		OrganizationID: orgID.String(),
	}, nil, 3600)

	require.True(t, runner.HasWorkOrderSurveyToken(env), "survey token must attach from context scope fields")
	require.Len(t, env, 2)
	assert.Equal(t, runner.EnvSuperplaneBaseURL, env[0].Name)
	assert.Equal(t, "https://public.example.test", env[0].Value)
	assert.Equal(t, runner.EnvSuperplaneRunToken, env[1].Name)
	assert.NotEmpty(t, env[1].Value)
}

func TestAttachWorkOrderSurveyEnvSkipsWhenWorkOrderScopeIsMissing(t *testing.T) {
	t.Setenv("JWT_SECRET", "survey-secret")
	t.Setenv("WEBHOOKS_BASE_URL", "https://public.example.test")

	env := runner.AttachWorkOrderSurveyEnv(core.ExecutionContext{
		ID:             uuid.New(),
		RunID:          uuid.New(),
		OrganizationID: uuid.New().String(),
	}, nil, 3600)

	assert.False(t, runner.HasWorkOrderSurveyToken(env))
}

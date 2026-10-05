package runner_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestParseMergeConfidenceChecks(t *testing.T) {
	assert.Equal(t, []string{"risk", "drift"}, runner.ParseMergeConfidenceChecks([]string{
		"Review the diff.\nEnabled checks: risk, drift.\n",
	}))
	assert.Empty(t, runner.ParseMergeConfidenceChecks([]string{"Enabled checks: none."}))
	assert.Empty(t, runner.ParseMergeConfidenceChecks([]string{"No checks line."}))
}

func TestMergeConfidenceTokenRoundTrip(t *testing.T) {
	signer := jwt.NewSigner("merge-secret")
	scope := runner.MergeConfidenceScope{
		OrganizationID:  uuid.New(),
		FactoryID:       uuid.New(),
		WorkOrderID:     uuid.New(),
		CanvasRunID:     uuid.New(),
		NodeExecutionID: uuid.New(),
		EnabledChecks:   []string{"risk", "security"},
	}
	token, err := runner.MintMergeConfidenceToken(signer, scope, time.Hour)
	require.NoError(t, err)

	got, err := runner.ParseMergeConfidenceToken(signer, token)
	require.NoError(t, err)
	assert.Equal(t, scope, *got)
}

func TestParseMergeConfidenceTokenRejectsPlanningPurpose(t *testing.T) {
	signer := jwt.NewSigner("merge-secret")
	token, err := signer.GenerateWithClaims(time.Hour, map[string]string{
		"purpose": runner.PlanningSessionTokenPurpose,
	})
	require.NoError(t, err)

	_, err = runner.ParseMergeConfidenceToken(signer, token)
	assert.ErrorContains(t, err, "purpose")
}

func TestAttachMergeConfidenceEnvLeavesOtherRunsAlone(t *testing.T) {
	environment := []runner.BrokerEnvironmentVariable{{Name: "REPO", Value: "acme/app"}}
	got, err := runner.AttachMergeConfidenceEnv(core.ExecutionContext{}, environment, nil, 60)
	require.NoError(t, err)
	assert.Equal(t, environment, got)
}

func TestAttachMergeConfidenceEnvMintsToken(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas, _ := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "merge", "start")
	orderID := uuid.New()
	runID := uuid.New()
	executionID := uuid.New()
	t.Setenv("JWT_SECRET", "merge-secret")
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("WEBHOOKS_BASE_URL", "https://app.example")
	t.Setenv("BASE_URL", "https://app.example")

	prompt := "Enabled checks: risk, security.\n"
	environment, err := runner.AttachMergeConfidenceEnv(core.ExecutionContext{
		ID:             executionID,
		RunID:          runID,
		WorkflowID:     canvas.ID.String(),
		OrganizationID: r.Organization.ID.String(),
	}, []runner.BrokerEnvironmentVariable{
		{Name: "REPO", Value: "acme/app"},
		{Name: runner.EnvSuperplaneMergeConfidenceOrderID, Value: orderID.String()},
	}, []runner.AgentStep{{
		Name:   "Review",
		Type:   runner.AgentStepPrompt,
		Prompt: &prompt,
	}}, 60)
	require.NoError(t, err)

	assert.False(t, environmentHas(environment, runner.EnvSuperplaneMergeConfidenceOrderID))
	token, ok := environmentLookup(environment, runner.EnvSuperplaneMergeConfidenceToken)
	require.True(t, ok)
	scope, err := runner.ParseMergeConfidenceToken(jwt.NewSigner("merge-secret"), token)
	require.NoError(t, err)
	assert.Equal(t, orderID, scope.WorkOrderID)
	assert.Equal(t, factoryModel.ID, scope.FactoryID)
	assert.Equal(t, r.Organization.ID, scope.OrganizationID)
	assert.Equal(t, runID, scope.CanvasRunID)
	assert.Equal(t, executionID, scope.NodeExecutionID)
	assert.Equal(t, []string{"risk", "security"}, scope.EnabledChecks)
	baseURL, ok := environmentLookup(environment, runner.EnvSuperplaneBaseURL)
	require.True(t, ok)
	assert.Equal(t, "https://app.example", baseURL)

	withFile := runner.AppendMergeConfidenceMCP(environment, []runner.BrokerTaskFile{{Path: "run.js"}})
	require.Len(t, withFile, 2)
	assert.Equal(t, "merge_confidence_mcp.js", withFile[1].Path)
}

func TestAttachMergeConfidenceEnvRequiresFactoryCanvas(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	t.Setenv("JWT_SECRET", "merge-secret")
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("WEBHOOKS_BASE_URL", "https://app.example")

	_, err := runner.AttachMergeConfidenceEnv(core.ExecutionContext{
		ID:             uuid.New(),
		RunID:          uuid.New(),
		WorkflowID:     canvas.ID.String(),
		OrganizationID: r.Organization.ID.String(),
	}, []runner.BrokerEnvironmentVariable{
		{Name: runner.EnvSuperplaneMergeConfidenceOrderID, Value: uuid.NewString()},
	}, nil, 60)
	require.Error(t, err)
	assert.ErrorContains(t, err, "factory canvas")
}

func environmentLookup(environment []runner.BrokerEnvironmentVariable, name string) (string, bool) {
	for _, item := range environment {
		if item.Name == name {
			return item.Value, true
		}
	}
	return "", false
}

func environmentHas(environment []runner.BrokerEnvironmentVariable, name string) bool {
	_, ok := environmentLookup(environment, name)
	return ok
}

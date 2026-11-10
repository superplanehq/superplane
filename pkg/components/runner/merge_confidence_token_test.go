package runner_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
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

func TestMergeConfidenceChecksFromSteps(t *testing.T) {
	risk := "Merge check: risk.\nScore the change."
	custom := "Merge check: api-latency.\nScore the query."
	ids, labels := runner.MergeConfidenceChecksFromSteps([]runner.AgentStep{
		{Name: "Blast radius", Type: runner.AgentStepPrompt, Prompt: &risk},
		{Name: "API latency", Type: runner.AgentStepPrompt, Prompt: &custom},
	})
	assert.Equal(t, []string{"risk", "api-latency"}, ids)
	assert.Equal(t, map[string]string{"risk": "Blast radius", "api-latency": "API latency"}, labels)

	legacy := "Enabled checks: risk, drift.\n"
	ids, labels = runner.MergeConfidenceChecksFromSteps([]runner.AgentStep{
		{Name: "Review Pull Request", Type: runner.AgentStepPrompt, Prompt: &legacy},
	})
	assert.Equal(t, []string{"risk", "drift"}, ids)
	assert.Nil(t, labels)
}

func TestAttachMergeConfidenceEnvLeavesOtherRunsAlone(t *testing.T) {
	environment := []runner.TaskEnvironmentVariable{{Name: "REPO", Value: "acme/app"}}
	got, scope, err := runner.AttachMergeConfidenceEnv(core.ExecutionContext{}, environment, nil)
	require.NoError(t, err)
	assert.Equal(t, environment, got)
	assert.Nil(t, scope)
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
	t.Setenv("RUNNER_API_BASE_URL", "https://app.example")
	t.Setenv("WEBHOOKS_BASE_URL", "https://app.example")
	t.Setenv("BASE_URL", "https://app.example")

	prompt := "Enabled checks: risk, security.\n"
	environment, scope, err := runner.AttachMergeConfidenceEnv(core.ExecutionContext{
		ID:             executionID,
		RunID:          runID,
		WorkflowID:     canvas.ID.String(),
		OrganizationID: r.Organization.ID.String(),
	}, []runner.TaskEnvironmentVariable{
		{Name: "REPO", Value: "acme/app"},
		{Name: runner.EnvSuperplaneMergeConfidenceOrderID, Value: orderID.String()},
		{Name: "SUPERPLANE_MERGE_CONFIDENCE_TOKEN", Value: "stale-token"},
	}, []runner.AgentStep{{
		Name:   "Review",
		Type:   runner.AgentStepPrompt,
		Prompt: &prompt,
	}})
	require.NoError(t, err)

	assert.False(t, environmentHas(environment, runner.EnvSuperplaneMergeConfidenceOrderID))
	assert.False(t, environmentHas(environment, "SUPERPLANE_MERGE_CONFIDENCE_TOKEN"))
	require.NotNil(t, scope)
	assert.Equal(t, orderID, scope.WorkOrderID)
	assert.Equal(t, factoryModel.ID, scope.FactoryID)
	assert.Equal(t, r.Organization.ID, scope.OrganizationID)
	assert.Equal(t, runID, scope.CanvasRunID)
	assert.Equal(t, executionID, scope.NodeExecutionID)
	assert.Equal(t, []string{"risk", "security"}, scope.EnabledChecks)
	baseURL, ok := environmentLookup(environment, runner.EnvSuperplaneBaseURL)
	require.True(t, ok)
	assert.Equal(t, "https://app.example", baseURL)
}

func TestAttachMergeConfidenceEnvRequiresFactoryCanvas(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	t.Setenv("RUNNER_API_BASE_URL", "https://app.example")
	t.Setenv("WEBHOOKS_BASE_URL", "https://app.example")

	_, scope, err := runner.AttachMergeConfidenceEnv(core.ExecutionContext{
		ID:             uuid.New(),
		RunID:          uuid.New(),
		WorkflowID:     canvas.ID.String(),
		OrganizationID: r.Organization.ID.String(),
	}, []runner.TaskEnvironmentVariable{
		{Name: runner.EnvSuperplaneMergeConfidenceOrderID, Value: uuid.NewString()},
	}, nil)
	assert.Nil(t, scope)
	require.Error(t, err)
	assert.ErrorContains(t, err, "factory canvas")
}

func environmentLookup(environment []runner.TaskEnvironmentVariable, name string) (string, bool) {
	for _, item := range environment {
		if item.Name == name {
			return item.Value, true
		}
	}
	return "", false
}

func environmentHas(environment []runner.TaskEnvironmentVariable, name string) bool {
	_, ok := environmentLookup(environment, name)
	return ok
}

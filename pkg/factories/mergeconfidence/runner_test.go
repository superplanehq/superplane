package mergeconfidence_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/factories/mergeconfidence"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestParseMergeConfidenceChecks(t *testing.T) {
	assert.Equal(t, []string{"risk", "drift"}, mergeconfidence.ParseMergeConfidenceChecks([]string{
		"Review the diff.\nEnabled checks: risk, drift.\n",
	}))
	assert.Empty(t, mergeconfidence.ParseMergeConfidenceChecks([]string{"Enabled checks: none."}))
	assert.Empty(t, mergeconfidence.ParseMergeConfidenceChecks([]string{"No checks line."}))
}

func TestMergeConfidenceChecksFromSteps(t *testing.T) {
	risk := "Merge check: risk.\nScore the change."
	custom := "Merge check: api-latency.\nScore the query."
	ids, labels := mergeconfidence.MergeConfidenceChecksFromSteps([]runner.AgentStep{
		{Name: "Blast radius", Type: runner.AgentStepPrompt, Prompt: &risk},
		{Name: "API latency", Type: runner.AgentStepPrompt, Prompt: &custom},
	})
	assert.Equal(t, []string{"risk", "api-latency"}, ids)
	assert.Equal(t, map[string]string{"risk": "Blast radius", "api-latency": "API latency"}, labels)

	legacy := "Enabled checks: risk, drift.\n"
	ids, labels = mergeconfidence.MergeConfidenceChecksFromSteps([]runner.AgentStep{
		{Name: "Review Pull Request", Type: runner.AgentStepPrompt, Prompt: &legacy},
	})
	assert.Equal(t, []string{"risk", "drift"}, ids)
	assert.Nil(t, labels)
}

func TestAttachMergeConfidenceEnvLeavesOtherRunsAlone(t *testing.T) {
	environment := []runner.TaskEnvironmentVariable{{Name: "REPO", Value: "acme/app"}}
	got, scope, err := mergeconfidence.AttachMergeConfidenceEnv(nil, core.ExecutionContext{}, environment, nil)
	require.NoError(t, err)
	assert.Equal(t, environment, got)
	assert.Nil(t, scope)
}

func TestMergeConfidenceScopeKeepsCheckAndRunIdentity(t *testing.T) {
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
	environment, scope, err := mergeconfidence.AttachMergeConfidenceEnv(database.DB(t.Context()), core.ExecutionContext{
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

	_, scope, err := mergeconfidence.AttachMergeConfidenceEnv(database.DB(t.Context()), core.ExecutionContext{
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

func TestPreparedRunnerCredentialsPreserveConfiguration(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	workspace, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := workspace.CreateWorkOrder(db, "Assess PR", "", &r.User, nil, nil)
	require.NoError(t, err)
	canvas, _ := support.CreateFactoryAppWithOnRunTrigger(t, r, workspace.ID, "merge", "start")
	root := support.EmitCanvasEventForNode(t, canvas.ID, "start", "default", nil)
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, "start", root.ID, root.ID)
	t.Setenv("JWT_SECRET", "test")
	signer := jwt.NewSigner("test")
	secrets := &contexts.SecretsContext{
		IntegrationKeys: map[string]map[string][]byte{
			"merge": {runner.EnvSuperplaneMergeConfidenceOrderID: []byte(order.ID.String())},
		},
		SecretKeys: map[string]map[string][]byte{
			"merge": {runner.EnvSuperplaneMergeConfidenceOrderID: []byte(order.ID.String())},
			"stale": {runner.EnvSuperplaneMergeConfidenceOrderID: []byte("invalid-order")},
		},
	}
	explicit := []map[string]any{{"name": runner.EnvSuperplaneMergeConfidenceOrderID, "value": order.ID.String(), "valueSource": "literal"}}
	for _, tc := range []struct {
		name            string
		environment     []map[string]any
		environmentFrom []map[string]any
		timeout         any
		lifetime        time.Duration
	}{
		{name: "default timeout", environment: explicit, lifetime: 90 * time.Minute},
		{name: "string timeout", environment: explicit, timeout: "1800", lifetime: time.Hour},
		{
			name:        "padded environment name",
			environment: []map[string]any{{"name": " " + runner.EnvSuperplaneMergeConfidenceOrderID + " ", "value": order.ID.String(), "valueSource": "literal"}},
			lifetime:    90 * time.Minute,
		},
		{
			name:            "integration environment",
			environmentFrom: []map[string]any{{"source": "integration", "integration": map[string]any{"name": "merge"}}},
			lifetime:        90 * time.Minute,
		},
		{
			name:            "secret environment",
			environmentFrom: []map[string]any{{"source": "secret", "secret": map[string]any{"secret": "merge"}}},
			lifetime:        90 * time.Minute,
		},
		{
			name:            "explicit environment overrides import",
			environment:     explicit,
			environmentFrom: []map[string]any{{"source": "secret", "secret": map[string]any{"secret": "stale"}}},
			lifetime:        90 * time.Minute,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := core.ExecutionContext{ID: execution.ID, RunID: execution.RunID, WorkflowID: canvas.ID.String(),
				OrganizationID: r.Organization.ID.String(), NodeID: execution.NodeID, BaseURL: "https://app.example",
				Secrets: secrets,
				Configuration: map[string]any{
					"environment":     tc.environment,
					"environmentFrom": tc.environmentFrom,
					"prompt":          "Enabled checks: risk, security.",
				}}
			if tc.timeout != nil {
				ctx.Configuration.(map[string]any)["executionTimeoutSeconds"] = tc.timeout
			}
			require.NoError(t, mergeconfidence.PrepareRunner(db, &ctx))
			environment := runner.AttachRunnerEnv(ctx, []runner.TaskEnvironmentVariable{
				{Name: runner.EnvSuperplaneMergeConfidenceOrderID, Value: order.ID.String()},
				{Name: "SUPERPLANE_MERGE_CONFIDENCE_TOKEN", Value: "stale"},
			}, runner.DefaultExecutionTimeoutSeconds)
			require.False(t, environmentHas(environment, runner.EnvSuperplaneMergeConfidenceOrderID))
			require.False(t, environmentHas(environment, "SUPERPLANE_MERGE_CONFIDENCE_TOKEN"))
			token, ok := environmentLookup(environment, runner.EnvSuperplaneRunnerToken)
			require.True(t, ok)
			scope, err := runner.ParseRunnerToken(signer, token)
			require.NoError(t, err)
			assert.Equal(t, runner.RunnerFlowMergeConfidence, scope.Flow)
			assert.Equal(t, order.ID, scope.WorkOrderID)
			assert.Equal(t, []string{"risk", "security"}, scope.MergeChecks)
			assert.Contains(t, scope.Scopes, runner.ScopeMergeChecksWrite)
			claims, err := signer.ValidateAndGetClaims(token)
			require.NoError(t, err)
			assert.Equal(t, tc.lifetime.Seconds(), claims["exp"].(float64)-claims["iat"].(float64))
		})
	}
}

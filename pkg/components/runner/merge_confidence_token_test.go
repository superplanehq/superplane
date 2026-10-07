package runner_test

import (
	"strings"
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

func TestMergeConfidenceCheckScales(t *testing.T) {
	legacy := "Merge check: performance.\nscore is an integer from 1 to 5.\nIf no performance practice applies, report the check with 5.\n5 means the change follows every practice that applies.\n"
	current := "Merge check: risk.\nscore is an integer from 1 to 3.\n"
	custom := "Merge check: api-latency.\nScore the query.\n"
	scales := runner.MergeConfidenceCheckScales([]runner.AgentStep{
		{Name: "Performance", Type: runner.AgentStepPrompt, Prompt: &legacy},
		{Name: "Blast radius", Type: runner.AgentStepPrompt, Prompt: &current},
		{Name: "API latency", Type: runner.AgentStepPrompt, Prompt: &custom},
	})
	assert.Equal(t, map[string]int{"performance": 5, "risk": 3, "api-latency": 5}, scales)

	combined := "Enabled checks: risk, performance.\nscore is an integer from 1 to 3.\n"
	assert.Equal(t, map[string]int{"risk": 3, "performance": 3}, runner.MergeConfidenceCheckScales([]runner.AgentStep{
		{Name: "Review", Type: runner.AgentStepPrompt, Prompt: &combined},
	}))

	oldCombined := "Enabled checks: risk.\nscore is an integer from 1 to 5.\nDocumentation only = 1 (very_low). Secrets = 5 (critical).\n"
	assert.Equal(t, map[string]int{"risk": 5}, runner.MergeConfidenceCheckScales([]runner.AgentStep{
		{Name: "Review", Type: runner.AgentStepPrompt, Prompt: &oldCombined},
	}))

	withTask := strings.Join([]string{
		"Merge check: risk.",
		"The original task is the intent for this change.",
		"Task title: Keep planning scores from 1 through 5",
		"Task description: Keep planning scores from 1 through 5.",
		"Cache changes = 5 (critical).",
		"",
		"Report this check with the report_merge_check tool.",
		"score is an integer from 1 to 3.",
		"Documentation only = 1 (healthy).",
	}, "\n")
	legacyWithTask := strings.Join([]string{
		"Merge check: performance.",
		"Task title: Use 1 to 3 for planning",
		"Task description: score is an integer from 1 to 3.",
		"",
		"Report this check with the report_merge_check tool.",
		"score is an integer from 1 to 5.",
		"If no performance practice applies, report the check with 5.",
	}, "\n")
	assert.Equal(t, map[string]int{"risk": 3, "performance": 5}, runner.MergeConfidenceCheckScales([]runner.AgentStep{
		{Name: "Blast radius", Type: runner.AgentStepPrompt, Prompt: &withTask},
		{Name: "Performance", Type: runner.AgentStepPrompt, Prompt: &legacyWithTask},
	}))
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
		CheckMaxScores:  map[string]int{"risk": 3, "security": 5},
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
	assert.Equal(t, map[string]int{"risk": 5, "security": 5}, scope.CheckMaxScores)
	assert.Equal(t, 5.0, scope.CheckScoreScale("risk"))
	maxScore, ok := environmentLookup(environment, runner.EnvSuperplaneMergeConfidenceMaxScore)
	require.True(t, ok)
	assert.Equal(t, "5", maxScore)
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

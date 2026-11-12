package public

import (
	"fmt"
	"github.com/superplanehq/superplane/pkg/factories/mergeconfidence"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestRunnerMergeConfidenceCheck(t *testing.T) {
	r := support.Setup(t)
	server, signer := mustRunnerLiveLogServer(t, r)
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Retry refunds", "Stop double charges.", &r.User, nil, nil)
	require.NoError(t, err)

	const nodeID = "assess-risk"
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
		NodeID: nodeID,
		Name:   "Assess risk",
		Type:   models.NodeTypeComponent,
	}}, nil)
	require.NoError(t, db.Model(canvas).Updates(map[string]any{
		"factory_id": factoryModel.ID,
		"name":       "Merge confidence",
	}).Error)
	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, nodeID, "default", nil)
	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(db, rootEvent)
	require.NoError(t, err)
	nodeExecution := createExecutionForCanvasRun(t, run, rootEvent.ID, nodeID)
	require.NoError(t, db.Model(nodeExecution).Update("state", models.CanvasNodeExecutionStateStarted).Error)

	t.Setenv("JWT_SECRET", "test")
	ctx := core.ExecutionContext{ID: nodeExecution.ID, RunID: run.ID, WorkflowID: canvas.ID.String(), OrganizationID: r.Organization.ID.String(), NodeID: nodeID, BaseURL: "http://app.test"}
	token := runnerTokenFrom(runneraction.AttachScopedRunnerEnv(db, ctx, nil, 60, &runneraction.RunnerScope{
		CanvasID: canvas.ID, NodeID: nodeID, Flow: runneraction.RunnerFlowMergeConfidence,
		OrganizationID:  r.Organization.ID,
		FactoryID:       factoryModel.ID,
		WorkOrderID:     order.ID,
		CanvasRunID:     run.ID,
		NodeExecutionID: nodeExecution.ID,
		MergeChecks:     []string{"risk"},
	}))
	require.NotEmpty(t, token)
	parsed, err := runneraction.ParseRunnerToken(signer, token)
	require.NoError(t, err)
	assert.Equal(t, order.ID, parsed.WorkOrderID)
	owners := suggestRunnerOwners(t, server, token, `{}`)
	require.Equal(t, http.StatusOK, owners.Code, owners.Body.String())
	assert.NotEmpty(t, decodeSuggestOwners(t, owners).Members)

	unknownScope := *parsed
	unknownScope.CanvasRunID = uuid.New()
	unknownRun, err := runneraction.MintRunnerToken(signer, unknownScope, time.Hour)
	require.NoError(t, err)

	const reportPath = "/api/v1/runner/task/merge-checks"
	const reportBody = `{"check":"risk","score":4,"summary":"Higher risk because the pull request raises the limit."}`
	unauthorized := runnerRequest(t, server, http.MethodPost, "", reportPath, reportBody)
	assert.Equal(t, http.StatusUnauthorized, unauthorized.Code)

	disabled := runnerRequest(t, server, http.MethodPost, token, reportPath, `{"check":"performance","score":5,"summary":"No performance practice applies."}`)
	assert.Equal(t, http.StatusBadRequest, disabled.Code)

	missingRun := runnerRequest(t, server, http.MethodPost, unknownRun, reportPath, reportBody)
	assert.Equal(t, http.StatusForbidden, missingRun.Code)

	report := runnerRequest(t, server, http.MethodPost, token, reportPath, reportBody)
	require.Equal(t, http.StatusOK, report.Code, report.Body.String())

	checks, err := order.ListChecks(db)
	require.NoError(t, err)
	require.Len(t, checks, 1)
	assert.Equal(t, "risk-review", checks[0].Key)
	assert.Equal(t, "Blast radius", checks[0].Name)
	assert.Equal(t, 4.0, checks[0].Score)
	assert.Equal(t, models.FactoryWorkOrderCheckLevelCritical, checks[0].Level)
	assert.Equal(t, "Higher risk because the pull request raises the limit.", checks[0].Summary)
	require.NotNil(t, checks[0].RunID)
	assert.Equal(t, run.ID, *checks[0].RunID)
	automation, err := checks[0].AutomationRef()
	require.NoError(t, err)
	require.NotNil(t, automation)
	assert.Equal(t, canvas.ID, automation.AppID)
	assert.Equal(t, "Merge confidence", automation.AppName)
	assert.Equal(t, nodeID, automation.NodeID)
	assert.Equal(t, "Assess risk", automation.NodeName)

	require.NoError(t, db.Model(nodeExecution).Update("state", models.CanvasNodeExecutionStateFinished).Error)
	ownerRec := assignRunnerOwner(t, server, token, `{"user":"`+r.User.String()+`","reason":"Owns pkg."}`)
	assert.Equal(t, http.StatusForbidden, ownerRec.Code)
	late := runnerRequest(t, server, http.MethodPost, token, reportPath, `{"check":"risk","score":1,"summary":"Lower risk because the pull request is introducing user interface changes only."}`)
	assert.Equal(t, http.StatusForbidden, late.Code)

	checks, err = order.ListChecks(db)
	require.NoError(t, err)
	require.Len(t, checks, 1)
	assert.Equal(t, 4.0, checks[0].Score)
}

func TestRunnerMergeConfidenceCheckKeepsEachRepositoryApart(t *testing.T) {
	pullRequestEvents := map[string]func(repository string, number int) map[string]any{
		"github": func(repository string, number int) map[string]any {
			return map[string]any{
				"type": "github.pullRequest",
				"data": map[string]any{"repository": map[string]any{"full_name": repository}, "pull_request": map[string]any{"number": number}},
			}
		},
		"bitbucket": func(repository string, number int) map[string]any {
			return map[string]any{
				"type": "bitbucket.pullRequest",
				"data": map[string]any{"repository": map[string]any{"full_name": repository}, "pullrequest": map[string]any{"id": number}},
			}
		},
	}
	for provider, pullRequestEvent := range pullRequestEvents {
		t.Run(provider, func(t *testing.T) {
			testRunnerMergeConfidenceCheckKeepsEachRepositoryApart(t, pullRequestEvent)
		})
	}
}

func testRunnerMergeConfidenceCheckKeepsEachRepositoryApart(t *testing.T, pullRequestEvent func(repository string, number int) map[string]any) {
	r := support.Setup(t)
	server, _ := mustRunnerLiveLogServer(t, r)
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Add AGENTS.md", "Add one to each repository.", &r.User, nil, nil)
	require.NoError(t, err)

	const nodeID = "assess-risk"
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
		NodeID: nodeID,
		Name:   "Assess risk",
		Type:   models.NodeTypeComponent,
	}}, nil)
	require.NoError(t, db.Model(canvas).Update("factory_id", factoryModel.ID).Error)
	t.Setenv("JWT_SECRET", "test")
	t.Setenv("RUNNER_API_BASE_URL", "http://app.test")
	t.Setenv("WEBHOOKS_BASE_URL", "http://app.test")
	t.Setenv("BASE_URL", "http://app.test")

	tokenForPullRequest := func(repository string, number int) string {
		rootEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, nodeID, "default", nil, pullRequestEvent(repository, number))
		run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(db, rootEvent)
		require.NoError(t, err)
		execution := createExecutionForCanvasRun(t, run, rootEvent.ID, nodeID)
		require.NoError(t, db.Model(execution).Update("state", models.CanvasNodeExecutionStateStarted).Error)
		ctx := core.ExecutionContext{ID: execution.ID, RunID: run.ID, WorkflowID: canvas.ID.String(), OrganizationID: r.Organization.ID.String(), NodeID: nodeID, BaseURL: "http://app.test"}
		prompt := "Merge check: risk.\nScore the diff."
		ctx.Configuration = map[string]any{"environment": []map[string]any{{"name": runneraction.EnvSuperplaneMergeConfidenceOrderID, "value": order.ID.String(), "valueSource": "literal"}}, "steps": []runneraction.AgentStep{{Name: "Blast radius", Type: runneraction.AgentStepPrompt, Prompt: &prompt}}, "executionTimeoutSeconds": 60}
		require.NoError(t, mergeconfidence.PrepareRunner(db, &ctx))
		return runnerTokenFrom(runneraction.AttachRunnerEnv(ctx, nil, 60))
	}

	const reportPath = "/api/v1/runner/task/merge-checks"
	report := func(token string, score int) {
		body := fmt.Sprintf(`{"check":"risk","score":%d,"summary":"Lower risk because the pull request adds documentation only."}`, score)
		rec := runnerRequest(t, server, http.MethodPost, token, reportPath, body)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	}

	apiToken := tokenForPullRequest("acme/api", 6)
	docsToken := tokenForPullRequest("acme/docs", 5)
	report(apiToken, 4)
	report(docsToken, 1)
	report(apiToken, 2)

	checks, err := order.ListChecks(db)
	require.NoError(t, err)
	require.Len(t, checks, 2)
	byRepository := map[string]models.FactoryWorkOrderCheck{}
	for _, check := range checks {
		byRepository[check.Repository] = check
	}
	assert.Equal(t, 2.0, byRepository["acme/api"].Score)
	require.NotNil(t, byRepository["acme/api"].PreviousScore)
	assert.Equal(t, 4.0, *byRepository["acme/api"].PreviousScore)
	assert.Equal(t, 1.0, byRepository["acme/docs"].Score)
	assert.Nil(t, byRepository["acme/docs"].PreviousScore)
}

package public

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/blob/filesystem"
	runner "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestRunnerGlobalPlanUpdate(t *testing.T) {
	r := support.Setup(t)
	t.Setenv("JWT_SECRET", "test")
	server, session, factoryModel, planningToken := mustPlanningRunnerSession(t, r)
	db := database.DB(t.Context())
	token := mustPlanningGlobalRunnerToken(t, r, session, planningToken)

	assert.Equal(t, append(append([]string{}, everyGlobalRunnerTool...), "propose_update", "inspect_attachment"), listRunnerTools(t, server, token))

	rec := runnerRequest(t, server, http.MethodPut, token, "/api/v1/runner/task/plan", `{
		"scores": {
			"clarity": {"score": 3, "summary": "Outcome, scope, and done are defined."},
			"complexity": {"score": 2, "summary": "One agent can finish this in one run."},
			"verifiability": {"score": 3, "summary": "Existing tests cover the change."}
		},
		"spec": "# Retry refunds\n\nStop double charges.\n"
	}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.JSONEq(t, `{"status":"shown"}`, rec.Body.String())

	require.NotNil(t, session.DraftWorkOrderID)
	order, err := factoryModel.FindWorkOrder(db, *session.DraftWorkOrderID)
	require.NoError(t, err)
	checks, err := order.ListChecks(db)
	require.NoError(t, err)
	assert.Len(t, checks, 3)
	artifacts, err := order.ListArtifacts(db)
	require.NoError(t, err)
	require.NotEmpty(t, artifacts)
	assert.Contains(t, string(artifacts[0].Data), "Stop double charges.")

	ctx, _, _ := mustTaskLinkedAgentRun(t, r)
	taskToken := runnerTokenFrom(runner.AttachRunnerEnv(ctx, nil, 60, nil))
	rec = runnerRequest(t, server, http.MethodPut, taskToken, "/api/v1/runner/task/plan", `{"spec":"# Plan"}`)
	assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	rec = runnerRequest(t, server, http.MethodPut, planningToken, "/api/v1/runner/task/plan", `{"spec":"# Plan"}`)
	assert.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())
}

func TestRunnerGlobalMergeCheck(t *testing.T) {
	r := support.Setup(t)
	t.Setenv("JWT_SECRET", "test")
	signer := jwt.NewSigner("test")
	server := newRunnerArtifactTestServer(t, r, signer)
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Retry refunds", "Stop double charges.", &r.User, nil, nil)
	require.NoError(t, err)
	const nodeID = "assess-risk"
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{NodeID: nodeID, Name: "Assess risk", Type: models.NodeTypeComponent}}, nil)
	require.NoError(t, db.Model(canvas).Update("factory_id", factoryModel.ID).Error)
	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, nodeID, "default", nil)
	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(db, rootEvent)
	require.NoError(t, err)
	execution := createExecutionForCanvasRun(t, run, rootEvent.ID, nodeID)
	require.NoError(t, db.Model(execution).Update("state", models.CanvasNodeExecutionStateStarted).Error)
	ctx := core.ExecutionContext{ID: execution.ID, RunID: run.ID, WorkflowID: canvas.ID.String(), OrganizationID: r.Organization.ID.String(), NodeID: nodeID, BaseURL: "http://app.test"}
	token := runnerTokenFrom(runner.AttachRunnerEnv(ctx, nil, 60, &runner.MergeConfidenceScope{
		OrganizationID:  r.Organization.ID,
		FactoryID:       factoryModel.ID,
		WorkOrderID:     order.ID,
		CanvasRunID:     run.ID,
		NodeExecutionID: execution.ID,
		EnabledChecks:   []string{"api-latency"},
		CheckLabels:     map[string]string{"api-latency": "API latency"},
	}))
	require.NotEmpty(t, token)

	assert.Equal(t, append(append([]string{}, everyGlobalRunnerTool...), "report_merge_check"), listRunnerTools(t, server, token))

	rec := runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/merge-checks", `{"check":"risk","score":4,"summary":"Risk is not enabled."}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	rec = runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/merge-checks", `{"check":"api-latency","score":2,"summary":"The change adds one query per request."}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	checks, err := order.ListChecks(db)
	require.NoError(t, err)
	require.Len(t, checks, 1)
	assert.Equal(t, "API latency", checks[0].Name)
	assert.Equal(t, "The change adds one query per request.", checks[0].Summary)
	require.NotNil(t, checks[0].RunID)
	assert.Equal(t, run.ID, *checks[0].RunID)

	taskCtx, _, _ := mustTaskLinkedAgentRun(t, r)
	taskToken := runnerTokenFrom(runner.AttachRunnerEnv(taskCtx, nil, 60, nil))
	rec = runnerRequest(t, server, http.MethodPost, taskToken, "/api/v1/runner/task/merge-checks", `{"check":"api-latency","score":2,"summary":"Wrong flow."}`)
	assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
}

func TestRunnerGlobalArtifactUpload(t *testing.T) {
	r := support.Setup(t)
	t.Setenv("JWT_SECRET", "test")
	t.Setenv("BASE_URL", "http://files.test")
	store, err := filesystem.New(t.TempDir())
	require.NoError(t, err)
	blob.SetCurrent(store)
	t.Cleanup(func() { blob.SetCurrent(nil) })
	server := newRunnerArtifactTestServer(t, r, jwt.NewSigner("test"))
	ctx, _, order := mustTaskLinkedAgentRun(t, r)

	env := runner.AttachArtifactUploadEnv(ctx, nil, 60, true)
	require.True(t, runner.HasVisualEvidence(env))
	token := runnerTokenFrom(runner.AttachRunnerEnv(ctx, env, 60, nil))
	require.NotEmpty(t, token)
	assert.Equal(t, append(append([]string{}, everyGlobalRunnerTool...),
		"inspect_screenshot", "upload_artifact", "report_visual_evidence_unavailable"), listRunnerTools(t, server, token))

	png := append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, 16)...)
	rec := postRunnerArtifactFile(t, server, token, png)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var uploaded runnerArtifactResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &uploaded))
	assert.Equal(t, "Checkout", uploaded.Title)
	artifacts, err := order.ListArtifacts(database.DB(t.Context()))
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	assert.Equal(t, models.FactoryWorkOrderArtifactTypeFile, artifacts[0].Type)

	links := runnerTokenFrom(runner.AttachRunnerEnv(ctx, []runner.TaskEnvironmentVariable{
		{Name: runner.EnvSuperplaneArtifactToken, Value: "artifact-token"},
		{Name: runner.EnvSuperplanePublishLinks, Value: "yes"},
	}, 60, nil))
	rec = postRunnerArtifactFile(t, server, links, png)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	withoutEvidence := runnerTokenFrom(runner.AttachRunnerEnv(ctx, nil, 60, nil))
	rec = postRunnerArtifactFile(t, server, withoutEvidence, png)
	assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
}

func TestRunnerGlobalArtifactLink(t *testing.T) {
	r := support.Setup(t)
	t.Setenv("JWT_SECRET", "test")
	signer := jwt.NewSigner("test")
	server := newRunnerArtifactTestServer(t, r, signer)
	ctx, _, order := mustTaskLinkedAgentRun(t, r)
	token := runnerTokenFrom(runner.AttachRunnerEnv(ctx, []runner.TaskEnvironmentVariable{
		{Name: runner.EnvSuperplaneArtifactToken, Value: "artifact-token"},
		{Name: runner.EnvSuperplanePublishLinks, Value: "yes"},
	}, 60, nil))
	require.NotEmpty(t, token)
	assert.Equal(t, append(append([]string{}, everyGlobalRunnerTool...), "publish_link"), listRunnerTools(t, server, token))

	body := `{"url":"https://staging.example.com/pr-42","title":"Staging","key":"staging"}`
	rec := runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/links", body)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	artifacts, err := order.ListArtifacts(database.DB(t.Context()))
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	assert.Equal(t, models.FactoryWorkOrderArtifactTypeLink, artifacts[0].Type)
	require.NotNil(t, artifacts[0].Key)
	assert.Equal(t, order.ID.String()+":staging", *artifacts[0].Key)

	markdown := `{"body":"# Note","title":"Note","key":"note"}`
	rec = runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/markdown", markdown)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	withoutLinks := runnerTokenFrom(runner.AttachRunnerEnv(ctx, nil, 60, nil))
	rec = runnerRequest(t, server, http.MethodPost, withoutLinks, "/api/v1/runner/task/links", body)
	assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	rec = runnerRequest(t, server, http.MethodPost, withoutLinks, "/api/v1/runner/task/markdown", markdown)
	assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
}

func mustPlanningGlobalRunnerToken(t *testing.T, r *support.ResourceRegistry, session *models.FactoryPlanningSession, planningToken string) string {
	t.Helper()
	db := database.DB(t.Context())
	require.NotNil(t, session.CanvasID)
	run, err := models.FindUnscopedCanvasRun(db, *session.CanvasRunID)
	require.NoError(t, err)
	event := support.EmitCanvasEventForNode(t, *session.CanvasID, "start", "default", nil)
	execution := createExecutionForCanvasRun(t, run, event.ID, "start")
	ctx := core.ExecutionContext{
		ID: execution.ID, RunID: run.ID, WorkflowID: session.CanvasID.String(),
		OrganizationID: r.Organization.ID.String(), NodeID: "start", BaseURL: "http://app.test",
	}
	token := runnerTokenFrom(runner.AttachRunnerEnv(ctx, []runner.TaskEnvironmentVariable{
		{Name: runner.EnvSuperplanePlanningID, Value: session.ID.String()},
		{Name: runner.EnvSuperplaneRunToken, Value: planningToken},
	}, 60, nil))
	require.NotEmpty(t, token)
	return token
}

func postRunnerArtifactFile(t *testing.T, server *Server, token string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/runner/task/artifacts", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "image/png")
	request.Header.Set("Content-Disposition", `attachment; filename="checkout.png"`)
	request.Header.Set("X-SuperPlane-Artifact-Title", "Checkout")
	response := httptest.NewRecorder()
	server.Router.ServeHTTP(response, request)
	return response
}

package public

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	runner "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

// noTaskRunnerToken mints the token of a webhook run of an automation in column.
func noTaskRunnerToken(t *testing.T, order *models.FactoryWorkOrder, column string) string {
	t.Helper()
	return customAutomationRunnerToken(t, order, column, "scheduled", map[string]any{"key": "value"})
}

// manualTaskRunnerToken mints the token of a manual run on order, as the task
// card starts it. The run is not a line step.
func manualTaskRunnerToken(t *testing.T, order *models.FactoryWorkOrder, column string) string {
	t.Helper()
	return customAutomationRunnerToken(t, order, column, "manual", map[string]any{"data": map[string]any{"workOrder": map[string]any{"id": order.ID.String()}}})
}

func customAutomationRunnerToken(t *testing.T, order *models.FactoryWorkOrder, column, startMode string, data map[string]any) string {
	t.Helper()
	db := database.DB(t.Context())
	var owner models.Canvas
	require.NoError(t, db.Where("organization_id = ? AND created_by IS NOT NULL", order.OrganizationID).First(&owner).Error)
	canvas, _ := support.CreateCanvas(t, order.OrganizationID, *owner.CreatedBy, []models.CanvasNode{{
		NodeID: "agent", Name: "Run webhook", Type: models.NodeTypeComponent,
		Metadata: datatypes.NewJSONType(map[string]any{
			models.FactoryAppTemplateMetadataKey: map[string]any{"id": models.FactoryAppTemplateCustomAutomationID, "startMode": startMode},
		}),
	}}, nil)
	require.NoError(t, db.Model(canvas).Updates(map[string]any{"factory_id": order.FactoryID, "column_key": column}).Error)
	event := support.EmitCanvasEventForNodeWithData(t, canvas.ID, "agent", "default", nil, data)
	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(db, event)
	require.NoError(t, err)
	execution := createExecutionForCanvasRun(t, run, event.ID, "agent")
	token := runnerTokenFrom(runner.AttachRunnerEnv(core.ExecutionContext{
		ID: execution.ID, RunID: run.ID, WorkflowID: canvas.ID.String(),
		OrganizationID: order.OrganizationID.String(), NodeID: "agent", BaseURL: "http://app.test",
	}, nil, 60))
	require.NotEmpty(t, token)
	return token
}

// waitInStage leaves the task in stage after its stage work passed, as a
// stage with auto advance off does.
func waitInStage(t *testing.T, dispatch *models.FactoryWorkOrderLineDispatch, stage string) {
	t.Helper()
	db := database.DB(t.Context())
	require.NoError(t, db.Model(&models.FactoryWorkOrderExecution{}).Where("line_dispatch_id = ?", dispatch.ID).
		Updates(map[string]any{"status": models.FactoryWorkOrderExecutionStatusFinished, "result": models.CanvasRunResultPassed}).Error)
	require.NoError(t, dispatch.EnterStage(db, stage))
	require.NoError(t, dispatch.Finish(db, models.CanvasRunResultPassed))
}

func TestRunnerMoveTaskFromWebhookRunMovesOnlyTasksWaitingInItsStage(t *testing.T) {
	server, taskToken, order, dispatch, _ := moveTaskFixture(t)
	db := database.DB(t.Context())
	factory, err := models.FindFactory(db, order.OrganizationID, order.FactoryID)
	require.NoError(t, err)
	key := factory.WorkOrderKey(order.Number)
	token := noTaskRunnerToken(t, order, "custom-wait")

	rec := runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/move", `{"task":"`+key+`","stage":"custom-review"}`)
	assert.Equal(t, http.StatusConflict, rec.Code, "a running stage is not waiting: %s", rec.Body.String())

	waitInStage(t, dispatch, "custom-wait")
	rec = runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/move", `{"stage":"custom-review"}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	other := noTaskRunnerToken(t, order, "custom-review")
	rec = runnerRequest(t, server, http.MethodPost, other, "/api/v1/runner/task/move", `{"task":"`+key+`","stage":"custom-review"}`)
	assert.Equal(t, http.StatusConflict, rec.Code, "another stage cannot move the task: %s", rec.Body.String())
	stranger, err := factory.CreateWorkOrder(db, "Other task", "", nil, nil, nil)
	require.NoError(t, err)
	rec = runnerRequest(t, server, http.MethodPost, taskToken, "/api/v1/runner/task/move", `{"task":"`+stranger.ID.String()+`","next":true}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code, "a task run cannot move another task: %s", rec.Body.String())

	rec = runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/move", `{"task":"`+key+`","stage":"custom-review"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NoError(t, db.First(dispatch, "id = ?", dispatch.ID).Error)
	assert.True(t, dispatch.IsActive())
	assert.Equal(t, "custom-review", dispatch.CurrentStageKey)
	var started []models.FactoryWorkOrderExecution
	require.NoError(t, db.Where("line_dispatch_id = ? AND step_index = 2", dispatch.ID).Find(&started).Error)
	require.Len(t, started, 1)
	assert.Equal(t, models.FactoryWorkOrderExecutionStatusPending, started[0].Status)
}

func TestRunnerMoveTaskFromManualTaskRunMovesItsWaitingTask(t *testing.T) {
	server, _, order, dispatch, _ := moveTaskFixture(t)
	db := database.DB(t.Context())
	token := manualTaskRunnerToken(t, order, "custom-wait")

	rec := runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/move", `{"next":true}`)
	assert.Equal(t, http.StatusConflict, rec.Code, "a running stage is not waiting: %s", rec.Body.String())

	waitInStage(t, dispatch, "custom-wait")
	rec = runnerRequest(t, server, http.MethodPost, manualTaskRunnerToken(t, order, "custom-review"), "/api/v1/runner/task/move", `{"next":true}`)
	assert.Equal(t, http.StatusConflict, rec.Code, "an automation of another stage cannot move the task: %s", rec.Body.String())

	rec = runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/move", `{"next":true}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NoError(t, db.First(order, "id = ?", order.ID).Error)
	assert.Equal(t, models.FactoryWorkOrderStateClosed, order.State, "the stage after Awaiting approval is Done")
}

func TestRunnerAttachTaskLinksWebhookRunToWaitingTask(t *testing.T) {
	server, taskToken, order, dispatch, _ := moveTaskFixture(t)
	db := database.DB(t.Context())
	factory, err := models.FindFactory(db, order.OrganizationID, order.FactoryID)
	require.NoError(t, err)
	body := `{"task":"` + factory.WorkOrderKey(order.Number) + `"}`
	token := noTaskRunnerToken(t, order, "custom-wait")

	rec := runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/attach", body)
	assert.Equal(t, http.StatusConflict, rec.Code, "a running stage is not waiting: %s", rec.Body.String())
	rec = runnerRequest(t, server, http.MethodPost, taskToken, "/api/v1/runner/task/attach", body)
	assert.Equal(t, http.StatusForbidden, rec.Code, "a task run cannot attach: %s", rec.Body.String())

	waitInStage(t, dispatch, "custom-wait")
	rec = runnerRequest(t, server, http.MethodPost, noTaskRunnerToken(t, order, "custom-review"), "/api/v1/runner/task/attach", body)
	assert.Equal(t, http.StatusConflict, rec.Code, "another stage cannot attach: %s", rec.Body.String())
	for range 2 {
		rec = runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/attach", body)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	}
	var task runnerTaskDetailView
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &task))
	assert.Equal(t, "custom-wait", task.Stage.Key)

	attached, err := models.ListWorkOrderAttachedRuns(db, order.ID)
	require.NoError(t, err)
	require.Len(t, attached, 1)
	assert.Equal(t, "custom-wait", attached[0].StageKey)
	assert.NotEmpty(t, attached[0].CanvasName)
}

func TestRunnerGetTaskFindsTaskByReference(t *testing.T) {
	server, _, order, dispatch, _ := moveTaskFixture(t)
	db := database.DB(t.Context())
	factory, err := models.FindFactory(db, order.OrganizationID, order.FactoryID)
	require.NoError(t, err)
	pullRequest, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{URL: "https://github.com/acme/app/pull/42", Title: "Fix retries"})
	require.NoError(t, err)
	sha := "0123456789abcdef0123456789abcdef01234567"
	_, err = pullRequest.ObserveRevision(db, sha)
	require.NoError(t, err)
	other, err := factory.CreateWorkOrder(db, "Other repository", "", nil, nil, nil)
	require.NoError(t, err)
	_, err = other.CreatePullRequest(db, models.FactoryPullRequestParams{URL: "https://github.com/acme/api/pull/42"})
	require.NoError(t, err)
	waitInStage(t, dispatch, "custom-wait")
	token := noTaskRunnerToken(t, order, "custom-wait")
	key := factory.WorkOrderKey(order.Number)

	for _, query := range []url.Values{
		{"task": {key}},
		{"pull_request": {"https://github.com/acme/app/pull/42"}},
		{"number": {"42"}, "repository": {"acme/app"}},
		{"commit": {sha}},
	} {
		rec := runnerRequest(t, server, http.MethodGet, token, "/api/v1/runner/task/get?"+query.Encode(), "")
		require.Equal(t, http.StatusOK, rec.Code, "%v: %s", query, rec.Body.String())
		var task runnerTaskDetailView
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &task))
		assert.Equal(t, key, task.Key, query)
		assert.Equal(t, &runnerStageView{Key: "custom-wait", Name: "Awaiting approval"}, task.Stage, query)
		assert.Equal(t, &runnerStageView{Key: "done", Name: "Done"}, task.NextStage, query)
		assert.Equal(t, &runnerStageView{Key: "custom-review", Name: "Review"}, task.PreviousStage, query)
		require.Len(t, task.PullRequests, 1, query)
		assert.Equal(t, int64(42), task.PullRequests[0].Number)
		assert.Equal(t, sha, task.PullRequests[0].HeadSHA)
	}

	rec := runnerRequest(t, server, http.MethodGet, token, "/api/v1/runner/task/get?number=42", "")
	assert.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), factory.WorkOrderKey(other.Number))
	for _, query := range []string{"", "task=" + key + "&commit=" + sha, "repository=acme/app", "number=-1"} {
		rec := runnerRequest(t, server, http.MethodGet, token, "/api/v1/runner/task/get?"+query, "")
		assert.Equal(t, http.StatusBadRequest, rec.Code, query)
	}
	rec = runnerRequest(t, server, http.MethodGet, token, "/api/v1/runner/task/get?task="+uuid.NewString(), "")
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

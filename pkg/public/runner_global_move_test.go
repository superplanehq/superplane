package public

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	runner "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func moveTaskFixture(t *testing.T) (*Server, string, *models.FactoryWorkOrder, *models.FactoryWorkOrderLineDispatch, core.ExecutionContext) {
	t.Helper()
	r := support.Setup(t)
	t.Setenv("JWT_SECRET", "test")
	ctx, factory, order := mustTaskLinkedAgentRun(t, r)
	db := database.DB(t.Context())
	_, err := order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateOpen})
	require.NoError(t, err)
	dispatch, err := order.FindActiveLineDispatch(db)
	require.NoError(t, err)
	app, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "Review", "start")
	sibling, siblingEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "Source", "start")
	dispatch.Steps = datatypes.NewJSONSlice([]models.FactoryLineStep{
		{AppID: uuid.MustParse(ctx.WorkflowID), Entrypoint: "agent", StageKey: "implement"},
		{AppID: sibling.ID, Entrypoint: siblingEntry, StageKey: "implement"},
		{AppID: app.ID, Entrypoint: entry, StageKey: "custom-review"},
	})
	no := false
	one := 1
	dispatch.Steps[2].MaxParallelism = &one
	dispatch.Stages = datatypes.NewJSONSlice([]models.FactoryLineStage{
		{Key: "backlog", Name: "Backlog", Kind: models.FactoryLineStageBacklog},
		{Key: "implement", Name: "Implement", Kind: models.FactoryLineStagePhase},
		{Key: "custom-review", Name: "Review", Kind: models.FactoryLineStagePhase},
		{Key: "custom-wait", Name: "Awaiting approval", Kind: models.FactoryLineStagePhase, AutoAdvance: &no},
		{Key: "done", Name: "Done", Kind: models.FactoryLineStageDone},
	})
	require.NoError(t, db.Save(dispatch).Error)
	require.NoError(t, dispatch.EnterStage(db, "implement"))
	return newRunnerArtifactTestServer(t, r, jwt.NewSigner("test")), runnerTokenFrom(runner.AttachRunnerEnv(ctx, nil, 60)), order, dispatch, ctx
}

func TestRunnerMoveTaskReplacesSourceWorkAndRejectsSupersededCallers(t *testing.T) {
	for _, body := range []string{`{"next":true}`, `{"stage":"custom-review"}`} {
		t.Run(body, func(t *testing.T) {
			server, token, order, dispatch, ctx := moveTaskFixture(t)
			db := database.DB(t.Context())
			sibling, err := dispatch.StartStep(db, order, 1)
			require.NoError(t, err)
			queue := &models.FactoryWorkOrderQueueItem{ID: uuid.New(), OrganizationID: order.OrganizationID, FactoryID: order.FactoryID, WorkOrderID: order.ID, LineID: dispatch.LineID, LineDispatchID: dispatch.ID, AppID: dispatch.Steps[0].AppID, StepIndex: 0, StageKey: "implement", CreatedAt: time.Now()}
			require.NoError(t, db.Create(queue).Error)
			rec := runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/move", body)
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			var moved struct {
				ID, State, Stage string
				FromStage        string `json:"from_stage"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &moved))
			assert.Equal(t, order.ID.String(), moved.ID)
			assert.Equal(t, "open", moved.State)
			assert.Equal(t, "implement", moved.FromStage)
			assert.Equal(t, "custom-review", moved.Stage)
			run, err := models.FindUnscopedCanvasRun(db, sibling.Run.ID)
			require.NoError(t, err)
			assert.Equal(t, models.CanvasRunStateCancelling, run.State)
			caller, err := models.FindUnscopedCanvasRun(db, ctx.RunID)
			require.NoError(t, err)
			assert.NotEqual(t, models.CanvasRunStateCancelling, caller.State)
			var queued int64
			require.NoError(t, db.Model(&models.FactoryWorkOrderQueueItem{}).Where("line_dispatch_id = ? AND stage_key = ?", dispatch.ID, "implement").Count(&queued).Error)
			assert.Zero(t, queued)
			var destination []models.FactoryWorkOrderExecution
			require.NoError(t, db.Where("line_dispatch_id = ? AND step_index = 2", dispatch.ID).Find(&destination).Error)
			require.Len(t, destination, 1)
			assert.Equal(t, models.FactoryWorkOrderExecutionStatusPending, destination[0].Status)
			rec = runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/move", `{"next":true}`)
			assert.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
			// Returning to the old stage must not restore the old run's authority.
			require.NoError(t, dispatch.EnterStage(db, "implement"))
			rec = runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/move", `{"next":true}`)
			assert.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		})
	}
}

func TestRunnerMoveTaskValidationAndLifecycle(t *testing.T) {
	for _, tc := range []struct {
		body, state, stage string
		status             int
	}{
		{`{}`, "open", "implement", http.StatusBadRequest},
		{`{"stage":"done","next":true}`, "open", "implement", http.StatusBadRequest},
		{`{"previous":true,"next":true}`, "open", "implement", http.StatusBadRequest},
		{`{"next":false}`, "open", "implement", http.StatusBadRequest},
		{`{"stage":"unknown"}`, "open", "implement", http.StatusBadRequest},
		{`{"stage":"implement"}`, "open", "implement", http.StatusOK},
		{`{"stage":"backlog"}`, "draft", "backlog", http.StatusOK},
		{`{"previous":true}`, "draft", "backlog", http.StatusOK},
		{`{"stage":"done"}`, "closed", "done", http.StatusOK},
		{`{"stage":" awaiting APPROVAL "}`, "open", "custom-wait", http.StatusOK},
		{`{"column":"custom-wait"}`, "open", "custom-wait", http.StatusOK},
	} {
		t.Run(tc.body, func(t *testing.T) {
			server, token, order, dispatch, ctx := moveTaskFixture(t)
			db := database.DB(t.Context())
			rec := runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/move", tc.body)
			require.Equal(t, tc.status, rec.Code, rec.Body.String())
			require.NoError(t, db.First(order, "id = ?", order.ID).Error)
			require.NoError(t, db.First(dispatch, "id = ?", dispatch.ID).Error)
			assert.Equal(t, tc.state, order.State)
			assert.Equal(t, tc.stage, dispatch.CurrentStageKey)
			if tc.stage == "implement" {
				execution, err := models.FindWorkOrderExecutionByRunID(db, ctx.RunID)
				require.NoError(t, err)
				assert.Equal(t, models.FactoryWorkOrderExecutionStatusRunning, execution.Status)
			}
		})
	}
	server, token, _, _, _ := moveTaskFixture(t)
	rec := runnerRequest(t, server, http.MethodPost, runnerTokenWithScopes(t, token, runner.ScopeTasksClose), "/api/v1/runner/task/move", `{"next":true}`)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRunnerMoveTaskQueuesDestinationAtCapacity(t *testing.T) {
	server, token, order, dispatch, _ := moveTaskFixture(t)
	db := database.DB(t.Context())
	factory, err := models.FindFactory(db, order.OrganizationID, order.FactoryID)
	require.NoError(t, err)
	other, err := factory.CreateWorkOrder(db, "Occupy Review", "", nil, nil, nil)
	require.NoError(t, err)
	_, err = other.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateOpen})
	require.NoError(t, err)
	blocker := support.CreateFactoryLineDispatch(t, order.OrganizationID, order.FactoryID, other.ID, dispatch.LineID, dispatch.LineName, []models.FactoryLineStep(dispatch.Steps))
	_, err = blocker.StartStep(db, other, 2)
	require.NoError(t, err)
	rec := runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/move", `{"next":true}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var queued []models.FactoryWorkOrderQueueItem
	require.NoError(t, db.Where("line_dispatch_id = ?", dispatch.ID).Find(&queued).Error)
	require.Len(t, queued, 1)
	assert.Equal(t, "custom-review", queued[0].StageKey)
	assert.Equal(t, 2, queued[0].StepIndex)
}

func TestRunnerMoveTaskStartsLeaveAndDestinationAutomations(t *testing.T) {
	for _, destination := range []string{"done", "backlog"} {
		t.Run(destination, func(t *testing.T) {
			server, token, order, dispatch, ctx := moveTaskFixture(t)
			db := database.DB(t.Context())
			// Normal lines use phase keys, while custom automations use board columns.
			dispatch.Steps[0].StageKey = "phase-0"
			dispatch.Steps[1].StageKey = "phase-0"
			dispatch.Stages[1].Key = "phase-0"
			require.NoError(t, db.Save(dispatch).Error)
			require.NoError(t, dispatch.EnterStage(db, "phase-0"))
			callerCanvas, err := models.FindCanvasInTransaction(db, order.OrganizationID, uuid.MustParse(ctx.WorkflowID))
			require.NoError(t, err)
			require.NotNil(t, callerCanvas.CreatedBy)
			var automations []*models.Canvas
			for _, automation := range []struct{ column, mode string }{
				{models.CanvasColumnKeyImplement, "leave"},
				{destination, "automatic"},
			} {
				canvas, _ := support.CreateCanvas(t, order.OrganizationID, *callerCanvas.CreatedBy, []models.CanvasNode{{
					NodeID: "on-task", Type: models.NodeTypeTrigger,
					Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "onWorkOrder"}}),
					Metadata: datatypes.NewJSONType(map[string]any{
						models.FactoryAppTemplateMetadataKey: map[string]any{
							"id": models.FactoryAppTemplateCustomAutomationID, "startMode": automation.mode,
						},
					}),
				}}, nil)
				require.NoError(t, db.Model(canvas).Updates(map[string]any{"factory_id": order.FactoryID, "column_key": automation.column}).Error)
				automations = append(automations, canvas)
			}
			rec := runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/move", `{"stage":"`+destination+`"}`)
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			for _, canvas := range automations {
				var events []models.CanvasEvent
				require.NoError(t, db.Where("workflow_id = ?", canvas.ID).Find(&events).Error)
				require.Len(t, events, 1, "move must start the configured automation")
				payload := models.RootEventSourcePayload(events[0].Data.Data()).(map[string]any)
				assert.Equal(t, order.ID.String(), payload["workOrder"].(map[string]any)["id"])
			}
		})
	}
}

func TestRunnerMoveTaskConcurrentRequestsDoNotAdvanceTwice(t *testing.T) {
	server, token, order, dispatch, _ := moveTaskFixture(t)
	start := make(chan struct{})
	codes := make(chan int, 2)
	for range 2 {
		go func() {
			<-start
			codes <- runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/move", `{"next":true}`).Code
		}()
	}
	close(start)
	assert.ElementsMatch(t, []int{http.StatusOK, http.StatusConflict}, []int{<-codes, <-codes})
	db := database.DB(t.Context())
	var count int64
	require.NoError(t, db.Model(&models.FactoryWorkOrderExecution{}).Where("line_dispatch_id = ? AND step_index = 2", dispatch.ID).Count(&count).Error)
	assert.EqualValues(t, 1, count)
	require.NoError(t, db.First(dispatch, "id = ?", dispatch.ID).Error)
	assert.Equal(t, "custom-review", dispatch.CurrentStageKey)
	assert.Equal(t, "open", order.State)
}

func TestRunnerMoveTaskRejectsMissingSuccessorAndLineContext(t *testing.T) {
	for _, withoutLine := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing successor", true: "missing line"}[withoutLine], func(t *testing.T) {
			server, token, order, dispatch, ctx := moveTaskFixture(t)
			db := database.DB(t.Context())
			if withoutLine {
				require.NoError(t, db.Where("run_id = ?", ctx.RunID).Delete(&models.FactoryWorkOrderExecution{}).Error)
			} else {
				dispatch.Stages = datatypes.NewJSONSlice([]models.FactoryLineStage{{Key: "implement", Kind: models.FactoryLineStagePhase}})
				require.NoError(t, db.Save(dispatch).Error)
			}
			rec := runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/move", `{"next":true}`)
			assert.NotEqual(t, http.StatusOK, rec.Code)
			require.NoError(t, db.First(dispatch, "id = ?", dispatch.ID).Error)
			assert.Equal(t, "implement", dispatch.CurrentStageKey)
			require.NoError(t, db.First(order, "id = ?", order.ID).Error)
			assert.Equal(t, models.FactoryWorkOrderStateOpen, order.State)
		})
	}
}

func TestRunnerMoveTaskRollsBackOnAdmissionFailure(t *testing.T) {
	server, token, _, dispatch, ctx := moveTaskFixture(t)
	db := database.DB(t.Context())
	require.NoError(t, db.Where("workflow_id = ? AND node_id = ?", dispatch.Steps[2].AppID, dispatch.Steps[2].Entrypoint).Delete(&models.CanvasNode{}).Error)
	rec := runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/move", `{"next":true}`)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		execution, err := models.FindWorkOrderExecutionByRunID(tx, ctx.RunID)
		require.NoError(t, err)
		assert.Equal(t, models.FactoryWorkOrderExecutionStatusRunning, execution.Status)
		return nil
	}))
	require.NoError(t, db.First(dispatch, "id = ?", dispatch.ID).Error)
	assert.Equal(t, "implement", dispatch.CurrentStageKey)
}

func TestRunnerMoveTaskRejectsClosedTasksAndFollowsEmptyStagePolicy(t *testing.T) {
	for _, result := range []string{models.FactoryWorkOrderResultCompleted, models.FactoryWorkOrderResultArchived} {
		t.Run(result, func(t *testing.T) {
			server, token, order, dispatch, ctx := moveTaskFixture(t)
			db := database.DB(t.Context())
			_, err := order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateClosed, Result: result, Run: &factoryevents.RunRef{ID: ctx.RunID}})
			require.NoError(t, err)
			rec := runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/move", `{"next":true}`)
			assert.Equal(t, http.StatusConflict, rec.Code)
			require.NoError(t, db.First(dispatch, "id = ?", dispatch.ID).Error)
			assert.Equal(t, "implement", dispatch.CurrentStageKey)
		})
	}
	server, token, order, dispatch, _ := moveTaskFixture(t)
	db := database.DB(t.Context())
	yes := true
	dispatch.Stages[3].AutoAdvance = &yes
	require.NoError(t, db.Save(dispatch).Error)
	rec := runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/move", `{"stage":"custom-wait"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NoError(t, db.First(order, "id = ?", order.ID).Error)
	assert.Equal(t, models.FactoryWorkOrderStateClosed, order.State)
	require.NoError(t, db.First(dispatch, "id = ?", dispatch.ID).Error)
	assert.Equal(t, "done", dispatch.CurrentStageKey)
}

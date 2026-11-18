package runner_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestClosedDoneAutomationReceivesRunnerToken(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	t.Setenv("JWT_SECRET", "test")
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas := doneAutomationCanvas(t, r, factory)

	t.Run("closed task", func(t *testing.T) {
		order := closedTask(t, db, factory, r, models.FactoryWorkOrderResultFailed)
		assert.True(t, runner.HasRunnerToken(runnerTokenEnv(t, db, r, factory, canvas, order)))
	})

	t.Run("archived task", func(t *testing.T) {
		order := closedTask(t, db, factory, r, models.FactoryWorkOrderResultArchived)
		assert.False(t, runner.HasRunnerToken(runnerTokenEnv(t, db, r, factory, canvas, order)))
	})

	t.Run("closed task on Verify", func(t *testing.T) {
		order := closedTask(t, db, factory, r, models.FactoryWorkOrderResultFailed)
		require.NoError(t, db.Model(canvas).Update("column_key", models.CanvasColumnKeyVerify).Error)
		t.Cleanup(func() {
			require.NoError(t, db.Model(canvas).Update("column_key", models.CanvasColumnKeyDone).Error)
		})
		assert.False(t, runner.HasRunnerToken(runnerTokenEnv(t, db, r, factory, canvas, order)))
	})
}

func runnerTokenEnv(
	t *testing.T,
	db *gorm.DB,
	r *support.ResourceRegistry,
	factory *models.Factory,
	canvas *models.Canvas,
	order *models.FactoryWorkOrder,
) []runner.TaskEnvironmentVariable {
	t.Helper()
	now := time.Now()
	event := &models.CanvasEvent{
		WorkflowID: canvas.ID,
		NodeID:     "on-task",
		Channel:    "default",
		Data: models.NewJSONValue(map[string]any{
			"data": map[string]any{"workOrder": map[string]any{"id": order.ID.String()}},
		}),
		State:     models.CanvasEventStatePending,
		CreatedAt: &now,
	}
	require.NoError(t, db.Create(event).Error)
	run, err := models.CreateCanvasRunInTransaction(db, canvas.ID, "on-task", models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, "on-task", event.ID, event.ID)
	require.NoError(t, db.Model(execution).Update("run_id", run.ID).Error)
	scope := &runner.RunnerScope{
		OrganizationID:  r.Organization.ID,
		FactoryID:       factory.ID,
		WorkOrderID:     order.ID,
		CanvasID:        canvas.ID,
		CanvasRunID:     run.ID,
		NodeExecutionID: execution.ID,
		NodeID:          "on-task",
		Flow:            runner.RunnerFlowTask,
	}
	return runner.AttachScopedRunnerEnv(db, core.ExecutionContext{
		ID:             execution.ID,
		RunID:          run.ID,
		WorkflowID:     canvas.ID.String(),
		OrganizationID: r.Organization.ID.String(),
		NodeID:         "on-task",
		BaseURL:        "http://runner.test",
	}, nil, 60, scope)
}

func doneAutomationCanvas(t *testing.T, r *support.ResourceRegistry, factory *models.Factory) *models.Canvas {
	t.Helper()
	db := database.DB(t.Context())
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
		NodeID: "on-task",
		Name:   "On Task",
		Type:   models.NodeTypeTrigger,
		Metadata: datatypes.NewJSONType(models.FactoryAppTemplateMetadata(
			models.FactoryAppTemplateCustomAutomationID,
			1,
		)),
	}}, nil)
	require.NoError(t, db.Model(canvas).Updates(map[string]any{
		"factory_id": factory.ID,
		"column_key": models.CanvasColumnKeyDone,
	}).Error)
	return canvas
}

func closedTask(
	t *testing.T,
	db *gorm.DB,
	factory *models.Factory,
	r *support.ResourceRegistry,
	result string,
) *models.FactoryWorkOrder {
	t.Helper()
	order, err := factory.CreateWorkOrder(db, support.RandomName("task"), "", &r.User, nil, nil)
	require.NoError(t, err)
	if result != models.FactoryWorkOrderResultArchived {
		_, err = order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateOpen})
		require.NoError(t, err)
		order, err = factory.FindWorkOrder(db, order.ID)
		require.NoError(t, err)
	}
	closed, err := order.Close(db, result, &r.User)
	require.NoError(t, err)
	return closed
}

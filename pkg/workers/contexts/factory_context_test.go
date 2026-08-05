package contexts

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestFactoryContext_CreateWorkOrder(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "")
	require.NoError(t, err)

	canvas, nodeExecution, run := setupFactoryAppExecution(t, r, factory.ID)

	t.Run("creates work order on factory-owned app", func(t *testing.T) {
		ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

		workOrder, err := ctx.CreateWorkOrder(core.WorkOrderParams{
			Title:       "From GitHub issue",
			Description: "Automated intake",
		})
		require.NoError(t, err)
		assert.Equal(t, "From GitHub issue", workOrder.Title)
		assert.Equal(t, "Automated intake", workOrder.Description)
		assert.NotEmpty(t, workOrder.ID)
	})

	t.Run("rejects blank title", func(t *testing.T) {
		ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

		_, err := ctx.CreateWorkOrder(core.WorkOrderParams{Title: "   "})
		require.Error(t, err)
		assert.ErrorIs(t, err, models.ErrFactoryWorkOrderTitleRequired)
	})

	t.Run("rejects non-factory app", func(t *testing.T) {
		regularCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		ctx := NewFactoryContext(database.Conn(), regularCanvas, nodeExecution)

		_, err := ctx.CreateWorkOrder(core.WorkOrderParams{Title: "Should fail"})
		require.Error(t, err)
		assert.EqualError(t, err, "app is not owned by a factory")
	})

	t.Run("rejects run that is executing a work order", func(t *testing.T) {
		line, err := factory.CreateLine(database.Conn(), "ship", nil)
		require.NoError(t, err)

		existingOrder, err := factory.CreateWorkOrder(database.Conn(), "Existing", "", &r.User, nil)
		require.NoError(t, err)

		now := time.Now()
		workOrderExecution := models.FactoryWorkOrderExecution{
			ID:             uuid.New(),
			OrganizationID: r.Organization.ID,
			FactoryID:      factory.ID,
			WorkOrderID:    existingOrder.ID,
			LineID:         line.ID,
			StepIndex:      0,
			StepName:       "step-one",
			RunID:          run.ID,
			Status:         models.FactoryWorkOrderExecutionStatusRunning,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		require.NoError(t, database.Conn().Create(&workOrderExecution).Error)

		ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)
		_, err = ctx.CreateWorkOrder(core.WorkOrderParams{Title: "Nested"})
		require.Error(t, err)
		assert.EqualError(t, err, "cannot create work order while executing another work order")
	})
}

func TestFactoryContext_UpdateWorkOrderStatus(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "")
	require.NoError(t, err)

	canvas, nodeExecution, run := setupFactoryAppExecution(t, r, factory.ID)
	order, err := factory.CreateWorkOrder(database.Conn(), "Status target", "", &r.User, nil)
	require.NoError(t, err)
	linkRunToWorkOrder(t, r, factory, order.ID, run.ID)

	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

	updated, err := ctx.UpdateWorkOrderStatus(core.UpdateWorkOrderStatusParams{
		State: models.FactoryWorkOrderStateOpen,
	})
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderStateOpen, updated.State)
}

func TestFactoryContext_UpdateWorkOrderStatus_RunNotAttached(t *testing.T) {
	//
	// When the run has no `factory_work_order_executions` row we have
	// no way to know which order the component is meant to update.
	// Rather than silently succeed on some default target, the context
	// must fail loudly so the component author sees the wiring is off.
	//
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "")
	require.NoError(t, err)

	canvas, nodeExecution, _ := setupFactoryAppExecution(t, r, factory.ID)

	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)
	_, err = ctx.UpdateWorkOrderStatus(core.UpdateWorkOrderStatusParams{
		State: models.FactoryWorkOrderStateOpen,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not attached to a work order")
}

func TestFactoryContext_AddWorkOrderComment(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "")
	require.NoError(t, err)

	canvas, nodeExecution, run := setupFactoryAppExecution(t, r, factory.ID)
	order, err := factory.CreateWorkOrder(database.Conn(), "Comment target", "", &r.User, nil)
	require.NoError(t, err)
	linkRunToWorkOrder(t, r, factory, order.ID, run.ID)

	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

	require.NoError(t, ctx.AddWorkOrderComment(core.AddWorkOrderCommentParams{
		Body: "Ready for review",
	}))

	events, err := order.ListEvents(database.Conn(), 10, nil)
	require.NoError(t, err)

	//
	// The comment must be attributed to the executing canvas node (kind
	// = automation, automation.nodeId matches the exec) so the timeline
	// can render "commented via <node> in <app>" instead of a free-form
	// author label.
	//
	var commentEvent *models.FactoryWorkOrderEvent
	for i := range events {
		if events[i].Type == "order.comment.added" {
			commentEvent = &events[i]
			break
		}
	}
	require.NotNil(t, commentEvent, "expected order.comment.added event")

	var payload struct {
		Author struct {
			Kind       string `json:"kind"`
			Automation struct {
				NodeID  string `json:"nodeId"`
				AppName string `json:"appName"`
			} `json:"automation"`
		} `json:"author"`
	}
	require.NoError(t, json.Unmarshal(commentEvent.Data, &payload))
	assert.Equal(t, "automation", payload.Author.Kind)
	assert.Equal(t, nodeExecution.NodeID, payload.Author.Automation.NodeID)
	assert.NotEmpty(t, payload.Author.Automation.AppName)
}

func TestFactoryContext_AddWorkOrderArtifact(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "")
	require.NoError(t, err)

	canvas, nodeExecution, run := setupFactoryAppExecution(t, r, factory.ID)
	order, err := factory.CreateWorkOrder(database.Conn(), "Artifact target", "", &r.User, nil)
	require.NoError(t, err)
	linkRunToWorkOrder(t, r, factory, order.ID, run.ID)

	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

	artifact, err := ctx.AddWorkOrderArtifact(core.AddWorkOrderArtifactParams{
		Type:  "pr",
		URL:   "https://github.com/example/repo/pull/1",
		Title: "Draft",
		Data:  map[string]any{"number": "1"},
	})
	require.NoError(t, err)
	require.NotNil(t, artifact)
	assert.Equal(t, "pr", artifact.Type)

	artifacts, err := order.ListArtifacts(database.Conn())
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	assert.Equal(t, "https://github.com/example/repo/pull/1", artifacts[0].URL)
}

// linkRunToWorkOrder creates the `factory_work_order_executions` row
// that the FactoryContext resolves from `execution.RunID` to find the
// "current" work order. Every factory work-order component now derives
// the target order from this link instead of taking it as config.
//
// A real (empty) factory line is created so the FK on `line_id`
// resolves; the test doesn't otherwise care about the line's shape.
func linkRunToWorkOrder(
	t *testing.T,
	r *support.ResourceRegistry,
	factory *models.Factory,
	workOrderID uuid.UUID,
	runID uuid.UUID,
) {
	t.Helper()

	line, err := factory.CreateLine(database.Conn(), support.RandomName("line"), nil)
	require.NoError(t, err)

	now := time.Now()
	execution := models.FactoryWorkOrderExecution{
		ID:             uuid.New(),
		OrganizationID: r.Organization.ID,
		FactoryID:      factory.ID,
		WorkOrderID:    workOrderID,
		LineID:         line.ID,
		StepIndex:      0,
		StepName:       "component-under-test",
		RunID:          runID,
		Status:         models.FactoryWorkOrderExecutionStatusRunning,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	require.NoError(t, database.Conn().Create(&execution).Error)
}

func setupFactoryAppExecution(
	t *testing.T,
	r *support.ResourceRegistry,
	factoryID uuid.UUID,
) (*models.Canvas, *models.CanvasNodeExecution, *models.CanvasRun) {
	t.Helper()

	const nodeID = "create-work-order"

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: nodeID, Type: models.NodeTypeComponent},
		},
		nil,
	)
	require.NoError(t, database.Conn().Model(canvas).Update("factory_id", factoryID).Error)
	canvas.FactoryID = &factoryID

	triggerEvent := support.EmitCanvasEventForNode(t, canvas.ID, nodeID, "default", nil)
	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(database.Conn(), triggerEvent)
	require.NoError(t, err)

	nodeExecution := support.CreateCanvasNodeExecution(t, canvas.ID, nodeID, triggerEvent.ID, triggerEvent.ID)
	nodeExecution.RunID = run.ID
	require.NoError(t, database.Conn().Save(nodeExecution).Error)

	return canvas, nodeExecution, run
}

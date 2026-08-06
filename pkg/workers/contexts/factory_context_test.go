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
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
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

		persisted, err := factory.FindWorkOrder(database.Conn(), uuid.MustParse(workOrder.ID))
		require.NoError(t, err)
		require.NotNil(t, persisted.SourceRunID)
		assert.Equal(t, run.ID, *persisted.SourceRunID)
		assert.Equal(t, models.FactoryWorkOrderStateDraft, persisted.State,
			"work orders now start as draft and are promoted to open on first dispatch")

		// Creation emits a single `order.status.updated` ("" → draft).
		events, err := persisted.ListEvents(database.Conn(), 0, nil)
		require.NoError(t, err)
		require.Len(t, events, 1)
		assert.Equal(t, factoryevents.EventTypeOrderStatusUpdated, events[0].Type)

		var initialStatus factoryevents.WorkOrderStatusUpdated
		require.NoError(t, json.Unmarshal(events[0].Data, &initialStatus))
		assert.Equal(t, "", initialStatus.FromState)
		assert.Equal(t, models.FactoryWorkOrderStateDraft, initialStatus.ToState)

		// On the first `draft → open` promotion the source-run snapshot from
		// `o.SourceRunID` is enriched into the status.updated event so the
		// timeline can link back to the originating canvas run.
		require.NoError(t, persisted.UpdateStatus(database.Conn(), models.FactoryWorkOrderStatusUpdate{
			ToState: models.FactoryWorkOrderStateOpen,
		}))

		statusEvents := listWorkOrderEvents(t, persisted, factoryevents.EventTypeOrderStatusUpdated)
		require.Len(t, statusEvents, 2)
		var opened factoryevents.WorkOrderStatusUpdated
		require.NoError(t, json.Unmarshal(statusEvents[0].Data, &opened))
		assert.Equal(t, models.FactoryWorkOrderStateDraft, opened.FromState)
		assert.Equal(t, models.FactoryWorkOrderStateOpen, opened.ToState)
		require.NotNil(t, opened.Run)
		assert.Equal(t, run.ID, opened.Run.ID)
		require.NotNil(t, opened.App)
		assert.Equal(t, canvas.ID, opened.App.ID)
		assert.Nil(t, opened.User)
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

		existingOrder, err := factory.CreateWorkOrder(database.Conn(), "Existing", "", &r.User, nil, nil)
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
	order, err := factory.CreateWorkOrder(database.Conn(), "Status target", "", &r.User, nil, nil)
	require.NoError(t, err)
	line := linkRunToWorkOrder(t, r, factory, order.ID, run.ID)

	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

	updated, err := ctx.UpdateWorkOrderStatus(core.UpdateWorkOrderStatusParams{
		State: models.FactoryWorkOrderStateOpen,
	})
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderStateOpen, updated.State)

	// `order.status.updated` is now the sole authoritative event. The
	// draft → open transition carries the automation ref (node/line/step).
	statusEvent := findWorkOrderEvent(t, order, "order.status.updated")
	statusAutomation := extractAutomationPayload(t, statusEvent)
	assert.Equal(t, nodeExecution.NodeID, statusAutomation.NodeID)
	assert.Equal(t, line.Name, statusAutomation.LineName)
	assert.Equal(t, "component-under-test", statusAutomation.StepName)
}

func TestFactoryContext_UpdateWorkOrderStatus_CloseAttributesAutomation(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "")
	require.NoError(t, err)

	canvas, nodeExecution, run := setupFactoryAppExecution(t, r, factory.ID)
	order, err := factory.CreateWorkOrder(database.Conn(), "Status target", "", &r.User, nil, nil)
	require.NoError(t, err)
	line := linkRunToWorkOrder(t, r, factory, order.ID, run.ID)

	require.NoError(t, order.UpdateStatus(database.Conn(), models.FactoryWorkOrderStatusUpdate{
		ToState: models.FactoryWorkOrderStateOpen,
		Actor:   &r.User,
	}))

	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)
	updated, err := ctx.UpdateWorkOrderStatus(core.UpdateWorkOrderStatusParams{
		State:  models.FactoryWorkOrderStateClosed,
		Result: models.FactoryWorkOrderResultCompleted,
	})
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderStateClosed, updated.State)

	// The most recent status.updated (open → closed) is the close event.
	closed := findWorkOrderEvent(t, order, "order.status.updated")
	closedAutomation := extractAutomationPayload(t, closed)
	assert.Equal(t, line.Name, closedAutomation.LineName)
	assert.Equal(t, "component-under-test", closedAutomation.StepName)
	assert.Equal(t, nodeExecution.NodeID, closedAutomation.NodeID)
}

func TestFactoryContext_UpdateWorkOrderStatus_RunNotAttached(t *testing.T) {
	// A run with no work-order execution link must fail loudly rather
	// than silently target some default order.
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
	order, err := factory.CreateWorkOrder(database.Conn(), "Comment target", "", &r.User, nil, nil)
	require.NoError(t, err)
	line := linkRunToWorkOrder(t, r, factory, order.ID, run.ID)

	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

	require.NoError(t, ctx.AddWorkOrderComment(core.AddWorkOrderCommentParams{
		Body: "Ready for review",
	}))

	commentEvent := findWorkOrderEvent(t, order, "order.comment.added")

	var payload struct {
		Author struct {
			Kind       string `json:"kind"`
			Automation struct {
				NodeID   string `json:"nodeId"`
				AppName  string `json:"appName"`
				LineName string `json:"lineName"`
				StepName string `json:"stepName"`
			} `json:"automation"`
		} `json:"author"`
	}
	require.NoError(t, json.Unmarshal(commentEvent.Data, &payload))
	assert.Equal(t, "automation", payload.Author.Kind)
	assert.Equal(t, nodeExecution.NodeID, payload.Author.Automation.NodeID)
	assert.NotEmpty(t, payload.Author.Automation.AppName)
	assert.Equal(t, line.Name, payload.Author.Automation.LineName)
	assert.Equal(t, "component-under-test", payload.Author.Automation.StepName)
}

func TestFactoryContext_AddWorkOrderArtifact(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "")
	require.NoError(t, err)

	canvas, nodeExecution, run := setupFactoryAppExecution(t, r, factory.ID)
	order, err := factory.CreateWorkOrder(database.Conn(), "Artifact target", "", &r.User, nil, nil)
	require.NoError(t, err)
	line := linkRunToWorkOrder(t, r, factory, order.ID, run.ID)

	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

	artifact, err := ctx.AddWorkOrderArtifact(core.AddWorkOrderArtifactParams{
		Type: "pr",
		Data: map[string]any{
			"url":    "https://github.com/example/repo/pull/1",
			"title":  "Draft",
			"number": "1",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, artifact)
	assert.Equal(t, "pr", artifact.Type)
	assert.Equal(t, "https://github.com/example/repo/pull/1", artifact.Data["url"])

	artifacts, err := order.ListArtifacts(database.Conn())
	require.NoError(t, err)
	require.Len(t, artifacts, 1)

	artifactEvent := findWorkOrderEvent(t, order, "order.artifact.added")
	artifactAutomation := extractAutomationPayload(t, artifactEvent)
	assert.Equal(t, nodeExecution.NodeID, artifactAutomation.NodeID)
	assert.Equal(t, line.Name, artifactAutomation.LineName)
	assert.Equal(t, "component-under-test", artifactAutomation.StepName)
}

type automationPayload struct {
	NodeID   string `json:"nodeId"`
	NodeName string `json:"nodeName"`
	AppName  string `json:"appName"`
	LineName string `json:"lineName"`
	StepName string `json:"stepName"`
}

func extractAutomationPayload(t *testing.T, event *models.FactoryWorkOrderEvent) automationPayload {
	t.Helper()

	var wrapper struct {
		Automation *automationPayload `json:"automation"`
	}
	require.NoError(t, json.Unmarshal(event.Data, &wrapper))
	require.NotNil(t, wrapper.Automation, "event %s missing automation payload", event.Type)
	return *wrapper.Automation
}

func findWorkOrderEvent(t *testing.T, order *models.FactoryWorkOrder, eventType string) *models.FactoryWorkOrderEvent {
	t.Helper()

	events, err := order.ListEvents(database.Conn(), 50, nil)
	require.NoError(t, err)

	// ListEvents sorts DESC by created_at, so the first match is the latest.
	for i := range events {
		if events[i].Type == eventType {
			return &events[i]
		}
	}

	t.Fatalf("expected %s event on work order %s", eventType, order.ID)
	return nil
}

// listWorkOrderEvents returns every event of the given type, newest first.
func listWorkOrderEvents(t *testing.T, order *models.FactoryWorkOrder, eventType string) []models.FactoryWorkOrderEvent {
	t.Helper()

	events, err := order.ListEvents(database.Conn(), 100, nil)
	require.NoError(t, err)

	matches := make([]models.FactoryWorkOrderEvent, 0)
	for _, event := range events {
		if event.Type == eventType {
			matches = append(matches, event)
		}
	}
	return matches
}

// linkRunToWorkOrder creates the `factory_work_order_executions` row
// FactoryContext looks up from `execution.RunID`; the returned line is
// used by tests that assert line-based automation attribution.
func linkRunToWorkOrder(
	t *testing.T,
	r *support.ResourceRegistry,
	factory *models.Factory,
	workOrderID uuid.UUID,
	runID uuid.UUID,
) *models.FactoryLine {
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
	return line
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

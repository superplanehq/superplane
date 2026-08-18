package contexts

import (
	"encoding/json"
	"errors"
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

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
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
		// The "" → draft creation event carries the same source-run
		// snapshot as the draft → open promotion so the very first
		// timeline entry already links back to the originating run.
		require.NotNil(t, initialStatus.Run)
		assert.Equal(t, run.ID, initialStatus.Run.ID)
		require.NotNil(t, initialStatus.App)
		assert.Equal(t, canvas.ID, initialStatus.App.ID)

		// On the first `draft → open` promotion the source-run snapshot from
		// `o.SourceRunID` is enriched into the status.updated event so the
		// timeline can link back to the originating canvas run.
		_, err = persisted.UpdateStatus(database.Conn(), models.FactoryWorkOrderStatusUpdate{
			ToState: models.FactoryWorkOrderStateOpen,
		})
		require.NoError(t, err)

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

		dispatch := support.CreateFactoryLineDispatch(t, r.Organization.ID, factory.ID, existingOrder.ID, line.ID, line.Name, nil)

		now := time.Now()
		workOrderExecution := models.FactoryWorkOrderExecution{
			ID:             uuid.New(),
			OrganizationID: r.Organization.ID,
			FactoryID:      factory.ID,
			WorkOrderID:    existingOrder.ID,
			LineID:         line.ID,
			LineDispatchID: dispatch.ID,
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

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, nodeExecution, run := setupFactoryAppExecution(t, r, factory.ID)
	order, err := factory.CreateWorkOrder(database.Conn(), "Status target", "", &r.User, nil, nil)
	require.NoError(t, err)
	line := linkRunToWorkOrder(t, r, factory, order.ID, run.ID)

	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

	updated, changed, err := ctx.UpdateWorkOrderStatus(core.UpdateWorkOrderStatusParams{
		OrderID: order.ID.String(),
		State:   models.FactoryWorkOrderStateOpen,
	})
	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, models.FactoryWorkOrderStateOpen, updated.State)

	// `order.status.updated` is now the sole authoritative event. The
	// draft → open transition carries the automation ref (node/line/step).
	statusEvent := findWorkOrderEvent(t, order, "order.status.updated")
	statusAutomation := extractAutomationPayload(t, statusEvent)
	assert.Equal(t, nodeExecution.NodeID, statusAutomation.NodeID)
	assert.Equal(t, line.Name, statusAutomation.LineName)
	assert.Equal(t, "component-under-test", statusAutomation.StepName)
}

// A re-run of the component with the same target state must be a
// silent no-op: no duplicate `order.status.updated` event, no
// websocket fan-out, and the caller sees `changed=false`.
func TestFactoryContext_UpdateWorkOrderStatus_NoopSkipsEmit(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, nodeExecution, run := setupFactoryAppExecution(t, r, factory.ID)
	order, err := factory.CreateWorkOrder(database.Conn(), "Status target", "", &r.User, nil, nil)
	require.NoError(t, err)
	linkRunToWorkOrder(t, r, factory, order.ID, run.ID)

	var notifications int
	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution).
		WithWorkOrderUpdated(func(_, _, _ string) { notifications++ })

	_, changed, err := ctx.UpdateWorkOrderStatus(core.UpdateWorkOrderStatusParams{
		OrderID: order.ID.String(),
		State:   models.FactoryWorkOrderStateOpen,
	})
	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, 1, notifications)

	_, changed, err = ctx.UpdateWorkOrderStatus(core.UpdateWorkOrderStatusParams{
		OrderID: order.ID.String(),
		State:   models.FactoryWorkOrderStateOpen,
	})
	require.NoError(t, err)
	assert.False(t, changed, "re-hitting the same state must report changed=false")
	assert.Equal(t, 1, notifications, "no-op transition must not fan out again")

	statusEvents := listWorkOrderEvents(t, order, factoryevents.EventTypeOrderStatusUpdated)
	// Creation ("" → draft) + one real draft → open transition = 2.
	// A second event would indicate the no-op leaked into the timeline.
	assert.Len(t, statusEvents, 2, "no-op must not record a second status.updated event")
}

func TestFactoryContext_UpdateWorkOrderStatus_CloseAttributesAutomation(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, nodeExecution, run := setupFactoryAppExecution(t, r, factory.ID)
	order, err := factory.CreateWorkOrder(database.Conn(), "Status target", "", &r.User, nil, nil)
	require.NoError(t, err)
	line := linkRunToWorkOrder(t, r, factory, order.ID, run.ID)

	_, err = order.UpdateStatus(database.Conn(), models.FactoryWorkOrderStatusUpdate{
		ToState: models.FactoryWorkOrderStateOpen,
		Actor:   &r.User,
	})
	require.NoError(t, err)

	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)
	updated, changed, err := ctx.UpdateWorkOrderStatus(core.UpdateWorkOrderStatusParams{
		OrderID: order.ID.String(),
		State:   models.FactoryWorkOrderStateClosed,
		Result:  models.FactoryWorkOrderResultCompleted,
	})
	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, models.FactoryWorkOrderStateClosed, updated.State)

	// The most recent status.updated (open → closed) is the close event.
	closed := findWorkOrderEvent(t, order, "order.status.updated")
	closedAutomation := extractAutomationPayload(t, closed)
	assert.Equal(t, line.Name, closedAutomation.LineName)
	assert.Equal(t, "component-under-test", closedAutomation.StepName)
	assert.Equal(t, nodeExecution.NodeID, closedAutomation.NodeID)
}

// orderId is required and explicit at the FactoryContext level too — there
// is no implicit fallback to "the work order driving the current run".
// That behavior now lives entirely in the `{{ order().id }}` default on the
// component field, resolved before Execute ever sees the configuration.
func TestFactoryContext_UpdateWorkOrderStatus_EmptyOrderIDIsRejected(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, nodeExecution, _ := setupFactoryAppExecution(t, r, factory.ID)

	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)
	_, _, err = ctx.UpdateWorkOrderStatus(core.UpdateWorkOrderStatusParams{
		State: models.FactoryWorkOrderStateOpen,
	})
	require.Error(t, err)
	assert.EqualError(t, err, "orderId is required")
}

func TestFactoryContext_AddWorkOrderComment_EmptyOrderIDIsRejected(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, nodeExecution, _ := setupFactoryAppExecution(t, r, factory.ID)

	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)
	err = ctx.AddWorkOrderComment(core.AddWorkOrderCommentParams{
		Body: "Ready for review",
	})
	require.Error(t, err)
	assert.EqualError(t, err, "orderId is required")
}

func TestFactoryContext_AddWorkOrderArtifact_EmptyOrderIDIsRejected(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, nodeExecution, _ := setupFactoryAppExecution(t, r, factory.ID)

	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)
	_, err = ctx.AddWorkOrderArtifact(core.AddWorkOrderArtifactParams{
		Type: "pr",
		Data: map[string]any{"url": "https://github.com/example/repo/pull/1"},
	})
	require.Error(t, err)
	assert.EqualError(t, err, "orderId is required")
}

// Regression coverage for the github.onPullRequest -> close-work-order bug:
// a run with no `factory_work_order_executions` row (e.g. one started by a
// plain webhook trigger, not a factory line dispatch) must still be able to
// target a work order explicitly via `orderId`.
func TestFactoryContext_UpdateWorkOrderStatus_ExplicitOrderIDOnUnattachedRun(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, nodeExecution, _ := setupFactoryAppExecution(t, r, factory.ID)
	order, err := factory.CreateWorkOrder(database.Conn(), "Unattached target", "", &r.User, nil, nil)
	require.NoError(t, err)

	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

	updated, changed, err := ctx.UpdateWorkOrderStatus(core.UpdateWorkOrderStatusParams{
		OrderID: order.ID.String(),
		State:   models.FactoryWorkOrderStateOpen,
	})
	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, models.FactoryWorkOrderStateOpen, updated.State)

	// Attribution still includes node/app but omits line/step (no
	// dispatch happened) and must not panic.
	statusEvent := findWorkOrderEvent(t, order, "order.status.updated")
	statusAutomation := extractAutomationPayload(t, statusEvent)
	assert.Equal(t, nodeExecution.NodeID, statusAutomation.NodeID)
	assert.NotEmpty(t, statusAutomation.AppName)
	assert.Empty(t, statusAutomation.LineName)
	assert.Empty(t, statusAutomation.StepName)
}

func TestFactoryContext_UpdateWorkOrderStatus_ExplicitOrderIDNotFound(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, nodeExecution, _ := setupFactoryAppExecution(t, r, factory.ID)
	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

	_, _, err = ctx.UpdateWorkOrderStatus(core.UpdateWorkOrderStatusParams{
		OrderID: uuid.New().String(),
		State:   models.FactoryWorkOrderStateOpen,
	})
	require.Error(t, err)
}

func TestFactoryContext_AddWorkOrderComment_ExplicitOrderIDOnUnattachedRun(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, nodeExecution, _ := setupFactoryAppExecution(t, r, factory.ID)
	order, err := factory.CreateWorkOrder(database.Conn(), "Unattached comment target", "", &r.User, nil, nil)
	require.NoError(t, err)

	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

	require.NoError(t, ctx.AddWorkOrderComment(core.AddWorkOrderCommentParams{
		OrderID: order.ID.String(),
		Body:    "Merged via PR",
	}))

	commentEvent := findWorkOrderEvent(t, order, "order.comment.added")
	assert.NotNil(t, commentEvent)
}

func TestFactoryContext_AddWorkOrderArtifact_ExplicitOrderIDOnUnattachedRun(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, nodeExecution, _ := setupFactoryAppExecution(t, r, factory.ID)
	order, err := factory.CreateWorkOrder(database.Conn(), "Unattached artifact target", "", &r.User, nil, nil)
	require.NoError(t, err)

	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

	artifact, err := ctx.AddWorkOrderArtifact(core.AddWorkOrderArtifactParams{
		OrderID: order.ID.String(),
		Type:    "pr",
		Data: map[string]any{
			"url": "https://github.com/example/repo/pull/7",
		},
		Key: "https://github.com/example/repo/pull/7",
	})
	require.NoError(t, err)
	require.NotNil(t, artifact)
	assert.Equal(t, order.ID.String(), artifact.WorkOrderID)
}

func TestFactoryContext_ReportWorkOrderCheck(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, nodeExecution, _ := setupFactoryAppExecution(t, r, factory.ID)
	order, err := factory.CreateWorkOrder(database.Conn(), "Check target", "", &r.User, nil, nil)
	require.NoError(t, err)

	notifications := 0
	lastReason := ""
	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution).
		WithWorkOrderUpdated(func(_, _, reason string) {
			notifications++
			lastReason = reason
		})

	check, err := ctx.ReportWorkOrderCheck(core.ReportWorkOrderCheckParams{
		OrderID:  order.ID.String(),
		CheckKey: "risk-review",
		Name:     "Risk review",
		Score:    7,
		MaxScore: 10,
		Level:    models.FactoryWorkOrderCheckLevelCaution,
	})
	require.NoError(t, err)
	assert.Equal(t, order.ID.String(), check.WorkOrderID)
	assert.Nil(t, check.PreviousScore)
	assert.Equal(t, 1, notifications)
	assert.Equal(t, factoryevents.EventTypeOrderCheckReported, lastReason)

	// A re-report of the same key updates in place and keeps the prior score.
	updated, err := ctx.ReportWorkOrderCheck(core.ReportWorkOrderCheckParams{
		OrderID:  order.ID.String(),
		CheckKey: "risk-review",
		Name:     "Risk review",
		Score:    3,
		MaxScore: 10,
		Level:    models.FactoryWorkOrderCheckLevelPositive,
	})
	require.NoError(t, err)
	assert.Equal(t, check.ID, updated.ID)
	require.NotNil(t, updated.PreviousScore)
	assert.Equal(t, float64(7), *updated.PreviousScore)
	assert.Equal(t, 2, notifications)

	// The timeline event carries automation + run attribution.
	event := findWorkOrderEvent(t, order, factoryevents.EventTypeOrderCheckReported)
	var payload factoryevents.WorkOrderCheckReported
	require.NoError(t, json.Unmarshal(event.Data, &payload))
	require.NotNil(t, payload.Automation)
	assert.Equal(t, canvas.ID, payload.Automation.AppID)
	require.NotNil(t, payload.Run)
	assert.Equal(t, nodeExecution.RunID, payload.Run.ID)
}

func TestFactoryContext_FindWorkOrder_ByID(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, nodeExecution, _ := setupFactoryAppExecution(t, r, factory.ID)
	order, err := factory.CreateWorkOrder(database.Conn(), "Find by id target", "", &r.User, nil, nil)
	require.NoError(t, err)

	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

	t.Run("finds the order", func(t *testing.T) {
		found, err := ctx.FindWorkOrder(core.FindWorkOrderParams{By: "id", OrderID: order.ID.String()})
		require.NoError(t, err)
		assert.Equal(t, order.ID.String(), found.ID)
	})

	t.Run("returns ErrWorkOrderNotFound for an unknown id", func(t *testing.T) {
		_, err := ctx.FindWorkOrder(core.FindWorkOrderParams{By: "id", OrderID: uuid.New().String()})
		assert.ErrorIs(t, err, core.ErrWorkOrderNotFound)
	})

	t.Run("returns a plain error for an invalid uuid", func(t *testing.T) {
		_, err := ctx.FindWorkOrder(core.FindWorkOrderParams{By: "id", OrderID: "not-a-uuid"})
		require.Error(t, err)
		assert.False(t, errors.Is(err, core.ErrWorkOrderNotFound))
	})

	t.Run("rejects a non-factory app", func(t *testing.T) {
		regularCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		regularCtx := NewFactoryContext(database.Conn(), regularCanvas, nodeExecution)

		_, err := regularCtx.FindWorkOrder(core.FindWorkOrderParams{By: "id", OrderID: order.ID.String()})
		require.Error(t, err)
		assert.EqualError(t, err, "app is not owned by a factory")
	})
}

func TestFactoryContext_FindWorkOrder_ByArtifactKey(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, nodeExecution, _ := setupFactoryAppExecution(t, r, factory.ID)
	order, err := factory.CreateWorkOrder(database.Conn(), "Find by artifact key target", "", &r.User, nil, nil)
	require.NoError(t, err)

	_, err = order.CreateArtifact(database.Conn(), models.FactoryWorkOrderArtifactParams{
		Type: "pr",
		Data: map[string]any{"url": "https://github.com/example/repo/pull/99"},
		Key:  "https://github.com/example/repo/pull/99",
	})
	require.NoError(t, err)

	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

	t.Run("finds the order", func(t *testing.T) {
		found, err := ctx.FindWorkOrder(core.FindWorkOrderParams{
			By:          "artifactKey",
			ArtifactKey: "https://github.com/example/repo/pull/99",
		})
		require.NoError(t, err)
		assert.Equal(t, order.ID.String(), found.ID)
	})

	t.Run("returns ErrWorkOrderNotFound for an unknown key", func(t *testing.T) {
		_, err := ctx.FindWorkOrder(core.FindWorkOrderParams{By: "artifactKey", ArtifactKey: "no-such-key"})
		assert.ErrorIs(t, err, core.ErrWorkOrderNotFound)
	})
}

func TestFactoryContext_AddWorkOrderComment(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, nodeExecution, run := setupFactoryAppExecution(t, r, factory.ID)
	order, err := factory.CreateWorkOrder(database.Conn(), "Comment target", "", &r.User, nil, nil)
	require.NoError(t, err)
	line := linkRunToWorkOrder(t, r, factory, order.ID, run.ID)

	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

	require.NoError(t, ctx.AddWorkOrderComment(core.AddWorkOrderCommentParams{
		OrderID: order.ID.String(),
		Body:    "Ready for review",
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

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, nodeExecution, run := setupFactoryAppExecution(t, r, factory.ID)
	order, err := factory.CreateWorkOrder(database.Conn(), "Artifact target", "", &r.User, nil, nil)
	require.NoError(t, err)
	line := linkRunToWorkOrder(t, r, factory, order.ID, run.ID)

	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

	artifact, err := ctx.AddWorkOrderArtifact(core.AddWorkOrderArtifactParams{
		OrderID: order.ID.String(),
		Type:    "pr",
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

	dispatch := support.CreateFactoryLineDispatch(t, r.Organization.ID, factory.ID, workOrderID, line.ID, line.Name, nil)

	now := time.Now()
	execution := models.FactoryWorkOrderExecution{
		ID:             uuid.New(),
		OrganizationID: r.Organization.ID,
		FactoryID:      factory.ID,
		WorkOrderID:    workOrderID,
		LineID:         line.ID,
		LineDispatchID: dispatch.ID,
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

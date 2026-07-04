package contexts

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/triggers/onerror"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func decodeEventPayload(t *testing.T, event models.CanvasEvent) map[string]any {
	raw, err := json.Marshal(event.Data)
	require.NoError(t, err)

	var structured map[string]any
	require.NoError(t, json.Unmarshal(raw, &structured))

	data, ok := structured["data"].(map[string]any)
	require.True(t, ok, "event payload missing data object")
	return data
}

func Test__DispatchOnError(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	triggerNodeID := "trigger-1"
	componentNodeID := "deploy-1"
	onErrorNodeID := "onerror-1"
	secondOnErrorNodeID := "onerror-2"

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNodeID,
				Name:   "Manual Run",
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID: componentNodeID,
				Name:   "Deploy to Prod",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
			{
				NodeID: onErrorNodeID,
				Name:   "On Error",
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: onerror.TriggerName}}),
			},
			{
				NodeID: secondOnErrorNodeID,
				Name:   "On Error 2",
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: onerror.TriggerName}}),
			},
		},
		[]models.Edge{
			{SourceID: triggerNodeID, TargetID: componentNodeID, Channel: "default"},
		},
	)

	t.Run("fails with error reason emits an event on every onError node", func(t *testing.T) {
		newEvents := []models.CanvasEvent{}
		onNewEvents := func(events []models.CanvasEvent) {
			newEvents = append(newEvents, events...)
		}

		rootEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, triggerNodeID, "default", nil, map[string]any{"version": "1.4.2"})
		execution := support.CreateCanvasNodeExecution(t, canvas.ID, componentNodeID, rootEvent.ID, rootEvent.ID)

		ctx := NewExecutionStateContext(database.Conn(), execution, onNewEvents)
		require.NoError(t, ctx.Fail(models.CanvasNodeExecutionResultReasonError, "request timeout after 30s"))

		require.Len(t, newEvents, 2)

		nodeIDs := []string{newEvents[0].NodeID, newEvents[1].NodeID}
		assert.Contains(t, nodeIDs, onErrorNodeID)
		assert.Contains(t, nodeIDs, secondOnErrorNodeID)

		payload := decodeEventPayload(t, newEvents[0])

		node, ok := payload["node"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, componentNodeID, node["id"])
		assert.Equal(t, "Deploy to Prod", node["name"])
		assert.Equal(t, "noop", node["component"])

		errInfo, ok := payload["error"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, models.CanvasNodeExecutionResultReasonError, errInfo["reason"])
		assert.Equal(t, "request timeout after 30s", errInfo["message"])

		_, ok = payload["payloads"].(map[string]any)
		assert.True(t, ok, "payload should include a payloads object")

		root, ok := payload["root"].(map[string]any)
		require.True(t, ok, "payload should include the run's root info")

		rootNode, ok := root["node"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, triggerNodeID, rootNode["id"])
		assert.Equal(t, "Manual Run", rootNode["name"])

		rootPayload, ok := root["payload"].(map[string]any)
		require.True(t, ok, "root payload should carry the triggering event data")
		assert.Equal(t, "1.4.2", rootPayload["version"])
	})

	t.Run("does not emit for non-error failures", func(t *testing.T) {
		newEvents := []models.CanvasEvent{}
		onNewEvents := func(events []models.CanvasEvent) {
			newEvents = append(newEvents, events...)
		}

		rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, triggerNodeID, "default", nil)
		execution := support.CreateCanvasNodeExecution(t, canvas.ID, componentNodeID, rootEvent.ID, rootEvent.ID)

		ctx := NewExecutionStateContext(database.Conn(), execution, onNewEvents)
		require.NoError(t, ctx.Fail("rejected", "approval was rejected"))

		assert.Empty(t, newEvents)
	})
}

func Test__DispatchOnError__NoOnErrorNodes(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	triggerNodeID := "trigger-1"
	componentNodeID := "deploy-1"

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNodeID,
				Name:   "Manual Run",
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID: componentNodeID,
				Name:   "Deploy to Prod",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{
			{SourceID: triggerNodeID, TargetID: componentNodeID, Channel: "default"},
		},
	)

	newEvents := []models.CanvasEvent{}
	onNewEvents := func(events []models.CanvasEvent) {
		newEvents = append(newEvents, events...)
	}

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, triggerNodeID, "default", nil)
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, componentNodeID, rootEvent.ID, rootEvent.ID)

	ctx := NewExecutionStateContext(database.Conn(), execution, onNewEvents)
	require.NoError(t, ctx.Fail(models.CanvasNodeExecutionResultReasonError, "boom"))

	assert.Empty(t, newEvents)
}

func Test__DispatchOnError__LoopPrevention(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	onErrorNodeID := "onerror-1"
	componentNodeID := "handler-1"

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: onErrorNodeID,
				Name:   "On Error",
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: onerror.TriggerName}}),
			},
			{
				NodeID: componentNodeID,
				Name:   "Error Handler",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{
			{SourceID: onErrorNodeID, TargetID: componentNodeID, Channel: "default"},
		},
	)

	newEvents := []models.CanvasEvent{}
	onNewEvents := func(events []models.CanvasEvent) {
		newEvents = append(newEvents, events...)
	}

	//
	// The run is rooted at the onError node itself, so a failure downstream
	// must not re-trigger the onError node (otherwise errors in error-handling
	// chains would loop forever).
	//
	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, onErrorNodeID, "default", nil)
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, componentNodeID, rootEvent.ID, rootEvent.ID)

	ctx := NewExecutionStateContext(database.Conn(), execution, onNewEvents)
	require.NoError(t, ctx.Fail(models.CanvasNodeExecutionResultReasonError, "handler blew up"))

	assert.Empty(t, newEvents)
}

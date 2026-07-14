package workers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"

	_ "github.com/superplanehq/superplane/pkg/triggers/messages"
)

func Test__AppMessageWorker_LockAndProcessMessage__deliversBroadcast(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	worker := NewAppMessageWorker(r.Registry)

	sourceCanvas, sourceNodes := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID:        "broadcast-message",
				Name:          "Broadcast Message",
				Type:          models.NodeTypeComponent,
				Ref:           datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "broadcastMessage"}}),
				Configuration: datatypes.NewJSONType(map[string]any{}),
			},
		},
		nil,
	)

	targetCanvas, targetNodes := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "on-broadcast",
				Name:   "On Broadcast",
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "onBroadcast"}}),
				Configuration: datatypes.NewJSONType(map[string]any{
					"app": sourceCanvas.ID.String(),
				}),
			},
		},
		nil,
	)

	require.NoError(t, database.Conn().Create(&models.CanvasSubscription{
		SourceCanvasID: sourceCanvas.ID,
		TargetCanvasID: targetCanvas.ID,
		TargetNodeID:   targetNodes[0].NodeID,
	}).Error)

	payload := map[string]any{"message": "hello"}
	require.NoError(t, models.CreateAppMessage(database.Conn(), sourceCanvas.ID, sourceNodes[0].NodeID, payload))

	messages, err := models.ListAppMessages()
	require.NoError(t, err)
	require.Len(t, messages, 1)

	require.NoError(t, worker.LockAndProcessMessage(messages[0]))

	var remaining int64
	require.NoError(t, database.Conn().Model(&models.AppMessage{}).Count(&remaining).Error)
	assert.Equal(t, int64(0), remaining)

	support.VerifyCanvasEventsCount(t, targetCanvas.ID, 1)

	var event models.CanvasEvent
	err = database.Conn().
		Where("workflow_id = ? AND node_id = ?", targetCanvas.ID, targetNodes[0].NodeID).
		First(&event).
		Error
	require.NoError(t, err)
	assert.Equal(t, "default", event.Channel)

	eventData, ok := event.Data.Data().(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "app.broadcast", eventData["type"])

	broadcastPayload, ok := eventData["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, payload, broadcastPayload["payload"])
	assert.Equal(t, map[string]any{
		"id":   sourceCanvas.ID.String(),
		"name": sourceCanvas.Name,
	}, broadcastPayload["app"])
	assert.Equal(t, map[string]any{
		"id":   sourceNodes[0].NodeID,
		"name": sourceNodes[0].Name,
	}, broadcastPayload["node"])
}

func Test__AppMessageWorker_LockAndProcessMessage__deletesMessageWithoutSubscribers(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	worker := NewAppMessageWorker(r.Registry)

	sourceCanvas, sourceNodes := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID:        "broadcast-message",
				Name:          "Broadcast Message",
				Type:          models.NodeTypeComponent,
				Ref:           datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "broadcastMessage"}}),
				Configuration: datatypes.NewJSONType(map[string]any{}),
			},
		},
		nil,
	)

	require.NoError(t, models.CreateAppMessage(
		database.Conn(),
		sourceCanvas.ID,
		sourceNodes[0].NodeID,
		map[string]any{"message": "ignored"},
	))

	messages, err := models.ListAppMessages()
	require.NoError(t, err)
	require.Len(t, messages, 1)

	require.NoError(t, worker.LockAndProcessMessage(messages[0]))

	var remaining int64
	require.NoError(t, database.Conn().Model(&models.AppMessage{}).Count(&remaining).Error)
	assert.Equal(t, int64(0), remaining)
}

func Test__AppMessageWorker_LockAndProcessMessage__skipsStaleSubscriptionAndDeliversToHealthySubscriber(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	worker := NewAppMessageWorker(r.Registry)

	sourceCanvas, sourceNodes := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID:        "broadcast-message",
				Name:          "Broadcast Message",
				Type:          models.NodeTypeComponent,
				Ref:           datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "broadcastMessage"}}),
				Configuration: datatypes.NewJSONType(map[string]any{}),
			},
		},
		nil,
	)

	targetCanvas, targetNodes := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "stale-on-broadcast",
				Name:   "Stale On Broadcast",
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "onBroadcast"}}),
				Configuration: datatypes.NewJSONType(map[string]any{
					"app": sourceCanvas.ID.String(),
				}),
			},
			{
				NodeID: "on-broadcast",
				Name:   "On Broadcast",
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "onBroadcast"}}),
				Configuration: datatypes.NewJSONType(map[string]any{
					"app": sourceCanvas.ID.String(),
				}),
			},
		},
		nil,
	)

	require.NoError(t, database.Conn().Create(&models.CanvasSubscription{
		SourceCanvasID: sourceCanvas.ID,
		TargetCanvasID: targetCanvas.ID,
		TargetNodeID:   "stale-on-broadcast",
	}).Error)
	require.NoError(t, database.Conn().Create(&models.CanvasSubscription{
		SourceCanvasID: sourceCanvas.ID,
		TargetCanvasID: targetCanvas.ID,
		TargetNodeID:   targetNodes[1].NodeID,
	}).Error)

	require.NoError(t, database.Conn().Exec(
		"UPDATE workflow_nodes SET deleted_at = NOW() WHERE workflow_id = ? AND node_id = ?",
		targetCanvas.ID,
		"stale-on-broadcast",
	).Error)

	payload := map[string]any{"message": "hello"}
	require.NoError(t, models.CreateAppMessage(database.Conn(), sourceCanvas.ID, sourceNodes[0].NodeID, payload))

	messages, err := models.ListAppMessages()
	require.NoError(t, err)
	require.Len(t, messages, 1)

	require.NoError(t, worker.LockAndProcessMessage(messages[0]))

	var remaining int64
	require.NoError(t, database.Conn().Model(&models.AppMessage{}).Count(&remaining).Error)
	assert.Equal(t, int64(0), remaining)

	support.VerifyCanvasEventsCount(t, targetCanvas.ID, 1)

	var staleSubs int64
	require.NoError(t, database.Conn().
		Model(&models.CanvasSubscription{}).
		Where("source_canvas_id = ? AND target_node_id = ?", sourceCanvas.ID, "stale-on-broadcast").
		Count(&staleSubs).
		Error)
	assert.Equal(t, int64(0), staleSubs)
}

func Test__AppMessageWorker_LockAndProcessMessage__deletesMessageWhenSourceNodeDeleted(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	worker := NewAppMessageWorker(r.Registry)

	sourceCanvas, sourceNodes := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID:        "broadcast-message",
				Name:          "Broadcast Message",
				Type:          models.NodeTypeComponent,
				Ref:           datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "broadcastMessage"}}),
				Configuration: datatypes.NewJSONType(map[string]any{}),
			},
		},
		nil,
	)

	require.NoError(t, models.CreateAppMessage(
		database.Conn(),
		sourceCanvas.ID,
		sourceNodes[0].NodeID,
		map[string]any{"message": "stale"},
	))

	require.NoError(t, models.DeleteCanvasNode(database.Conn(), sourceNodes[0]))

	messages, err := models.ListAppMessages()
	require.NoError(t, err)
	require.Len(t, messages, 1)

	require.NoError(t, worker.LockAndProcessMessage(messages[0]))

	var remaining int64
	require.NoError(t, database.Conn().Model(&models.AppMessage{}).Count(&remaining).Error)
	assert.Equal(t, int64(0), remaining)
}

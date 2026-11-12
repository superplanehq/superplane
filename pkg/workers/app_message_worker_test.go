package workers

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/triggers/messages"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

type failingBroadcastTrigger struct {
	messages.OnBroadcast
	deliveries int
	err        error
}

func (t *failingBroadcastTrigger) OnAppMessage(ctx core.AppMessageContext) error {
	t.deliveries++
	if t.deliveries == 2 {
		return t.err
	}
	return t.OnBroadcast.OnAppMessage(ctx)
}

func Test__AppMessageWorker_LockAndProcessMessage__retriesBroadcastAtomically(t *testing.T) {
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
			{
				NodeID: "other-on-broadcast",
				Name:   "Other On Broadcast",
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
	require.NoError(t, database.Conn().Create(&models.CanvasSubscription{
		SourceCanvasID: sourceCanvas.ID,
		TargetCanvasID: targetCanvas.ID,
		TargetNodeID:   targetNodes[1].NodeID,
	}).Error)

	payload := map[string]any{"message": "hello"}
	require.NoError(t, models.CreateAppMessage(database.Conn(), sourceCanvas.ID, sourceNodes[0].NodeID, payload))

	pending, err := models.ListAppMessages(database.Conn())
	require.NoError(t, err)
	require.Len(t, pending, 1)

	deliveryError := errors.New("subscriber unavailable")
	trigger := &failingBroadcastTrigger{err: deliveryError}
	r.Registry.Triggers["onBroadcast"] = trigger
	assert.ErrorIs(t, worker.LockAndProcessMessage(pending[0]), deliveryError)
	assert.Equal(t, 2, trigger.deliveries, "the failure must follow a successful subscriber insert")
	remainingMessages, err := models.ListAppMessages(database.Conn())
	require.NoError(t, err)
	assert.Len(t, remainingMessages, 1)
	support.VerifyCanvasEventsCount(t, targetCanvas.ID, 0)

	r.Registry.Triggers["onBroadcast"] = &messages.OnBroadcast{}
	require.NoError(t, worker.LockAndProcessMessage(pending[0]))
	// A stale poll result can process the consumed message again.
	require.NoError(t, worker.LockAndProcessMessage(pending[0]))

	var remaining int64
	require.NoError(t, database.Conn().Model(&models.AppMessage{}).Count(&remaining).Error)
	assert.Equal(t, int64(0), remaining)

	for _, node := range targetNodes {
		support.VerifyCanvasNodeEventsCount(t, targetCanvas.ID, node.NodeID, 1)
	}

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

func Test__AppMessageWorker_LockAndProcessMessage__discardsOversizedBroadcast(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	worker := NewAppMessageWorker(r.Registry)
	sourceCanvas, sourceNodes := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{
			NodeID: "broadcast-message",
			Name:   "Broadcast Message",
			Type:   models.NodeTypeComponent,
			Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "broadcastMessage"}}),
		},
	}, nil)
	targetCanvas, targetNodes := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{
			NodeID: "on-broadcast",
			Name:   "On Broadcast",
			Type:   models.NodeTypeTrigger,
			Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "onBroadcast"}}),
			Configuration: datatypes.NewJSONType(map[string]any{
				"app": sourceCanvas.ID.String(),
			}),
		},
	}, nil)
	require.NoError(t, database.Conn().Create(&models.CanvasSubscription{
		SourceCanvasID: sourceCanvas.ID,
		TargetCanvasID: targetCanvas.ID,
		TargetNodeID:   targetNodes[0].NodeID,
	}).Error)

	// The payload fits on its own, but the broadcast envelope exceeds the limit.
	payload := strings.Repeat("a", config.MaxPayloadSize()-2)
	require.NoError(t, models.CreateAppMessage(database.Conn(), sourceCanvas.ID, sourceNodes[0].NodeID, payload))
	pending, err := models.ListAppMessages(database.Conn())
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.NoError(t, worker.LockAndProcessMessage(pending[0]))

	remaining, err := models.ListAppMessages(database.Conn())
	require.NoError(t, err)
	assert.Empty(t, remaining)
	support.VerifyCanvasEventsCount(t, targetCanvas.ID, 0)

	require.NoError(t, models.CreateAppMessage(database.Conn(), sourceCanvas.ID, sourceNodes[0].NodeID, "hello"))
	pending, err = models.ListAppMessages(database.Conn())
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.NoError(t, worker.LockAndProcessMessage(pending[0]))

	remaining, err = models.ListAppMessages(database.Conn())
	require.NoError(t, err)
	assert.Empty(t, remaining)
	support.VerifyCanvasEventsCount(t, targetCanvas.ID, 1)
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

	messages, err := models.ListAppMessages(database.Conn())
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

	messages, err := models.ListAppMessages(database.Conn())
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

func Test__AppMessageWorker_LockAndProcessMessage__skipsTargetNodeInErrorState(t *testing.T) {
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

	require.NoError(t, database.Conn().
		Model(&models.CanvasNode{}).
		Where("workflow_id = ? AND node_id = ?", targetCanvas.ID, targetNodes[0].NodeID).
		Update("state", models.CanvasNodeStateError).
		Error)

	require.NoError(t, models.CreateAppMessage(
		database.Conn(),
		sourceCanvas.ID,
		sourceNodes[0].NodeID,
		map[string]any{"message": "hello"},
	))

	messages, err := models.ListAppMessages(database.Conn())
	require.NoError(t, err)
	require.Len(t, messages, 1)

	require.NoError(t, worker.LockAndProcessMessage(messages[0]))

	var remaining int64
	require.NoError(t, database.Conn().Model(&models.AppMessage{}).Count(&remaining).Error)
	assert.Equal(t, int64(0), remaining)

	support.VerifyCanvasEventsCount(t, targetCanvas.ID, 0)
}

func Test__AppMessageWorker_LockAndProcessMessage__skipsSubscriptionOnDeletedTargetCanvas(t *testing.T) {
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

	require.NoError(t, targetCanvas.SoftDelete())

	require.NoError(t, models.CreateAppMessage(
		database.Conn(),
		sourceCanvas.ID,
		sourceNodes[0].NodeID,
		map[string]any{"message": "hello"},
	))

	messages, err := models.ListAppMessages(database.Conn())
	require.NoError(t, err)
	require.Len(t, messages, 1)

	require.NoError(t, worker.LockAndProcessMessage(messages[0]))

	var remaining int64
	require.NoError(t, database.Conn().Model(&models.AppMessage{}).Count(&remaining).Error)
	assert.Equal(t, int64(0), remaining)

	support.VerifyCanvasEventsCount(t, targetCanvas.ID, 0)

	var subs int64
	require.NoError(t, database.Conn().
		Model(&models.CanvasSubscription{}).
		Where("source_canvas_id = ? AND target_canvas_id = ? AND target_node_id = ?",
			sourceCanvas.ID, targetCanvas.ID, targetNodes[0].NodeID).
		Count(&subs).
		Error)
	assert.Equal(t, int64(0), subs)
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

	messages, err := models.ListAppMessages(database.Conn())
	require.NoError(t, err)
	require.Len(t, messages, 1)

	require.NoError(t, worker.LockAndProcessMessage(messages[0]))

	var remaining int64
	require.NoError(t, database.Conn().Model(&models.AppMessage{}).Count(&remaining).Error)
	assert.Equal(t, int64(0), remaining)
}

func TestAppMessageWorkerFilterErrorRollsBackDeliveryAndNonmatchAcknowledges(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	db := database.DB(t.Context())
	source, sourceNodes := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{NodeID: "source", Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "broadcastMessage"}})}}, nil)
	listeners := []models.CanvasNode{}
	for i, filter := range []string{`message.payload.type == "completed"`, `len(message.payload.missing.value) > 0`} {
		listeners = append(listeners, models.CanvasNode{NodeID: fmt.Sprintf("listener-%d", i), Type: models.NodeTypeTrigger, Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "onBroadcast"}}), Configuration: datatypes.NewJSONType(map[string]any{"app": source.ID.String(), "filter": filter})})
	}
	target, nodes := support.CreateCanvas(t, r.Organization.ID, r.User, listeners, nil)
	for _, node := range nodes {
		require.NoError(t, db.Create(&models.CanvasSubscription{SourceCanvasID: source.ID, TargetCanvasID: target.ID, TargetNodeID: node.NodeID}).Error)
	}
	require.NoError(t, models.CreateAppMessage(db, source.ID, sourceNodes[0].NodeID, map[string]any{"type": "completed"}))
	pending, err := models.ListAppMessages(db)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	worker := NewAppMessageWorker(r.Registry)
	require.ErrorContains(t, worker.LockAndProcessMessage(pending[0]), "broadcast filter")
	support.VerifyCanvasEventsCount(t, target.ID, 0)
	remaining, err := models.ListAppMessages(db)
	require.NoError(t, err)
	require.Len(t, remaining, 1)
	nodes[1].Configuration = datatypes.NewJSONType(map[string]any{"app": source.ID.String(), "filter": "false"})
	require.NoError(t, db.Model(&nodes[1]).Update("configuration", nodes[1].Configuration).Error)
	require.NoError(t, worker.LockAndProcessMessage(pending[0]))
	support.VerifyCanvasNodeEventsCount(t, target.ID, nodes[0].NodeID, 1)
	support.VerifyCanvasNodeEventsCount(t, target.ID, nodes[1].NodeID, 0)
	remaining, err = models.ListAppMessages(db)
	require.NoError(t, err)
	require.Empty(t, remaining)
	var runs int64
	require.NoError(t, db.Model(&models.CanvasRun{}).Where("workflow_id = ? AND node_id = ?", target.ID, nodes[1].NodeID).Count(&runs).Error)
	require.Zero(t, runs, "filtered delivery must not create a run")
}

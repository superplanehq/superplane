package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func Test__MaybeScheduleCanvasOnErrorInTransaction__SchedulesOnErrorNode(t *testing.T) {
	r := support.Setup(t)

	onErrorNodeID := "on-error-node"
	failingNodeID := "failing-node"

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: "trigger", Type: models.NodeTypeTrigger},
			{NodeID: failingNodeID, Type: models.NodeTypeComponent},
			{NodeID: onErrorNodeID, Type: models.NodeTypeComponent},
		},
		[]models.Edge{},
	)

	version, err := models.FindLiveCanvasVersionInTransaction(database.Conn(), canvas.ID)
	require.NoError(t, err)
	nodes := version.Nodes
	for i := range nodes {
		if nodes[i].ID == onErrorNodeID {
			nodes[i].OnError = true
		}
	}
	version.Nodes = nodes
	require.NoError(t, database.Conn().Save(version).Error)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, failingNodeID, rootEvent.ID, rootEvent.ID, nil)

	var dispatch *models.OnErrorDispatch
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		var failErr error
		dispatch, failErr = execution.FailInTransaction(tx, models.CanvasNodeExecutionResultReasonError, "boom")
		return failErr
	})
	require.NoError(t, err)
	require.NotNil(t, dispatch)
	assert.Equal(t, onErrorNodeID, dispatch.QueueItem.NodeID)
	assert.Equal(t, models.CanvasOnErrorEventSourceNodeID, dispatch.Event.NodeID)

	eventData, ok := dispatch.Event.Data.Data().(map[string]any)
	require.True(t, ok)
	assert.Equal(t, models.CanvasOnErrorEventType, eventData["type"])

	payload, ok := eventData["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, failingNodeID, payload["failedNodeId"])
	assert.Equal(t, "boom", payload["errorMessage"])
	assert.Equal(t, execution.ID.String(), payload["executionId"])
}

func Test__MaybeScheduleCanvasOnErrorInTransaction__SkipsWhenUnset(t *testing.T) {
	r := support.Setup(t)

	failingNodeID := "failing-node"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: "trigger", Type: models.NodeTypeTrigger},
			{NodeID: failingNodeID, Type: models.NodeTypeComponent},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, failingNodeID, rootEvent.ID, rootEvent.ID, nil)

	var dispatch *models.OnErrorDispatch
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		var failErr error
		dispatch, failErr = execution.FailInTransaction(tx, models.CanvasNodeExecutionResultReasonError, "boom")
		return failErr
	})
	require.NoError(t, err)
	assert.Nil(t, dispatch)
}

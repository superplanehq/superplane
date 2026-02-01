package canvases

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func createNodeQueueItem(t *testing.T, workflowID uuid.UUID, nodeID string, eventID uuid.UUID, rootEventID *uuid.UUID) *models.CanvasNodeQueueItem {
	now := time.Now()

	queueItem := models.CanvasNodeQueueItem{
		ID:         uuid.New(),
		WorkflowID: workflowID,
		NodeID:     nodeID,
		EventID:    eventID,
		CreatedAt:  &now,
	}

	if rootEventID != nil {
		queueItem.RootEventID = *rootEventID
	} else {
		queueItem.RootEventID = eventID
	}

	err := database.Conn().Create(&queueItem).Error
	require.NoError(t, err)

	return &queueItem
}

func Test__ListNodeQueueItems__ReturnsEmptyListWhenNoQueueItemsExist(t *testing.T) {
	r := support.Setup(t)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Name:   "Node 1",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	response, err := ListNodeQueueItems(context.Background(), r.Registry, canvas.ID.String(), "node-1", 10, nil)
	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Empty(t, response.Items)
	assert.Equal(t, uint32(0), response.TotalCount)
	assert.False(t, response.HasNextPage)
	assert.Nil(t, response.LastTimestamp)
}

func Test__ListNodeQueueItems__ReturnsQueueItemsWithInputData(t *testing.T) {
	r := support.Setup(t)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Name:   "Node 1",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	inputEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, "node-1", "default", nil, map[string]interface{}{
		"test_field": "test_value",
	})

	queueItem := createNodeQueueItem(t, canvas.ID, "node-1", inputEvent.ID, nil)

	response, err := ListNodeQueueItems(context.Background(), r.Registry, canvas.ID.String(), "node-1", 10, nil)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Len(t, response.Items, 1)
	assert.Equal(t, uint32(1), response.TotalCount)
	assert.False(t, response.HasNextPage)

	item := response.Items[0]
	assert.Equal(t, queueItem.ID.String(), item.Id)
	assert.Equal(t, canvas.ID.String(), item.CanvasId)
	assert.Equal(t, "node-1", item.NodeId)
	assert.NotNil(t, item.CreatedAt)
	assert.NotNil(t, item.Input)
	assert.NotNil(t, item.RootEvent)

	inputData := item.Input.AsMap()
	assert.Equal(t, "test_value", inputData["test_field"])
}

func Test__ListNodeQueueItems__ReturnsQueueItemsWithRootEvent(t *testing.T) {
	r := support.Setup(t)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Name:   "Node 1",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "root-node", "default", nil)
	inputEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, "node-1", "default", nil, map[string]interface{}{
		"data": "value",
	})

	queueItem := createNodeQueueItem(t, canvas.ID, "node-1", inputEvent.ID, &rootEvent.ID)

	response, err := ListNodeQueueItems(context.Background(), r.Registry, canvas.ID.String(), "node-1", 10, nil)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Len(t, response.Items, 1)

	item := response.Items[0]
	assert.Equal(t, queueItem.ID.String(), item.Id)
	assert.NotNil(t, item.RootEvent)
	assert.Equal(t, rootEvent.ID.String(), item.RootEvent.Id)
}

func Test__ListNodeQueueItems__HandlesPaginationCorrectly(t *testing.T) {
	r := support.Setup(t)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Name:   "Node 1",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	var queueItems []models.CanvasNodeQueueItem
	for i := 0; i < 5; i++ {
		inputEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, "node-1", "default", nil, map[string]interface{}{
			"index": i,
		})
		queueItem := createNodeQueueItem(t, canvas.ID, "node-1", inputEvent.ID, nil)
		queueItems = append(queueItems, *queueItem)
	}

	response, err := ListNodeQueueItems(context.Background(), r.Registry, canvas.ID.String(), "node-1", 3, nil)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Len(t, response.Items, 3)
	assert.Equal(t, uint32(5), response.TotalCount)
	assert.True(t, response.HasNextPage)
	assert.NotNil(t, response.LastTimestamp)
}

func Test__ListNodeQueueItems__FiltersQueueItemsByNodeID(t *testing.T) {
	r := support.Setup(t)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Name:   "Node 1",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
			{
				NodeID: "node-2",
				Name:   "Node 2",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	inputEvent1 := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
	inputEvent2 := support.EmitCanvasEventForNode(t, canvas.ID, "node-2", "default", nil)

	queueItem1 := createNodeQueueItem(t, canvas.ID, "node-1", inputEvent1.ID, nil)
	createNodeQueueItem(t, canvas.ID, "node-2", inputEvent2.ID, nil)

	response, err := ListNodeQueueItems(context.Background(), r.Registry, canvas.ID.String(), "node-1", 10, nil)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Len(t, response.Items, 1)
	assert.Equal(t, uint32(1), response.TotalCount)

	item := response.Items[0]
	assert.Equal(t, queueItem1.ID.String(), item.Id)
	assert.Equal(t, "node-1", item.NodeId)
}

func Test__ListNodeQueueItems__HandlesPaginationWithTimestamp(t *testing.T) {
	r := support.Setup(t)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Name:   "Node 1",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	for i := 0; i < 3; i++ {
		inputEvent := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
		createNodeQueueItem(t, canvas.ID, "node-1", inputEvent.ID, nil)
	}

	firstResponse, err := ListNodeQueueItems(context.Background(), r.Registry, canvas.ID.String(), "node-1", 2, nil)
	require.NoError(t, err)
	require.Len(t, firstResponse.Items, 2)
	assert.True(t, firstResponse.HasNextPage)

	secondResponse, err := ListNodeQueueItems(context.Background(), r.Registry, canvas.ID.String(), "node-1", 2, firstResponse.LastTimestamp)
	require.NoError(t, err)
	require.Len(t, secondResponse.Items, 1)
	assert.False(t, secondResponse.HasNextPage)
}

func Test__ListNodeQueueItems__ReturnsErrorForInvalidCanvasID(t *testing.T) {
	r := support.Setup(t)

	response, err := ListNodeQueueItems(context.Background(), r.Registry, "invalid-uuid", "node-1", 10, nil)
	require.Error(t, err)
	assert.Nil(t, response)
}

func Test__SerializeNodeQueueItems__HandlesEmptyList(t *testing.T) {
	result, err := SerializeNodeQueueItems([]models.CanvasNodeQueueItem{})
	require.NoError(t, err)
	assert.Empty(t, result)
}

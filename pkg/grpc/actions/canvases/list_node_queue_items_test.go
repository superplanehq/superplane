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

	response, err := ListNodeQueueItems(context.Background(), database.DB(t.Context()), canvas, "node-1", 10, nil, "")
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

	response, err := ListNodeQueueItems(context.Background(), database.DB(t.Context()), canvas, "node-1", 10, nil, "")
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

	response, err := ListNodeQueueItems(context.Background(), database.DB(t.Context()), canvas, "node-1", 10, nil, "")
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

	response, err := ListNodeQueueItems(context.Background(), database.DB(t.Context()), canvas, "node-1", 3, nil, "")
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

	response, err := ListNodeQueueItems(context.Background(), database.DB(t.Context()), canvas, "node-1", 10, nil, "")
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

	firstResponse, err := ListNodeQueueItems(context.Background(), database.DB(t.Context()), canvas, "node-1", 2, nil, "")
	require.NoError(t, err)
	require.Len(t, firstResponse.Items, 2)
	assert.True(t, firstResponse.HasNextPage)

	secondResponse, err := ListNodeQueueItems(context.Background(), database.DB(t.Context()), canvas, "node-1", 2, firstResponse.LastTimestamp, "")
	require.NoError(t, err)
	require.Len(t, secondResponse.Items, 1)
	assert.False(t, secondResponse.HasNextPage)
}

func Test__ListNodeQueueItems__PaginatesAcrossRowsSharingATimestamp(t *testing.T) {
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

	//
	// Three items sharing one created_at, the shape a batch insert produces.
	// Paging two at a time ends page 1 inside the group, so the cursor has to
	// carry the id to reach the third row - a timestamp alone excludes it.
	//
	sharedCreatedAt := time.Now().UTC().Truncate(time.Microsecond)
	expectedIDs := []string{}
	for i := 0; i < 3; i++ {
		inputEvent := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
		queueItem := createNodeQueueItem(t, canvas.ID, "node-1", inputEvent.ID, nil)
		require.NoError(t, database.Conn().
			Model(&models.CanvasNodeQueueItem{}).
			Where("id = ?", queueItem.ID).
			Update("created_at", sharedCreatedAt).Error)
		expectedIDs = append(expectedIDs, queueItem.ID.String())
	}

	firstResponse, err := ListNodeQueueItems(context.Background(), database.DB(t.Context()), canvas, "node-1", 2, nil, "")
	require.NoError(t, err)
	require.Len(t, firstResponse.Items, 2)
	require.NotEmpty(t, firstResponse.LastId)

	secondResponse, err := ListNodeQueueItems(context.Background(), database.DB(t.Context()), canvas, "node-1", 2, firstResponse.LastTimestamp, firstResponse.LastId)
	require.NoError(t, err)
	require.Len(t, secondResponse.Items, 1)

	returnedIDs := []string{}
	for _, item := range append(firstResponse.Items, secondResponse.Items...) {
		returnedIDs = append(returnedIDs, item.Id)
	}
	assert.ElementsMatch(t, expectedIDs, returnedIDs, "no row sharing the boundary timestamp may be skipped")
}

func Test__ListNodeQueueItems__TimestampOnlyCursorKeepsWorking(t *testing.T) {
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

	//
	// A client that has not been updated sends `before` without `before_id`.
	// Distinct timestamps, so it still pages the way it always has.
	//
	for i := 0; i < 3; i++ {
		inputEvent := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
		createNodeQueueItem(t, canvas.ID, "node-1", inputEvent.ID, nil)
	}

	firstResponse, err := ListNodeQueueItems(context.Background(), database.DB(t.Context()), canvas, "node-1", 2, nil, "")
	require.NoError(t, err)
	require.Len(t, firstResponse.Items, 2)

	secondResponse, err := ListNodeQueueItems(context.Background(), database.DB(t.Context()), canvas, "node-1", 2, firstResponse.LastTimestamp, "")
	require.NoError(t, err)
	require.Len(t, secondResponse.Items, 1)
}

func Test__SerializeNodeQueueItems__HandlesEmptyList(t *testing.T) {
	result, err := SerializeNodeQueueItems(database.Conn(), []models.CanvasNodeQueueItem{})
	require.NoError(t, err)
	assert.Empty(t, result)
}

func Test__SerializeNodeQueueItemsWithInputEvents__KeepsItemsWithMissingInput(t *testing.T) {
	now := time.Now()
	validEventID := uuid.New()
	missingEventID := uuid.New()
	missingQueueItemID := uuid.New()
	validQueueItemID := uuid.New()

	result, err := serializeNodeQueueItemsWithInputEvents(
		[]models.CanvasNodeQueueItem{
			{
				ID:         missingQueueItemID,
				WorkflowID: uuid.New(),
				NodeID:     "node-1",
				EventID:    missingEventID,
				CreatedAt:  &now,
			},
			{
				ID:         validQueueItemID,
				WorkflowID: uuid.New(),
				NodeID:     "node-1",
				EventID:    validEventID,
				CreatedAt:  &now,
			},
		},
		[]models.CanvasEvent{
			{
				ID:   validEventID,
				Data: models.NewJSONValue(map[string]any{"message": "queued"}),
			},
		},
	)

	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, missingQueueItemID.String(), result[0].Id)
	assert.Empty(t, result[0].Input.AsMap())
	assert.Equal(t, validQueueItemID.String(), result[1].Id)
	assert.Equal(t, "queued", result[1].Input.AsMap()["message"])
}

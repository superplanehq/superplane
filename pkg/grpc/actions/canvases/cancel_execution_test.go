package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__CancelExecution__CancelsExecutionSuccessfully(t *testing.T) {
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

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", rootEvent.ID, rootEvent.ID, nil)

	response, err := CancelExecution(context.Background(), r.AuthService, r.Encryptor, r.Organization.ID.String(), r.Registry, canvas.ID, execution.ID)
	require.NoError(t, err)
	require.NotNil(t, response)

	updatedExecution, err := models.FindNodeExecution(canvas.ID, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateFinished, updatedExecution.State)
	assert.Equal(t, models.CanvasNodeExecutionResultCancelled, updatedExecution.Result)
}

func Test__CancelExecution__ReturnsNotFoundForNonExistentExecution(t *testing.T) {
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

	nonExistentID := uuid.New()
	_, err := CancelExecution(context.Background(), r.AuthService, r.Encryptor, r.Organization.ID.String(), r.Registry, canvas.ID, nonExistentID)
	require.Error(t, err)
}

func Test__CancelExecution__ReturnsErrorWhenCancellingChild(t *testing.T) {
	r := support.Setup(t)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Name:   "Node 1",
				Type:   models.NodeTypeBlueprint,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Blueprint: &models.BlueprintRef{ID: "test-blueprint"},
				}),
			},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
	parentExecution := support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", rootEvent.ID, rootEvent.ID, nil)

	childEvent := support.EmitCanvasEventForNode(t, canvas.ID, "child-node-1", "default", nil)
	childExecution := support.CreateCanvasNodeExecution(t, canvas.ID, "child-node-1", rootEvent.ID, childEvent.ID, &parentExecution.ID)

	_, err := CancelExecution(context.Background(), r.AuthService, r.Encryptor, r.Organization.ID.String(), r.Registry, canvas.ID, childExecution.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot cancel child execution directly")

	updatedChildExecution, err := models.FindNodeExecution(canvas.ID, childExecution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStatePending, updatedChildExecution.State)
	assert.Equal(t, "", updatedChildExecution.Result)

	updatedParentExecution, err := models.FindNodeExecution(canvas.ID, parentExecution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStatePending, updatedParentExecution.State)
	assert.Equal(t, "", updatedParentExecution.Result)
}

func Test__CancelExecution__CancelsAllChildrenWhenCancellingParent(t *testing.T) {
	r := support.Setup(t)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Name:   "Node 1",
				Type:   models.NodeTypeBlueprint,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Blueprint: &models.BlueprintRef{ID: "test-blueprint"},
				}),
			},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
	parentExecution := support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", rootEvent.ID, rootEvent.ID, nil)

	childEvent1 := support.EmitCanvasEventForNode(t, canvas.ID, "child-node-1", "default", nil)
	childEvent2 := support.EmitCanvasEventForNode(t, canvas.ID, "child-node-2", "default", nil)

	childExecution1 := support.CreateCanvasNodeExecution(t, canvas.ID, "child-node-1", rootEvent.ID, childEvent1.ID, &parentExecution.ID)
	childExecution2 := support.CreateCanvasNodeExecution(t, canvas.ID, "child-node-2", rootEvent.ID, childEvent2.ID, &parentExecution.ID)

	response, err := CancelExecution(context.Background(), r.AuthService, r.Encryptor, r.Organization.ID.String(), r.Registry, canvas.ID, parentExecution.ID)
	require.NoError(t, err)
	require.NotNil(t, response)

	updatedParentExecution, err := models.FindNodeExecution(canvas.ID, parentExecution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateFinished, updatedParentExecution.State)
	assert.Equal(t, models.CanvasNodeExecutionResultCancelled, updatedParentExecution.Result)

	updatedChildExecution1, err := models.FindNodeExecution(canvas.ID, childExecution1.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateFinished, updatedChildExecution1.State)
	assert.Equal(t, models.CanvasNodeExecutionResultCancelled, updatedChildExecution1.Result)

	updatedChildExecution2, err := models.FindNodeExecution(canvas.ID, childExecution2.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateFinished, updatedChildExecution2.State)
	assert.Equal(t, models.CanvasNodeExecutionResultCancelled, updatedChildExecution2.Result)
}

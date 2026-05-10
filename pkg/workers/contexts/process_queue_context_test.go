package contexts

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__BuildProcessQueueContext__ResolvesExecutionIdToCreatedExecution(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	triggerNodeID := "trigger-1"
	componentNodeID := "component-1"
	canvas, nodes := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNodeID,
				Name:   triggerNodeID,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID: componentNodeID,
				Name:   componentNodeID,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
				Configuration: datatypes.NewJSONType(map[string]any{
					"runUrl": "/canvases/x/runs/{{ executionId() }}?event={{ eventId() }}",
				}),
			},
		},
		[]models.Edge{
			{SourceID: triggerNodeID, TargetID: componentNodeID, Channel: "default"},
		},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, triggerNodeID, "default", nil)
	queueItem := support.CreateQueueItem(t, canvas.ID, componentNodeID, rootEvent.ID, rootEvent.ID)

	var componentNode *models.CanvasNode
	for i, n := range nodes {
		if n.NodeID == componentNodeID {
			componentNode = &nodes[i]
			break
		}
	}
	require.NotNil(t, componentNode)

	ctx, err := BuildProcessQueueContext(nil, database.Conn(), componentNode, queueItem, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, ctx)

	resolvedConfig, ok := ctx.Configuration.(map[string]any)
	require.True(t, ok, "expected Configuration to be a map after resolution")
	resolvedURL, ok := resolvedConfig["runUrl"].(string)
	require.True(t, ok, "expected runUrl to be a string after resolution")
	assert.Contains(t, resolvedURL, "/canvases/x/runs/")
	assert.Contains(t, resolvedURL, "?event="+rootEvent.ID.String())

	executionCtx, err := ctx.CreateExecution()
	require.NoError(t, err)
	require.NotNil(t, executionCtx)

	expected := "/canvases/x/runs/" + executionCtx.ID.String() + "?event=" + rootEvent.ID.String()
	assert.Equal(t, expected, resolvedURL,
		"resolved executionId() must match the created execution's actual ID")
}

func Test__BuildProcessQueueContext__RebindsExpressionsToExistingExecutionAfterFindByKV(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	triggerNodeID := "trigger-1"
	mergeNodeID := "merge-1"
	canvas, nodes := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNodeID,
				Name:   triggerNodeID,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID: mergeNodeID,
				Name:   mergeNodeID,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "merge"}}),
			},
		},
		[]models.Edge{
			{SourceID: triggerNodeID, TargetID: mergeNodeID, Channel: "default"},
		},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, triggerNodeID, "default", nil)
	queueItem := support.CreateQueueItem(t, canvas.ID, mergeNodeID, rootEvent.ID, rootEvent.ID)

	var mergeNode *models.CanvasNode
	for i, n := range nodes {
		if n.NodeID == mergeNodeID {
			mergeNode = &nodes[i]
			break
		}
	}
	require.NotNil(t, mergeNode)

	existingExecution := support.CreateNodeExecutionWithConfiguration(
		t, canvas.ID, mergeNodeID, rootEvent.ID, rootEvent.ID, nil, map[string]any{},
	)
	require.NoError(t, models.CreateNodeExecutionKVInTransaction(
		database.Conn(), canvas.ID, mergeNodeID, existingExecution.ID, "merge_group", "g1",
	))

	ctx, err := BuildProcessQueueContext(nil, database.Conn(), mergeNode, queueItem, nil, nil)
	require.NoError(t, err)

	preBindResult, err := ctx.Expressions.Run("executionId()")
	require.NoError(t, err)
	preBindID, ok := preBindResult.(string)
	require.True(t, ok)
	assert.NotEqual(t, existingExecution.ID.String(), preBindID,
		"before FindExecutionByKV, ctx.Expressions returns the pre-generated id")

	executionCtx, err := ctx.FindExecutionByKV("merge_group", "g1")
	require.NoError(t, err)
	require.NotNil(t, executionCtx)
	assert.Equal(t, existingExecution.ID, executionCtx.ID)

	postBindResult, err := ctx.Expressions.Run("executionId()")
	require.NoError(t, err)
	postBindID, ok := postBindResult.(string)
	require.True(t, ok)
	assert.Equal(t, existingExecution.ID.String(), postBindID,
		"after FindExecutionByKV, ctx.Expressions must rebind to the existing execution's id")
}

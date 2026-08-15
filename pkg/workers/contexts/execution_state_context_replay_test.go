package contexts

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/triggers/onerror"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Given a replay that fails on a canvas with an on-error node
// configured, the on-error node receives no event - unlike a normal
// (non-replay) failure, which does dispatch to it (see on_error_dispatch_test.go).
func Test__Fail_ReplayExecutionSkipsOnErrorDispatch(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	triggerNodeID := "trigger-1"
	componentNodeID := "deploy-1"
	onErrorNodeID := "onerror-1"

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNodeID,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID: componentNodeID,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
			{
				NodeID: onErrorNodeID,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: onerror.TriggerName}}),
			},
		},
		[]models.Edge{
			{SourceID: triggerNodeID, TargetID: componentNodeID, Channel: "default"},
		},
	)

	node, err := models.FindCanvasNode(database.Conn(), canvas.ID, componentNodeID)
	require.NoError(t, err)

	var run *models.CanvasRun
	var queueItems []*models.CanvasNodeQueueItem
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		var txErr error
		run, queueItems, txErr = models.CreateMultiInputNodeReplay(tx, node, nil, []models.ReplayInput{{Payload: map[string]any{"a": 1}}})
		return txErr
	})
	require.NoError(t, err)
	require.True(t, run.IsReplay)

	//
	// Create the pending execution for the replay directly (this test targets
	// ExecutionStateContext.Fail in isolation, not the whole queue pipeline).
	//
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, componentNodeID, queueItems[0].RootEventID, queueItems[0].EventID)
	execution.RunID = run.ID
	require.NoError(t, database.Conn().Save(execution).Error)

	newEvents := []models.CanvasEvent{}
	onNewEvents := func(events []models.CanvasEvent) {
		newEvents = append(newEvents, events...)
	}

	ctx := NewExecutionStateContext(database.Conn(), execution, onNewEvents)
	require.NoError(t, ctx.Fail(models.CanvasNodeExecutionResultReasonError, "boom"))

	assert.Empty(t, newEvents, "a failing replay must not notify On Error nodes")

	onErrorEvents, err := models.ListCanvasEvents(database.Conn(), canvas.ID, onErrorNodeID, 10, nil)
	require.NoError(t, err)
	assert.Empty(t, onErrorEvents, "no event should have been stored for the on-error node")
}

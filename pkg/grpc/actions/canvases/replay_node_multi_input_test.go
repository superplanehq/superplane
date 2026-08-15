package canvases_test

// This file uses the external canvases_test package, not the internal
// canvases package replay_node_test.go uses, because it needs pkg/workers to
// drive queue items to a real execution - and pkg/workers transitively
// imports pkg/grpc/actions/canvases (via pkg/workers/eventdistributer), so
// importing pkg/workers from *inside* package canvases is an import cycle.
// Being a separate compiled test binary, canvases_test has no such problem.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

// drainQueue drives n queue items for nodeID through the real
// NodeQueueWorker loop, the same way pkg/workers/replay_test.go's own
// helpers do - so a multi-input replay's queue items actually accumulate
// into one execution instead of the test only asserting on rows this
// package created directly.
func drainQueue(t *testing.T, r *support.ResourceRegistry, canvas *models.Canvas, nodeID string, n int) {
	logger := log.NewEntry(log.New())
	worker := workers.NewNodeQueueWorker(r.Registry, r.GitProvider, "")
	for i := 0; i < n; i++ {
		node, err := models.FindCanvasNode(database.Conn(), canvas.ID, nodeID)
		require.NoError(t, err)
		require.NoError(t, worker.LockAndProcessNode(logger, *node, time.Now()))
	}
}

func createJoinCanvas(t *testing.T, r *support.ResourceRegistry) (*models.Canvas, string, string, string) {
	laneA, laneB := "lane-a", "lane-b"
	joinNodeID := "join"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: laneA, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
			{NodeID: laneB, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
			{NodeID: joinNodeID, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "merge"}})},
		},
		[]models.Edge{
			{SourceID: laneA, TargetID: joinNodeID, Channel: "default"},
			{SourceID: laneB, TargetID: joinNodeID, Channel: "default"},
		},
	)
	return canvas, laneA, laneB, joinNodeID
}

func Test__ReplayNode_ExplicitMultiInputCreatesOneRunTwoQueueItemsOneFinishedExecution(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, laneA, laneB, joinNodeID := createJoinCanvas(t, r)

	explicitInputs := []models.ReplayInput{
		{Payload: map[string]any{"v": "from-a"}, SourceNodeID: &laneA},
		{Payload: map[string]any{"v": "from-b"}, SourceNodeID: &laneB},
	}

	response, err := canvases.ReplayNode(context.Background(), database.DB(t.Context()), canvas, joinNodeID, nil, nil, nil, explicitInputs)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Len(t, response.QueueItemIds, 2, "one queue item per input")

	runs, err := models.ListCanvasRuns(canvas.ID, 100, nil, models.CanvasRunFilters{})
	require.NoError(t, err)
	require.Len(t, runs, 1, "exactly one replay run")

	drainQueue(t, r, canvas, joinNodeID, len(response.QueueItemIds))

	executions, err := models.ListNodeExecutions(database.Conn(), canvas.ID, joinNodeID, nil, nil, 10, nil)
	require.NoError(t, err)
	require.Len(t, executions, 1, "exactly one join execution")
	assert.Equal(t, models.CanvasNodeExecutionStateFinished, executions[0].State, "the join execution must reach Finished")
}

func Test__ReplayNode_EmptyInputsFallsBackToLegacyPayload(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	sourceNodeID := "source-1"
	targetNodeID := "target-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: sourceNodeID, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
			{NodeID: targetNodeID, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
		},
		[]models.Edge{
			{SourceID: sourceNodeID, TargetID: targetNodeID, Channel: "default"},
		},
	)

	response, err := canvases.ReplayNode(context.Background(), database.DB(t.Context()), canvas, targetNodeID, nil, map[string]any{"v": "legacy"}, nil, []models.ReplayInput{})
	require.NoError(t, err)
	require.Len(t, response.QueueItemIds, 1, "empty inputs must fall back to the single legacy payload")

	drainQueue(t, r, canvas, targetNodeID, 1)

	executions, err := models.ListNodeExecutions(database.Conn(), canvas.ID, targetNodeID, nil, nil, 10, nil)
	require.NoError(t, err)
	require.Len(t, executions, 1)
}

func Test__ReplayNode_InputOrderDoesNotAffectMergeResult(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, laneA, laneB, joinNodeID := createJoinCanvas(t, r)

	inputA := models.ReplayInput{Payload: map[string]any{"v": "from-a"}, SourceNodeID: &laneA}
	inputB := models.ReplayInput{Payload: map[string]any{"v": "from-b"}, SourceNodeID: &laneB}

	responseAB, err := canvases.ReplayNode(context.Background(), database.DB(t.Context()), canvas, joinNodeID, nil, nil, nil, []models.ReplayInput{inputA, inputB})
	require.NoError(t, err)
	drainQueue(t, r, canvas, joinNodeID, len(responseAB.QueueItemIds))

	responseBA, err := canvases.ReplayNode(context.Background(), database.DB(t.Context()), canvas, joinNodeID, nil, nil, nil, []models.ReplayInput{inputB, inputA})
	require.NoError(t, err)
	drainQueue(t, r, canvas, joinNodeID, len(responseBA.QueueItemIds))

	executions, err := models.ListNodeExecutions(database.Conn(), canvas.ID, joinNodeID, nil, nil, 10, nil)
	require.NoError(t, err)
	require.Len(t, executions, 2, "one execution per replay call, regardless of input order")

	for _, execution := range executions {
		assert.Equal(t, models.CanvasNodeExecutionStateFinished, execution.State)
		assert.Equal(t, models.CanvasNodeExecutionResultPassed, execution.Result, "merge must reach the same passing result regardless of input order")
	}
}

func Test__ReplayNode_ReplayOfMultiInputReplayExpandsSameSources(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, laneA, laneB, joinNodeID := createJoinCanvas(t, r)

	explicitInputs := []models.ReplayInput{
		{Payload: map[string]any{"v": "from-a"}, SourceNodeID: &laneA},
		{Payload: map[string]any{"v": "from-b"}, SourceNodeID: &laneB},
	}

	response1, err := canvases.ReplayNode(context.Background(), database.DB(t.Context()), canvas, joinNodeID, nil, nil, nil, explicitInputs)
	require.NoError(t, err)
	drainQueue(t, r, canvas, joinNodeID, len(response1.QueueItemIds))

	executions, err := models.ListNodeExecutions(database.Conn(), canvas.ID, joinNodeID, nil, nil, 10, nil)
	require.NoError(t, err)
	require.Len(t, executions, 1)
	joinExecution := executions[0]

	response2, err := canvases.ReplayNode(context.Background(), database.DB(t.Context()), canvas, joinNodeID, &joinExecution.ID, nil, nil, nil)
	require.NoError(t, err, "replaying a multi-input replay's execution must succeed, not 400")
	require.Len(t, response2.QueueItemIds, 2, "must expand back into 2 inputs, one per original source, not collapse into 1")

	gotSources := make([]string, 0, 2)
	for _, queueItemID := range response2.QueueItemIds {
		id, err := uuid.Parse(queueItemID)
		require.NoError(t, err)
		var queueItem models.CanvasNodeQueueItem
		require.NoError(t, database.Conn().Where("id = ?", id).First(&queueItem).Error)
		event, err := models.FindCanvasEvent(queueItem.EventID)
		require.NoError(t, err)
		gotSources = append(gotSources, event.NodeID)
	}
	assert.ElementsMatch(t, []string{laneA, laneB}, gotSources, "the replay-of-replay must keep the same source composition")
}

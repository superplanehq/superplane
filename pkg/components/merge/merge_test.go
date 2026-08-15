package merge

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test_Merge(t *testing.T) {
	steps := NewMergeTestSteps(t)

	steps.CreateWorkflow()
	steps.CreateEvents()
	steps.CreateQueueItems()

	m := &Merge{}

	steps.ProcessFirstEvent(m)
	steps.AssertNodeExecutionCount(1)
	steps.AssertExecutionPending()
	steps.AssertNodeIsAllowedToProcessNextQueueItem()

	steps.ProcessSecondEvent(m)
	steps.AssertNodeExecutionCount(1)
	steps.AssertExecutionFinished()
	steps.AssertNodeIsAllowedToProcessNextQueueItem()

	steps.AssertQueueIsEmpty()
}

func Test_Merge_StopIfExpression(t *testing.T) {
	steps := NewMergeTestSteps(t)

	steps.CreateWorkflow()
	steps.SetMergeConfiguration(map[string]any{
		"stopIfExpression": "$[\"process-1\"].result == \"fail\"",
	})

	steps.CreateEventsWithData(
		map[string]any{"result": "fail"},
		map[string]any{"result": "ok"},
	)
	steps.CreateQueueItems()

	m := &Merge{}

	// First event should immediately finish the merge due to stop expression
	steps.ProcessFirstEventExpectFinish(m)
	steps.AssertNodeExecutionCount(1)
	steps.AssertExecutionFailed()
	steps.AssertNodeIsAllowedToProcessNextQueueItem()

	// Second event should be dequeued and not re-finish the execution
	steps.ProcessSecondEventExpectNoFinish(m)
	steps.AssertNodeExecutionCount(1)
	steps.AssertExecutionFinished()
	steps.AssertNodeIsAllowedToProcessNextQueueItem()

	steps.AssertQueueIsEmpty()
}

func Test_Merge_BadStopIfExpression(t *testing.T) {
	steps := NewMergeTestSteps(t)
	steps.CreateWorkflow()

	//
	// Set up an expression that does not evaluate to a boolean value.
	//
	steps.SetMergeConfiguration(map[string]any{
		"stopIfExpression": "$[\"process-1\"].result",
	})

	steps.CreateEventsWithData(
		map[string]any{"result": "fail"},
		map[string]any{"result": "ok"},
	)
	steps.CreateQueueItems()

	m := &Merge{}

	//
	// First event should fail the execution and mark the merge as stopped early.
	//
	steps.ProcessFirstEventExpectFinish(m)
	steps.AssertNodeExecutionCount(1)
	steps.AssertExecutionFailedWithError("stop expression must evaluate to boolean, got string: fail")
	steps.AssertNodeIsAllowedToProcessNextQueueItem()

	// Second event should be dequeued and not re-finish the execution
	steps.ProcessSecondEventExpectNoFinish(m)
	steps.AssertNodeExecutionCount(1)
	steps.AssertExecutionFinished()
	steps.AssertNodeIsAllowedToProcessNextQueueItem()

	steps.AssertQueueIsEmpty()
}

func Test_Merge_StopIfExpressionEvaluationError(t *testing.T) {
	steps := NewMergeTestSteps(t)
	steps.CreateWorkflow()

	steps.SetMergeConfiguration(map[string]any{
		"stopIfExpression": "$[\"missing-node\"].result == \"fail\"",
	})

	steps.CreateEventsWithData(
		map[string]any{"result": "fail"},
		map[string]any{"result": "ok"},
	)
	steps.CreateQueueItems()

	m := &Merge{}

	steps.ProcessFirstEventExpectFinish(m)
	steps.AssertNodeExecutionCount(1)
	steps.AssertExecutionFailedWithErrorContaining("expression evaluation failed")
	steps.AssertNodeIsAllowedToProcessNextQueueItem()

	steps.ProcessSecondEventExpectNoFinish(m)
	steps.AssertNodeExecutionCount(1)
	steps.AssertExecutionFinished()
	steps.AssertNodeIsAllowedToProcessNextQueueItem()

	steps.AssertQueueIsEmpty()
}

func Test_Merge_StopIfExpression_SourceNodeReference(t *testing.T) {
	steps := NewMergeTestSteps(t)

	steps.CreateWorkflow()
	steps.SetMergeConfiguration(map[string]any{
		"stopIfExpression": "$[\"process-1\"].result == \"fail\"",
	})

	steps.CreateEventsWithData(
		map[string]any{"result": "fail"},
		map[string]any{"result": "ok"},
	)
	steps.CreateQueueItems()

	m := &Merge{}

	steps.ProcessFirstEventExpectFinish(m)
	steps.AssertNodeExecutionCount(1)
	steps.AssertExecutionFailed()
	steps.AssertNodeIsAllowedToProcessNextQueueItem()

	steps.ProcessSecondEventExpectNoFinish(m)
	steps.AssertNodeExecutionCount(1)
	steps.AssertExecutionFinished()
	steps.AssertNodeIsAllowedToProcessNextQueueItem()

	steps.AssertQueueIsEmpty()
}

func Test_Merge_ReplayStyleQueueItemsAccumulateIntoSingleExecution(t *testing.T) {
	steps := NewMergeTestSteps(t)
	steps.CreateWorkflow()
	steps.CreateReplayStyleEventsAndQueueItems()

	m := &Merge{}

	steps.ProcessFirstEvent(m)
	steps.AssertNodeExecutionCount(1)
	steps.AssertExecutionPending()

	steps.ProcessSecondEvent(m)
	steps.AssertNodeExecutionCount(1)
	steps.AssertExecutionFinished()

	steps.AssertQueueIsEmpty()
}

func Test_Merge_WaitsForDistinctSources(t *testing.T) {
	steps := NewMergeTestSteps(t)

	steps.CreateWorkflowSingleSourceMultipleEdges()
	steps.CreateSingleEventForProcess1()
	steps.CreateSingleQueueItemForProcess1()

	m := &Merge{}

	// Processing a single event from the single source should finish the merge
	// because we now wait for distinct sources (1), not number of edges (2).
	steps.ProcessFirstEventExpectFinish(m)
	steps.AssertNodeExecutionCount(1)
	steps.AssertExecutionFinished()
	steps.AssertNodeIsAllowedToProcessNextQueueItem()

	steps.AssertQueueIsEmpty()
}

// The engine writes a consumed-event link row only when ProcessQueueItem
// returns a non-nil execution id. The tests below reproduce that choke point
// directly, without pulling in the worker, because a replay rebuilds its inputs
// from those rows: an event recorded there but never folded into the execution
// would be replayed as an input the original run never saw.

func recordIfConsumed(t *testing.T, tx *gorm.DB, id *uuid.UUID, eventID uuid.UUID) {
	if id == nil {
		return
	}
	require.NoError(t, models.CreateConsumedEvent(tx, *id, eventID))
}

func Test_Merge_StopEarlyThenSecondSourceDiscardsIt(t *testing.T) {
	steps := NewMergeTestSteps(t)
	steps.CreateWorkflow()
	steps.SetMergeConfiguration(map[string]any{
		"stopIfExpression": "$[\"process-1\"].result == \"fail\"",
	})
	steps.CreateEventsWithData(
		map[string]any{"result": "fail"},
		map[string]any{"result": "ok"},
	)
	steps.CreateQueueItems()

	m := &Merge{}

	ctx1, err := contexts.BuildProcessQueueContext(http.DefaultClient, steps.Tx, steps.MergeNode, steps.QueureItem1, nil, nil, nil)
	require.NoError(t, err)
	id1, err := m.ProcessQueueItem(*ctx1)
	require.NoError(t, err)
	require.NotNil(t, id1, "the event that triggers the stop expression must produce an execution id")
	recordIfConsumed(t, steps.Tx, id1, steps.Process1Event.ID)

	// The stop expression finished the execution on the first event, so the
	// second one arrives at an already-finished execution and is dequeued
	// without being folded into its metadata.
	ctx2, err := contexts.BuildProcessQueueContext(http.DefaultClient, steps.Tx, steps.MergeNode, steps.QueureItem2, nil, nil, nil)
	require.NoError(t, err)
	id2, err := m.ProcessQueueItem(*ctx2)
	require.NoError(t, err)
	assert.Nil(t, id2, "an event the execution never folded in must not be reported as consumed")
	recordIfConsumed(t, steps.Tx, id2, steps.Process2Event.ID)

	consumed, err := models.ListConsumedEventsForExecution(steps.Tx, *id1)
	require.NoError(t, err)
	require.Len(t, consumed, 1, "only the event that triggered the stop expression fed this execution")
	require.NotNil(t, consumed[0].EventID)
	assert.Equal(t, steps.Process1Event.ID, *consumed[0].EventID)
}

// A late/duplicate queue item arriving after the execution already finished is
// dequeued and discarded, never folded into the execution's metadata, so it
// must not be recorded as consumed: a replay rebuilds its inputs from those
// rows and would otherwise feed the node an input the original run threw away.
func Test_Merge_AlreadyFinishedPathDiscardsLateRedelivery(t *testing.T) {
	steps := NewMergeTestSteps(t)
	steps.CreateWorkflow()
	steps.CreateEvents()
	steps.CreateQueueItems()

	m := &Merge{}

	ctx1, err := contexts.BuildProcessQueueContext(http.DefaultClient, steps.Tx, steps.MergeNode, steps.QueureItem1, nil, nil, nil)
	require.NoError(t, err)
	id1, err := m.ProcessQueueItem(*ctx1)
	require.NoError(t, err)
	require.NotNil(t, id1)
	recordIfConsumed(t, steps.Tx, id1, steps.Process1Event.ID)

	ctx2, err := contexts.BuildProcessQueueContext(http.DefaultClient, steps.Tx, steps.MergeNode, steps.QueureItem2, nil, nil, nil)
	require.NoError(t, err)
	id2, err := m.ProcessQueueItem(*ctx2)
	require.NoError(t, err)
	require.NotNil(t, id2)
	recordIfConsumed(t, steps.Tx, id2, steps.Process2Event.ID)
	steps.AssertExecutionFinished()

	// A late/duplicate redelivery for process-1: same RootEventID (so
	// findOrCreateExecution finds the now-finished execution), a fresh
	// EventID.
	duplicateEvent := &models.CanvasEvent{
		WorkflowID: steps.Wf.ID,
		NodeID:     steps.ProcessNode1.NodeID,
		Channel:    "default",
		Data:       models.JSONValue{},
	}
	require.NoError(t, steps.Tx.Create(duplicateEvent).Error)
	duplicateQueueItem := &models.CanvasNodeQueueItem{
		WorkflowID:  steps.Wf.ID,
		NodeID:      steps.ProcessNode1.NodeID,
		EventID:     duplicateEvent.ID,
		RootEventID: steps.RootEvent.ID,
	}
	require.NoError(t, steps.Tx.Create(duplicateQueueItem).Error)

	ctx3, err := contexts.BuildProcessQueueContext(http.DefaultClient, steps.Tx, steps.MergeNode, duplicateQueueItem, nil, nil, nil)
	require.NoError(t, err)
	id3, err := m.ProcessQueueItem(*ctx3)
	require.NoError(t, err)
	assert.Nil(t, id3, "a redelivery the execution discarded must not be reported as consumed by it")
	recordIfConsumed(t, steps.Tx, id3, duplicateEvent.ID)

	consumed, err := models.ListConsumedEventsForExecution(steps.Tx, *id1)
	require.NoError(t, err)
	require.Len(t, consumed, 2, "only the two events actually folded into the execution are consumed")

	eventIDs := []uuid.UUID{*consumed[0].EventID, *consumed[1].EventID}
	assert.NotContains(t, eventIDs, duplicateEvent.ID, "the discarded redelivery must not become a replay input")
}

// A stop-early only survives into a later queue item when it could not finish
// the execution, and the one state where it cannot is `cancelling`: Pass and
// Fail are both no-ops there, and RequestCancellation leaves the execution in
// that state until ExecutionTerminator picks it up in a separate transaction,
// without draining the node's queue. An event arriving in that window is folded
// into the execution's metadata, so it must be recorded as consumed.
func Test_Merge_StopEarlyWithoutFinishing_FoldsInTheNextEventAndRecordsIt(t *testing.T) {
	steps := NewMergeTestSteps(t)
	steps.CreateWorkflow()
	steps.SetMergeConfiguration(map[string]any{
		"stopIfExpression": "$[\"process-1\"].result == \"fail\"",
	})
	steps.CreateEventsWithData(
		map[string]any{"result": "ok"},
		map[string]any{"result": "ok"},
	)
	steps.CreateQueueItems()

	m := &Merge{}

	ctx1, err := contexts.BuildProcessQueueContext(http.DefaultClient, steps.Tx, steps.MergeNode, steps.QueureItem1, nil, nil, nil)
	require.NoError(t, err)
	id1, err := m.ProcessQueueItem(*ctx1)
	require.NoError(t, err)
	require.NotNil(t, id1)
	recordIfConsumed(t, steps.Tx, id1, steps.Process1Event.ID)

	execution, err := models.FindNodeExecutionInTransaction(steps.Tx, steps.Wf.ID, *id1)
	require.NoError(t, err)
	require.NoError(t, execution.RequestCancellation(steps.Tx, nil))

	// A redelivery from process-1 whose payload trips the stop expression.
	stopEvent := &models.CanvasEvent{
		WorkflowID: steps.Wf.ID,
		NodeID:     steps.ProcessNode1.NodeID,
		Channel:    "default",
		Data:       models.NewJSONValue(map[string]any{"result": "fail"}),
	}
	require.NoError(t, steps.Tx.Create(stopEvent).Error)
	stopQueueItem := &models.CanvasNodeQueueItem{
		WorkflowID:  steps.Wf.ID,
		NodeID:      steps.ProcessNode1.NodeID,
		EventID:     stopEvent.ID,
		RootEventID: steps.RootEvent.ID,
	}
	require.NoError(t, steps.Tx.Create(stopQueueItem).Error)

	ctx2, err := contexts.BuildProcessQueueContext(http.DefaultClient, steps.Tx, steps.MergeNode, stopQueueItem, nil, nil, nil)
	require.NoError(t, err)
	id2, err := m.ProcessQueueItem(*ctx2)
	require.NoError(t, err)
	require.NotNil(t, id2)
	recordIfConsumed(t, steps.Tx, id2, stopEvent.ID)

	stopped, err := models.FindNodeExecutionInTransaction(steps.Tx, steps.Wf.ID, *id1)
	require.NoError(t, err)
	require.Equal(t, models.CanvasNodeExecutionStateCancelling, stopped.State, "the stop expression cannot finish a cancelling execution")

	// process-3's event now reaches an execution that stopped early and never
	// finished.
	ctx3, err := contexts.BuildProcessQueueContext(http.DefaultClient, steps.Tx, steps.MergeNode, steps.QueureItem2, nil, nil, nil)
	require.NoError(t, err)
	id3, err := m.ProcessQueueItem(*ctx3)
	require.NoError(t, err)
	require.NotNil(t, id3, "an event folded into the execution's metadata must be reported as consumed")
	assert.Equal(t, *id1, *id3)
	recordIfConsumed(t, steps.Tx, id3, steps.Process2Event.ID)

	stopped, err = models.FindNodeExecutionInTransaction(steps.Tx, steps.Wf.ID, *id1)
	require.NoError(t, err)
	md := &ExecutionMetadata{}
	require.NoError(t, mapstructure.Decode(stopped.Metadata.Data(), md))
	require.True(t, md.StopEarly)
	assert.Contains(t, md.EventIDs, steps.Process2Event.ID.String(), "the event was folded into the execution's metadata")

	consumed, err := models.ListConsumedEventsForExecution(steps.Tx, *id1)
	require.NoError(t, err)
	require.Len(t, consumed, 3)
	eventIDs := []uuid.UUID{}
	for _, link := range consumed {
		require.NotNil(t, link.EventID)
		eventIDs = append(eventIDs, *link.EventID)
	}
	assert.Contains(t, eventIDs, steps.Process2Event.ID, "an event a replay must feed back has to be linked to the execution")
}

type MergeTestSteps struct {
	t  *testing.T
	Tx *gorm.DB

	Wf           *models.Canvas
	StartNode    *models.CanvasNode
	ProcessNode1 *models.CanvasNode
	ProcessNode2 *models.CanvasNode
	MergeNode    *models.CanvasNode

	RootEvent     *models.CanvasEvent
	Process1Event *models.CanvasEvent
	Process2Event *models.CanvasEvent

	QueureItem1 *models.CanvasNodeQueueItem
	QueureItem2 *models.CanvasNodeQueueItem
}

func NewMergeTestSteps(t *testing.T) *MergeTestSteps {
	require.NoError(t, database.TruncateTables())
	return &MergeTestSteps{
		Tx: database.Conn(),
		t:  t,
	}
}

/*
* Creates a canvas with the following structure:
*
*   (start) +--> (n1) \
*           |           -> (merge)
*           +--> (n2) /
 */
func (s *MergeTestSteps) CreateWorkflow() {
	now := time.Now()
	liveVersionID := uuid.New()
	wf := &models.Canvas{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		LiveVersionID:  &liveVersionID,
		Name:           "merge-workflow",
		Description:    "merge workflow test",
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	n1 := &models.CanvasNode{
		WorkflowID: wf.ID,
		NodeID:     "start-node",
		Name:       "start-node",
		Type:       models.NodeTypeComponent,
		Ref:        datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "start"}}),
	}
	n2 := &models.CanvasNode{
		WorkflowID: wf.ID,
		NodeID:     "process-1",
		Name:       "process-1",
		Type:       models.NodeTypeComponent,
		Ref:        datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "process-1"}}),
	}
	n3 := &models.CanvasNode{
		WorkflowID: wf.ID,
		NodeID:     "process-3",
		Name:       "process-3",
		Type:       models.NodeTypeComponent,
		Ref:        datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "process-2"}}),
	}
	n4 := &models.CanvasNode{
		WorkflowID: wf.ID,
		NodeID:     "merge-node",
		Name:       "merge-node",
		Type:       models.NodeTypeComponent,
		Ref:        datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "merge"}}),
	}

	require.NoError(s.t, s.Tx.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(wf).Error; err != nil {
			return err
		}
		if err := tx.Create(n1).Error; err != nil {
			return err
		}
		if err := tx.Create(n2).Error; err != nil {
			return err
		}
		if err := tx.Create(n3).Error; err != nil {
			return err
		}
		if err := tx.Create(n4).Error; err != nil {
			return err
		}

		edges := []models.Edge{
			{
				SourceID: n1.NodeID,
				TargetID: n2.NodeID,
				Channel:  "default",
			},
			{
				SourceID: n1.NodeID,
				TargetID: n3.NodeID,
				Channel:  "default",
			},
			{
				SourceID: n2.NodeID,
				TargetID: n4.NodeID,
				Channel:  "default",
			},
			{
				SourceID: n3.NodeID,
				TargetID: n4.NodeID,
				Channel:  "default",
			},
		}

		nodes := []models.Node{
			{ID: n1.NodeID, Name: n1.Name, Type: n1.Type, Ref: n1.Ref.Data()},
			{ID: n2.NodeID, Name: n2.Name, Type: n2.Type, Ref: n2.Ref.Data()},
			{ID: n3.NodeID, Name: n3.Name, Type: n3.Type, Ref: n3.Ref.Data()},
			{ID: n4.NodeID, Name: n4.Name, Type: n4.Type, Ref: n4.Ref.Data()},
		}

		return tx.Create(&models.CanvasVersion{
			ID:         liveVersionID,
			WorkflowID: wf.ID,
			Nodes:      datatypes.NewJSONSlice(nodes),
			Edges:      datatypes.NewJSONSlice(edges),
			CreatedAt:  &now,
			UpdatedAt:  &now,
		}).Error
	}))

	s.Wf = wf
	s.StartNode = n1
	s.ProcessNode1 = n2
	s.ProcessNode2 = n3
	s.MergeNode = n4
}

// Create a canvas where a single upstream node connects to the merge node
// via two separate edges/channels. With the updated semantics, the merge should
// require only one event (distinct source) to finish.
func (s *MergeTestSteps) CreateWorkflowSingleSourceMultipleEdges() {
	now := time.Now()
	liveVersionID := uuid.New()
	wf := &models.Canvas{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		LiveVersionID:  &liveVersionID,
		Name:           "merge-workflow-single-source",
		Description:    "merge workflow test",
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	n1 := &models.CanvasNode{
		WorkflowID: wf.ID,
		NodeID:     "start-node",
		Name:       "start-node",
		Type:       models.NodeTypeComponent,
		Ref:        datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "start"}}),
	}
	n2 := &models.CanvasNode{
		WorkflowID: wf.ID,
		NodeID:     "process-1",
		Name:       "process-1",
		Type:       models.NodeTypeComponent,
		Ref:        datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "process-1"}}),
	}
	n4 := &models.CanvasNode{
		WorkflowID: wf.ID,
		NodeID:     "merge-node",
		Name:       "merge-node",
		Type:       models.NodeTypeComponent,
		Ref:        datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "merge"}}),
	}

	require.NoError(s.t, s.Tx.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(wf).Error; err != nil {
			return err
		}
		if err := tx.Create(n1).Error; err != nil {
			return err
		}
		if err := tx.Create(n2).Error; err != nil {
			return err
		}
		if err := tx.Create(n4).Error; err != nil {
			return err
		}

		edges := []models.Edge{
			{SourceID: n1.NodeID, TargetID: n2.NodeID, Channel: "default"},
			// Two edges from process-1 to merge-node on different channels
			{SourceID: n2.NodeID, TargetID: n4.NodeID, Channel: "default"},
			{SourceID: n2.NodeID, TargetID: n4.NodeID, Channel: "alt"},
		}
		nodes := []models.Node{
			{ID: n1.NodeID, Name: n1.Name, Type: n1.Type, Ref: n1.Ref.Data()},
			{ID: n2.NodeID, Name: n2.Name, Type: n2.Type, Ref: n2.Ref.Data()},
			{ID: n4.NodeID, Name: n4.Name, Type: n4.Type, Ref: n4.Ref.Data()},
		}

		return tx.Create(&models.CanvasVersion{
			ID:         liveVersionID,
			WorkflowID: wf.ID,
			Nodes:      datatypes.NewJSONSlice(nodes),
			Edges:      datatypes.NewJSONSlice(edges),
			CreatedAt:  &now,
			UpdatedAt:  &now,
		}).Error
	}))

	s.Wf = wf
	s.StartNode = n1
	s.ProcessNode1 = n2
	s.MergeNode = n4
}

// CreateReplayStyleEventsAndQueueItems builds the exact shape
// models.CreateMultiInputNodeReplay produces for a multi-payload replay: one CanvasRun
// shared by every synthetic event (so CanvasEvent.BeforeCreate does not
// manufacture a stray run per event), one synthetic root event, and one
// synthetic input event per distinct source, each Channel "replay" and
// already State Routed. All queue items share the root event's ID as their
// RootEventID but carry distinct EventIDs.
func (s *MergeTestSteps) CreateReplayStyleEventsAndQueueItems() {
	now := time.Now()

	run, err := models.CreateCanvasRunInTransaction(s.Tx, s.Wf.ID, s.MergeNode.NodeID, models.CanvasRunStateStarted, "")
	require.NoError(s.t, err)

	rootEvent := &models.CanvasEvent{
		WorkflowID: s.Wf.ID,
		NodeID:     s.MergeNode.NodeID,
		Channel:    "replay",
		Data:       models.JSONValue{},
		RunID:      run.ID,
		State:      models.CanvasEventStateRouted,
		CreatedAt:  &now,
	}
	require.NoError(s.t, s.Tx.Create(rootEvent).Error)

	event1 := &models.CanvasEvent{
		WorkflowID: s.Wf.ID,
		NodeID:     s.ProcessNode1.NodeID,
		Channel:    "replay",
		Data:       models.NewJSONValue(map[string]any{"payload": 1}),
		RunID:      run.ID,
		State:      models.CanvasEventStateRouted,
		CreatedAt:  &now,
	}
	require.NoError(s.t, s.Tx.Create(event1).Error)

	event2 := &models.CanvasEvent{
		WorkflowID: s.Wf.ID,
		NodeID:     s.ProcessNode2.NodeID,
		Channel:    "replay",
		Data:       models.NewJSONValue(map[string]any{"payload": 2}),
		RunID:      run.ID,
		State:      models.CanvasEventStateRouted,
		CreatedAt:  &now,
	}
	require.NoError(s.t, s.Tx.Create(event2).Error)

	s.RootEvent = rootEvent
	s.Process1Event = event1
	s.Process2Event = event2

	queueItem1 := &models.CanvasNodeQueueItem{
		WorkflowID:  s.Wf.ID,
		NodeID:      s.MergeNode.NodeID,
		RunID:       run.ID,
		EventID:     event1.ID,
		RootEventID: rootEvent.ID,
	}
	require.NoError(s.t, s.Tx.Create(queueItem1).Error)

	queueItem2 := &models.CanvasNodeQueueItem{
		WorkflowID:  s.Wf.ID,
		NodeID:      s.MergeNode.NodeID,
		RunID:       run.ID,
		EventID:     event2.ID,
		RootEventID: rootEvent.ID,
	}
	require.NoError(s.t, s.Tx.Create(queueItem2).Error)

	s.QueureItem1 = queueItem1
	s.QueureItem2 = queueItem2
}

func (s *MergeTestSteps) SetMergeConfiguration(cfg map[string]any) {
	s.MergeNode.Configuration = datatypes.NewJSONType(cfg)
	require.NoError(s.t, s.Tx.Save(s.MergeNode).Error)
}

func (s *MergeTestSteps) CreateEvents() {
	rootEvent := &models.CanvasEvent{
		WorkflowID: s.Wf.ID,
		NodeID:     "start-node",
		Channel:    "default",
		Data:       models.JSONValue{},
	}
	require.NoError(s.t, s.Tx.Create(rootEvent).Error)

	event1 := &models.CanvasEvent{
		WorkflowID: s.Wf.ID,
		NodeID:     s.ProcessNode1.NodeID,
		Channel:    "default",
		Data:       models.JSONValue{},
	}
	require.NoError(s.t, s.Tx.Create(event1).Error)

	event2 := &models.CanvasEvent{
		WorkflowID: s.Wf.ID,
		NodeID:     s.ProcessNode2.NodeID,
		Channel:    "default",
		Data:       models.JSONValue{},
	}
	require.NoError(s.t, s.Tx.Create(event2).Error)

	s.RootEvent = rootEvent
	s.Process1Event = event1
	s.Process2Event = event2
}

func (s *MergeTestSteps) CreateSingleEventForProcess1() {
	rootEvent := &models.CanvasEvent{
		WorkflowID: s.Wf.ID,
		NodeID:     "start-node",
		Channel:    "default",
		Data:       models.JSONValue{},
	}
	require.NoError(s.t, s.Tx.Create(rootEvent).Error)

	event1 := &models.CanvasEvent{
		WorkflowID: s.Wf.ID,
		NodeID:     s.ProcessNode1.NodeID,
		Channel:    "default",
		Data:       models.JSONValue{},
	}
	require.NoError(s.t, s.Tx.Create(event1).Error)

	s.RootEvent = rootEvent
	s.Process1Event = event1
}

func (s *MergeTestSteps) CreateEventsWithData(data1 any, data2 any) {
	rootEvent := &models.CanvasEvent{
		WorkflowID: s.Wf.ID,
		NodeID:     "start-node",
		Channel:    "default",
		Data:       models.JSONValue{},
	}
	require.NoError(s.t, s.Tx.Create(rootEvent).Error)

	event1 := &models.CanvasEvent{
		WorkflowID: s.Wf.ID,
		NodeID:     s.ProcessNode1.NodeID,
		Channel:    "default",
		Data:       models.NewJSONValue(data1),
	}
	require.NoError(s.t, s.Tx.Create(event1).Error)

	event2 := &models.CanvasEvent{
		WorkflowID: s.Wf.ID,
		NodeID:     s.ProcessNode2.NodeID,
		Channel:    "default",
		Data:       models.NewJSONValue(data2),
	}
	require.NoError(s.t, s.Tx.Create(event2).Error)

	s.RootEvent = rootEvent
	s.Process1Event = event1
	s.Process2Event = event2
}

func (s *MergeTestSteps) CreateQueueItems() {
	queueItem1 := &models.CanvasNodeQueueItem{
		WorkflowID:  s.Wf.ID,
		NodeID:      s.ProcessNode1.NodeID,
		EventID:     s.Process1Event.ID,
		RootEventID: s.RootEvent.ID,
	}
	require.NoError(s.t, s.Tx.Create(queueItem1).Error)

	queueItem2 := &models.CanvasNodeQueueItem{
		WorkflowID:  s.Wf.ID,
		NodeID:      s.ProcessNode2.NodeID,
		EventID:     s.Process2Event.ID,
		RootEventID: s.RootEvent.ID,
	}
	require.NoError(s.t, s.Tx.Create(queueItem2).Error)

	s.QueureItem1 = queueItem1
	s.QueureItem2 = queueItem2
}

func (s *MergeTestSteps) CreateSingleQueueItemForProcess1() {
	queueItem1 := &models.CanvasNodeQueueItem{
		WorkflowID:  s.Wf.ID,
		NodeID:      s.ProcessNode1.NodeID,
		EventID:     s.Process1Event.ID,
		RootEventID: s.RootEvent.ID,
	}
	require.NoError(s.t, s.Tx.Create(queueItem1).Error)

	s.QueureItem1 = queueItem1
}

func (s *MergeTestSteps) ProcessFirstEvent(m *Merge) {
	fmt.Println("Processing first event")

	ctx1, err := contexts.BuildProcessQueueContext(http.DefaultClient, s.Tx, s.MergeNode, s.QueureItem1, nil, nil, nil)
	assert.NoError(s.t, err)

	execution, err := m.ProcessQueueItem(*ctx1)
	require.NoError(s.t, err)
	require.NotNil(s.t, execution)
}

// ProcessFirstEventExpectFinish is used when the first event should
// finish the merge immediately (e.g. due to stopIfExpression)
func (s *MergeTestSteps) ProcessFirstEventExpectFinish(m *Merge) {
	fmt.Println("Processing first event (expect finish)")

	ctx1, err := contexts.BuildProcessQueueContext(http.DefaultClient, s.Tx, s.MergeNode, s.QueureItem1, nil, nil, nil)
	assert.NoError(s.t, err)

	execution, err := m.ProcessQueueItem(*ctx1)
	require.NoError(s.t, err)
	require.NotNil(s.t, execution)
}

func (s *MergeTestSteps) ProcessSecondEvent(m *Merge) {
	fmt.Println("Processing second event")

	ctx2, err := contexts.BuildProcessQueueContext(http.DefaultClient, s.Tx, s.MergeNode, s.QueureItem2, nil, nil, nil)
	assert.NoError(s.t, err)

	execution, err := m.ProcessQueueItem(*ctx2)
	require.NoError(s.t, err)
	require.NotNil(s.t, execution)
}

// ProcessSecondEventExpectNoFinish is used when the group already
// short-circuited (stopped early) before this event arrived: the event is
// still dequeued and folded into metadata, but must not re-finish the
// execution.
func (s *MergeTestSteps) ProcessSecondEventExpectNoFinish(m *Merge) {
	fmt.Println("Processing second event")

	ctx2, err := contexts.BuildProcessQueueContext(http.DefaultClient, s.Tx, s.MergeNode, s.QueureItem2, nil, nil, nil)
	assert.NoError(s.t, err)

	execution, err := m.ProcessQueueItem(*ctx2)
	require.NoError(s.t, err)
	require.Nil(s.t, execution, "the execution had already finished, so this event is discarded rather than consumed")
}

func (s *MergeTestSteps) AssertNodeExecutionCount(expectedCount int) {
	var executions []models.CanvasNodeExecution
	require.NoError(s.t, s.Tx.Where("node_id = ?", s.MergeNode.NodeID).Find(&executions).Error)
	assert.Equal(s.t, expectedCount, len(executions))
}

func (s *MergeTestSteps) AssertExecutionFinished() {
	var execution models.CanvasNodeExecution
	require.NoError(s.t, s.Tx.Where("node_id = ?", s.MergeNode.NodeID).First(&execution).Error)
	assert.Equal(s.t, models.CanvasNodeExecutionStateFinished, execution.State)
}

func (s *MergeTestSteps) AssertExecutionFailedWithError(errorMessage string) {
	var execution models.CanvasNodeExecution
	require.NoError(s.t, s.Tx.Where("node_id = ?", s.MergeNode.NodeID).First(&execution).Error)
	assert.Equal(s.t, models.CanvasNodeExecutionStateFinished, execution.State)
	assert.Equal(s.t, models.CanvasNodeExecutionResultFailed, execution.Result)
	assert.Equal(s.t, "error", execution.ResultReason)
	assert.Equal(s.t, errorMessage, execution.ResultMessage)
}

func (s *MergeTestSteps) AssertExecutionFailedWithErrorContaining(errorMessage string) {
	var execution models.CanvasNodeExecution
	require.NoError(s.t, s.Tx.Where("node_id = ?", s.MergeNode.NodeID).First(&execution).Error)
	assert.Equal(s.t, models.CanvasNodeExecutionStateFinished, execution.State)
	assert.Equal(s.t, models.CanvasNodeExecutionResultFailed, execution.Result)
	assert.Equal(s.t, "error", execution.ResultReason)
	assert.Contains(s.t, execution.ResultMessage, errorMessage)
}

// AssertExecutionFailed checks that the execution finished and emitted to the fail channel.
// Note: With output channels, conditional stop "passes" the execution but routes to the
// "fail" channel, similar to how the `if` component routes to true/false channels.
func (s *MergeTestSteps) AssertExecutionFailed() {
	var execution models.CanvasNodeExecution
	require.NoError(s.t, s.Tx.Where("node_id = ?", s.MergeNode.NodeID).First(&execution).Error)
	assert.Equal(s.t, models.CanvasNodeExecutionStateFinished, execution.State)
	assert.Equal(s.t, models.CanvasNodeExecutionResultPassed, execution.Result)
}

func (s *MergeTestSteps) AssertExecutionPending() {
	var execution models.CanvasNodeExecution
	require.NoError(s.t, s.Tx.Where("node_id = ?", s.MergeNode.NodeID).First(&execution).Error)
	assert.Equal(s.t, models.CanvasNodeExecutionStatePending, execution.State)
}

func (s *MergeTestSteps) AssertQueueIsEmpty() {
	var count int64
	require.NoError(s.t, s.Tx.Model(&models.CanvasNodeQueueItem{}).Where("node_id = ?", s.MergeNode.NodeID).Count(&count).Error)
	assert.Equal(s.t, int64(0), count)
}

func (s *MergeTestSteps) AssertNodeIsAllowedToProcessNextQueueItem() {
	var node models.CanvasNode
	require.NoError(s.t, s.Tx.Where("node_id = ?", s.MergeNode.NodeID).First(&node).Error)
	assert.Equal(s.t, models.CanvasNodeStateReady, node.State)
}

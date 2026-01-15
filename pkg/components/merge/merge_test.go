package merge

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/expr-lang/expr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
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
		"stopIfExpression": "$.result == \"fail\"",
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

func Test_Merge_StopIfExpression_SourceNodeReference(t *testing.T) {
	steps := NewMergeTestSteps(t)

	steps.CreateWorkflow()
	steps.SetMergeConfiguration(map[string]any{
		"stopIfExpression": "$[\"process-1\"].data.result == \"fail\"",
	})

	steps.CreateEventsWithData(
		map[string]any{"data": map[string]any{"result": "fail"}},
		map[string]any{"data": map[string]any{"result": "ok"}},
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

func TestMerge_ExpressionEnv_UsesContextEnv(t *testing.T) {
	ctx := core.ProcessQueueContext{
		Input:        map[string]any{"result": "ignored"},
		SourceNodeID: "source-node",
		ExpressionEnv: func(expression string) (map[string]any, error) {
			return map[string]any{
				"$": map[string]any{
					"other-node": map[string]any{
						"data": map[string]any{
							"result": "ok",
						},
					},
				},
			}, nil
		},
	}

	env, err := expressionEnv(ctx, "$[\"other-node\"].data.result == \"ok\"")
	require.NoError(t, err)

	vm, err := expr.Compile("$[\"other-node\"].data.result == \"ok\"", expr.Env(env), expr.AsBool())
	require.NoError(t, err)

	out, err := expr.Run(vm, env)
	require.NoError(t, err)
	assert.True(t, out.(bool))
}

type MergeTestSteps struct {
	t  *testing.T
	Tx *gorm.DB

	Wf           *models.Workflow
	StartNode    *models.WorkflowNode
	ProcessNode1 *models.WorkflowNode
	ProcessNode2 *models.WorkflowNode
	MergeNode    *models.WorkflowNode

	RootEvent     *models.WorkflowEvent
	Process1Event *models.WorkflowEvent
	Process2Event *models.WorkflowEvent

	QueureItem1 *models.WorkflowNodeQueueItem
	QueureItem2 *models.WorkflowNodeQueueItem
}

func NewMergeTestSteps(t *testing.T) *MergeTestSteps {
	require.NoError(t, database.TruncateTables())
	return &MergeTestSteps{
		Tx: database.Conn(),
		t:  t,
	}
}

/*
* Creates a workflow with the following structure:
*
*   (start) +--> (n1) \
*           |           -> (merge)
*           +--> (n2) /
 */
func (s *MergeTestSteps) CreateWorkflow() {
	wf := &models.Workflow{ID: uuid.New()}
	require.NoError(s.t, s.Tx.Create(wf).Error)

	n1 := &models.WorkflowNode{
		WorkflowID: wf.ID,
		NodeID:     "start-node",
		Type:       models.NodeTypeComponent,
		Ref:        datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "start"}}),
	}
	require.NoError(s.t, s.Tx.Create(n1).Error)

	n2 := &models.WorkflowNode{
		WorkflowID: wf.ID,
		NodeID:     "process-1",
		Type:       models.NodeTypeComponent,
		Ref:        datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "process-1"}}),
	}
	require.NoError(s.t, s.Tx.Create(n2).Error)

	n3 := &models.WorkflowNode{
		WorkflowID: wf.ID,
		NodeID:     "process-3",
		Type:       models.NodeTypeComponent,
		Ref:        datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "process-2"}}),
	}
	require.NoError(s.t, s.Tx.Create(n3).Error)

	n4 := &models.WorkflowNode{
		WorkflowID: wf.ID,
		NodeID:     "merge-node",
		Type:       models.NodeTypeComponent,
		Ref:        datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "merge"}}),
	}
	require.NoError(s.t, s.Tx.Create(n4).Error)

	wf.Edges = []models.Edge{
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

	require.NoError(s.t, s.Tx.Updates(&wf).Error)

	s.Wf = wf
	s.StartNode = n1
	s.ProcessNode1 = n2
	s.ProcessNode2 = n3
	s.MergeNode = n4
}

// Create a workflow where a single upstream node connects to the merge node
// via two separate edges/channels. With the updated semantics, the merge should
// require only one event (distinct source) to finish.
func (s *MergeTestSteps) CreateWorkflowSingleSourceMultipleEdges() {
	wf := &models.Workflow{ID: uuid.New()}
	require.NoError(s.t, s.Tx.Create(wf).Error)

	n1 := &models.WorkflowNode{
		WorkflowID: wf.ID,
		NodeID:     "start-node",
		Type:       models.NodeTypeComponent,
		Ref:        datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "start"}}),
	}
	require.NoError(s.t, s.Tx.Create(n1).Error)

	n2 := &models.WorkflowNode{
		WorkflowID: wf.ID,
		NodeID:     "process-1",
		Type:       models.NodeTypeComponent,
		Ref:        datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "process-1"}}),
	}
	require.NoError(s.t, s.Tx.Create(n2).Error)

	n4 := &models.WorkflowNode{
		WorkflowID: wf.ID,
		NodeID:     "merge-node",
		Type:       models.NodeTypeComponent,
		Ref:        datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "merge"}}),
	}
	require.NoError(s.t, s.Tx.Create(n4).Error)

	wf.Edges = []models.Edge{
		{SourceID: n1.NodeID, TargetID: n2.NodeID, Channel: "default"},
		// Two edges from process-1 to merge-node on different channels
		{SourceID: n2.NodeID, TargetID: n4.NodeID, Channel: "default"},
		{SourceID: n2.NodeID, TargetID: n4.NodeID, Channel: "alt"},
	}
	require.NoError(s.t, s.Tx.Updates(&wf).Error)

	s.Wf = wf
	s.StartNode = n1
	s.ProcessNode1 = n2
	s.MergeNode = n4
}

func (s *MergeTestSteps) SetMergeConfiguration(cfg map[string]any) {
	s.MergeNode.Configuration = datatypes.NewJSONType(cfg)
	require.NoError(s.t, s.Tx.Save(s.MergeNode).Error)
}

func (s *MergeTestSteps) CreateEvents() {
	rootEvent := &models.WorkflowEvent{
		WorkflowID: s.Wf.ID,
		NodeID:     "start-node",
		Channel:    "default",
		Data:       datatypes.JSONType[any]{},
	}
	require.NoError(s.t, s.Tx.Create(rootEvent).Error)

	event1 := &models.WorkflowEvent{
		WorkflowID: s.Wf.ID,
		NodeID:     s.ProcessNode1.NodeID,
		Channel:    "default",
		Data:       datatypes.JSONType[any]{},
	}
	require.NoError(s.t, s.Tx.Create(event1).Error)

	event2 := &models.WorkflowEvent{
		WorkflowID: s.Wf.ID,
		NodeID:     s.ProcessNode2.NodeID,
		Channel:    "default",
		Data:       datatypes.JSONType[any]{},
	}
	require.NoError(s.t, s.Tx.Create(event2).Error)

	s.RootEvent = rootEvent
	s.Process1Event = event1
	s.Process2Event = event2
}

func (s *MergeTestSteps) CreateSingleEventForProcess1() {
	rootEvent := &models.WorkflowEvent{
		WorkflowID: s.Wf.ID,
		NodeID:     "start-node",
		Channel:    "default",
		Data:       datatypes.JSONType[any]{},
	}
	require.NoError(s.t, s.Tx.Create(rootEvent).Error)

	event1 := &models.WorkflowEvent{
		WorkflowID: s.Wf.ID,
		NodeID:     s.ProcessNode1.NodeID,
		Channel:    "default",
		Data:       datatypes.JSONType[any]{},
	}
	require.NoError(s.t, s.Tx.Create(event1).Error)

	s.RootEvent = rootEvent
	s.Process1Event = event1
}

func (s *MergeTestSteps) CreateEventsWithData(data1 any, data2 any) {
	rootEvent := &models.WorkflowEvent{
		WorkflowID: s.Wf.ID,
		NodeID:     "start-node",
		Channel:    "default",
		Data:       datatypes.JSONType[any]{},
	}
	require.NoError(s.t, s.Tx.Create(rootEvent).Error)

	event1 := &models.WorkflowEvent{
		WorkflowID: s.Wf.ID,
		NodeID:     s.ProcessNode1.NodeID,
		Channel:    "default",
		Data:       datatypes.NewJSONType(data1),
	}
	require.NoError(s.t, s.Tx.Create(event1).Error)

	event2 := &models.WorkflowEvent{
		WorkflowID: s.Wf.ID,
		NodeID:     s.ProcessNode2.NodeID,
		Channel:    "default",
		Data:       datatypes.NewJSONType(data2),
	}
	require.NoError(s.t, s.Tx.Create(event2).Error)

	s.RootEvent = rootEvent
	s.Process1Event = event1
	s.Process2Event = event2
}

func (s *MergeTestSteps) CreateQueueItems() {
	queueItem1 := &models.WorkflowNodeQueueItem{
		WorkflowID:  s.Wf.ID,
		NodeID:      s.ProcessNode1.NodeID,
		EventID:     s.Process1Event.ID,
		RootEventID: s.RootEvent.ID,
	}
	require.NoError(s.t, s.Tx.Create(queueItem1).Error)

	queueItem2 := &models.WorkflowNodeQueueItem{
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
	queueItem1 := &models.WorkflowNodeQueueItem{
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

	ctx1, err := contexts.BuildProcessQueueContext(http.DefaultClient, s.Tx, s.MergeNode, s.QueureItem1, nil)
	assert.NoError(s.t, err)

	execution, err := m.ProcessQueueItem(*ctx1)
	require.NoError(s.t, err)
	require.Nil(s.t, execution)
}

// ProcessFirstEventExpectFinish is used when the first event should
// finish the merge immediately (e.g. due to stopIfExpression)
func (s *MergeTestSteps) ProcessFirstEventExpectFinish(m *Merge) {
	fmt.Println("Processing first event (expect finish)")

	ctx1, err := contexts.BuildProcessQueueContext(http.DefaultClient, s.Tx, s.MergeNode, s.QueureItem1, nil)
	assert.NoError(s.t, err)

	execution, err := m.ProcessQueueItem(*ctx1)
	require.NoError(s.t, err)
	require.NotNil(s.t, execution)
}

func (s *MergeTestSteps) ProcessSecondEvent(m *Merge) {
	fmt.Println("Processing second event")

	ctx2, err := contexts.BuildProcessQueueContext(http.DefaultClient, s.Tx, s.MergeNode, s.QueureItem2, nil)
	assert.NoError(s.t, err)

	execution, err := m.ProcessQueueItem(*ctx2)
	require.NoError(s.t, err)
	require.NotNil(s.t, execution)
}

func (s *MergeTestSteps) ProcessSecondEventExpectNoFinish(m *Merge) {
	fmt.Println("Processing second event")

	ctx2, err := contexts.BuildProcessQueueContext(http.DefaultClient, s.Tx, s.MergeNode, s.QueureItem2, nil)
	assert.NoError(s.t, err)

	execution, err := m.ProcessQueueItem(*ctx2)
	require.NoError(s.t, err)
	require.Nil(s.t, execution)
}

func (s *MergeTestSteps) AssertNodeExecutionCount(expectedCount int) {
	var executions []models.WorkflowNodeExecution
	require.NoError(s.t, s.Tx.Where("node_id = ?", s.MergeNode.NodeID).Find(&executions).Error)
	assert.Equal(s.t, expectedCount, len(executions))
}

func (s *MergeTestSteps) AssertExecutionFinished() {
	var execution models.WorkflowNodeExecution
	require.NoError(s.t, s.Tx.Where("node_id = ?", s.MergeNode.NodeID).First(&execution).Error)
	assert.Equal(s.t, execution.State, models.WorkflowNodeExecutionStateFinished)
}

func (s *MergeTestSteps) AssertExecutionFailed() {
	var execution models.WorkflowNodeExecution
	require.NoError(s.t, s.Tx.Where("node_id = ?", s.MergeNode.NodeID).First(&execution).Error)
	assert.Equal(s.t, execution.State, models.WorkflowNodeExecutionStateFinished)
	assert.Equal(s.t, execution.Result, models.WorkflowNodeExecutionResultFailed)
}

func (s *MergeTestSteps) AssertExecutionPending() {
	var execution models.WorkflowNodeExecution
	require.NoError(s.t, s.Tx.Where("node_id = ?", s.MergeNode.NodeID).First(&execution).Error)
	assert.Equal(s.t, execution.State, models.WorkflowNodeExecutionStatePending)
}

func (s *MergeTestSteps) AssertQueueIsEmpty() {
	var count int64
	require.NoError(s.t, s.Tx.Model(&models.WorkflowNodeQueueItem{}).Where("node_id = ?", s.MergeNode.NodeID).Count(&count).Error)
	assert.Equal(s.t, int64(0), count)
}

func (s *MergeTestSteps) AssertNodeIsAllowedToProcessNextQueueItem() {
	var node models.WorkflowNode
	require.NoError(s.t, s.Tx.Where("node_id = ?", s.MergeNode.NodeID).First(&node).Error)
	assert.Equal(s.t, models.WorkflowNodeStateReady, node.State)
}

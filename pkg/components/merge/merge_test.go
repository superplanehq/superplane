package merge

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test_QueueProcessing(t *testing.T) {
	steps := NewQueueProcessingSteps(t)

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

func Test_ExecutionTimeout(t *testing.T) {
	steps := NewExecutionTimeoutSteps(t)

	steps.CreateMergeNodeWithTimeout(2, "seconds")
	steps.CreateNodeExecution()

	m := &Merge{}

	steps.ExecuteMergeNode(m)
	steps.AssertTimeoutScheduled()

	steps.SimulateTimeoutReached(m)
	steps.AssertExecutionFinished()
}

type QueueProcessingSteps struct {
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

	QueueItem1 *models.WorkflowNodeQueueItem
	QueueItem2 *models.WorkflowNodeQueueItem
}

func NewQueueProcessingSteps(t *testing.T) *QueueProcessingSteps {
	require.NoError(t, database.TruncateTables())
	return &QueueProcessingSteps{
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
func (s *QueueProcessingSteps) CreateWorkflow() {
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

func (s *QueueProcessingSteps) CreateEvents() {
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

func (s *QueueProcessingSteps) CreateQueueItems() {
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

	s.QueueItem1 = queueItem1
	s.QueueItem2 = queueItem2
}

func (s *QueueProcessingSteps) ProcessFirstEvent(m *Merge) {
	fmt.Println("Processing first event")

	ctx1, err := contexts.BuildProcessQueueContext(s.Tx, s.MergeNode, s.QueueItem1)
	assert.NoError(s.t, err)

	err = m.ProcessQueueItem(*ctx1)
	require.NoError(s.t, err)
}

func (s *QueueProcessingSteps) ProcessSecondEvent(m *Merge) {
	fmt.Println("Processing second event")

	ctx2, err := contexts.BuildProcessQueueContext(s.Tx, s.MergeNode, s.QueueItem2)
	assert.NoError(s.t, err)

	err = m.ProcessQueueItem(*ctx2)
	require.NoError(s.t, err)
}

func (s *QueueProcessingSteps) AssertNodeExecutionCount(expectedCount int) {
	var executions []models.WorkflowNodeExecution
	require.NoError(s.t, s.Tx.Where("node_id = ?", s.MergeNode.NodeID).Find(&executions).Error)
	assert.Equal(s.t, expectedCount, len(executions))
}

func (s *QueueProcessingSteps) AssertExecutionFinished() {
	var execution models.WorkflowNodeExecution
	require.NoError(s.t, s.Tx.Where("node_id = ?", s.MergeNode.NodeID).First(&execution).Error)
	assert.Equal(s.t, execution.State, models.WorkflowNodeExecutionStateFinished)
}

func (s *QueueProcessingSteps) AssertExecutionPending() {
	var execution models.WorkflowNodeExecution
	require.NoError(s.t, s.Tx.Where("node_id = ?", s.MergeNode.NodeID).First(&execution).Error)
	assert.Equal(s.t, execution.State, models.WorkflowNodeExecutionStatePending)
}

func (s *QueueProcessingSteps) AssertQueueIsEmpty() {
	var count int64
	require.NoError(s.t, s.Tx.Model(&models.WorkflowNodeQueueItem{}).Where("node_id = ?", s.MergeNode.NodeID).Count(&count).Error)
	assert.Equal(s.t, int64(0), count)
}

func (s *QueueProcessingSteps) AssertNodeIsAllowedToProcessNextQueueItem() {
	var node models.WorkflowNode
	require.NoError(s.t, s.Tx.Where("node_id = ?", s.MergeNode.NodeID).First(&node).Error)
	assert.Equal(s.t, models.WorkflowNodeStateReady, node.State)
}

type ExecutionTimeoutSteps struct {
	t  *testing.T
	Tx *gorm.DB

	Wf        *models.Workflow
	MergeNode *models.WorkflowNode

	NodeExec *models.WorkflowNodeExecution
}

func NewExecutionTimeoutSteps(t *testing.T) *ExecutionTimeoutSteps {
	require.NoError(t, database.TruncateTables())
	return &ExecutionTimeoutSteps{
		Tx: database.Conn(),
		t:  t,
	}
}

func (s *ExecutionTimeoutSteps) CreateMergeNodeWithTimeout(value int, unit string) {
	wf := &models.Workflow{ID: uuid.New()}
	require.NoError(s.t, s.Tx.Create(wf).Error)

	mergeNode := &models.WorkflowNode{
		WorkflowID: wf.ID,
		NodeID:     "merge-node",
		Type:       models.NodeTypeComponent,
		Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{
			Name: "merge",
			Configuration: map[string]any{
				"executionTimeout": map[string]any{
					"value": value,
					"unit":  unit,
				},
			},
		}}),
	}
	require.NoError(s.t, s.Tx.Create(mergeNode).Error)

	s.Wf = wf
	s.MergeNode = mergeNode
}

func (s *ExecutionTimeoutSteps) CreateNodeExecution() {
	nodeExec := &models.WorkflowNodeExecution{
		WorkflowID: s.Wf.ID,
		NodeID:     s.MergeNode.NodeID,
		State:      models.WorkflowNodeExecutionStatePending,
	}
	require.NoError(s.t, s.Tx.Create(nodeExec).Error)

	s.NodeExec = nodeExec
}

func (s *ExecutionTimeoutSteps) ExecuteMergeNode(m *Merge) {
	ctx, err := contexts.BuildNodeExecutionContext(s.Tx, s.MergeNode, s.NodeExec)
	assert.NoError(s.t, err)

	err = m.Execute(*ctx)
	require.NoError(s.t, err)
}

func (s *ExecutionTimeoutSteps) AssertTimeoutScheduled() {
	var actionCalls []models.WorkflowNodeActionCall
	require.NoError(s.t, s.Tx.Where("node_id = ?", s.MergeNode.NodeID).Find(&actionCalls).Error)

	found := false
	for _, ac := range actionCalls {
		if ac.Name == "timeoutReached" {
			found = true
			break
		}
	}

	assert.True(s.t, found, "Expected timeoutReached action to be scheduled")
}

func (s *ExecutionTimeoutSteps) SimulateTimeoutReached(m *Merge) {
	ctx, err := contexts.BuildActionContext(s.Tx, s.MergeNode, s.NodeExec, "timeoutReached")
	assert.NoError(s.t, err)

	err = m.HandleAction(*ctx)
	require.NoError(s.t, err)
}

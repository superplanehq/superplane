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

func Test_Merge(t *testing.T) {
	steps := NewMergeTestSteps(t)

	steps.CreateWorkflow()
	steps.CreateEvents()
	steps.CreateQueueItems()

	m := &Merge{}

	steps.ProcessFirstEvent(m)
	steps.ProcessSecondEvent(m)

	steps.AssertOneExecutionCreated()
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
	}

	require.NoError(s.t, s.Tx.Updates(&wf).Error)

	s.Wf = wf
	s.StartNode = n1
	s.ProcessNode1 = n2
	s.ProcessNode2 = n3
	s.MergeNode = n4
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

func (s *MergeTestSteps) ProcessFirstEvent(m *Merge) {
	fmt.Println("Processing first event")

	ctx1, err := contexts.BuildProcessQueueContext(s.Tx, s.MergeNode, s.QueureItem1)
	assert.NoError(s.t, err)

	err = m.ProcessQueueItem(*ctx1)
	require.NoError(s.t, err)
}

func (s *MergeTestSteps) ProcessSecondEvent(m *Merge) {
	fmt.Println("Processing second event")

	ctx2, err := contexts.BuildProcessQueueContext(s.Tx, s.MergeNode, s.QueureItem2)
	assert.NoError(s.t, err)

	err = m.ProcessQueueItem(*ctx2)
	require.NoError(s.t, err)
}

func (s *MergeTestSteps) AssertOneExecutionCreated() {
	var executions []models.WorkflowNodeExecution
	require.NoError(s.t, s.Tx.Where("node_id = ?", s.MergeNode.NodeID).Find(&executions).Error)
	assert.Equal(s.t, 1, len(executions))
}

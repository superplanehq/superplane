package telemetry

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
)

func TestCountStuckQueueNodes_NodeWithQueueAndNoExecutionsIsCounted(t *testing.T) {
	database.TruncateTables()

	steps := stuckQueueItemsTestSteps{t: t}
	steps.CreateWorkflow()
	steps.CreateWorkflowNode()
	steps.CreateRootEvent()

	db := database.Conn()

	queueItem := &models.CanvasNodeQueueItem{
		WorkflowID:  steps.workflow.ID,
		NodeID:      steps.node.NodeID,
		RootEventID: steps.rootEvent.ID,
		EventID:     steps.rootEvent.ID,
	}

	require.NoError(t, db.Create(queueItem).Error)

	count, err := countStuckQueueNodes()
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
}

func TestCountStuckQueueNodes_NodeWithOnlyFinishedExecutionsIsCounted(t *testing.T) {
	database.TruncateTables()

	steps := stuckQueueItemsTestSteps{t: t}
	steps.CreateWorkflow()
	steps.CreateWorkflowNode()
	steps.CreateRootEvent()

	db := database.Conn()

	queueItem := &models.CanvasNodeQueueItem{
		WorkflowID:  steps.workflow.ID,
		NodeID:      steps.node.NodeID,
		RootEventID: steps.rootEvent.ID,
		EventID:     steps.rootEvent.ID,
	}

	require.NoError(t, db.Create(queueItem).Error)

	exec := &models.CanvasNodeExecution{
		WorkflowID:  steps.workflow.ID,
		NodeID:      steps.node.NodeID,
		RootEventID: steps.rootEvent.ID,
		EventID:     steps.rootEvent.ID,
		State:       models.CanvasNodeExecutionStateFinished,
	}

	require.NoError(t, db.Create(exec).Error)

	count, err := countStuckQueueNodes()
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
}

func TestCountStuckQueueNodes_NodeWithNonFinishedExecutionIsNotCounted(t *testing.T) {
	database.TruncateTables()

	steps := stuckQueueItemsTestSteps{t: t}
	steps.CreateWorkflow()
	steps.CreateWorkflowNode()
	steps.CreateRootEvent()

	db := database.Conn()

	queueItem := &models.CanvasNodeQueueItem{
		WorkflowID:  steps.workflow.ID,
		NodeID:      steps.node.NodeID,
		RootEventID: steps.rootEvent.ID,
		EventID:     steps.rootEvent.ID,
	}

	require.NoError(t, db.Create(queueItem).Error)

	exec := &models.CanvasNodeExecution{
		WorkflowID:  steps.workflow.ID,
		NodeID:      steps.node.NodeID,
		RootEventID: steps.rootEvent.ID,
		EventID:     steps.rootEvent.ID,
		State:       models.CanvasNodeExecutionStateStarted,
	}

	require.NoError(t, db.Create(exec).Error)

	count, err := countStuckQueueNodes()
	require.NoError(t, err)
	require.Equal(t, int64(0), count)
}

type stuckQueueItemsTestSteps struct {
	t         *testing.T
	workflow  *models.Canvas
	node      *models.CanvasNode
	rootEvent *models.CanvasEvent
}

func (s *stuckQueueItemsTestSteps) CreateWorkflow() {
	s.workflow = &models.Canvas{
		OrganizationID: uuid.New(),
		Name:           "Test Workflow",
		Description:    "This is a test workflow",
	}

	require.NoError(s.t, database.Conn().Create(s.workflow).Error)
}

func (s *stuckQueueItemsTestSteps) CreateWorkflowNode() {
	s.node = &models.CanvasNode{
		WorkflowID: s.workflow.ID,
		NodeID:     "node-1",
	}

	require.NoError(s.t, database.Conn().Create(s.node).Error)
}

func (s *stuckQueueItemsTestSteps) CreateRootEvent() {
	s.rootEvent = &models.CanvasEvent{
		WorkflowID: s.workflow.ID,
		NodeID:     s.node.NodeID,
		Channel:    "default",
		Data:       datatypes.JSONType[any]{},
		State:      models.CanvasEventStatePending,
	}

	require.NoError(s.t, database.Conn().Create(s.rootEvent).Error)
}

package telemetry

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
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

func TestCountPendingEvents(t *testing.T) {
	database.TruncateTables()

	steps := stuckQueueItemsTestSteps{t: t}
	steps.CreateWorkflow()
	steps.CreateWorkflowNode()
	steps.CreateRootEvent()

	routedEvent := &models.CanvasEvent{
		WorkflowID: steps.workflow.ID,
		NodeID:     steps.node.NodeID,
		Channel:    "default",
		Data:       datatypes.JSONType[any]{},
		State:      models.CanvasEventStateRouted,
	}

	require.NoError(t, database.Conn().Create(routedEvent).Error)

	count, err := countPendingEvents()
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
}

func TestCountPendingEvents_DeletedWorkflowIsNotCounted(t *testing.T) {
	database.TruncateTables()

	activeSteps := stuckQueueItemsTestSteps{t: t}
	activeSteps.CreateWorkflow()
	activeSteps.CreateWorkflowNode()
	activeSteps.CreateRootEvent()

	deletedSteps := stuckQueueItemsTestSteps{t: t}
	deletedSteps.CreateWorkflow()
	deletedSteps.CreateWorkflowNode()
	deletedSteps.CreateRootEvent()

	require.NoError(t, deletedSteps.workflow.SoftDelete())

	count, err := countPendingEvents()
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
}

func TestCountPendingExecutions(t *testing.T) {
	database.TruncateTables()

	steps := stuckQueueItemsTestSteps{t: t}
	steps.CreateWorkflow()
	steps.CreateWorkflowNode()
	steps.CreateRootEvent()

	pendingExecution := &models.CanvasNodeExecution{
		WorkflowID:  steps.workflow.ID,
		NodeID:      steps.node.NodeID,
		RootEventID: steps.rootEvent.ID,
		EventID:     steps.rootEvent.ID,
		State:       models.CanvasNodeExecutionStatePending,
	}

	require.NoError(t, database.Conn().Create(pendingExecution).Error)

	startedExecution := &models.CanvasNodeExecution{
		WorkflowID:  steps.workflow.ID,
		NodeID:      steps.node.NodeID,
		RootEventID: steps.rootEvent.ID,
		EventID:     steps.rootEvent.ID,
		State:       models.CanvasNodeExecutionStateStarted,
	}

	require.NoError(t, database.Conn().Create(startedExecution).Error)

	count, err := countPendingExecutions()
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
}

func TestCountPendingExecutions_DeletedWorkflowIsNotCounted(t *testing.T) {
	database.TruncateTables()

	activeSteps := stuckQueueItemsTestSteps{t: t}
	activeSteps.CreateWorkflow()
	activeSteps.CreateWorkflowNode()
	activeSteps.CreateRootEvent()

	deletedSteps := stuckQueueItemsTestSteps{t: t}
	deletedSteps.CreateWorkflow()
	deletedSteps.CreateWorkflowNode()
	deletedSteps.CreateRootEvent()

	activeExecution := &models.CanvasNodeExecution{
		WorkflowID:  activeSteps.workflow.ID,
		NodeID:      activeSteps.node.NodeID,
		RootEventID: activeSteps.rootEvent.ID,
		EventID:     activeSteps.rootEvent.ID,
		State:       models.CanvasNodeExecutionStatePending,
	}
	require.NoError(t, database.Conn().Create(activeExecution).Error)

	deletedExecution := &models.CanvasNodeExecution{
		WorkflowID:  deletedSteps.workflow.ID,
		NodeID:      deletedSteps.node.NodeID,
		RootEventID: deletedSteps.rootEvent.ID,
		EventID:     deletedSteps.rootEvent.ID,
		State:       models.CanvasNodeExecutionStatePending,
	}
	require.NoError(t, database.Conn().Create(deletedExecution).Error)

	require.NoError(t, deletedSteps.workflow.SoftDelete())

	count, err := countPendingExecutions()
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
}

type stuckQueueItemsTestSteps struct {
	t         *testing.T
	workflow  *models.Canvas
	node      *models.CanvasNode
	rootEvent *models.CanvasEvent
}

func (s *stuckQueueItemsTestSteps) CreateWorkflow() {
	now := time.Now()
	liveVersionID := uuid.New()
	workflow := &models.Canvas{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		LiveVersionID:  &liveVersionID,
		Name:           "Test Workflow",
		Description:    "This is a test workflow",
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}

	require.NoError(s.t, database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(workflow).Error; err != nil {
			return err
		}
		return tx.Create(&models.CanvasVersion{
			ID:          liveVersionID,
			WorkflowID:  workflow.ID,
			State:       models.CanvasVersionStatePublished,
			PublishedAt: &now,
			Nodes:       datatypes.NewJSONSlice([]models.Node{}),
			Edges:       datatypes.NewJSONSlice([]models.Edge{}),
			CreatedAt:   &now,
			UpdatedAt:   &now,
		}).Error
	}))

	s.workflow = workflow
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

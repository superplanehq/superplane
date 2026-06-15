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
		Data:       models.JSONValue{},
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

func TestCountOrganizations(t *testing.T) {
	database.TruncateTables()

	org := &models.Organization{Name: "Acme"}
	require.NoError(t, database.Conn().Create(org).Error)

	deletedOrg := &models.Organization{Name: "Deleted Co"}
	require.NoError(t, database.Conn().Create(deletedOrg).Error)
	require.NoError(t, database.Conn().Delete(deletedOrg).Error)

	count, err := countOrganizations()
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
}

func TestCountUsers(t *testing.T) {
	database.TruncateTables()

	org := &models.Organization{Name: "Acme"}
	require.NoError(t, database.Conn().Create(org).Error)

	user := &models.User{
		OrganizationID: org.ID,
		Name:           "Alice",
		Type:           models.UserTypeHuman,
		TokenHash:      "hash",
	}
	require.NoError(t, database.Conn().Create(user).Error)

	deletedUser := &models.User{
		OrganizationID: org.ID,
		Name:           "Bob",
		Type:           models.UserTypeHuman,
		TokenHash:      "hash",
	}
	require.NoError(t, database.Conn().Create(deletedUser).Error)
	require.NoError(t, database.Conn().Delete(deletedUser).Error)

	count, err := countUsers()
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
}

func TestCountWorkflows(t *testing.T) {
	database.TruncateTables()

	steps := stuckQueueItemsTestSteps{t: t}
	steps.CreateWorkflow()

	deletedSteps := stuckQueueItemsTestSteps{t: t}
	deletedSteps.CreateWorkflow()
	require.NoError(t, deletedSteps.workflow.SoftDelete())

	count, err := countWorkflows()
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
}

func TestCountWorkflowNodes(t *testing.T) {
	database.TruncateTables()

	steps := stuckQueueItemsTestSteps{t: t}
	steps.CreateWorkflow()
	steps.CreateWorkflowNode()

	deletedNode := &models.CanvasNode{
		WorkflowID: steps.workflow.ID,
		NodeID:     "node-2",
	}
	require.NoError(t, database.Conn().Create(deletedNode).Error)
	require.NoError(t, database.Conn().Delete(deletedNode).Error)

	count, err := countWorkflowNodes()
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
}

func TestCountWorkflowNodes_DeletedWorkflowIsNotCounted(t *testing.T) {
	database.TruncateTables()

	steps := stuckQueueItemsTestSteps{t: t}
	steps.CreateWorkflow()
	steps.CreateWorkflowNode()
	require.NoError(t, steps.workflow.SoftDelete())

	count, err := countWorkflowNodes()
	require.NoError(t, err)
	require.Equal(t, int64(0), count)
}

func TestCountDrafts(t *testing.T) {
	database.TruncateTables()

	steps := stuckQueueItemsTestSteps{t: t}
	steps.CreateWorkflow()

	branchName := "drafts/" + uuid.New().String()
	now := time.Now()
	draft := &models.CanvasVersion{
		ID:          uuid.New(),
		WorkflowID:  steps.workflow.ID,
		State:       models.CanvasVersionStateDraft,
		BranchName:  &branchName,
		DisplayName: "Draft #1",
		CreatedAt:   &now,
		UpdatedAt:   &now,
	}
	require.NoError(t, database.Conn().Create(draft).Error)

	count, err := countDrafts()
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
}

func TestCountDrafts_DeletedWorkflowIsNotCounted(t *testing.T) {
	database.TruncateTables()

	steps := stuckQueueItemsTestSteps{t: t}
	steps.CreateWorkflow()

	branchName := "drafts/" + uuid.New().String()
	now := time.Now()
	draft := &models.CanvasVersion{
		ID:          uuid.New(),
		WorkflowID:  steps.workflow.ID,
		State:       models.CanvasVersionStateDraft,
		BranchName:  &branchName,
		DisplayName: "Draft #1",
		CreatedAt:   &now,
		UpdatedAt:   &now,
	}
	require.NoError(t, database.Conn().Create(draft).Error)
	require.NoError(t, steps.workflow.SoftDelete())

	count, err := countDrafts()
	require.NoError(t, err)
	require.Equal(t, int64(0), count)
}

func TestCountIntegrations(t *testing.T) {
	database.TruncateTables()

	org := &models.Organization{Name: "Acme"}
	require.NoError(t, database.Conn().Create(org).Error)

	_, err := models.CreateIntegration(uuid.New(), org.ID, "slack", "active", map[string]any{})
	require.NoError(t, err)

	_, err = models.CreateIntegration(uuid.New(), org.ID, "github", "deleted", map[string]any{})
	require.NoError(t, err)
	deletedIntegration, err := models.FindIntegrationByName(org.ID, "deleted")
	require.NoError(t, err)
	require.NoError(t, deletedIntegration.SoftDelete())

	count, err := countIntegrations()
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
}

func TestCountIntegrationSecrets(t *testing.T) {
	database.TruncateTables()

	org := &models.Organization{Name: "Acme"}
	require.NoError(t, database.Conn().Create(org).Error)

	integration, err := models.CreateIntegration(uuid.New(), org.ID, "slack", "active", map[string]any{})
	require.NoError(t, err)

	now := time.Now()
	secret := &models.IntegrationSecret{
		OrganizationID: org.ID,
		InstallationID: integration.ID,
		Name:           "token",
		Value:          []byte("secret"),
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	require.NoError(t, database.Conn().Create(secret).Error)

	deletedIntegration, err := models.CreateIntegration(uuid.New(), org.ID, "github", "deleted", map[string]any{})
	require.NoError(t, err)
	deletedSecret := &models.IntegrationSecret{
		OrganizationID: org.ID,
		InstallationID: deletedIntegration.ID,
		Name:           "token",
		Value:          []byte("secret"),
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	require.NoError(t, database.Conn().Create(deletedSecret).Error)
	require.NoError(t, deletedIntegration.SoftDelete())

	count, err := countIntegrationSecrets()
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
		Data:       models.JSONValue{},
		State:      models.CanvasEventStatePending,
	}

	require.NoError(s.t, database.Conn().Create(s.rootEvent).Error)
}

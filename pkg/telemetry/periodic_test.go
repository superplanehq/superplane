package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"google.golang.org/grpc/metadata"
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

func TestCountActiveUsers(t *testing.T) {
	database.TruncateTables()

	org, userID := createActiveUserTestFixtures(t)
	now := time.Now()

	require.NoError(t, models.TouchUserLastActiveAt(userID, now))

	inactiveUser, err := models.CreateServiceAccount(database.Conn(), org.ID, "inactive", nil, userID)
	require.NoError(t, err)
	require.NoError(t, models.TouchUserLastActiveAt(inactiveUser.ID, now.Add(-25*time.Hour)))

	deletedUser, err := models.CreateServiceAccount(database.Conn(), org.ID, "deleted", nil, userID)
	require.NoError(t, err)
	require.NoError(t, models.TouchUserLastActiveAt(deletedUser.ID, now))
	require.NoError(t, deletedUser.Delete())

	count, err := countActiveUsers(24 * time.Hour)
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
}

func TestCountActiveWorkflows(t *testing.T) {
	database.TruncateTables()

	activeSteps := stuckQueueItemsTestSteps{t: t}
	activeSteps.CreateWorkflow()
	inactiveSteps := stuckQueueItemsTestSteps{t: t}
	inactiveSteps.CreateWorkflow()
	deletedSteps := stuckQueueItemsTestSteps{t: t}
	deletedSteps.CreateWorkflow()

	now := time.Now()
	createWorkflowRun(t, activeSteps.workflow.ID, *activeSteps.workflow.LiveVersionID, now)
	createWorkflowRun(t, inactiveSteps.workflow.ID, *inactiveSteps.workflow.LiveVersionID, now.Add(-25*time.Hour))
	createWorkflowRun(t, deletedSteps.workflow.ID, *deletedSteps.workflow.LiveVersionID, now)
	createWorkflowRun(t, activeSteps.workflow.ID, *activeSteps.workflow.LiveVersionID, now.Add(-time.Minute))
	require.NoError(t, deletedSteps.workflow.SoftDelete())

	count, err := countActiveWorkflows(24 * time.Hour)
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
}

func TestCountDailyWorkflowMetrics(t *testing.T) {
	database.TruncateTables()

	activeSteps := stuckQueueItemsTestSteps{t: t}
	activeSteps.CreateWorkflow()
	activeSteps.CreateWorkflowNode()

	inactiveSteps := stuckQueueItemsTestSteps{t: t}
	inactiveSteps.CreateWorkflow()
	inactiveSteps.CreateWorkflowNode()

	deletedSteps := stuckQueueItemsTestSteps{t: t}
	deletedSteps.CreateWorkflow()
	deletedSteps.CreateWorkflowNode()

	now := time.Now()
	activeRun := createWorkflowRunReturning(t, activeSteps.workflow.ID, *activeSteps.workflow.LiveVersionID, now)
	inactiveRun := createWorkflowRunReturning(t, inactiveSteps.workflow.ID, *inactiveSteps.workflow.LiveVersionID, now.Add(-25*time.Hour))
	deletedRun := createWorkflowRunReturning(t, deletedSteps.workflow.ID, *deletedSteps.workflow.LiveVersionID, now)
	require.NoError(t, deletedSteps.workflow.SoftDelete())

	activeEvent := createWorkflowEventReturning(t, activeSteps.workflow.ID, activeSteps.node.NodeID, activeRun.ID, now)
	createWorkflowEvent(t, activeSteps.workflow.ID, activeSteps.node.NodeID, activeRun.ID, now.Add(-time.Minute))
	inactiveEvent := createWorkflowEventReturning(t, inactiveSteps.workflow.ID, inactiveSteps.node.NodeID, inactiveRun.ID, now.Add(-25*time.Hour))
	createWorkflowEvent(t, deletedSteps.workflow.ID, deletedSteps.node.NodeID, deletedRun.ID, now)

	createWorkflowNodeExecution(t, activeSteps.workflow.ID, activeSteps.node.NodeID, activeEvent.ID, activeEvent.ID, activeRun.ID, now)
	createWorkflowNodeExecution(t, activeSteps.workflow.ID, activeSteps.node.NodeID, activeEvent.ID, activeEvent.ID, activeRun.ID, now.Add(-time.Minute))
	createWorkflowNodeExecution(t, inactiveSteps.workflow.ID, inactiveSteps.node.NodeID, inactiveEvent.ID, inactiveEvent.ID, inactiveRun.ID, now.Add(-25*time.Hour))
	deletedEvent := createWorkflowEventReturning(t, deletedSteps.workflow.ID, deletedSteps.node.NodeID, deletedRun.ID, now)
	createWorkflowNodeExecution(t, deletedSteps.workflow.ID, deletedSteps.node.NodeID, deletedEvent.ID, deletedEvent.ID, deletedRun.ID, now)

	runCount, err := countWorkflowRunsCreated(24 * time.Hour)
	require.NoError(t, err)
	require.Equal(t, int64(1), runCount)

	eventCount, err := countWorkflowEventsCreated(24 * time.Hour)
	require.NoError(t, err)
	require.Equal(t, int64(2), eventCount)

	executionCount, err := countWorkflowNodeExecutionsCreated(24 * time.Hour)
	require.NoError(t, err)
	require.Equal(t, int64(2), executionCount)
}

func TestRecordUserDatabaseActivity_UpdatesUserLastActiveAt(t *testing.T) {
	database.TruncateTables()
	resetUserActivityThrottleForTests()

	org, userID := createActiveUserTestFixtures(t)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-user-id", userID.String()))
	recordUserDatabaseActivity(ctx)

	require.Eventually(t, func() bool {
		user, err := models.FindActiveUserByID(org.ID.String(), userID.String())
		if err != nil {
			return false
		}

		return user.LastActiveAt != nil
	}, time.Second, 10*time.Millisecond)
}

func createActiveUserTestFixtures(t *testing.T) (*models.Organization, uuid.UUID) {
	t.Helper()

	tx := database.Conn().Begin()
	org, err := models.CreateOrganizationInTransaction(tx, "active-user-org", "Active User Org")
	require.NoError(t, err)

	account, err := models.CreateAccountInTransaction(tx, "active-user@example.com", "Active User")
	require.NoError(t, err)

	user, err := models.CreateUserInTransaction(tx, org.ID, account.ID, account.Email, account.Name)
	require.NoError(t, err)
	require.NoError(t, tx.Commit().Error)

	return org, user.ID
}

func createWorkflowRun(t *testing.T, workflowID, versionID uuid.UUID, createdAt time.Time) {
	t.Helper()
	_ = createWorkflowRunReturning(t, workflowID, versionID, createdAt)
}

func createWorkflowRunReturning(t *testing.T, workflowID, versionID uuid.UUID, createdAt time.Time) *models.CanvasRun {
	t.Helper()

	run := &models.CanvasRun{
		ID:         uuid.New(),
		WorkflowID: workflowID,
		VersionID:  versionID,
		State:      models.CanvasRunStateStarted,
		CreatedAt:  &createdAt,
		UpdatedAt:  &createdAt,
	}
	require.NoError(t, database.Conn().Create(run).Error)
	return run
}

func createWorkflowEvent(t *testing.T, workflowID uuid.UUID, nodeID string, runID uuid.UUID, createdAt time.Time) {
	t.Helper()
	_ = createWorkflowEventReturning(t, workflowID, nodeID, runID, createdAt)
}

func createWorkflowEventReturning(t *testing.T, workflowID uuid.UUID, nodeID string, runID uuid.UUID, createdAt time.Time) *models.CanvasEvent {
	t.Helper()

	event := &models.CanvasEvent{
		ID:         uuid.New(),
		WorkflowID: workflowID,
		NodeID:     nodeID,
		Channel:    "default",
		Data:       models.JSONValue{},
		RunID:      runID,
		State:      models.CanvasEventStatePending,
		CreatedAt:  &createdAt,
	}
	require.NoError(t, database.Conn().Create(event).Error)
	return event
}

func createWorkflowNodeExecution(
	t *testing.T,
	workflowID uuid.UUID,
	nodeID string,
	rootEventID, eventID, runID uuid.UUID,
	createdAt time.Time,
) {
	t.Helper()

	execution := &models.CanvasNodeExecution{
		ID:          uuid.New(),
		WorkflowID:  workflowID,
		NodeID:      nodeID,
		RootEventID: rootEventID,
		RunID:       runID,
		EventID:     eventID,
		State:       models.CanvasNodeExecutionStatePending,
		CreatedAt:   &createdAt,
		UpdatedAt:   &createdAt,
	}
	require.NoError(t, database.Conn().Create(execution).Error)
}

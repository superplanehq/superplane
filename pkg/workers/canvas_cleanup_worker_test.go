package workers

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/database"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type cleanupProvider struct {
	deleted []string
	err     error
}

func (p *cleanupProvider) Name() string { return "test" }

func (p *cleanupProvider) CreateSession(context.Context, agents.CreateSessionOptions) (*agents.CreateSessionResult, error) {
	return nil, errors.New("not used")
}

func (p *cleanupProvider) SendMessage(context.Context, string, string, agents.SendMessageOptions) error {
	return errors.New("not used")
}

func (p *cleanupProvider) InterruptSession(context.Context, string) error {
	return errors.New("not used")
}

func (p *cleanupProvider) DefineOutcome(context.Context, string, agents.DefineOutcomeOptions) error {
	return errors.New("not used")
}

func (p *cleanupProvider) StreamEvents(context.Context, string, func(agents.ProviderEvent) error) error {
	return errors.New("not used")
}

func (p *cleanupProvider) DeleteSession(_ context.Context, providerSessionID string) error {
	p.deleted = append(p.deleted, providerSessionID)
	return p.err
}

type cleanupGitProvider struct {
	deleted []string
	err     error
}

func (p *cleanupGitProvider) Name() string {
	return "test-git"
}

func (p *cleanupGitProvider) GetRepositoryID(options git.RepositoryOptions) string {
	return "repo-" + options.CanvasID.String()
}

func (p *cleanupGitProvider) CreateRepository(context.Context, string) (*git.Repository, error) {
	return nil, errors.New("not used")
}

func (p *cleanupGitProvider) DeleteRepository(_ context.Context, repoID string) error {
	p.deleted = append(p.deleted, repoID)
	return p.err
}

func (p *cleanupGitProvider) ListFiles(context.Context, string, string) ([]string, error) {
	return nil, errors.New("not used")
}

func (p *cleanupGitProvider) GetFile(context.Context, string, string, string) (io.ReadCloser, error) {
	return nil, errors.New("not used")
}

func (p *cleanupGitProvider) Commit(context.Context, string, git.CommitOptions) (string, error) {
	return "", errors.New("not used")
}

func (p *cleanupGitProvider) Head(context.Context, string, string) (string, error) {
	return "", errors.New("not used")
}

func (p *cleanupGitProvider) ListBranches(context.Context, string, string) ([]string, error) {
	return nil, errors.New("not used")
}

func (p *cleanupGitProvider) CreateBranch(context.Context, string, string, string) error {
	return errors.New("not used")
}

func (p *cleanupGitProvider) MergeBranch(context.Context, string, string, string, string, git.CommitAuthor) (string, error) {
	return "", errors.New("not used")
}

func (p *cleanupGitProvider) DeleteBranch(context.Context, string, string) error {
	return errors.New("not used")
}

func createAgentSessionWithMessage(t *testing.T, organizationID, userID, canvasID uuid.UUID) *models.AgentSession {
	t.Helper()

	session := &models.AgentSession{
		OrganizationID:    organizationID,
		UserID:            userID,
		CanvasID:          canvasID,
		Provider:          "test",
		ProviderSessionID: "provider-session-" + uuid.NewString(),
		Status:            models.AgentSessionStatusIdle,
	}

	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := models.CreateAgentSessionInTransaction(tx, session); err != nil {
			return err
		}

		return models.AppendAgentSessionMessageInTransaction(tx, &models.AgentSessionMessage{
			SessionID: session.ID,
			Role:      models.AgentMessageRoleUser,
			Content:   "hello",
		})
	}))

	return session
}

func countAgentSessions(t *testing.T, sessionID uuid.UUID) int64 {
	t.Helper()

	var count int64
	require.NoError(t, database.Conn().Model(&models.AgentSession{}).Where("id = ?", sessionID).Count(&count).Error)
	return count
}

func countAgentSessionMessages(t *testing.T, sessionID uuid.UUID) int64 {
	t.Helper()

	var count int64
	require.NoError(t, database.Conn().Model(&models.AgentSessionMessage{}).Where("session_id = ?", sessionID).Count(&count).Error)
	return count
}

func Test__CanvasCleanupWorker_ProcessesDeletedWorkflow(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	cleaner := &cleanupProvider{}
	worker := NewCanvasCleanupWorker(r.GitProvider, cleaner)

	//
	// Create a canvas with nodes, events, executions, and queue items
	//
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
			{
				NodeID: "node-2",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	session := createAgentSessionWithMessage(t, r.Organization.ID, r.User, canvas.ID)

	// Create associated data
	event1 := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
	event2 := support.EmitCanvasEventForNode(t, canvas.ID, "node-2", "default", nil)
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", event1.ID, event2.ID)
	support.CreateQueueItem(t, canvas.ID, "node-1", event1.ID, event2.ID)

	// Create canvas node execution KV
	require.NoError(t, models.CreateNodeExecutionKVInTransaction(
		database.Conn(),
		canvas.ID,
		"node-1",
		execution.ID,
		"test-key",
		"test-value",
	))

	// Create workflow node request
	nodeRequest := models.CanvasNodeRequest{
		ID:         uuid.New(),
		WorkflowID: canvas.ID,
		NodeID:     "node-1",
		Type:       models.NodeRequestTypeInvokeAction,
		State:      models.NodeExecutionRequestStatePending,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: "test",
				Parameters: map[string]any{},
			},
		}),
	}
	require.NoError(t, database.Conn().Create(&nodeRequest).Error)

	//
	// Verify all data exists before soft delete
	//
	_, err := models.FindCanvas(r.Organization.ID, canvas.ID)
	require.NoError(t, err)
	nodes, err := models.FindCanvasNodes(canvas.ID)
	require.NoError(t, err)
	assert.Len(t, nodes, 2)
	support.VerifyCanvasEventsCount(t, canvas.ID, 2)
	support.VerifyNodeExecutionsCount(t, canvas.ID, 1)
	support.VerifyNodeQueueCount(t, canvas.ID, 1)

	// Verify KV and request exist
	support.VerifyNodeExecutionKVCount(t, canvas.ID, 1)
	support.VerifyNodeRequestCount(t, canvas.ID, 1)
	assert.Equal(t, int64(1), countAgentSessions(t, session.ID))
	assert.Equal(t, int64(1), countAgentSessionMessages(t, session.ID))

	//
	// Soft delete the canvas using the new soft delete method
	//
	err = canvas.SoftDelete()
	require.NoError(t, err)

	// Verify workflow is soft deleted
	_, err = models.FindCanvas(r.Organization.ID, canvas.ID)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	//
	// Fetch the updated workflow with deleted_at set
	//
	deletedCanvas, err := models.FindUnscopedCanvas(canvas.ID)
	require.NoError(t, err)
	require.True(t, deletedCanvas.DeletedAt.Valid, "DeletedAt should be set")
	require.NoError(t, database.Conn().Unscoped().Model(&models.Canvas{}).Where("id = ?", canvas.ID).Update("deleted_at", time.Now().AddDate(0, 0, -31)).Error)
	deletedCanvas, err = models.FindUnscopedCanvas(canvas.ID)
	require.NoError(t, err)

	//
	// Process the deleted canvas with cleanup worker
	// The worker now processes resources in batches, so it might take multiple calls
	//

	// Process until everything is cleaned up (with a reasonable limit)
	maxAttempts := 10
	for i := 0; i < maxAttempts; i++ {
		err = worker.LockAndProcessCanvas(*deletedCanvas)
		require.NoError(t, err)

		// Check if workflow is completely deleted
		var canvasCount int64
		database.Conn().Unscoped().Model(&models.Canvas{}).Where("id = ?", canvas.ID).Count(&canvasCount)
		if canvasCount == 0 {
			break
		}
	}

	//
	// Verify everything is permanently deleted
	//

	// Canvas should be permanently deleted
	var canvasCount int64
	database.Conn().Unscoped().Model(&models.Canvas{}).Where("id = ?", canvas.ID).Count(&canvasCount)
	assert.Equal(t, int64(0), canvasCount)

	// All associated data should be permanently deleted
	nodes, err = models.FindCanvasNodes(canvas.ID)
	require.NoError(t, err)
	assert.Len(t, nodes, 0)

	support.VerifyCanvasEventsCount(t, canvas.ID, 0)
	support.VerifyNodeExecutionsCount(t, canvas.ID, 0)
	support.VerifyNodeQueueCount(t, canvas.ID, 0)

	// KV and request should be deleted
	support.VerifyNodeExecutionKVCount(t, canvas.ID, 0)
	support.VerifyNodeRequestCount(t, canvas.ID, 0)

	assert.Equal(t, int64(0), countAgentSessions(t, session.ID))
	assert.Equal(t, int64(0), countAgentSessionMessages(t, session.ID))
	assert.Contains(t, cleaner.deleted, session.ProviderSessionID)
}

func Test__CanvasCleanupWorker_ProviderCleanupFailureDoesNotBlockDatabaseCleanup(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	cleaner := &cleanupProvider{err: errors.New("provider unavailable")}
	worker := NewCanvasCleanupWorker(r.GitProvider, cleaner)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
	session := createAgentSessionWithMessage(t, r.Organization.ID, r.User, canvas.ID)

	require.NoError(t, canvas.SoftDelete())
	deletedAtOutsideGracePeriod := time.Now().AddDate(0, 0, -31)
	require.NoError(t, database.Conn().Unscoped().Model(&models.Canvas{}).Where("id = ?", canvas.ID).Update("deleted_at", deletedAtOutsideGracePeriod).Error)

	deletedCanvas, err := models.FindUnscopedCanvas(canvas.ID)
	require.NoError(t, err)
	require.NoError(t, worker.LockAndProcessCanvas(*deletedCanvas))

	var canvasCount int64
	require.NoError(t, database.Conn().Unscoped().Model(&models.Canvas{}).Where("id = ?", canvas.ID).Count(&canvasCount).Error)
	assert.Equal(t, int64(0), canvasCount)
	assert.Equal(t, int64(0), countAgentSessions(t, session.ID))
	assert.Equal(t, int64(0), countAgentSessionMessages(t, session.ID))
	assert.Contains(t, cleaner.deleted, session.ProviderSessionID)
}

func Test__CanvasCleanupWorker_DeletesGitRepositoryAfterCanvasCleanup(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	gitProvider := &cleanupGitProvider{}
	worker := NewCanvasCleanupWorker(gitProvider)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
	repoID := "repo-" + canvas.ID.String()
	_, err := canvas.CreatePendingRepository(gitProvider.Name(), repoID)
	require.NoError(t, err)

	require.NoError(t, canvas.SoftDelete())
	deletedAtOutsideGracePeriod := time.Now().AddDate(0, 0, -31)
	require.NoError(t, database.Conn().Unscoped().Model(&models.Canvas{}).Where("id = ?", canvas.ID).Update("deleted_at", deletedAtOutsideGracePeriod).Error)

	deletedCanvas, err := models.FindUnscopedCanvas(canvas.ID)
	require.NoError(t, err)
	require.NoError(t, worker.LockAndProcessCanvas(*deletedCanvas))

	var canvasCount int64
	require.NoError(t, database.Conn().Unscoped().Model(&models.Canvas{}).Where("id = ?", canvas.ID).Count(&canvasCount).Error)
	assert.Equal(t, int64(0), canvasCount)

	var repositoryCount int64
	require.NoError(t, database.Conn().Model(&models.Repository{}).Where("canvas_id = ?", canvas.ID).Count(&repositoryCount).Error)
	assert.Equal(t, int64(0), repositoryCount)
	assert.Contains(t, gitProvider.deleted, repoID)
}

func Test__CanvasCleanupWorker_ProcessesWorkflowFromSoftDeletedOrganization(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewCanvasCleanupWorker(r.GitProvider)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
			{
				NodeID: "node-2",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	now := time.Now()
	nodeRequest := models.CanvasNodeRequest{
		ID:         uuid.New(),
		WorkflowID: canvas.ID,
		NodeID:     "node-1",
		Type:       models.NodeRequestTypeInvokeAction,
		State:      models.NodeExecutionRequestStatePending,
		RunAt:      now,
		CreatedAt:  now,
		UpdatedAt:  now,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: "test",
				Parameters: map[string]any{},
			},
		}),
	}
	require.NoError(t, database.Conn().Create(&nodeRequest).Error)

	require.NoError(t, models.SoftDeleteOrganization(r.Organization.ID.String()))
	deletedAtOutsideGracePeriod := time.Now().AddDate(0, 0, -31)
	require.NoError(t, database.Conn().
		Unscoped().
		Model(&models.Organization{}).
		Where("id = ?", r.Organization.ID).
		Update("deleted_at", deletedAtOutsideGracePeriod).
		Error)

	canvases, err := models.ListDeletedCanvases()
	require.NoError(t, err)
	require.Len(t, canvases, 1)
	require.Equal(t, canvas.ID, canvases[0].ID)
	require.True(t, canvases[0].DeletedAt.Valid)

	require.NoError(t, worker.LockAndProcessCanvas(canvases[0]))

	var canvasCount int64
	require.NoError(t, database.Conn().Unscoped().Model(&models.Canvas{}).Where("id = ?", canvas.ID).Count(&canvasCount).Error)
	assert.Equal(t, int64(0), canvasCount)

	nodes, err := models.FindCanvasNodes(canvas.ID)
	require.NoError(t, err)
	assert.Empty(t, nodes)

	support.VerifyNodeRequestCount(t, canvas.ID, 0)

	var organizationCount int64
	require.NoError(t, database.Conn().Unscoped().Model(&models.Organization{}).Where("id = ?", r.Organization.ID).Count(&organizationCount).Error)
	assert.Equal(t, int64(1), organizationCount)
}

func Test__CanvasCleanupWorker_ProcessesWorkflowWithWebhook(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewCanvasCleanupWorker(r.GitProvider)

	//
	// Create webhook
	//
	webhookID := uuid.New()
	webhook := models.Webhook{
		ID:     webhookID,
		State:  models.WebhookStatePending,
		Secret: []byte("secret"),
	}
	require.NoError(t, database.Conn().Create(&webhook).Error)

	//
	// Create a canvas with node that has webhook
	//
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
				WebhookID: &webhookID,
			},
		},
		[]models.Edge{},
	)

	//
	// Soft delete the canvas using the new soft delete method
	//
	err := canvas.SoftDelete()
	require.NoError(t, err)

	//
	// Verify webhook exists before cleanup
	//
	_, err = models.FindWebhook(webhookID)
	require.NoError(t, err)

	//
	// Fetch the updated workflow with deleted_at set
	//
	deletedCanvas, err := models.FindUnscopedCanvas(canvas.ID)
	require.NoError(t, err)
	require.True(t, deletedCanvas.DeletedAt.Valid, "DeletedAt should be set")
	require.NoError(t, database.Conn().Unscoped().Model(&models.Canvas{}).Where("id = ?", canvas.ID).Update("deleted_at", time.Now().AddDate(0, 0, -31)).Error)
	deletedCanvas, err = models.FindUnscopedCanvas(canvas.ID)
	require.NoError(t, err)

	//
	// Process the deleted canvas with cleanup worker
	// May take multiple calls due to batched resource deletion
	//
	maxAttempts := 10
	for i := 0; i < maxAttempts; i++ {
		err = worker.LockAndProcessCanvas(*deletedCanvas)
		require.NoError(t, err)

		// Check if workflow is completely deleted
		var canvasCount int64
		database.Conn().Unscoped().Model(&models.Canvas{}).Where("id = ?", canvas.ID).Count(&canvasCount)
		if canvasCount == 0 {
			break
		}
	}

	//
	// Verify webhook is soft deleted (marked for cleanup by webhook cleanup worker)
	//
	var webhookInDb models.Webhook
	err = database.Conn().Unscoped().Where("id = ?", webhookID).First(&webhookInDb).Error
	require.NoError(t, err)
	assert.NotNil(t, webhookInDb.DeletedAt)
}

func Test__CanvasCleanupWorker_HandlesEmptyWorkflow(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewCanvasCleanupWorker(r.GitProvider)

	//
	// Create a minimal canvas with no nodes, events, etc.
	//
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{},
		[]models.Edge{},
	)

	//
	// Soft delete the canvas using the new soft delete method
	//
	err := canvas.SoftDelete()
	require.NoError(t, err)

	//
	// Fetch the updated workflow with deleted_at set
	//
	deletedCanvas, err := models.FindUnscopedCanvas(canvas.ID)
	require.NoError(t, err)
	require.True(t, deletedCanvas.DeletedAt.Valid, "DeletedAt should be set")
	require.NoError(t, database.Conn().Unscoped().Model(&models.Canvas{}).Where("id = ?", canvas.ID).Update("deleted_at", time.Now().AddDate(0, 0, -31)).Error)
	deletedCanvas, err = models.FindUnscopedCanvas(canvas.ID)
	require.NoError(t, err)

	//
	// Process the deleted canvas with cleanup worker
	//
	err = worker.LockAndProcessCanvas(*deletedCanvas)
	require.NoError(t, err)

	//
	// Verify canvas is permanently deleted
	//
	var canvasCount int64
	database.Conn().Unscoped().Model(&models.Canvas{}).Where("id = ?", canvas.ID).Count(&canvasCount)
	assert.Equal(t, int64(0), canvasCount)
}

func Test__CanvasCleanupWorker_HandlesConcurrentProcessing(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	//
	// Create a canvas with some data
	//
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	event := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
	support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", event.ID, event.ID)

	//
	// Soft delete the canvas using the new soft delete method
	//
	err := canvas.SoftDelete()
	require.NoError(t, err)

	//
	// Fetch the updated workflow with deleted_at set
	//
	deletedCanvas, err := models.FindUnscopedCanvas(canvas.ID)
	require.NoError(t, err)
	require.True(t, deletedCanvas.DeletedAt.Valid, "DeletedAt should be set")
	require.NoError(t, database.Conn().Unscoped().Model(&models.Canvas{}).Where("id = ?", canvas.ID).Update("deleted_at", time.Now().AddDate(0, 0, -31)).Error)
	deletedCanvas, err = models.FindUnscopedCanvas(canvas.ID)
	require.NoError(t, err)

	//
	// Have two workers try to process the same canvas concurrently
	//
	results := make(chan error, 2)

	go func() {
		worker1 := NewCanvasCleanupWorker(r.GitProvider)
		results <- worker1.LockAndProcessCanvas(*deletedCanvas)
	}()

	go func() {
		worker2 := NewCanvasCleanupWorker(r.GitProvider)
		results <- worker2.LockAndProcessCanvas(*deletedCanvas)
	}()

	// Collect results - both should succeed (return nil)
	result1 := <-results
	result2 := <-results
	assert.NoError(t, result1)
	assert.NoError(t, result2)

	// Process remaining work until fully cleaned up
	maxAttempts := 10
	for i := 0; i < maxAttempts; i++ {
		worker := NewCanvasCleanupWorker(r.GitProvider)
		err := worker.LockAndProcessCanvas(*deletedCanvas)
		require.NoError(t, err)

		// Check if workflow is completely deleted
		var canvasCount int64
		database.Conn().Unscoped().Model(&models.Canvas{}).Where("id = ?", canvas.ID).Count(&canvasCount)
		if canvasCount == 0 {
			break
		}
	}

	//
	// Verify canvas is permanently deleted
	//
	var canvasCount int64
	database.Conn().Unscoped().Model(&models.Canvas{}).Where("id = ?", canvas.ID).Count(&canvasCount)
	assert.Equal(t, int64(0), canvasCount)

	// Verify associated data is cleaned up
	support.VerifyCanvasEventsCount(t, canvas.ID, 0)
	support.VerifyNodeExecutionsCount(t, canvas.ID, 0)
}

func Test__CanvasCleanupWorker_IgnoresNonDeletedWorkflows(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	worker := NewCanvasCleanupWorker(r.GitProvider)

	//
	// Create a normal (non-deleted) canvas
	//
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	_ = support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)

	//
	// Try to process a non-deleted canvas (should be harmless)
	//
	err := worker.LockAndProcessCanvas(*canvas)
	require.NoError(t, err)

	//
	// Verify canvas and data still exist
	//
	_, err = models.FindCanvas(r.Organization.ID, canvas.ID)
	require.NoError(t, err)

	nodes, err := models.FindCanvasNodes(canvas.ID)
	require.NoError(t, err)
	assert.Len(t, nodes, 1)

	support.VerifyCanvasEventsCount(t, canvas.ID, 1)
}

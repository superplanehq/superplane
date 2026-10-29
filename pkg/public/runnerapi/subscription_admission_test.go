package runnerapi

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	runnerlogs "github.com/superplanehq/superplane/pkg/runners/logs"
	"github.com/superplanehq/superplane/pkg/workers"
	"github.com/superplanehq/superplane/test/support"
)

func TestRunnerReservationWithoutTaskKeepsRunnerIdle(t *testing.T) {
	resource := support.Setup(t)
	defer resource.Close()
	runner, _, fleet := createRegistrationFixture(t)
	connectionID := uuid.New()
	db := database.DB(t.Context())
	require.NoError(t, db.Model(runner).Updates(map[string]any{"state": models.RunnerStateIdle, "current_connection_id": connectionID}).Error)
	server, err := NewServer(jwt.NewSigner("secret"), resource.Encryptor, runnerlogs.StoreFS, nil)
	require.NoError(t, err)

	require.NoError(t, server.reconcileRunnerConnection(t.Context(), nil, runner, connectionID, "", fleet.Slug, fleet.ScopeType))
	stored, err := models.FindRunner(db, runner.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerStateIdle, stored.State)
	_, err = stored.FindActiveTask(db)
	assert.ErrorIs(t, err, models.ErrRunnerTaskNotFound)
}

func TestRunnerReservationFailureProcessesSavedErrorAfterCommit(t *testing.T) {
	resource := support.Setup(t)
	defer resource.Close()
	db := database.DB(t.Context())
	runner, _, fleet := createRegistrationFixture(t)
	connectionID := uuid.New()
	require.NoError(t, db.Model(runner).Updates(map[string]any{"state": models.RunnerStateIdle, "current_connection_id": connectionID}).Error)
	task := &models.RunnerTask{ID: uuid.New(), OrganizationID: resource.Organization.ID, FleetID: fleet.ID,
		Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateQueued,
		QueuedAt: time.Now(), PayloadCiphertext: []byte(`{}`)}
	require.NoError(t, db.Create(task).Error)
	execution, request := createRunnerTaskPoll(t, resource, task)
	require.NoError(t, models.EnableExperimentalFeature(resource.Organization.ID, features.FeatureCodexSubscriptions))
	factory, err := models.CreateFactory(db, resource.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	connection, err := models.CreateFactoryCodexConnection(db, resource.Organization.ID, factory.ID, resource.User, time.Now())
	require.NoError(t, err)
	require.NoError(t, connection.BeginLogin(db, task.ID, time.Now()))
	require.NoError(t, db.Model(&models.FactoryCodexLease{}).Where("task_id = ?", task.ID).Update("revoked_at", time.Now()).Error)
	executor := workers.NewNodeRequestExecutor(resource.Encryptor, resource.Registry, "", resource.AuthService)
	server, err := NewServer(jwt.NewSigner("secret"), resource.Encryptor, runnerlogs.StoreFS, executor)
	require.NoError(t, err)

	require.NoError(t, server.reconcileRunnerConnection(t.Context(), nil, runner, connectionID, "", fleet.Slug, fleet.ScopeType))
	stored, err := models.FindRunnerTask(db, task.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerTaskStateFailed, stored.State)
	require.NotNil(t, stored.ErrorMessage)
	assert.Equal(t, models.ErrCodexConnectionStale.Error(), *stored.ErrorMessage)
	require.Eventually(t, func() bool {
		updated, err := models.FindCanvasNodeRequest(db, request.ID)
		return err == nil && updated.State == models.NodeExecutionRequestStateCompleted
	}, 5*time.Second, 10*time.Millisecond)
	require.NoError(t, db.First(execution, "id = ?", execution.ID).Error)
	assert.Equal(t, models.CanvasNodeExecutionStateFinished, execution.State)
	events, err := models.FindCanvasEventsForExecutions(db, []string{execution.ID.String()})
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "failed", events[0].Channel)
	eventPayload, err := events[0].Data.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(eventPayload), models.ErrCodexConnectionStale.Error())
	storedRunner, err := models.FindRunner(db, runner.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerStateIdle, storedRunner.State)
}

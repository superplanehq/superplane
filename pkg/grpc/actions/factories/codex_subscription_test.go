package factories

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/jwt"
	runnerpayload "github.com/superplanehq/superplane/pkg/runners/api"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestPersonalCodexConnectionDoesNotActivateWorkspace(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureCodexSubscriptions))
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	ctx := onboardingContext(r)
	before, err := DescribeFactoryCodexConnection(ctx, r.Organization.ID.String(), &pb.DescribeFactoryCodexConnectionRequest{})
	require.NoError(t, err)
	assert.True(t, before.Connection.Eligible)
	assert.Equal(t, "disconnected", before.Connection.Status)
	fleets := models.DefaultInstallationRunnerFleets(models.DefaultRunnerVersion)
	require.NoError(t, fleets[0].Create(db))
	t.Setenv("JWT_SECRET", "personal-codex-test")
	t.Setenv("RUNNER_API_BASE_URL", "https://superplane.example")
	started, err := StartFactoryCodexConnection(ctx, r.Encryptor, r.Organization.ID.String(), &pb.StartFactoryCodexConnectionRequest{})
	require.NoError(t, err)
	assert.Equal(t, "pending", started.Connection.Status)
	connection, err := models.FindCodexConnection(db, r.Organization.ID, r.User)
	require.NoError(t, err)
	lease, err := models.FindCodexLoginLease(db, connection.ID)
	require.NoError(t, err)
	require.NotNil(t, lease)
	assert.Nil(t, lease.FactoryID)
	view, err := DescribeFactoryCodexConnection(ctx, r.Organization.ID.String(), &pb.DescribeFactoryCodexConnectionRequest{FactoryId: factory.ID.String()})
	require.NoError(t, err)
	assert.Equal(t, started.Connection.Id, view.Connection.Id)
	saved, err := models.FindFactory(db, r.Organization.ID, factory.ID)
	require.NoError(t, err)
	assert.False(t, saved.OnboardingConfigValue().CodexSubscription)
	assert.Empty(t, saved.OnboardingConfigValue().CodexDefaultOwnerID)
}

func TestCancelFactoryCodexConnectionReportsCancellationUntilRunnerDeletion(t *testing.T) {
	for _, test := range []struct {
		status          string
		cancelingStatus string
		finalStatus     string
	}{
		{models.CodexConnectionStatusPending, "canceling", models.CodexConnectionStatusDisconnected},
		{models.CodexConnectionStatusFinishing, models.CodexConnectionStatusFinishing, models.CodexConnectionStatusReady},
	} {
		t.Run(test.status, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureCodexSubscriptions))
			factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
			require.NoError(t, err)
			fleet := models.DefaultInstallationRunnerFleets("test")[0]
			require.NoError(t, fleet.Create(db))
			now := time.Now()
			runner := &models.Runner{
				ID: uuid.New(), FleetID: fleet.ID, State: models.RunnerStateIdle,
				RunnerVersion: "test", Ephemeral: true, CreatedAt: now, UpdatedAt: now,
			}
			require.NoError(t, db.Create(runner).Error)
			task := &models.RunnerTask{
				ID: uuid.New(), OrganizationID: r.Organization.ID, FleetID: fleet.ID,
				Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateQueued,
				PayloadCiphertext: []byte("login"), QueuedAt: now, CreatedAt: now, UpdatedAt: now,
			}
			require.NoError(t, db.Create(task).Error)
			connection, err := models.CreateCodexConnection(db, r.Organization.ID, r.User, now)
			require.NoError(t, err)
			require.NoError(t, connection.BeginLogin(db, task.ID, now))
			require.NoError(t, connection.RecordDeviceCode(db, task.ID, connection.ReservationGeneration, "https://auth.openai.com/codex/device", "TEST-CODE", now))
			require.NoError(t, task.Reserve(db, runner.ID))
			require.NoError(t, task.Start(db, runner, "test", now))
			if test.status == models.CodexConnectionStatusFinishing {
				require.NoError(t, connection.StoreBundle(db, task.ID, connection.ReservationGeneration, 1, []byte("saved-bundle"), "member@example.com", true, now))
			}

			ctx := authentication.SetUserIdInMetadata(t.Context(), r.User.String())
			before, err := DescribeFactoryCodexConnection(ctx, r.Organization.ID.String(), &pb.DescribeFactoryCodexConnectionRequest{FactoryId: factory.ID.String()})
			require.NoError(t, err)
			assert.Equal(t, test.status, before.Connection.Status)
			if test.status == models.CodexConnectionStatusPending {
				assert.Equal(t, "TEST-CODE", before.Connection.VerificationCode)
			}

			canceled, err := CancelFactoryCodexConnection(ctx, crypto.NewNoOpEncryptor(), r.Organization.ID.String(), &pb.CancelFactoryCodexConnectionRequest{FactoryId: factory.ID.String()})
			require.NoError(t, err)
			assert.Equal(t, test.cancelingStatus, canceled.Connection.Status)
			assert.Empty(t, canceled.Connection.VerificationUrl)
			assert.Empty(t, canceled.Connection.VerificationCode)
			assert.True(t, canceled.Connection.ReservationHeld)
			assert.Equal(t, task.ID.String(), canceled.Connection.ActiveTaskId)

			refreshed, err := DescribeFactoryCodexConnection(ctx, r.Organization.ID.String(), &pb.DescribeFactoryCodexConnectionRequest{FactoryId: factory.ID.String()})
			require.NoError(t, err)
			assert.Equal(t, test.cancelingStatus, refreshed.Connection.Status)
			assert.Empty(t, refreshed.Connection.VerificationCode)
			stored, err := models.FindCodexConnection(db, r.Organization.ID, r.User)
			require.NoError(t, err)
			assert.Equal(t, test.status, stored.Status)

			completed, err := task.Complete(db, runner, "canceled", datatypes.JSON([]byte(`{}`)), 1, "", true, time.Now())
			require.NoError(t, err)
			require.True(t, completed)
			stopped, err := DescribeFactoryCodexConnection(ctx, r.Organization.ID.String(), &pb.DescribeFactoryCodexConnectionRequest{FactoryId: factory.ID.String()})
			require.NoError(t, err)
			assert.Equal(t, test.cancelingStatus, stopped.Connection.Status)
			assert.True(t, stopped.Connection.ReservationHeld)

			require.NoError(t, runner.MarkProviderDeleted(db, time.Now()))
			finished, err := DescribeFactoryCodexConnection(ctx, r.Organization.ID.String(), &pb.DescribeFactoryCodexConnectionRequest{FactoryId: factory.ID.String()})
			require.NoError(t, err)
			assert.Equal(t, test.finalStatus, finished.Connection.Status)
			assert.False(t, finished.Connection.ReservationHeld)
			assert.Empty(t, finished.Connection.ActiveTaskId)
		})
	}
}

func TestCatalogRefreshCoalescesAndCancellationTargetsOnlyItsTask(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureCodexSubscriptions))
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	fleet := models.DefaultInstallationRunnerFleets("test")[0]
	require.NoError(t, fleet.Create(db))
	now := time.Now()
	connection, err := models.CreateCodexConnection(db, r.Organization.ID, r.User, now)
	require.NoError(t, err)
	bundle := []byte(`{"access_token":"access-token","refresh_token":"provider-refresh","id_token":"id-token"}`)
	require.NoError(t, db.Model(connection).Updates(map[string]any{
		"status": models.CodexConnectionStatusReady, "credential_version": 1, "bundle_ciphertext": bundle,
	}).Error)
	t.Setenv("JWT_SECRET", "catalog-task-test")
	t.Setenv("RUNNER_API_BASE_URL", "https://superplane.example")
	ctx := onboardingContext(r)
	start := func() (*pb.StartFactoryCodexConnectionResponse, error) {
		return StartFactoryCodexConnection(ctx, r.Encryptor, r.Organization.ID.String(), &pb.StartFactoryCodexConnectionRequest{RefreshCatalog: true})
	}
	type startResult struct {
		response *pb.StartFactoryCodexConnectionResponse
		err error
	}
	results := make(chan startResult, 2)
	ready := make(chan struct{})
	for range 2 {
		go func() { <-ready; response, err := start(); results <- startResult{response, err} }()
	}
	close(ready)
	var taskID string
	for range 2 {
		result := <-results
		require.NoError(t, result.err)
		view := result.response.Connection
		assert.Equal(t, models.CodexConnectionStatusReady, view.Status)
		assert.False(t, view.ReservationHeld)
		assert.Empty(t, view.ActiveTaskId)
		require.NotEmpty(t, view.CatalogRefreshTaskId)
		if taskID == "" { taskID = view.CatalogRefreshTaskId }
		assert.Equal(t, taskID, view.CatalogRefreshTaskId)
	}
	var taskCount int64
	require.NoError(t, db.Model(&models.RunnerTask{}).Count(&taskCount).Error)
	assert.Equal(t, int64(1), taskCount)
	task, err := models.FindRunnerTask(db, uuid.MustParse(taskID))
	require.NoError(t, err)
	var payload runnerpayload.TaskPayload
	require.NoError(t, json.Unmarshal(task.PayloadCiphertext, &payload))
	environment := map[string]string{}
	for _, variable := range payload.Environment { environment[variable.Name] = variable.Value }
	assert.Equal(t, "https://superplane.example"+runner.CodexSubscriptionRefreshPath, environment[runner.EnvCodexRefreshTokenURL])
	scope, err := runner.ParseCodexSubscriptionToken(jwt.NewSigner("catalog-task-test"), environment[runner.EnvCodexSubscriptionToken])
	require.NoError(t, err)
	lease, err := models.FindCodexLeaseByTask(db, task.ID)
	require.NoError(t, err)
	require.NotNil(t, lease)
	assert.Equal(t, models.CodexLeasePurposeCatalog, lease.Purpose)
	assert.Equal(t, lease.Generation, scope.Generation)
	assert.Equal(t, uuid.Nil, scope.FactoryID)
	require.NotNil(t, payload.ExecutionTimeoutSeconds)
	assert.Equal(t, int(models.CodexCatalogTimeout/time.Second), *payload.ExecutionTimeoutSeconds)
	worker := &models.Runner{ID: uuid.New(), FleetID: fleet.ID, State: models.RunnerStateIdle, RunnerVersion: "test", Ephemeral: true, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, db.Create(worker).Error)
	require.NoError(t, task.Reserve(db, worker.ID))
	require.NoError(t, task.Start(db, worker, "test", time.Now()))
	runTask := &models.RunnerTask{ID: uuid.New(), OrganizationID: r.Organization.ID, FleetID: fleet.ID, Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateQueued, PayloadCiphertext: []byte("run"), QueuedAt: now}
	require.NoError(t, db.Create(runTask).Error)
	_, err = models.ReserveFactoryCodexConnection(db, r.Organization.ID, factory.ID, support.CreateCodexOwnedRun(t, db, connection, runTask.ID), runTask.ID, now, r.Encryptor)
	require.NoError(t, err)
	cancel := func(target string) *pb.FactoryCodexConnection {
		result, err := CancelFactoryCodexConnection(ctx, r.Encryptor, r.Organization.ID.String(), &pb.CancelFactoryCodexConnectionRequest{CatalogRefreshTaskId: target})
		require.NoError(t, err)
		return result.Connection
	}
	for _, target := range []string{runTask.ID.String(), uuid.NewString()} {
		view := cancel(target)
		assert.Equal(t, taskID, view.CatalogRefreshTaskId)
		assert.False(t, view.CatalogRefreshCanceling)
	}
	for range 2 {
		view := cancel(taskID)
		assert.Equal(t, taskID, view.CatalogRefreshTaskId)
		assert.True(t, view.CatalogRefreshCanceling)
		assert.Equal(t, models.CodexConnectionStatusReady, view.Status)
		assert.False(t, view.ReservationHeld)
		assert.Empty(t, view.ActiveTaskId)
	}
	duplicate, err := start()
	require.NoError(t, err)
	assert.Equal(t, taskID, duplicate.Connection.CatalogRefreshTaskId)
	assert.True(t, duplicate.Connection.CatalogRefreshCanceling)
	storedLease, err := models.FindCodexLeaseByTask(db, task.ID)
	require.NoError(t, err)
	require.NotNil(t, storedLease.RevokedAt)
	stillRunning, err := models.FindCodexLeaseByTask(db, runTask.ID)
	require.NoError(t, err)
	require.NotNil(t, stillRunning)
	assert.Nil(t, stillRunning.RevokedAt)
	storedRun, err := models.FindRunnerTask(db, runTask.ID)
	require.NoError(t, err)
	assert.Nil(t, storedRun.CancelRequestedAt)
	assert.Equal(t, models.RunnerTaskStateQueued, storedRun.State)
	completed, err := task.Complete(db, worker, "canceled", datatypes.JSON([]byte(`{}`)), 1, "", true, time.Now())
	require.NoError(t, err)
	require.True(t, completed)
	described, err := DescribeFactoryCodexConnection(ctx, r.Organization.ID.String(), &pb.DescribeFactoryCodexConnectionRequest{})
	require.NoError(t, err)
	assert.Empty(t, described.Connection.CatalogRefreshTaskId, "terminal catalog cleanup must not wait for provider VM deletion")
	assert.Equal(t, models.CodexConnectionStatusReady, described.Connection.Status)
	require.NoError(t, db.First(connection, "id = ?", connection.ID).Error)
	assert.Equal(t, bundle, []byte(connection.BundleCiphertext))
	next, err := start()
	require.NoError(t, err)
	require.NotEqual(t, taskID, next.Connection.CatalogRefreshTaskId)
	stale := cancel(taskID)
	assert.Equal(t, next.Connection.CatalogRefreshTaskId, stale.CatalogRefreshTaskId)
	assert.False(t, stale.CatalogRefreshCanceling)
}

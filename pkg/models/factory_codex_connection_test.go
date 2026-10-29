package models_test

import (
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestCodexConnectionStaysReservedUntilTheLoginVMIsDeleted(t *testing.T) {
	db, connection, task, runner := newReservedCodexLogin(t)
	now := time.Now()
	require.NoError(t, clearProviderDeletion(db, runner))
	require.NoError(t, connection.StoreBundle(db, task.ID, connection.ReservationGeneration, connection.CredentialVersion+1, []byte("bundle-2"), "ada@example.com", true, now))
	require.NoError(t, markRunnerTask(db, task, models.RunnerTaskStateSucceeded, &runner.ID, now))

	require.NoError(t, models.ReleaseConfirmedCodexReservations(db, now))
	held := reloadCodexConnection(t, db, connection.ID)
	heldLeases := codexLeases(t, db, connection.ID)
	require.Len(t, heldLeases, 1)
	assert.Equal(t, task.ID, heldLeases[0].TaskID)
	assert.Equal(t, models.CodexLeasePurposeLogin, heldLeases[0].Purpose)
	assert.Equal(t, models.CodexConnectionStatusFinishing, held.Status)

	require.NoError(t, runner.MarkProviderDeleted(db, now))
	released := reloadCodexConnection(t, db, connection.ID)
	assert.Empty(t, codexLeases(t, db, connection.ID))
	assert.Equal(t, models.CodexConnectionStatusReady, released.Status)
	assert.NotEqual(t, held.ReservationGeneration, released.ReservationGeneration)
}

func TestCodexRunLeaseReleasesWhenTheTaskIsLost(t *testing.T) {
	db, connection, task, runner := newReadyCodexConnection(t)
	now := time.Now()
	_, err := reserveCodexRun(t, db, connection, task.ID, now)
	require.NoError(t, err)
	require.NoError(t, clearProviderDeletion(db, runner))
	require.NoError(t, markRunnerTask(db, task, models.RunnerTaskStateLost, &runner.ID, now))
	require.NoError(t, models.MarkCodexConnectionUncertain(db, task.ID, now))

	held := reloadCodexConnection(t, db, connection.ID)
	assert.Equal(t, models.CodexConnectionStatusReady, held.Status)
	require.Len(t, codexLeases(t, db, connection.ID), 1)

	require.NoError(t, models.ReleaseConfirmedCodexReservations(db, now))
	released := reloadCodexConnection(t, db, connection.ID)
	assert.Empty(t, codexLeases(t, db, connection.ID))
	assert.Equal(t, models.CodexConnectionStatusReady, released.Status)
	assert.Equal(t, []byte("bundle"), []byte(released.BundleCiphertext))
}

func TestCodexLoginLostRunnerStaysReservedUntilVMDeletion(t *testing.T) {
	db, connection, task, runner := newReservedCodexLogin(t)
	now := time.Now()
	require.NoError(t, clearProviderDeletion(db, runner))
	require.NoError(t, markRunnerTask(db, task, models.RunnerTaskStateLost, &runner.ID, now))
	require.NoError(t, models.MarkCodexConnectionUncertain(db, task.ID, now))

	require.NoError(t, models.ReleaseConfirmedCodexReservations(db, now))
	held := reloadCodexConnection(t, db, connection.ID)
	require.Len(t, codexLeases(t, db, connection.ID), 1)
	assert.Equal(t, models.CodexConnectionStatusReconnectRequired, held.Status)

	require.NoError(t, runner.MarkProviderDeleted(db, now))
	released := reloadCodexConnection(t, db, connection.ID)
	assert.Empty(t, codexLeases(t, db, connection.ID))
	assert.Equal(t, models.CodexConnectionStatusReconnectRequired, released.Status)
}

func TestCodexConnectionRejectsStaleCheckpoint(t *testing.T) {
	db, connection, task, _ := newReservedCodexLogin(t)
	now := time.Now()
	err := connection.StoreBundle(db, task.ID, connection.ReservationGeneration, connection.CredentialVersion+2, []byte("bundle"), "ada@example.com", false, now)
	require.ErrorIs(t, err, models.ErrCodexConnectionStale)

	require.NoError(t, connection.StoreBundle(db, task.ID, connection.ReservationGeneration, connection.CredentialVersion+1, []byte("bundle-1"), "ada@example.com", false, now))
	err = connection.StoreBundle(db, task.ID, connection.ReservationGeneration, connection.CredentialVersion, []byte("bundle-old"), "ada@example.com", true, now)
	require.ErrorIs(t, err, models.ErrCodexConnectionStale)
}

func TestCodexConnectionAllowsParallelRunLeases(t *testing.T) {
	db, connection, task, _ := newReadyCodexConnection(t)
	now := time.Now()
	first, err := reserveCodexRun(t, db, connection, task.ID, now)
	require.NoError(t, err)
	second, err := reserveCodexRun(t, db, connection, uuid.New(), now)
	require.NoError(t, err)
	_, err = reserveCodexRun(t, db, connection, uuid.New(), now)
	require.NoError(t, err)

	leases := codexLeases(t, db, connection.ID)
	require.Len(t, leases, 3)
	assert.NotEqual(t, first.ReservationGeneration, second.ReservationGeneration)
	for _, lease := range leases {
		assert.Equal(t, models.CodexLeasePurposeRun, lease.Purpose)
		assert.NotEmpty(t, lease.SecretHash)
	}
}

func TestCodexConnectionsArePersonalWithinAWorkspace(t *testing.T) {
	db, first, _, _ := newReadyCodexConnection(t)
	secondUser := &models.User{ID: uuid.New(), OrganizationID: first.OrganizationID, Type: models.UserTypeHuman, Name: "Second member"}
	require.NoError(t, db.Create(secondUser).Error)
	second, err := models.CreateFactoryCodexConnection(db, first.OrganizationID, first.FactoryID, secondUser.ID, time.Now())
	require.NoError(t, err)
	require.NoError(t, db.Model(second).Updates(map[string]any{"status": models.CodexConnectionStatusReady, "credential_version": 1, "bundle_ciphertext": []byte("second-bundle")}).Error)
	firstRun := support.CreateCodexOwnedRun(t, db, first)
	secondRun := support.CreateCodexOwnedRun(t, db, second)
	for _, route := range []struct{ run, connection uuid.UUID }{{firstRun, first.ID}, {secondRun, second.ID}, {firstRun, first.ID}} {
		reserved, err := models.ReserveFactoryCodexConnection(db, first.OrganizationID, first.FactoryID, route.run, uuid.New(), time.Now(), crypto.NewNoOpEncryptor())
		require.NoError(t, err)
		assert.Equal(t, route.connection, reserved.ID)
	}
	assert.Len(t, codexLeases(t, db, first.ID), 2)
	assert.Len(t, codexLeases(t, db, second.ID), 1)
	_, err = models.ReserveFactoryCodexConnection(db, first.OrganizationID, first.FactoryID, uuid.New(), uuid.New(), time.Now(), crypto.NewNoOpEncryptor())
	require.ErrorIs(t, err, models.ErrSubscriptionTaskAssociation)
	assert.NotEqual(t, first.ID, second.ID)
	assert.Equal(t, []byte("bundle"), reloadCodexConnection(t, db, first.ID).BundleCiphertext)
}

func TestSubscriptionReservationUsesStoredTaskAssociations(t *testing.T) {
	db, connection, _, _ := newReadyCodexConnection(t)
	f, err := models.FindFactory(db, connection.OrganizationID, connection.FactoryID)
	require.NoError(t, err)
	subscription := true
	require.NoError(t, f.UpdateOnboarding(db, models.FactoryOnboardingPatch{CodexSubscription: &subscription}))
	order, err := f.CreateWorkOrder(db, "Route every task activity", "", &connection.OwnerUserID, []uuid.UUID{connection.OwnerUserID}, nil)
	require.NoError(t, err)
	canvas, _ := support.CreateCanvas(t, f.OrganizationID, connection.OwnerUserID, []models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger, Metadata: datatypes.NewJSONType(models.FactoryAppTemplateMetadata("risk-score", 1))}}, nil)
	require.NoError(t, db.Model(canvas).Update("factory_id", f.ID).Error)
	canvas.FactoryID = &f.ID
	newRun := func() *models.CanvasRun {
		t.Helper()
		run, err := models.CreateCanvasRunInTransaction(db, canvas.ID, "trigger", models.CanvasRunStateStarted, "")
		require.NoError(t, err)
		return run
	}
	line, err := f.CreateLine(db, "ship", nil)
	require.NoError(t, err)
	dispatch := support.CreateFactoryLineDispatch(t, f.OrganizationID, f.ID, order.ID, line.ID, line.Name, nil)
	lineRun := newRun()
	require.NoError(t, db.Create(&models.FactoryWorkOrderExecution{ID: uuid.New(), OrganizationID: f.OrganizationID, FactoryID: f.ID, WorkOrderID: order.ID, LineID: line.ID, LineDispatchID: dispatch.ID, StepName: "Implement", RunID: &lineRun.ID, Status: models.FactoryWorkOrderExecutionStatusRunning}).Error)
	refineRun := newRun()
	require.NoError(t, db.Create(&models.FactoryPlanningSession{ID: uuid.New(), OrganizationID: f.OrganizationID, FactoryID: f.ID, CreatedByUserID: &connection.OwnerUserID, Kind: models.PlanningSessionKindWorkOrderAnalysis, State: models.PlanningSessionStateRunning, CanvasID: &canvas.ID, CanvasRunID: &refineRun.ID, DraftWorkOrderID: &order.ID}).Error)
	pr, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{Provider: models.FactoryPullRequestProviderGitHub, Repository: "acme/app", Number: 42, URL: "https://github.com/acme/app/pull/42", State: models.FactoryPullRequestStateOpen})
	require.NoError(t, err)
	feedback := newRun()
	_, err = pr.CreateActivity(db, models.FactoryPullRequestActivityParams{RunID: feedback.ID, Title: "Address feedback", Access: models.FactoryPullRequestAccessConcurrent})
	require.NoError(t, err)
	merge := newRun()
	// The stored PR selects the task even when the event names a different task.
	event := &models.CanvasEvent{WorkflowID: canvas.ID, Data: models.NewJSONValue(map[string]any{"data": map[string]any{"pullRequest": map[string]any{"id": pr.ID.String()}, "workOrder": map[string]any{"id": uuid.NewString()}}})}
	require.NoError(t, models.MaybeLinkMergeConfidenceRun(db, canvas, event, merge))
	require.NoError(t, models.MaybeLinkMergeConfidenceRun(db, canvas, event, merge))
	child := newRun()
	require.NoError(t, db.Model(child).Update("parent_run_id", merge.ID).Error)
	for _, run := range []*models.CanvasRun{lineRun, refineRun, feedback, merge, child} {
		reserved, err := models.ReserveFactoryCodexConnection(db, f.OrganizationID, f.ID, run.ID, uuid.New(), time.Now(), crypto.NewNoOpEncryptor())
		require.NoError(t, err)
		assert.Equal(t, connection.ID, reserved.ID)
	}
	runs, err := order.ListRuns(db)
	require.NoError(t, err)
	assert.Len(t, runs, 5, "cancellation and owner restrictions must use the routing associations")
	untrusted := newRun()
	require.NoError(t, db.Model(untrusted).Update("input", models.NewJSONValue(map[string]any{"work_order": map[string]any{"id": order.ID.String()}})).Error)
	_, err = models.ReserveFactoryCodexConnection(db, f.OrganizationID, f.ID, untrusted.ID, uuid.New(), time.Now(), crypto.NewNoOpEncryptor())
	require.ErrorIs(t, err, models.ErrSubscriptionTaskAssociation)
	other, err := f.CreateWorkOrder(db, "Conflicting task", "", &connection.OwnerUserID, []uuid.UUID{connection.OwnerUserID}, nil)
	require.NoError(t, err)
	require.NoError(t, db.Create(&models.FactoryPlanningSession{ID: uuid.New(), OrganizationID: f.OrganizationID, FactoryID: f.ID, CreatedByUserID: &connection.OwnerUserID, Kind: models.PlanningSessionKindWorkOrderAnalysis, State: models.PlanningSessionStateRunning, CanvasID: &canvas.ID, CanvasRunID: &child.ID, DraftWorkOrderID: &other.ID}).Error)
	_, err = models.ReserveFactoryCodexConnection(db, f.OrganizationID, f.ID, child.ID, uuid.New(), time.Now(), crypto.NewNoOpEncryptor())
	require.ErrorIs(t, err, models.ErrSubscriptionTaskAssociation)
}

func TestCodexRunRejectsAReplacementMember(t *testing.T) {
	db, connection, task, _ := newReadyCodexConnection(t)
	// The connection remains after its owner leaves the organization.
	require.NoError(t, db.Delete(&models.User{}, "id = ?", connection.OwnerUserID).Error)
	replacement := &models.User{ID: uuid.New(), OrganizationID: connection.OrganizationID, Type: models.UserTypeHuman, Name: "Replacement"}
	require.NoError(t, db.Create(replacement).Error)
	_, err := reserveCodexRun(t, db, connection, task.ID, time.Now())
	require.ErrorIs(t, err, models.ErrSubscriptionOwnerInvalid)
	assert.Empty(t, codexLeases(t, db, connection.ID))
}

func TestCodexCredentialsParticipateInKeyMigration(t *testing.T) {
	db, connection, task, _ := newReadyCodexConnection(t)
	_, err := reserveCodexRun(t, db, connection, task.ID, time.Now())
	require.NoError(t, err)
	lease := codexLeases(t, db, connection.ID)[0]
	for _, store := range []struct {
		name, id string
		value    []byte
	}{
		{"factory_codex_connections.bundle_ciphertext", connection.ID.String(), connection.BundleCiphertext},
		{"factory_codex_leases.secret_ciphertext", lease.ID.String(), lease.SecretCiphertext},
	} {
		t.Run(store.name, func(t *testing.T) {
			var location *models.EncryptedStorageLocation
			for _, candidate := range models.ApplicationEncryptedStorage() {
				if candidate.Name == store.name {
					location = &candidate
					break
				}
			}
			require.NotNil(t, location)
			rows, err := location.ReadPage(db, "", 1)
			require.NoError(t, err)
			require.Len(t, rows, 1)
			assert.Equal(t, store.id, rows[0].Cursor)
			assert.Equal(t, connection.ID.String(), rows[0].AssociatedData)
			assert.Equal(t, store.value, rows[0].Value)
			replacement := []byte("rotated-ciphertext")
			changed, err := location.Replace(db, rows[0], replacement)
			require.NoError(t, err)
			assert.True(t, changed)
			changed, err = location.Replace(db, rows[0], []byte("stale-rotation"))
			require.NoError(t, err)
			assert.False(t, changed)
			rotated, err := location.ReadPage(db, "", 1)
			require.NoError(t, err)
			require.Len(t, rotated, 1)
			assert.Equal(t, replacement, rotated[0].Value)
			remaining, err := location.ReadPage(db, store.id, 1)
			require.NoError(t, err)
			assert.Empty(t, remaining)
		})
	}
}

func TestCodexLoginCleanupSkipsATaskLockedByCompletion(t *testing.T) {
	db, connection, task, _ := newDisconnectedCodexLogin(t)
	completion := db.Begin()
	require.NoError(t, completion.Error)
	t.Cleanup(func() { completion.Rollback() })
	require.NoError(t, completion.Exec("SELECT id FROM runner_tasks WHERE id = ? FOR UPDATE", task.ID).Error)
	cleanup := db.Begin()
	require.NoError(t, cleanup.Error)
	t.Cleanup(func() { cleanup.Rollback() })
	require.NoError(t, cleanup.Exec("SET LOCAL lock_timeout = '250ms'").Error)
	require.NoError(t, models.ReleaseConfirmedCodexReservations(cleanup, time.Now().Add(16*time.Minute)))
	// Cleanup must not hold the connection while completion holds the task.
	require.NoError(t, completion.Exec("SET LOCAL lock_timeout = '250ms'").Error)
	_, err := models.LockFactoryCodexConnection(completion, connection.OrganizationID, connection.FactoryID, connection.OwnerUserID)
	require.NoError(t, err)
}

func TestCodexConnectionLoginAndRunsExcludeEachOther(t *testing.T) {
	db, connection, task, _ := newReadyCodexConnection(t)
	now := time.Now()
	_, err := reserveCodexRun(t, db, connection, task.ID, now)
	require.NoError(t, err)
	err = connection.BeginLogin(db, uuid.New(), now)
	var busy *models.CodexConnectionBusyError
	require.ErrorAs(t, err, &busy)
	assert.Equal(t, task.ID, busy.TaskID)

	require.NoError(t, models.ReleaseCodexReservationForTask(db, task.ID, now))
	require.NoError(t, connection.BeginLogin(db, uuid.New(), now))
	_, err = reserveCodexRun(t, db, reloadCodexConnection(t, db, connection.ID), uuid.New(), now)
	require.ErrorAs(t, err, &busy)
}

func TestCodexRunLeaseCannotReplaceTheRefreshToken(t *testing.T) {
	db, connection, task, runner := newReadyCodexConnection(t)
	now := time.Now()
	reserved, err := reserveCodexRun(t, db, connection, task.ID, now)
	require.NoError(t, err)
	err = reserved.StoreBundle(db, task.ID, reserved.ReservationGeneration, reserved.CredentialVersion+1, []byte("rotated"), "ada@example.com", true, now)
	require.ErrorIs(t, err, models.ErrCodexConnectionStale)
	require.NoError(t, markRunnerTask(db, task, models.RunnerTaskStateSucceeded, &runner.ID, now))
	require.NoError(t, models.ReleaseConfirmedCodexReservations(db, now))

	released := reloadCodexConnection(t, db, connection.ID)
	assert.Empty(t, codexLeases(t, db, connection.ID))
	assert.Equal(t, models.CodexConnectionStatusReady, released.Status)
	assert.Equal(t, []byte("bundle"), []byte(released.BundleCiphertext))
}

func TestCodexLoginSuccessWithoutFinalCheckpointRequiresReconnect(t *testing.T) {
	db, connection, task, runner := newReservedCodexLogin(t)
	now := time.Now()
	require.NoError(t, clearProviderDeletion(db, runner))
	require.NoError(t, markRunnerTask(db, task, models.RunnerTaskStateSucceeded, &runner.ID, now))
	require.NoError(t, runner.MarkProviderDeleted(db, now))

	released := reloadCodexConnection(t, db, connection.ID)
	assert.Empty(t, codexLeases(t, db, connection.ID))
	assert.Equal(t, models.CodexConnectionStatusReconnectRequired, released.Status)
}

func TestCodexCatalogRefreshFailureBeforeCheckpointKeepsConnectionReady(t *testing.T) {
	for _, state := range []string{models.RunnerTaskStateFailed, models.RunnerTaskStateCanceled} {
		t.Run(state, func(t *testing.T) {
			db, connection, previousTask, _ := newReadyCodexConnection(t)
			now := time.Now()
			task := &models.RunnerTask{
				ID: uuid.New(), OrganizationID: connection.OrganizationID, FleetID: previousTask.FleetID,
				Backend: models.RunnerTaskBackendIntegrated, State: state,
				PayloadCiphertext: []byte("ciphertext"), QueuedAt: now,
				CreatedAt: now, UpdatedAt: now, FinishedAt: &now,
			}
			require.NoError(t, db.Create(task).Error)
			require.NoError(t, connection.BeginCatalogRefresh(db, task.ID, now))
			require.NoError(t, models.ReleaseConfirmedCodexReservations(db, now))
			released := reloadCodexConnection(t, db, connection.ID)
			assert.Equal(t, models.CodexConnectionStatusReady, released.Status)
			assert.Equal(t, []byte("bundle"), []byte(released.BundleCiphertext))
			assert.Empty(t, codexLeases(t, db, connection.ID))
			_, err := reserveCodexRun(t, db, released, uuid.New(), now)
			require.NoError(t, err)
		})
	}
}

func TestCodexConnectionCancelBeforeCheckpointRestoresPreviousStatus(t *testing.T) {
	db, connection, task, _ := newDisconnectedCodexLogin(t)
	now := time.Now()
	require.NoError(t, markRunnerTask(db, task, models.RunnerTaskStateCanceled, nil, now))
	require.NoError(t, models.ReleaseConfirmedCodexReservations(db, now))

	released := reloadCodexConnection(t, db, connection.ID)
	assert.Empty(t, codexLeases(t, db, connection.ID))
	assert.Equal(t, models.CodexConnectionStatusDisconnected, released.Status)
}

func TestCodexLoginRecoversFinalCheckpointAfterTaskFailure(t *testing.T) {
	for _, state := range []string{models.RunnerTaskStateFailed, models.RunnerTaskStateLost} {
		t.Run(state, func(t *testing.T) {
			db, connection, task, runner := newReservedCodexLogin(t)
			now := time.Now()
			require.NoError(t, clearProviderDeletion(db, runner))
			require.NoError(t, connection.StoreBundle(db, task.ID, connection.ReservationGeneration, connection.CredentialVersion+1, []byte("saved-login"), "ada@example.com", true, now))
			require.NoError(t, markRunnerTask(db, task, state, &runner.ID, now))
			require.NoError(t, runner.MarkProviderDeleted(db, now))

			recovered := reloadCodexConnection(t, db, connection.ID)
			assert.Equal(t, models.CodexConnectionStatusReady, recovered.Status)
			assert.Equal(t, []byte("saved-login"), []byte(recovered.BundleCiphertext))
			assert.Empty(t, codexLeases(t, db, connection.ID))
		})
	}
}

func TestCodexLoginExpiresWhileWaitingForARunner(t *testing.T) {
	db, connection, task, _ := newDisconnectedCodexLogin(t)
	now := time.Now().Add(16 * time.Minute)
	require.NoError(t, models.ReleaseConfirmedCodexReservations(db, now))

	expired, err := models.FindRunnerTask(db, task.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerTaskStateCanceled, expired.State)
	assert.Equal(t, models.CodexConnectionStatusDisconnected, reloadCodexConnection(t, db, connection.ID).Status)
	assert.Empty(t, codexLeases(t, db, connection.ID))
}

func TestCodexLoginRequiresReconnectAfterANewerUnfinishedCheckpoint(t *testing.T) {
	for _, state := range []string{models.RunnerTaskStateFailed, models.RunnerTaskStateCanceled} {
		t.Run(state, func(t *testing.T) {
			db, connection, task, runner := newReservedCodexLogin(t)
			now := time.Now()
			require.NoError(t, clearProviderDeletion(db, runner))
			require.NoError(t, connection.StoreBundle(db, task.ID, connection.ReservationGeneration, connection.CredentialVersion+1, []byte("final"), "ada@example.com", true, now))
			require.NoError(t, connection.StoreBundle(db, task.ID, connection.ReservationGeneration, connection.CredentialVersion+1, []byte("unfinished"), "ada@example.com", false, now))
			require.NoError(t, markRunnerTask(db, task, state, &runner.ID, now))
			require.NoError(t, runner.MarkProviderDeleted(db, now))

			assert.Equal(t, models.CodexConnectionStatusReconnectRequired, reloadCodexConnection(t, db, connection.ID).Status)
		})
	}
}

func reserveCodexRun(t *testing.T, db *gorm.DB, connection *models.FactoryCodexConnection, taskID uuid.UUID, now time.Time) (*models.FactoryCodexConnection, error) {
	runID := support.CreateCodexOwnedRun(t, db, connection)
	return models.ReserveFactoryCodexConnection(db, connection.OrganizationID, connection.FactoryID, runID, taskID, now, crypto.NewNoOpEncryptor())
}

func codexLeases(t *testing.T, db *gorm.DB, connectionID uuid.UUID) []models.FactoryCodexLease {
	t.Helper()
	var leases []models.FactoryCodexLease
	require.NoError(t, db.Where("connection_id = ?", connectionID).Order("created_at, id").Find(&leases).Error)
	return leases
}

func clearProviderDeletion(db *gorm.DB, runner *models.Runner) error {
	runner.ProviderDeletedAt = nil
	return db.Model(runner).Update("provider_deleted_at", nil).Error
}

func newDisconnectedCodexLogin(t *testing.T) (*gorm.DB, *models.FactoryCodexConnection, *models.RunnerTask, *models.Runner) {
	t.Helper()
	require.NoError(t, database.TruncateTables())
	resource := support.Setup(t)
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(resource.Organization.ID, features.FeatureCodexSubscriptions))
	factory, err := models.CreateFactory(db, resource.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	fleet := newTestRunnerFleet()
	require.NoError(t, fleet.Create(db))
	now := time.Now()
	runner := &models.Runner{
		ID:            uuid.New(),
		FleetID:       fleet.ID,
		State:         models.RunnerStateTerminated,
		RunnerVersion: "0.1.0",
		TerminatedAt:  &now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	require.NoError(t, db.Create(runner).Error)
	task := &models.RunnerTask{
		ID:                uuid.New(),
		OrganizationID:    resource.Organization.ID,
		FleetID:           fleet.ID,
		Backend:           models.RunnerTaskBackendIntegrated,
		State:             models.RunnerTaskStateQueued,
		PayloadCiphertext: []byte("ciphertext"),
		QueuedAt:          now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	require.NoError(t, db.Create(task).Error)
	connection, err := models.CreateFactoryCodexConnection(db, resource.Organization.ID, factory.ID, resource.User, now)
	require.NoError(t, err)
	require.NoError(t, connection.BeginLogin(db, task.ID, now))
	return db, reloadCodexConnection(t, db, connection.ID), task, runner
}

func newReservedCodexLogin(t *testing.T) (*gorm.DB, *models.FactoryCodexConnection, *models.RunnerTask, *models.Runner) {
	t.Helper()
	db, connection, task, runner := newReadyCodexConnection(t)
	require.NoError(t, connection.BeginLogin(db, task.ID, time.Now()))
	return db, reloadCodexConnection(t, db, connection.ID), task, runner
}

func newReadyCodexConnection(t *testing.T) (*gorm.DB, *models.FactoryCodexConnection, *models.RunnerTask, *models.Runner) {
	t.Helper()
	require.NoError(t, database.TruncateTables())
	resource := support.Setup(t)
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(resource.Organization.ID, features.FeatureCodexSubscriptions))

	factory, err := models.CreateFactory(db, resource.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	fleet := newTestRunnerFleet()
	require.NoError(t, fleet.Create(db))

	now := time.Now()
	runner := &models.Runner{
		ID:            uuid.New(),
		FleetID:       fleet.ID,
		State:         models.RunnerStateTerminated,
		RunnerVersion: "0.1.0",
		TerminatedAt:  &now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	require.NoError(t, db.Create(runner).Error)
	task := &models.RunnerTask{
		ID:                uuid.New(),
		OrganizationID:    resource.Organization.ID,
		FleetID:           fleet.ID,
		Backend:           models.RunnerTaskBackendIntegrated,
		State:             models.RunnerTaskStateQueued,
		PayloadCiphertext: []byte("ciphertext"),
		QueuedAt:          now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	require.NoError(t, db.Create(task).Error)

	connection, err := models.CreateFactoryCodexConnection(db, resource.Organization.ID, factory.ID, resource.User, now)
	require.NoError(t, err)
	require.NoError(t, connection.BeginLogin(db, task.ID, now))
	require.NoError(t, connection.StoreBundle(db, task.ID, connection.ReservationGeneration, 1, []byte("bundle"), "ada@example.com", true, now))
	require.NoError(t, markRunnerTask(db, task, models.RunnerTaskStateSucceeded, &runner.ID, now))
	require.NoError(t, runner.MarkProviderDeleted(db, now))
	return db, reloadCodexConnection(t, db, connection.ID), task, reloadRunner(t, db, runner.ID)
}

func markRunnerTask(db *gorm.DB, task *models.RunnerTask, state string, runnerID *uuid.UUID, now time.Time) error {
	return db.Model(task).Updates(map[string]any{
		"state":       state,
		"runner_id":   runnerID,
		"finished_at": now,
		"updated_at":  now,
	}).Error
}

func reloadCodexConnection(t *testing.T, db *gorm.DB, id uuid.UUID) *models.FactoryCodexConnection {
	t.Helper()
	var connection models.FactoryCodexConnection
	require.NoError(t, db.First(&connection, "id = ?", id).Error)
	return &connection
}

func reloadRunner(t *testing.T, db *gorm.DB, id uuid.UUID) *models.Runner {
	t.Helper()
	var runner models.Runner
	require.NoError(t, db.First(&runner, "id = ?", id).Error)
	return &runner
}

func TestMemberRemovalRevokesCredentialsBeforeRunnerCleanup(t *testing.T) {
	db, connection, task, runner := newReadyCodexConnection(t)
	require.NoError(t, clearProviderDeletion(db, runner))
	require.NoError(t, markRunnerTask(db, task, models.RunnerTaskStateRunning, &runner.ID, time.Now()))
	_, err := reserveCodexRun(t, db, connection, task.ID, time.Now())
	require.NoError(t, err)
	lease := codexLeases(t, db, connection.ID)[0]
	member, err := models.FindActiveUserByIDInTransaction(db, connection.OrganizationID.String(), connection.OwnerUserID.String())
	require.NoError(t, err)
	require.NoError(t, member.SoftDelete(db, time.Now(), ""))
	revoked := reloadCodexConnection(t, db, connection.ID)
	require.NotNil(t, revoked.RevocationPendingAt)
	assert.Equal(t, models.CodexConnectionStatusDisconnected, revoked.Status)
	assert.Empty(t, revoked.BundleCiphertext)
	assert.Empty(t, revoked.AccountLabel)
	_, err = models.FindCodexLeaseForTask(db, connection.ID, task.ID, lease.Generation)
	require.ErrorIs(t, err, models.ErrCodexConnectionStale)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, err := revoked.CleanupRevocation(tx, time.Now())
		return err
	}))
	stopping, err := models.FindRunnerTask(db, task.ID)
	require.NoError(t, err)
	assert.NotNil(t, stopping.CancelRequestedAt)
	require.NoError(t, markRunnerTask(db, task, models.RunnerTaskStateLost, &runner.ID, time.Now()))
	require.NoError(t, models.ReleaseConfirmedCodexReservations(db, time.Now()))
	assert.Len(t, codexLeases(t, db, connection.ID), 1, "a lost runner retains revocation evidence until provider deletion")
	require.NoError(t, runner.MarkProviderDeleted(db, time.Now()))
	assert.Empty(t, codexLeases(t, db, connection.ID))
	assert.Equal(t, models.CodexConnectionStatusDisconnected, reloadCodexConnection(t, db, connection.ID).Status)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, err := revoked.CleanupRevocation(tx, time.Now())
		return err
	}))
	require.NoError(t, member.RestoreInTransaction(db))
	eligible, err := models.SubscriptionOwnerEligible(db, connection.OrganizationID, connection.FactoryID, connection.OwnerUserID)
	require.NoError(t, err)
	assert.False(t, eligible, "rejoining must not restore credentials")
	reconnected := reloadCodexConnection(t, db, connection.ID)
	require.Nil(t, reconnected.RevocationPendingAt)
	require.NoError(t, reconnected.BeginLogin(db, uuid.New(), time.Now()))
	assert.Empty(t, reconnected.BundleCiphertext)
}

func TestSubscriptionOwnerChangesValidateBeforeMutation(t *testing.T) {
	db, connection, _, _ := newReadyCodexConnection(t)
	f, err := models.FindFactory(db, connection.OrganizationID, connection.FactoryID)
	require.NoError(t, err)
	subscription := true
	require.NoError(t, f.UpdateOnboarding(db, models.FactoryOnboardingPatch{CodexSubscription: &subscription}))
	order, err := f.CreateWorkOrder(db, "Assign an owner", "", &connection.OwnerUserID, []uuid.UUID{connection.OwnerUserID}, nil)
	require.NoError(t, err)
	other := &models.User{ID: uuid.New(), OrganizationID: f.OrganizationID, Type: models.UserTypeHuman, Name: "No subscription"}
	require.NoError(t, db.Create(other).Error)
	for _, ids := range [][]uuid.UUID{{connection.OwnerUserID, other.ID}, {other.ID}} {
		require.Error(t, order.UpdateAssignees(db, ids, connection.OwnerUserID))
		loaded, err := f.FindWorkOrder(db, order.ID)
		require.NoError(t, err)
		assert.Equal(t, []uuid.UUID{connection.OwnerUserID}, loaded.AssigneeIDs())
	}
	require.NoError(t, order.UpdateAssignees(db, nil, connection.OwnerUserID))
	loaded, err := f.FindWorkOrder(db, order.ID)
	require.NoError(t, err)
	assert.Empty(t, loaded.Assignees)
	runID := support.CreateCodexOwnedRun(t, db, connection)
	running, err := models.FindSubscriptionWorkOrderForRun(db, f.OrganizationID, f.ID, runID)
	require.NoError(t, err)
	require.ErrorIs(t, running.UpdateAssignees(db, nil, connection.OwnerUserID), models.ErrSubscriptionOwnerBusy)
	// A busy task must not prevent changes to an idle task for the same member.
	require.NoError(t, loaded.UpdateAssignees(db, []uuid.UUID{connection.OwnerUserID}, connection.OwnerUserID))
	admission := db.Begin()
	require.NoError(t, admission.Error)
	t.Cleanup(func() { admission.Rollback() })
	require.NoError(t, running.LockForUpdate(admission))
	_, err = models.ReserveFactoryCodexConnection(db, f.OrganizationID, f.ID, runID, uuid.New(), time.Now(), crypto.NewNoOpEncryptor())
	var busy *models.CodexConnectionBusyError
	require.ErrorAs(t, err, &busy)
	assert.Empty(t, codexLeases(t, db, connection.ID))
}

func TestMemberRemovalCleansQueuedWorkWithoutARunnerLease(t *testing.T) {
	db, connection, _, _ := newReadyCodexConnection(t)
	f, err := models.FindFactory(db, connection.OrganizationID, connection.FactoryID)
	require.NoError(t, err)
	subscription := true
	require.NoError(t, f.UpdateOnboarding(db, models.FactoryOnboardingPatch{CodexSubscription: &subscription}))
	order, err := f.CreateWorkOrder(db, "Queued task", "", &connection.OwnerUserID, []uuid.UUID{connection.OwnerUserID}, nil)
	require.NoError(t, err)
	line, err := f.CreateLine(db, "ship", nil)
	require.NoError(t, err)
	dispatch := &models.FactoryWorkOrderLineDispatch{ID: uuid.New(), OrganizationID: f.OrganizationID, FactoryID: f.ID, WorkOrderID: order.ID, LineID: line.ID, LineName: line.Name, State: models.FactoryWorkOrderLineDispatchStateActive}
	require.NoError(t, db.Create(dispatch).Error)
	item := &models.FactoryWorkOrderQueueItem{ID: uuid.New(), OrganizationID: f.OrganizationID, FactoryID: f.ID, WorkOrderID: order.ID, LineID: line.ID, LineDispatchID: dispatch.ID, StepIndex: 0, StepName: "Implement", CreatedAt: time.Now()}
	require.NoError(t, db.Create(item).Error)
	assert.Empty(t, codexLeases(t, db, connection.ID))
	member, err := models.FindActiveUserByIDInTransaction(db, f.OrganizationID.String(), connection.OwnerUserID.String())
	require.NoError(t, err)
	require.NoError(t, member.SoftDelete(db, time.Now(), ""))
	// Re-open the durable cleanup state as a new worker would after a restart.
	pending, err := models.ListPendingCodexRevocations(db)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, err := pending[0].CleanupRevocation(tx, time.Now())
		return err
	}))
	require.NoError(t, db.First(dispatch, "id = ?", dispatch.ID).Error)
	assert.Equal(t, models.FactoryWorkOrderLineDispatchStateFinished, dispatch.State)
	assert.Equal(t, models.CanvasRunResultCancelled, dispatch.Result)
	blocked, err := order.SubscriptionOwnershipBlocked(db)
	require.NoError(t, err)
	assert.False(t, blocked)
	assert.Nil(t, reloadCodexConnection(t, db, connection.ID).RevocationPendingAt)
}

func TestStaleRevocationCleanupLeavesCurrentLeasesAlone(t *testing.T) {
	db, connection, task, runner := newReadyCodexConnection(t)
	require.NoError(t, clearProviderDeletion(db, runner))
	require.NoError(t, markRunnerTask(db, task, models.RunnerTaskStateRunning, &runner.ID, time.Now()))
	_, err := reserveCodexRun(t, db, connection, task.ID, time.Now())
	require.NoError(t, err)
	// A worker read this snapshot while the revocation was pending. Another
	// worker then finished the cleanup, and the lease belongs to new work.
	stale := reloadCodexConnection(t, db, connection.ID)
	pendingAt := time.Now()
	stale.RevocationPendingAt = &pendingAt

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		results, err := stale.CleanupRevocation(tx, time.Now())
		assert.Empty(t, results)
		return err
	}))

	current, err := models.FindRunnerTask(db, task.ID)
	require.NoError(t, err)
	assert.Nil(t, current.CancelRequestedAt)
	assert.Len(t, codexLeases(t, db, connection.ID), 1)
	assert.Nil(t, codexLeases(t, db, connection.ID)[0].RevokedAt)
}

func TestPersonalCodexMigrationPreservesCredentialsAndLeases(t *testing.T) {
	db := database.DB(t.Context())
	migration, err := os.ReadFile("../../db/migrations/20261029101500_personal-codex-connections.up.sql")
	require.NoError(t, err)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		// Temporary tables isolate the old schema from the live test tables.
		require.NoError(t, tx.Exec(`CREATE TEMP TABLE factory_codex_connections (
   id uuid PRIMARY KEY, factory_id uuid NOT NULL, owner_user_id uuid NOT NULL,
   bundle_ciphertext bytea, credential_version bigint, reservation_generation bigint,
   CONSTRAINT factory_codex_connections_factory_key UNIQUE (factory_id)
  ) ON COMMIT DROP;
  CREATE TEMP TABLE factory_codex_leases (
   id uuid PRIMARY KEY, connection_id uuid, task_id uuid, generation bigint,
   secret_hash text, secret_ciphertext bytea
  ) ON COMMIT DROP;
  CREATE TEMP TABLE factory_work_orders (id uuid PRIMARY KEY) ON COMMIT DROP;`).Error)
		factoryID, ownerID, connectionID, leaseID, taskID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
		require.NoError(t, tx.Exec("INSERT INTO factory_codex_connections VALUES (?, ?, ?, ?, 7, 12)", connectionID, factoryID, ownerID, []byte("encrypted-auth-bundle")).Error)
		require.NoError(t, tx.Exec("INSERT INTO factory_codex_leases VALUES (?, ?, ?, 12, ?, ?)", leaseID, connectionID, taskID, "lease-hash", []byte("encrypted-lease-secret")).Error)
		require.NoError(t, tx.Exec(string(migration)).Error)
		var connection models.FactoryCodexConnection
		require.NoError(t, tx.Table("factory_codex_connections").Where("id = ?", connectionID).First(&connection).Error)
		assert.Equal(t, []byte("encrypted-auth-bundle"), connection.BundleCiphertext)
		assert.EqualValues(t, 7, connection.CredentialVersion)
		assert.EqualValues(t, 12, connection.ReservationGeneration)
		assert.Nil(t, connection.RevocationPendingAt)
		var lease models.FactoryCodexLease
		require.NoError(t, tx.Table("factory_codex_leases").Where("id = ?", leaseID).First(&lease).Error)
		assert.Equal(t, connectionID, lease.ConnectionID)
		assert.Equal(t, taskID, lease.TaskID)
		assert.EqualValues(t, 12, lease.Generation)
		assert.Equal(t, "lease-hash", lease.SecretHash)
		assert.Equal(t, []byte("encrypted-lease-secret"), lease.SecretCiphertext)
		assert.Nil(t, lease.RevokedAt)
		require.NoError(t, tx.Exec("INSERT INTO factory_codex_connections (id, factory_id, owner_user_id) VALUES (?, ?, ?)", uuid.New(), factoryID, uuid.New()).Error)
		duplicate := tx.Transaction(func(inner *gorm.DB) error {
			return inner.Exec("INSERT INTO factory_codex_connections (id, factory_id, owner_user_id) VALUES (?, ?, ?)", uuid.New(), factoryID, ownerID).Error
		})
		require.Error(t, duplicate)
		return nil
	}))
}

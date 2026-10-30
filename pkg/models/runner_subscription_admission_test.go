package models_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestRunnerAdmissionRejectsQueuedModelRemovedFromWorkspace(t *testing.T) {
	for _, scenario := range []struct {
		name, component, source                  string
		nativeSnapshot, blankSnapshot, permitted bool
	}{
		{name: "BYOK", component: "runnerCodex", source: "integration"},
		{name: "legacy Hosted", component: "runnerCodex", source: "hosted"},
		{name: "legacy Hosted blank snapshot", component: "runnerCodex", source: "hosted", blankSnapshot: true},
		{name: "SuperPlane Hosted key", component: models.SuperPlaneRunnerComponent, source: "hosted"},
		{name: "SuperPlane Hosted snapshot excluded", component: models.SuperPlaneRunnerComponent, source: "hosted", nativeSnapshot: true},
		{name: "SuperPlane Hosted blank snapshot", component: models.SuperPlaneRunnerComponent, source: "hosted", nativeSnapshot: true, blankSnapshot: true},
		{name: "SuperPlane Hosted snapshot permitted", component: models.SuperPlaneRunnerComponent, source: "hosted", nativeSnapshot: true, permitted: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			db, connection, task, previousRunner := newReadyCodexConnection(t)
			funding := models.UsageFundingSourceBYOK
			if scenario.source == "integration" {
				require.NoError(t, models.EnableExperimentalFeature(connection.OrganizationID, features.FeatureOrganizationBYOK))
				_, err := models.UpsertOrganizationBYOKModelAllowlist(db, connection.OrganizationID, models.UsageProviderOpenAI, []string{"permitted-model", "excluded-model"})
				require.NoError(t, err)
			}
			if scenario.source == "hosted" {
				funding = models.UsageFundingSourceHosted
				previous, err := models.FindHostedLLMProvider(db, models.UsageProviderOpenAI)
				t.Cleanup(func() {
					cleanup := db.WithContext(context.Background())
					if previous != nil {
						_, restoreErr := models.UpsertHostedLLMProvider(cleanup, *previous)
						require.NoError(t, restoreErr)
						return
					}
					require.NoError(t, cleanup.Where("provider = ?", models.UsageProviderOpenAI).Delete(&models.HostedLLMProvider{}).Error)
				})
				require.True(t, err == nil || errors.Is(err, models.ErrHostedLLMProviderNotFound))
				_, err = models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{Provider: models.UsageProviderOpenAI, Enabled: true, APIKey: []byte("key"), AllowedModels: []string{"permitted-model", "excluded-model"}})
				require.NoError(t, err)
			}
			require.NoError(t, db.Model(task).Updates(map[string]any{"state": models.RunnerTaskStateQueued, "runner_id": nil, "finished_at": nil}).Error)
			support.CreateCodexOwnedRun(t, db, connection, task.ID)
			execution, err := task.FindExecution(db)
			require.NoError(t, err)
			model := "excluded-model"
			if scenario.permitted {
				model = "permitted-model"
			}
			if scenario.component == models.SuperPlaneRunnerComponent && !scenario.nativeSnapshot {
				model = models.FormatSelectableLLMModelKey(funding, models.UsageProviderOpenAI, model)
			}
			if scenario.blankSnapshot {
				model = ""
			}
			configuration := map[string]any{"model": model, "credentials": map[string]any{"source": scenario.source}}
			if scenario.nativeSnapshot {
				configuration["hostedProvider"] = models.UsageProviderOpenAI
			}
			require.NoError(t, db.Model(execution).Updates(map[string]any{
				"configuration": datatypes.NewJSONType(configuration),
				"metadata":      datatypes.NewJSONType(map[string]any{models.CanvasNodeExecutionFrozenComponentKey: scenario.component}),
			}).Error)
			inherit := false
			_, err = models.UpsertFactoryLLMModelAllowlist(db, connection.OrganizationID, connection.FactoryID, models.UsageProviderOpenAI, funding, []string{"permitted-model"}, &inherit)
			require.NoError(t, err)
			runner := &models.Runner{ID: uuid.New(), FleetID: previousRunner.FleetID, State: models.RunnerStateIdle, RunnerVersion: previousRunner.RunnerVersion}
			require.NoError(t, db.Create(runner).Error)
			reserved, err := runner.ReserveNextTask(db)
			require.NoError(t, err)
			if scenario.permitted {
				require.NotNil(t, reserved.Task)
				assert.Equal(t, task.ID, reserved.Task.ID)
				assert.Empty(t, reserved.FailedTasks)
				return
			}
			assert.Nil(t, reserved.Task)
			require.Len(t, reserved.FailedTasks, 1)
			stored, err := models.FindRunnerTask(db, task.ID)
			require.NoError(t, err)
			assert.Equal(t, models.RunnerTaskStateFailed, stored.State)
			require.NotNil(t, stored.ErrorMessage)
			assert.Equal(t, models.ErrSelectableLLMModelNotAllowed.Error(), *stored.ErrorMessage)
			execution, err = task.FindExecution(db)
			require.NoError(t, err)
			assert.Equal(t, model, execution.Configuration.Data()["model"])
		})
	}
}

func TestRunnerAdmissionRejectsInvalidSubscriptionAndContinuesFleetScan(t *testing.T) {
	for _, reason := range []string{"revoked lease", "disconnected connection", "missing connection", "missing owner", "unavailable model", "workspace disables subscription models", "blank queued model under restriction", "missing task association"} {
		t.Run(reason, func(t *testing.T) {
			db, connection, rejected, runner, order := queuedSubscriptionRunnerTask(t)
			expectedError := models.ErrCodexConnectionStale
			switch reason {
			case "revoked lease":
				require.NoError(t, db.Model(&models.FactoryCodexLease{}).Where("task_id = ?", rejected.ID).Update("revoked_at", time.Now()).Error)
			case "disconnected connection":
				expectedError = models.ErrCodexConnectionNotReady
				require.NoError(t, db.Model(connection).Update("status", models.CodexConnectionStatusDisconnected).Error)
			case "missing connection":
				expectedError = models.ErrCodexConnectionNotFound
				execution, err := rejected.FindExecution(db)
				require.NoError(t, err)
				require.NoError(t, db.Model(execution).Update("configuration", []byte(`{"credentials":{"source":"subscription"}}`)).Error)
				require.NoError(t, db.Delete(connection).Error)
			case "missing owner":
				expectedError = models.ErrSubscriptionOwnerRequired
				require.NoError(t, db.Where("work_order_id = ?", order.ID).Delete(&models.FactoryWorkOrderAssignee{}).Error)
			case "unavailable model":
				expectedError = models.ErrSubscriptionModelUnavailable
				execution, err := rejected.FindExecution(db)
				require.NoError(t, err)
				require.NoError(t, db.Model(execution).Update("configuration", []byte(`{"model":"removed-model"}`)).Error)
			case "workspace disables subscription models", "blank queued model under restriction":
				expectedError = models.ErrSubscriptionModelUnavailable
				inherit := false
				selected := []string{}
				if reason == "blank queued model under restriction" {
					require.NoError(t, connection.StoreUsage(db, nil, []models.CodexModel{{Model: "allowed-model"}}, time.Now()))
					selected = []string{"allowed-model"}
				}
				_, err := models.UpsertFactoryLLMModelAllowlist(db, connection.OrganizationID, connection.FactoryID, models.UsageProviderOpenAI, models.UsageFundingSourceSubscription, selected, &inherit)
				require.NoError(t, err)
			case "missing task association":
				expectedError = models.ErrSubscriptionTaskAssociation
				require.NoError(t, db.Where("key = ? AND value = ?", "task_id", rejected.ID.String()).Delete(&models.CanvasNodeExecutionKV{}).Error)
			}
			otherOrganization, err := models.CreateOrganization("Healthy fleet organization", "")
			require.NoError(t, err)
			healthy := &models.RunnerTask{ID: uuid.New(), OrganizationID: otherOrganization.ID, FleetID: runner.FleetID,
				Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateQueued,
				QueuedAt: time.Now().Add(time.Minute), PayloadCiphertext: []byte("healthy")}
			require.NoError(t, db.Create(healthy).Error)

			reserved, err := runner.ReserveNextTask(db)
			require.NoError(t, err)
			require.NotNil(t, reserved.Task)
			assert.Equal(t, healthy.ID, reserved.Task.ID)
			require.Len(t, reserved.FailedTasks, 1)
			assert.Equal(t, rejected.ID, reserved.FailedTasks[0].ID)
			stored, err := models.FindRunnerTask(db, rejected.ID)
			require.NoError(t, err)
			assert.Equal(t, models.RunnerTaskStateFailed, stored.State)
			require.NotNil(t, stored.ErrorMessage)
			if reason == "workspace disables subscription models" {
				assert.Contains(t, *stored.ErrorMessage, "No ChatGPT models are enabled")
				assert.Contains(t, *stored.ErrorMessage, "Update Workspace Models")
			}
			assert.Contains(t, *stored.ErrorMessage, expectedError.Error())
			assert.NotNil(t, stored.FinishedAt)
			assert.Nil(t, stored.RunnerID)
			assert.Empty(t, codexLeases(t, db, connection.ID))
		})
	}
}

func TestRunnerAdmissionStopsAfterTenCandidatesAndResumesOnNextCall(t *testing.T) {
	db, connection, first, runner, _ := queuedSubscriptionRunnerTask(t)
	now := time.Now()
	require.NoError(t, db.Model(&models.FactoryCodexLease{}).Where("task_id = ?", first.ID).Update("revoked_at", now).Error)
	for i := 1; i < 10; i++ {
		task := *first
		task.ID = uuid.New()
		task.QueuedAt = first.QueuedAt.Add(time.Duration(i) * time.Millisecond)
		require.NoError(t, db.Create(&task).Error)
		require.NoError(t, db.Create(&models.FactoryCodexLease{
			ID: uuid.New(), ConnectionID: connection.ID, TaskID: task.ID,
			Generation: int64(i + 1), Purpose: models.CodexLeasePurposeRun, RevokedAt: &now,
		}).Error)
	}
	healthy := *first
	healthy.ID = uuid.New()
	healthy.QueuedAt = first.QueuedAt.Add(time.Minute)
	require.NoError(t, db.Create(&healthy).Error)

	reservation, err := runner.ReserveNextTask(db)
	require.NoError(t, err)
	assert.Nil(t, reservation.Task)
	require.Len(t, reservation.FailedTasks, 10)
	for _, failed := range reservation.FailedTasks {
		stored, err := models.FindRunnerTask(db, failed.ID)
		require.NoError(t, err)
		assert.Equal(t, models.RunnerTaskStateFailed, stored.State)
	}
	assert.Empty(t, codexLeases(t, db, connection.ID))
	stored, err := models.FindRunnerTask(db, healthy.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerTaskStateQueued, stored.State)

	reservation, err = runner.ReserveNextTask(db)
	require.NoError(t, err)
	require.NotNil(t, reservation.Task)
	assert.Equal(t, healthy.ID, reservation.Task.ID)
	assert.Empty(t, reservation.FailedTasks)
}

func TestRunnerAdmissionCommitsLoneSubscriptionFailureAndReleasesLogin(t *testing.T) {
	db, connection, task, previousRunner := newDisconnectedCodexLogin(t)
	runner := &models.Runner{ID: uuid.New(), FleetID: previousRunner.FleetID, State: models.RunnerStateIdle, RunnerVersion: previousRunner.RunnerVersion}
	require.NoError(t, db.Create(runner).Error)
	require.NoError(t, db.Model(&models.FactoryCodexLease{}).Where("task_id = ?", task.ID).Update("revoked_at", time.Now()).Error)

	reserved, err := runner.ReserveNextTask(db)
	require.NoError(t, err)
	assert.Nil(t, reserved.Task)
	require.Len(t, reserved.FailedTasks, 1)
	assert.Equal(t, task.ID, reserved.FailedTasks[0].ID)
	require.Len(t, reserved.LoginConnections, 1)
	assert.Equal(t, connection.ID, reserved.LoginConnections[0].ID)
	stored, err := models.FindRunnerTask(db, task.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerTaskStateFailed, stored.State)
	require.NotNil(t, stored.ErrorMessage)
	assert.Equal(t, models.ErrCodexConnectionStale.Error(), *stored.ErrorMessage)
	assert.NotNil(t, stored.FinishedAt)
	assert.Empty(t, codexLeases(t, db, connection.ID))
}

func TestRunnerAdmissionKeepsContendedSubscriptionQueuedAndReleasesLocks(t *testing.T) {
	db, connection, task, runner, order := queuedSubscriptionRunnerTask(t)
	otherOrganization, err := models.CreateOrganization("Healthy fleet organization", "")
	require.NoError(t, err)
	later := *task
	later.ID = uuid.New()
	later.QueuedAt = task.QueuedAt.Add(time.Minute)
	require.NoError(t, db.Create(&later).Error)
	healthy := later
	healthy.ID = uuid.New()
	healthy.OrganizationID = otherOrganization.ID
	healthy.QueuedAt = later.QueuedAt.Add(time.Minute)
	require.NoError(t, db.Create(&healthy).Error)
	holder := db.Begin()
	require.NoError(t, holder.Error)
	defer holder.Rollback()
	require.NoError(t, order.LockForUpdate(holder))

	scan := db.Begin()
	require.NoError(t, scan.Error)
	defer scan.Rollback()
	reserved, err := runner.ReserveNextTask(scan)
	require.NoError(t, err)
	require.NotNil(t, reserved.Task)
	assert.Equal(t, healthy.ID, reserved.Task.ID)
	assert.Empty(t, reserved.FailedTasks)
	stored, err := models.FindRunnerTask(db, task.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerTaskStateQueued, stored.State)
	assert.Nil(t, stored.ErrorMessage)
	assert.Nil(t, stored.FinishedAt)
	require.Len(t, codexLeases(t, db, connection.ID), 1)
	require.NoError(t, db.Transaction(func(probe *gorm.DB) error {
		return probe.Clauses(clause.Locking{Strength: "UPDATE", Options: "NOWAIT"}).First(&models.RunnerTask{}, "id = ?", task.ID).Error
	}))
	require.NoError(t, scan.Commit().Error)
	require.NoError(t, holder.Rollback().Error)
	nextRunner := *runner
	nextRunner.ID = uuid.New()
	require.NoError(t, db.Create(&nextRunner).Error)
	reserved, err = nextRunner.ReserveNextTask(db)
	require.NoError(t, err)
	require.NotNil(t, reserved.Task)
	assert.Equal(t, task.ID, reserved.Task.ID)
}

func queuedSubscriptionRunnerTask(t *testing.T) (*gorm.DB, *models.FactoryCodexConnection, *models.RunnerTask, *models.Runner, *models.FactoryWorkOrder) {
	t.Helper()
	db, connection, task, previousRunner := newReadyCodexConnection(t)
	require.NoError(t, db.Model(task).Updates(map[string]any{"state": models.RunnerTaskStateQueued, "runner_id": nil, "finished_at": nil}).Error)
	runner := &models.Runner{ID: uuid.New(), FleetID: previousRunner.FleetID, State: models.RunnerStateIdle, RunnerVersion: previousRunner.RunnerVersion}
	require.NoError(t, db.Create(runner).Error)
	runID := support.CreateCodexOwnedRun(t, db, connection, task.ID)
	_, err := models.ReserveFactoryCodexConnection(db, connection.OrganizationID, connection.FactoryID, runID, task.ID, time.Now(), crypto.NewNoOpEncryptor())
	require.NoError(t, err)
	order, err := models.FindSubscriptionWorkOrderForRun(db, connection.OrganizationID, connection.FactoryID, runID)
	require.NoError(t, err)
	return db, connection, task, runner, order
}

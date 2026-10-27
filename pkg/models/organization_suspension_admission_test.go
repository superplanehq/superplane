package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestSuspendedOrganizationRunnerAdmission(t *testing.T) {
	f := newRunnerCapacityTest(t, nil)
	fleet := f.fleet("suspension-amd64")
	queued := f.task(f.organizationID, fleet.ID)
	reserved := f.task(f.organizationID, fleet.ID)
	runner := f.runner(fleet.ID)
	require.NoError(t, reserved.Reserve(f.db, runner.ID))
	other, err := models.CreateOrganization("Active fleet user", "")
	require.NoError(t, err)
	active := f.task(other.ID, fleet.ID)
	organization := &models.Organization{ID: f.organizationID}
	require.NoError(t, organization.Suspend(f.db, uuid.New(), "Abuse"))
	require.ErrorIs(t, queued.Reserve(f.db, f.runner(fleet.ID).ID), models.ErrOrganizationSuspended)
	require.ErrorIs(t, reserved.Start(f.db, runner, "test", time.Now()), models.ErrOrganizationSuspended)
	selected, err := f.runner(fleet.ID).ReserveNextTask(f.db)
	require.NoError(t, err)
	require.Equal(t, active.ID, selected.ID)
	newTask := &models.RunnerTask{ID: uuid.New(), OrganizationID: organization.ID, FleetID: fleet.ID, Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateQueued}
	require.ErrorIs(t, f.db.Create(newTask).Error, models.ErrOrganizationSuspended)
	require.NoError(t, reserved.RequestCancel(f.db, time.Now()))
}

func TestSuspensionWaitsForAdmittedRunnerTask(t *testing.T) {
	f := newRunnerCapacityTest(t, nil)
	fleet := f.fleet("suspension-race")
	admission := f.db.Begin()
	require.NoError(t, admission.Error)
	defer admission.Rollback()
	task := &models.RunnerTask{ID: uuid.New(), OrganizationID: f.organizationID, FleetID: fleet.ID, Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateQueued, PayloadCiphertext: []byte("payload")}
	require.NoError(t, admission.Create(task).Error)
	suspension := f.db.Begin()
	require.NoError(t, suspension.Error)
	defer suspension.Rollback()
	require.NoError(t, suspension.Exec("SET LOCAL lock_timeout = '100ms'").Error)
	organization := &models.Organization{ID: f.organizationID}
	// Suspension cannot commit while an admitted task is still invisible to
	// the cancellation worker in another transaction.
	require.Error(t, organization.Suspend(suspension, uuid.New(), "Abuse"))
	require.NoError(t, suspension.Rollback().Error)
	require.NoError(t, admission.Commit().Error)
	require.NoError(t, organization.Suspend(f.db, uuid.New(), "Abuse"))
}

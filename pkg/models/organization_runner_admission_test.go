package models_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Scheduling and reservation are separate admission boundaries. This test
// protects queued work from flag bypass while preserving committed work.
func TestOrganizationRunnerDisablementPreservesWork(t *testing.T) {
	for _, disable := range []string{"feature", "fleet"} {
		t.Run(disable, func(t *testing.T) {
			f := newRunnerCapacityTest(t, new(3))
			fleet := newOrganizationTestRunnerFleet(f.organizationID, "private-amd64")
			require.NoError(t, fleet.Create(f.db))
			require.NoError(t, models.EnableExperimentalFeature(f.organizationID, features.FeatureOrganizationRunners))
			active, queued := f.task(f.organizationID, fleet.ID), f.task(f.organizationID, fleet.ID)
			scheduled, err := models.ScheduleOrganizationRunnerTasks(f.db, f.organizationID, 3)
			require.NoError(t, err)
			require.Len(t, scheduled, 2)
			runner := f.runner(fleet.ID)
			require.NoError(t, active.Reserve(f.db, runner.ID))
			if disable == "feature" {
				require.NoError(t, models.DisableExperimentalFeature(f.organizationID, features.FeatureOrganizationRunners))
			} else {
				require.NoError(t, fleet.SetEnabled(f.db, false))
			}
			idle := f.runner(fleet.ID)
			reservation, err := idle.ReserveNextTask(f.db)
			require.NoError(t, err)
			assert.Nil(t, reservation.Task)
			assert.Error(t, queued.Reserve(f.db, idle.ID))
			// A reservation committed before disablement remains startable.
			require.NoError(t, active.Start(f.db, runner, "test-store", time.Now()))
			require.NoError(t, active.RequestCancel(f.db, time.Now()))
			_, err = active.Complete(f.db, runner, "cancelled", datatypes.JSON([]byte(`{}`)), 0, "", true, time.Now())
			require.NoError(t, err)
			_, err = active.Complete(f.db, runner, "cancelled", datatypes.JSON([]byte(`{}`)), 0, "", true, time.Now())
			require.NoError(t, err)
			requeued, err := models.RequeueUnavailableFleetRunnerTasks(f.db, f.organizationID, 3)
			require.NoError(t, err)
			require.Len(t, requeued, 1)
			assert.Equal(t, queued.ID, requeued[0].ID)
			assert.Equal(t, fleet.ID, requeued[0].FleetID)
			assert.Equal(t, models.RunnerTaskStatePending, requeued[0].State)
			scheduled, err = models.ScheduleOrganizationRunnerTasks(f.db, f.organizationID, 3)
			require.NoError(t, err)
			assert.Empty(t, scheduled)
			require.NoError(t, models.EnableExperimentalFeature(f.organizationID, features.FeatureOrganizationRunners))
			require.NoError(t, fleet.SetEnabled(f.db, true))
			scheduled, err = models.ScheduleOrganizationRunnerTasks(f.db, f.organizationID, 3)
			require.NoError(t, err)
			require.Len(t, scheduled, 1)
			assert.Equal(t, fleet.ID, scheduled[0].FleetID)
		})
	}
}

func TestOrganizationRunnerAdmissionSerializesWithDisablement(t *testing.T) {
	for _, disable := range []string{"feature", "fleet"} {
		for _, first := range []string{"assignment", "disablement"} {
			t.Run(disable+"/"+first, func(t *testing.T) {
				f := newRunnerCapacityTest(t, new(1))
				fleet := newOrganizationTestRunnerFleet(f.organizationID, "private-amd64")
				require.NoError(t, fleet.Create(f.db))
				require.NoError(t, models.EnableExperimentalFeature(f.organizationID, features.FeatureOrganizationRunners))
				task := f.task(f.organizationID, fleet.ID)
				_, err := models.ScheduleOrganizationRunnerTasks(f.db, f.organizationID, 1)
				require.NoError(t, err)
				runner := f.runner(fleet.ID)
				holder := f.db.Begin()
				defer holder.Rollback()
				assignment := func(tx *gorm.DB) error { return task.Reserve(tx, runner.ID) }
				disablement := func(tx *gorm.DB) error {
					if disable == "fleet" {
						return fleet.SetEnabled(tx, false)
					}
					return models.DisableExperimentalFeatureInTransaction(tx, f.organizationID, features.FeatureOrganizationRunners)
				}
				lead, follow := assignment, disablement
				if first == "disablement" {
					lead, follow = disablement, assignment
				}
				require.NoError(t, lead(holder))
				// The opposing admission cannot pass an uncommitted row lock.
				blockedCtx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
				err = f.db.WithContext(blockedCtx).Transaction(follow)
				cancel()
				require.Error(t, err)
				require.NoError(t, holder.Commit().Error)
				err = f.db.Transaction(follow)
				if first == "disablement" {
					expectedError := models.ErrOrganizationRunnersDisabled
					if disable == "fleet" {
						expectedError = models.ErrRunnerFleetDisabled
					}
					require.ErrorIs(t, err, expectedError)
				} else {
					require.NoError(t, err)
				}
				stored, err := models.FindRunnerTask(f.db, task.ID)
				require.NoError(t, err)
				expected := models.RunnerTaskStateReserved
				if first == "disablement" {
					expected = models.RunnerTaskStateQueued
				}
				assert.Equal(t, expected, stored.State)
			})
		}
	}
}

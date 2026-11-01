package workers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

// Suspension must wait for real admission writes, then cancellation must see
// those committed writes. Transition-only tests cannot prove this ordering.
func TestSuspensionAfterAdmissionCancelsCommittedWork(t *testing.T) {
	for _, operation := range []string{"canvas run", "factory dispatch", "factory queue admission", "runner creation", "runner scheduling", "runner reservation", "runner start"} {
		t.Run(operation, func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()
			db := database.DB(t.Context())
			var run *models.CanvasRun
			var task *models.RunnerTask
			var admit func(*gorm.DB) error
			switch operation {
			case "canvas run":
				canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{NodeID: "start", Type: models.NodeTypeTrigger}}, nil)
				admit = func(tx *gorm.DB) error {
					var err error
					run, err = models.CreateCanvasRunInTransaction(tx, canvas.ID, "start", models.CanvasRunStatePending, "")
					return err
				}
			case "factory dispatch", "factory queue admission":
				factory, err := models.CreateFactory(db, r.Organization.ID, "Admission race", "", "")
				require.NoError(t, err)
				app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "build", "start")
				limit := 1
				line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint, MaxParallelism: &limit}})
				require.NoError(t, err)
				order, err := factory.CreateWorkOrder(db, "Build", "", &r.User, nil, nil)
				require.NoError(t, err)
				_, err = order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateOpen})
				require.NoError(t, err)
				if operation == "factory queue admission" {
					blocker, err := factory.CreateWorkOrder(db, "Occupy slot", "", &r.User, nil, nil)
					require.NoError(t, err)
					_, err = blocker.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateOpen})
					require.NoError(t, err)
					_, occupied, err := line.Dispatch(db, blocker)
					require.NoError(t, err)
					_, queued, err := line.Dispatch(db, order)
					require.NoError(t, err)
					require.NotNil(t, queued.QueueItem)
					require.NoError(t, db.Model(occupied.Execution).Update("status", models.FactoryWorkOrderExecutionStatusFinished).Error)
					admit = func(tx *gorm.DB) error {
						results, err := models.AdmitQueuedForStep(tx, line.ID, 0)
						if err == nil {
							require.Len(t, results, 1)
							run = results[0].Run
						}
						return err
					}
				} else {
					admit = func(tx *gorm.DB) error {
						_, result, err := line.Dispatch(tx, order)
						if err == nil {
							run = result.Run
						}
						return err
					}
				}
			default:
				fleets := models.DefaultInstallationRunnerFleets("test")
				require.NoError(t, fleets[0].Create(db))
				runner := &models.Runner{ID: uuid.New(), FleetID: fleets[0].ID, State: models.RunnerStateIdle, RunnerVersion: "test"}
				require.NoError(t, db.Create(runner).Error)
				task = &models.RunnerTask{ID: uuid.New(), OrganizationID: r.Organization.ID, FleetID: fleets[0].ID, Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStatePending, PayloadCiphertext: []byte("encrypted"), QueuedAt: time.Now()}
				if operation != "runner creation" {
					require.NoError(t, db.Create(task).Error)
				}
				if operation == "runner reservation" || operation == "runner start" {
					_, err := models.ScheduleOrganizationRunnerTasks(db, r.Organization.ID, 1)
					require.NoError(t, err)
					task, err = models.FindRunnerTask(db, task.ID)
					require.NoError(t, err)
				}
				switch operation {
				case "runner creation":
					admit = func(tx *gorm.DB) error {
						return contexts.NewRunnerTaskContext(tx, r.Encryptor, r.Organization.ID).Create(task.ID.String(), fleets[0].Slug, []byte(`{"run_mode":"bash_script","script":"true"}`))
					}
				case "runner scheduling":
					admit = func(tx *gorm.DB) error {
						scheduled, err := models.ScheduleOrganizationRunnerTasks(tx, r.Organization.ID, 1)
						if err == nil {
							require.Len(t, scheduled, 1)
						}
						return err
					}
				case "runner reservation":
					admit = func(tx *gorm.DB) error { return task.Reserve(tx, runner.ID) }
				case "runner start":
					require.NoError(t, task.Reserve(db, runner.ID))
					admit = func(tx *gorm.DB) error { return task.Start(tx, runner, "test-store", time.Now()) }
				}
			}

			admission := db.Begin()
			require.NoError(t, admission.Error)
			defer admission.Rollback()
			require.NoError(t, admit(admission))
			suspension := db.Begin()
			require.NoError(t, suspension.Error)
			defer suspension.Rollback()
			require.NoError(t, suspension.Exec("SET LOCAL lock_timeout = '5s'").Error)
			var pid int
			require.NoError(t, suspension.Raw("SELECT pg_backend_pid()").Scan(&pid).Error)
			done := make(chan error, 1)
			go func() {
				if err := r.Organization.Suspend(suspension, r.Account.ID, "Stop admitted work"); err != nil {
					suspension.Rollback()
					done <- err
					return
				}
				done <- suspension.Commit().Error
			}()
			finished := false
			defer func() {
				admission.Rollback()
				if !finished {
					<-done
				}
			}()
			require.Eventually(t, func() bool {
				var waiting bool
				err := db.Raw("SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE pid = ? AND wait_event_type = 'Lock')", pid).Scan(&waiting).Error
				return err == nil && waiting
			}, 2*time.Second, 10*time.Millisecond)
			require.NoError(t, admission.Commit().Error)
			err := <-done
			finished = true
			require.NoError(t, err)
			require.NoError(t, NewOrganizationSuspensionWorker().Process(t.Context()))
			if run != nil {
				stored, err := models.FindUnscopedCanvasRun(db, run.ID)
				require.NoError(t, err)
				require.Equal(t, models.CanvasRunStateCancelling, stored.State)
				return
			}
			require.NotNil(t, task)
			stored, err := models.FindRunnerTask(db, task.ID)
			require.NoError(t, err)
			if operation == "runner creation" || operation == "runner scheduling" {
				require.Equal(t, models.RunnerTaskStateCanceled, stored.State)
				return
			}
			require.NotNil(t, stored.CancelRequestedAt)
			require.ErrorIs(t, r.Organization.Reactivate(db, r.Account.ID), models.ErrSuspensionCancellationPending)
		})
	}
}

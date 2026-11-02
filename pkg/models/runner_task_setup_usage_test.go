package models_test

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/billing"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestCodexSetupAdmissionRechecksHostedRequirementsBeforeAllocation(t *testing.T) {
	for _, scenario := range []string{"flag", "payment", "plan", "credit", "workspace budget", "billing disabled"} {
		t.Run(scenario, func(t *testing.T) {
			db, connection, task, runner, runtime := newSetupUsageTask(t, models.RunnerTaskSetupPurposeCodexLogin)
			require.NoError(t, models.RequireCodexSetupAdmission(db, task.OrganizationID))
			var expected error
			switch scenario {
			case "flag":
				expected = models.ErrCodexConnectionUnavailable
				require.NoError(t, models.DisableExperimentalFeature(task.OrganizationID, features.FeatureCodexSubscriptions))
			case "payment":
				expected = billing.ErrPaymentMethodRequired
				runtime.paymentError = expected
			case "plan":
				expected = billing.ErrHostedSubscriptionRequired
				require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", task.OrganizationID).
					Update("trial_ends_at", time.Now().Add(-time.Hour)).Error)
			case "credit":
				expected = billing.ErrHostedCreditEmpty
				require.NoError(t, db.Where("organization_id = ?", task.OrganizationID).Delete(&billingmodels.OrganizationLLMCreditGrant{}).Error)
			case "workspace budget":
				factory, err := models.FindFactory(db, task.OrganizationID, support.CodexWorkspaceID(t, db, task.OrganizationID))
				require.NoError(t, err)
				zero := int64(0)
				require.NoError(t, factory.UpdateHostedSpendBudget(db, &zero))
			case "billing disabled":
				t.Setenv(billing.BillingEnabledEnv, "no")
				runtime.paymentError = billing.ErrPaymentMethodRequired
				require.NoError(t, db.Where("organization_id = ?", task.OrganizationID).Delete(&billingmodels.OrganizationLLMCreditGrant{}).Error)
			}
			err := task.Reserve(db, runner.ID)
			if expected != nil {
				require.ErrorIs(t, err, expected)
				stored, err := models.FindRunnerTask(db, task.ID)
				require.NoError(t, err)
				assert.Nil(t, stored.ReservedAt)
				assert.Nil(t, stored.RunnerID)
				assert.False(t, stored.SetupBillable)
				assert.Empty(t, setupUsageEvents(t, db, task.ID))
				return
			}
			require.NoError(t, err)
			assert.Equal(t, scenario != "billing disabled", task.SetupBillable)
			require.NoError(t, models.DisableExperimentalFeature(task.OrganizationID, features.FeatureCodexSubscriptions))
			require.NoError(t, task.RequireSubscriptionLeaseContinuation(db))
			require.NoError(t, db.Model(&models.FactoryCodexLease{}).Where("connection_id = ?", connection.ID).Update("revoked_at", time.Now()).Error)
			require.ErrorIs(t, task.RequireSubscriptionLeaseContinuation(db), models.ErrCodexConnectionStale)
		})
	}
}

func TestCodexSetupRecordsOneOrganizationChargeForEachCompletedAllocation(t *testing.T) {
	for _, scenario := range []struct {
		name, purpose    string
		exitCode         int32
		canceled         bool
		expired          bool
		deleteConnection bool
	}{
		{name: "sign in success", purpose: models.RunnerTaskSetupPurposeCodexLogin},
		{name: "catalog failure", purpose: models.RunnerTaskSetupPurposeCodexCatalog, exitCode: 1},
		{name: "sign in canceled", purpose: models.RunnerTaskSetupPurposeCodexLogin, canceled: true},
		{name: "catalog expiry", purpose: models.RunnerTaskSetupPurposeCodexCatalog, canceled: true, expired: true},
		{name: "connection deleted", purpose: models.RunnerTaskSetupPurposeCodexLogin, deleteConnection: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			db, connection, task, runner, _ := newSetupUsageTask(t, scenario.purpose)
			require.NoError(t, db.Model(runner).Update("ephemeral", true).Error)
			before, err := billing.RemainingCreditMicros(db, []uuid.UUID{task.OrganizationID})
			require.NoError(t, err)
			require.NoError(t, task.Reserve(db, runner.ID))
			assert.Empty(t, setupUsageEvents(t, db, task.ID))
			startedAt := time.Now().Add(-10*time.Second - 100*time.Millisecond)
			require.NoError(t, task.Start(db, runner, "test-store", startedAt))
			if scenario.expired {
				expiredAt := startedAt.Add(5 * time.Second)
				require.NoError(t, db.Model(&models.FactoryCodexLease{}).Where("task_id = ?", task.ID).
					Update("created_at", expiredAt.Add(-models.CodexCatalogTimeout)).Error)
				require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
					return models.ReleaseConfirmedCodexReservations(tx, expiredAt)
				}))
				stored, err := models.FindRunnerTask(db, task.ID)
				require.NoError(t, err)
				assert.NotNil(t, stored.CancelRequestedAt)
				assert.Empty(t, setupUsageEvents(t, db, task.ID))
			} else if scenario.canceled {
				require.NoError(t, task.RequestCancel(db, startedAt.Add(time.Second)))
				assert.Empty(t, setupUsageEvents(t, db, task.ID))
			}
			if scenario.deleteConnection {
				require.NoError(t, db.Delete(connection).Error)
			}
			// Billing mode is captured at allocation and must survive later policy changes.
			t.Setenv(billing.BillingEnabledEnv, "no")
			finishedAt := startedAt.Add(10*time.Second + 100*time.Millisecond)
			complete := func() (bool, error) {
				return task.Complete(db, runner, "same-completion", datatypes.JSON(`{}`), scenario.exitCode, "", scenario.canceled, finishedAt)
			}
			completed, err := complete()
			require.NoError(t, err)
			assert.True(t, completed)
			completed, err = complete()
			require.NoError(t, err)
			assert.False(t, completed)
			require.NoError(t, runner.MarkProviderDeleted(db, finishedAt.Add(time.Hour)))
			events := setupUsageEvents(t, db, task.ID)
			require.Len(t, events, 1)
			event := events[0]
			assert.Equal(t, scenario.purpose, event.SetupPurpose)
			assert.EqualValues(t, 11, event.DurationSeconds)
			assert.Positive(t, event.CostMicros)
			assert.Equal(t, event.CostMicros, event.ProviderCostMicros)
			assert.NotEmpty(t, event.PriceBookVersion)
			assert.Nil(t, event.FactoryID)
			assert.Nil(t, event.WorkOrderID)
			var attributed int64
			require.NoError(t, db.Model(&models.WorkspaceUsageEvent{}).Where("runner_task_id = ? AND (canvas_run_id IS NOT NULL OR node_execution_id IS NOT NULL)", task.ID).Count(&attributed).Error)
			assert.Zero(t, attributed)
			after, err := billing.RemainingCreditMicros(db, []uuid.UUID{task.OrganizationID})
			require.NoError(t, err)
			assert.Equal(t, before[task.OrganizationID]-event.CostMicros, after[task.OrganizationID])
			factoryID := support.CodexWorkspaceID(t, db, task.OrganizationID)
			workspace, _, err := billing.SummarizeComputeUsage(db, billing.UsageReportFilter{OrganizationID: task.OrganizationID, FactoryID: &factoryID})
			require.NoError(t, err)
			assert.Zero(t, workspace.CostMicros)
			catalogs, err := billingmodels.ListSpendingFilterCatalogs(db, billing.UsageReportFilter{OrganizationID: task.OrganizationID})
			require.NoError(t, err)
			report, err := billingmodels.SummarizeSpendingExplorer(db, billing.UsageReportFilter{OrganizationID: task.OrganizationID}, billingmodels.SpendingGroupByWorkspace, billingmodels.SpendingTimeGrainDay)
			require.NoError(t, err)
			require.Len(t, report.Breakdown, 1)
			assert.Equal(t, "ChatGPT setup", billingmodels.SpendingBreakdownLabel(report.Breakdown[0].ID, catalogs, billingmodels.SpendingGroupByWorkspace))
		})
	}
}

func TestCodexSetupReservationReleaseAndLossPreserveAllocationCharges(t *testing.T) {
	for _, scenario := range []string{"release and reallocate", "reserved loss", "running loss", "canceled reservation", "unmarked task", "billing disabled", "local fleet"} {
		t.Run(scenario, func(t *testing.T) {
			db, _, task, runner, _ := newSetupUsageTask(t, models.RunnerTaskSetupPurposeCodexLogin)
			if scenario == "unmarked task" {
				require.NoError(t, db.Model(task).Update("setup_purpose", "").Error)
			}
			if scenario == "billing disabled" {
				t.Setenv(billing.BillingEnabledEnv, "no")
			}
			if scenario == "local fleet" {
				require.NoError(t, db.Model(&models.RunnerFleet{}).Where("id = ?", task.FleetID).Update("slug", "local").Error)
			}
			require.NoError(t, task.Reserve(db, runner.ID))
			now := time.Now()
			reservedAt := now.Add(-2*time.Second - 100*time.Millisecond)
			require.NoError(t, db.Model(task).Update("reserved_at", reservedAt).Error)
			if scenario == "canceled reservation" {
				require.NoError(t, task.RequestCancel(db, now))
			}
			if scenario == "reserved loss" || scenario == "running loss" {
				if scenario == "running loss" {
					require.NoError(t, task.Start(db, runner, "test-store", reservedAt))
				}
				lastSeen := now.Add(-time.Minute)
				require.NoError(t, db.Model(runner).Update("last_seen_at", lastSeen).Error)
				_, err := runner.MarkLost(db, now.Add(-time.Second), now)
				require.NoError(t, err)
			} else {
				require.NoError(t, runner.Terminate(db, models.RunnerTerminationRegistrationExpired))
			}
			events := setupUsageEvents(t, db, task.ID)
			if scenario == "unmarked task" || scenario == "billing disabled" {
				assert.Empty(t, events)
				return
			}
			require.Len(t, events, 1)
			expectedSeconds := int64(math.Ceil(runner.TerminatedAt.Sub(reservedAt).Seconds()))
			assert.Equal(t, expectedSeconds, events[0].DurationSeconds)
			if scenario == "local fleet" {
				assert.Zero(t, events[0].CostMicros)
			}
			if scenario != "release and reallocate" {
				return
			}
			stored, err := models.FindRunnerTask(db, task.ID)
			require.NoError(t, err)
			assert.Nil(t, stored.ReservedAt)
			next := &models.Runner{ID: uuid.New(), FleetID: runner.FleetID, State: models.RunnerStateIdle, RunnerVersion: runner.RunnerVersion}
			require.NoError(t, db.Create(next).Error)
			require.NoError(t, models.DisableExperimentalFeature(task.OrganizationID, features.FeatureCodexSubscriptions))
			require.ErrorIs(t, stored.Reserve(db, next.ID), models.ErrCodexConnectionUnavailable)
			require.Len(t, setupUsageEvents(t, db, task.ID), 1)
			require.NoError(t, models.EnableExperimentalFeature(task.OrganizationID, features.FeatureCodexSubscriptions))
			require.NoError(t, stored.Reserve(db, next.ID))
			require.NoError(t, next.Terminate(db, models.RunnerTerminationRegistrationExpired))
			require.NoError(t, next.Terminate(db, models.RunnerTerminationRegistrationExpired))
			events = setupUsageEvents(t, db, task.ID)
			require.Len(t, events, 2)
			assert.NotEqual(t, events[0].IdempotencyKey, events[1].IdempotencyKey)
		})
	}
}

func TestCodexSetupCancellationBeforeAllocationIsFree(t *testing.T) {
	for _, state := range []string{models.RunnerTaskStatePending, models.RunnerTaskStateQueued} {
		t.Run(state, func(t *testing.T) {
			db, _, task, _, _ := newSetupUsageTask(t, models.RunnerTaskSetupPurposeCodexLogin)
			require.NoError(t, db.Model(task).Update("state", state).Error)
			require.NoError(t, task.RequestCancel(db, time.Now()))
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
				return models.ReleaseCodexReservationForTask(tx, task.ID, time.Now())
			}))
			stored, err := models.FindRunnerTask(db, task.ID)
			require.NoError(t, err)
			assert.Equal(t, models.RunnerTaskStateCanceled, stored.State)
			assert.Nil(t, stored.ReservedAt)
			assert.Empty(t, setupUsageEvents(t, db, task.ID))
		})
	}
}

func TestCodexSetupAccountingFailureRollsBackLifecycleTransition(t *testing.T) {
	for _, transition := range []string{"completion", "loss", "release"} {
		t.Run(transition, func(t *testing.T) {
			db, _, task, runner, runtime := newSetupUsageTask(t, models.RunnerTaskSetupPurposeCodexLogin)
			require.NoError(t, task.Reserve(db, runner.ID))
			if transition == "completion" {
				require.NoError(t, task.Start(db, runner, "test-store", time.Now()))
			}
			lastSeen := time.Now().Add(-time.Minute)
			require.NoError(t, db.Model(runner).Update("last_seen_at", lastSeen).Error)
			runtime.pricingError = errors.New("price book is unavailable")
			var err error
			switch transition {
			case "completion":
				_, err = task.Complete(db, runner, "completion", datatypes.JSON(`{}`), 0, "", false, time.Now())
			case "loss":
				_, err = runner.MarkLost(db, time.Now().Add(-time.Second), time.Now())
			case "release":
				err = runner.Terminate(db, models.RunnerTerminationRegistrationExpired)
			}
			require.ErrorIs(t, err, runtime.pricingError)
			stored, err := models.FindRunnerTask(db, task.ID)
			require.NoError(t, err)
			assert.Equal(t, task.State, stored.State)
			assert.NotNil(t, stored.ReservedAt)
			assert.Nil(t, stored.FinishedAt)
			assert.Empty(t, setupUsageEvents(t, db, task.ID))
			storedRunner := reloadRunner(t, db, runner.ID)
			assert.NotEqual(t, models.RunnerStateTerminated, storedRunner.State)
			runtime.pricingError = nil
			switch transition {
			case "completion":
				completed, err := task.Complete(db, runner, "completion", datatypes.JSON(`{}`), 0, "", false, time.Now())
				require.NoError(t, err)
				assert.True(t, completed)
			case "loss":
				_, err = runner.MarkLost(db, time.Now().Add(-time.Second), time.Now())
				require.NoError(t, err)
			case "release":
				require.NoError(t, runner.Terminate(db, models.RunnerTerminationRegistrationExpired))
			}
			require.Len(t, setupUsageEvents(t, db, task.ID), 1)
			stored, err = models.FindRunnerTask(db, task.ID)
			require.NoError(t, err)
			expectedState := map[string]string{
				"completion": models.RunnerTaskStateSucceeded,
				"loss":       models.RunnerTaskStateLost,
				"release":    models.RunnerTaskStateQueued,
			}[transition]
			assert.Equal(t, expectedState, stored.State)
			if transition != "completion" {
				assert.Equal(t, models.RunnerStateTerminated, reloadRunner(t, db, runner.ID).State)
			}
		})
	}
}

type setupUsageBilling struct {
	billing.Runtime
	paymentError error
	pricingError error
}

func (b *setupUsageBilling) AssertOrganizationPaymentMethod(*gorm.DB, uuid.UUID) error {
	return b.paymentError
}

func (b *setupUsageBilling) PriceCompute(usage billing.ComputeUsage) (billing.UsagePrice, error) {
	if b.pricingError != nil {
		return billing.UsagePrice{}, b.pricingError
	}
	return b.Runtime.PriceCompute(usage)
}

func newSetupUsageTask(t *testing.T, purpose string) (*gorm.DB, *models.FactoryCodexConnection, *models.RunnerTask, *models.Runner, *setupUsageBilling) {
	t.Helper()
	db, connection, task, runner := newDisconnectedCodexLogin(t)
	if purpose == models.RunnerTaskSetupPurposeCodexCatalog {
		require.NoError(t, db.Where("connection_id = ?", connection.ID).Delete(&models.FactoryCodexLease{}).Error)
		require.NoError(t, db.Model(connection).Updates(map[string]any{
			"status": models.CodexConnectionStatusReady, "credential_version": 1, "bundle_ciphertext": []byte("saved"),
		}).Error)
		_, err := connection.BeginCatalogRefresh(db, task.ID, crypto.NewNoOpEncryptor(), time.Now())
		require.NoError(t, err)
	}
	require.NoError(t, db.Model(task).Update("setup_purpose", purpose).Error)
	require.NoError(t, db.Model(runner).Updates(map[string]any{"state": models.RunnerStateIdle, "terminated_at": nil}).Error)
	require.NoError(t, db.Model(&models.RunnerFleet{}).Where("id = ?", task.FleetID).Update("slug", models.RunnerFleetE1LargeAMD64).Error)
	t.Setenv(billing.BillingEnabledEnv, "yes")
	_, err := billingmodels.EnsureOrganizationBillingPlan(db, task.OrganizationID)
	require.NoError(t, err)
	support.AssignOpenSystemTrial(t, task.OrganizationID)
	_, err = billingmodels.AddAdminLLMCreditGrant(db, task.OrganizationID, 1_000_000, "Setup compute test", nil)
	require.NoError(t, err)
	runtime := &setupUsageBilling{Runtime: billing.Require()}
	t.Cleanup(billing.Bind(runtime))
	return db, connection, task, runner, runtime
}

func setupUsageEvents(t *testing.T, db *gorm.DB, taskID uuid.UUID) []models.WorkspaceUsageEvent {
	t.Helper()
	var events []models.WorkspaceUsageEvent
	require.NoError(t, db.Where("runner_task_id = ?", taskID).Order("occurred_at, id").Find(&events).Error)
	return events
}

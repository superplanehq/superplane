package workers

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	runnercontrol "github.com/superplanehq/superplane/pkg/runners/control"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"gorm.io/gorm"
)

const runnerRegistrationCleanupBatchSize = 100
const terminatedRunnerCredentialRetryWindow = 24 * time.Hour
const runnerConnectionLossTimeout = 15 * time.Minute

type RunnerCleanupWorker struct {
	interval time.Duration
	logger   *log.Entry
}

func NewRunnerCleanupWorker(interval time.Duration) *RunnerCleanupWorker {
	return &RunnerCleanupWorker{
		interval: interval,
		logger:   log.WithField("worker", "RunnerCleanupWorker"),
	}
}

func (w *RunnerCleanupWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.Process(); err != nil {
				w.logger.WithError(err).Error("Failed to clean up runner registrations")
			}
		}
	}
}

func (w *RunnerCleanupWorker) Process() error {
	now := time.Now()
	ids, err := models.ListExpiredPendingRunnerIDs(
		database.Conn(),
		now,
		runnerRegistrationCleanupBatchSize,
	)
	if err != nil {
		return err
	}
	for _, id := range ids {
		var organizationID uuid.UUID
		err := database.Conn().Transaction(func(tx *gorm.DB) error {
			runner, err := models.LockRunner(tx, id)
			if errors.Is(err, models.ErrRunnerNotFound) {
				return nil
			}
			if err != nil {
				return err
			}
			if task, err := runner.FindActiveTask(tx); err == nil {
				organizationID = task.OrganizationID
			}
			return runner.Expire(tx, now)
		})
		if err != nil {
			w.logger.WithError(err).
				WithField("runner_id", id).
				Error("Failed to expire runner registration")
		}
		if err == nil && organizationID != uuid.Nil {
			_ = runnercontrol.PublishOrganizationCapacity(context.Background(), organizationID)
		}
	}
	staleRunners, err := models.ListStaleRunners(
		database.Conn(),
		now.Add(-runnerConnectionLossTimeout),
		runnerRegistrationCleanupBatchSize,
	)
	if err != nil {
		return err
	}
	for _, staleRunner := range staleRunners {
		var organizationID uuid.UUID
		var lostRunningTask *models.RunnerTask
		err := database.Conn().Transaction(func(tx *gorm.DB) error {
			runner, err := models.LockRunner(tx, staleRunner.ID)
			if errors.Is(err, models.ErrRunnerNotFound) {
				return nil
			}
			if err != nil {
				return err
			}
			if task, err := runner.FindActiveTask(tx); err == nil {
				organizationID = task.OrganizationID
			}
			lostRunningTask, err = runner.MarkLost(tx, now.Add(-runnerConnectionLossTimeout), now)
			return err
		})
		if err != nil {
			w.logger.WithError(err).
				WithField("runner_id", staleRunner.ID).
				Error("Failed to mark disconnected runner as lost")
			continue
		}
		if organizationID != uuid.Nil {
			_ = runnercontrol.PublishOrganizationCapacity(context.Background(), organizationID)
		}
		if lostRunningTask != nil && telemetry.MetricsEnabled() {
			telemetry.RecordRunnerTaskExecutionDuration(context.Background(), lostRunningTask, staleRunner.FleetSlug, staleRunner.FleetScope)
		}
	}
	return models.RevokeTerminatedRunnerCredentials(
		database.Conn(),
		now.Add(-terminatedRunnerCredentialRetryWindow),
		now,
	)
}

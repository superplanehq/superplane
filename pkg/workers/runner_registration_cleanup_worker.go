package workers

import (
	"context"
	"errors"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
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
		err := database.Conn().Transaction(func(tx *gorm.DB) error {
			runner, err := models.LockRunner(tx, id)
			if errors.Is(err, models.ErrRunnerNotFound) {
				return nil
			}
			if err != nil {
				return err
			}
			return runner.Expire(tx, now)
		})
		if err != nil {
			w.logger.WithError(err).
				WithField("runner_id", id).
				Error("Failed to expire runner registration")
		}
	}
	staleIDs, err := models.ListStaleRunnerIDs(
		database.Conn(),
		now.Add(-runnerConnectionLossTimeout),
		runnerRegistrationCleanupBatchSize,
	)
	if err != nil {
		return err
	}
	for _, id := range staleIDs {
		var lostRunningTask *models.RunnerTask
		err := database.Conn().Transaction(func(tx *gorm.DB) error {
			runner, err := models.LockRunner(tx, id)
			if errors.Is(err, models.ErrRunnerNotFound) {
				return nil
			}
			if err != nil {
				return err
			}
			lostRunningTask, err = runner.MarkLost(tx, now.Add(-runnerConnectionLossTimeout), now)
			return err
		})
		if err != nil {
			w.logger.WithError(err).
				WithField("runner_id", id).
				Error("Failed to mark disconnected runner as lost")
			continue
		}
		if lostRunningTask != nil {
			fleet, err := models.FindRunnerFleet(database.Conn().Unscoped(), lostRunningTask.FleetID)
			if err != nil {
				w.logger.WithError(err).
					WithField("task_id", lostRunningTask.ID).
					Error("Failed to find fleet for lost runner task metric")
				continue
			}
			telemetry.RecordRunnerTaskExecutionDuration(context.Background(), lostRunningTask, fleet.Slug)
		}
	}
	return models.RevokeTerminatedRunnerCredentials(
		database.Conn(),
		now.Add(-terminatedRunnerCredentialRetryWindow),
		now,
	)
}

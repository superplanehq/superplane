package workers

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
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
		if err := models.ExpirePendingRunner(database.Conn(), id, now); err != nil {
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
		if err := models.MarkRunnerConnectionLost(
			database.Conn(),
			id,
			now.Add(-runnerConnectionLossTimeout),
			now,
		); err != nil {
			w.logger.WithError(err).
				WithField("runner_id", id).
				Error("Failed to mark disconnected runner as lost")
		}
	}
	return models.RevokeTerminatedRunnerCredentials(
		database.Conn(),
		now.Add(-terminatedRunnerCredentialRetryWindow),
		now,
	)
}

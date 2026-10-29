package workers

import (
	"context"
	"errors"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
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
		var releasedTask *models.RunnerTask
		err := database.Conn().Transaction(func(tx *gorm.DB) error {
			runner, err := models.LockRunner(tx, id)
			if errors.Is(err, models.ErrRunnerNotFound) {
				return nil
			}
			if err != nil {
				return err
			}
			releasedTask, err = runner.FindActiveTask(tx)
			if err != nil && !errors.Is(err, models.ErrRunnerTaskNotFound) {
				return err
			}
			if err := runner.Expire(tx, now); err != nil {
				return err
			}
			if runner.State != models.RunnerStateTerminated {
				releasedTask = nil
			}
			return nil
		})
		if err != nil {
			w.logger.WithError(err).
				WithField("runner_id", id).
				Error("Failed to expire runner registration")
			continue
		}
		if releasedTask != nil {
			messages.PublishPlanningRunnerStatus(database.DB(context.Background()), releasedTask)
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
		var lostRunningTask *models.RunnerTask
		var lostTask *models.RunnerTask
		err := database.Conn().Transaction(func(tx *gorm.DB) error {
			runner, err := models.LockRunner(tx, staleRunner.ID)
			if errors.Is(err, models.ErrRunnerNotFound) {
				return nil
			}
			if err != nil {
				return err
			}
			lostTask, err = runner.FindActiveTask(tx)
			if err != nil && !errors.Is(err, models.ErrRunnerTaskNotFound) {
				return err
			}
			lostRunningTask, err = runner.MarkLost(tx, now.Add(-runnerConnectionLossTimeout), now)
			if runner.State != models.RunnerStateTerminated {
				lostTask = nil
			}
			if err != nil {
				return err
			}
			if lostRunningTask != nil {
				return models.MarkCodexConnectionUncertain(tx, lostRunningTask.ID, now)
			}
			return nil
		})
		if err != nil {
			w.logger.WithError(err).
				WithField("runner_id", staleRunner.ID).
				Error("Failed to mark disconnected runner as lost")
			continue
		}
		if lostTask != nil {
			messages.PublishPlanningRunnerStatus(database.DB(context.Background()), lostTask)
		}
		if lostRunningTask != nil && telemetry.MetricsEnabled() {
			telemetry.RecordRunnerTaskExecutionDuration(context.Background(), lostRunningTask, staleRunner.FleetSlug, staleRunner.FleetScope)
		}
	}
	if err := models.RevokeTerminatedRunnerCredentials(
		database.Conn(),
		now.Add(-terminatedRunnerCredentialRetryWindow),
		now,
	); err != nil {
		return err
	}
	connections, err := models.ListPendingCodexRevocations(database.DB(context.Background()))
	if err != nil {
		return err
	}
	for i := range connections {
		var results []*models.RunCancellationResult
		err := database.DB(context.Background()).Transaction(func(tx *gorm.DB) error {
			var err error
			results, err = connections[i].CleanupRevocation(tx, now)
			return err
		})
		if err != nil {
			return err
		}
		notifier := &RunCancellationNotifier{}
		notifier.RecordRuns(results)
		notifier.Publish()
		if len(results) > 0 || connections[i].RevocationPendingAt == nil {
			_ = messages.PublishFactoryCodexConnectionUpdated(connections[i].FactoryID.String())
		}
	}
	var logins []models.FactoryCodexConnection
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		var err error
		logins, err = models.ListCodexLoginConnections(tx)
		if err != nil {
			return err
		}
		return models.ReleaseConfirmedCodexReservations(tx, now)
	})
	if err != nil {
		return err
	}
	messages.PublishCodexConnectionChanges(database.DB(context.Background()), logins)
	return nil
}

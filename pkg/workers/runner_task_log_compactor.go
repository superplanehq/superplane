package workers

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	runnerlogs "github.com/superplanehq/superplane/pkg/runners/logs"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	runnerTaskLogArchivingBatchSize = 25
	runnerTaskLogProcessingLease    = 15 * time.Minute
)

type RunnerTaskLogCompactor struct {
	provider        blob.Provider
	activeStore     runnerlogs.Store
	installationID  string
	interval        time.Duration
	liveReaderGrace time.Duration
	logger          *log.Entry
}

func NewRunnerTaskLogCompactor(
	provider blob.Provider,
	activeStore runnerlogs.Store,
	installationID string,
	interval time.Duration,
	liveReaderGrace time.Duration,
) *RunnerTaskLogCompactor {
	return &RunnerTaskLogCompactor{
		provider:        provider,
		activeStore:     activeStore,
		installationID:  installationID,
		interval:        interval,
		liveReaderGrace: liveReaderGrace,
		logger:          log.WithField("worker", "RunnerTaskLogCompactor"),
	}
}

func (w *RunnerTaskLogCompactor) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.Process(ctx); err != nil {
				w.logger.WithError(err).Error("Failed to compact runner task logs")
			}
		}
	}
}

func (w *RunnerTaskLogCompactor) Process(ctx context.Context) error {
	for range runnerTaskLogArchivingBatchSize {
		now := time.Now()
		candidate, err := claimLogArchiving(
			database.DB(ctx),
			now,
			now.Add(runnerTaskLogProcessingLease),
		)
		if err != nil {
			return err
		}
		if candidate == nil {
			return nil
		}

		taskCtx, cancel := context.WithDeadline(ctx, candidate.ProcessingUntil)
		err = w.processTask(taskCtx, *candidate)
		cancel()
		if err != nil {
			releaseErr := candidate.release(database.DB(ctx), time.Now())
			w.logger.WithError(err).
				WithField("task_id", candidate.TaskID).
				Error("Failed to compact task logs")
			if releaseErr != nil {
				w.logger.WithError(releaseErr).
					WithField("task_id", candidate.TaskID).
					Error("Failed to release task log compaction claim")
			}
			return nil
		}
	}
	return nil
}

func (w *RunnerTaskLogCompactor) processTask(ctx context.Context, candidate LogArchivingCandidate) error {
	if candidate.ActiveStore != w.activeStore.Name() {
		return fmt.Errorf("active log store %q is unavailable", candidate.ActiveStore)
	}
	if candidate.State == models.RunnerTaskLogStateArchived {
		err := w.activeStore.Delete(ctx, candidate.TaskID)
		if err != nil {
			return err
		}
		return candidate.finishCleanup(database.DB(ctx), time.Now())
	}

	active, err := w.activeStore.ReadAfter(ctx, candidate.TaskID, "")
	if err != nil {
		return fmt.Errorf("read active task logs: %w", err)
	}
	defer active.Content.Close()
	if err := candidate.recordFinalCursor(database.DB(ctx), active.Cursor, time.Now()); err != nil {
		return err
	}

	key := runnerlogs.FinalKey(w.installationID, candidate.OrganizationID, candidate.TaskID)
	logSize, err := w.writeFinalObject(ctx, key, active.Content)
	if err != nil {
		return err
	}
	now := time.Now()
	if err := candidate.markArchived(
		database.DB(ctx),
		key,
		active.Truncated,
		now.Add(w.liveReaderGrace),
		now,
	); err != nil {
		return err
	}
	if candidate.FleetScopeType == models.RunnerFleetScopeInstallation {
		telemetry.RecordRunnerTaskLogSize(ctx, candidate.FleetSlug, logSize)
	}
	return nil
}

func (w *RunnerTaskLogCompactor) writeFinalObject(ctx context.Context, key string, active io.Reader) (int64, error) {
	reader, writer := io.Pipe()
	writeDone := make(chan gzipWriteResult, 1)
	go func() {
		written, err := writeGzipStream(writer, active)
		writeDone <- gzipWriteResult{written: written, err: err}
	}()

	err := w.provider.Put(
		ctx,
		key,
		reader,
		blob.PutOptions{
			ContentType:     "application/x-ndjson",
			ContentEncoding: "gzip",
		},
	)
	_ = reader.Close()
	result := <-writeDone
	if result.err != nil {
		return 0, result.err
	}
	if err != nil {
		return 0, fmt.Errorf("store final log object: %w", err)
	}
	return result.written, nil
}

type gzipWriteResult struct {
	written int64
	err     error
}

func writeGzipStream(writer *io.PipeWriter, active io.Reader) (int64, error) {
	gzipWriter := gzip.NewWriter(writer)
	written, err := io.Copy(gzipWriter, active)
	if err != nil {
		_ = writer.CloseWithError(err)
		return 0, fmt.Errorf("copy active task log: %w", err)
	}
	if err := gzipWriter.Close(); err != nil {
		_ = writer.CloseWithError(err)
		return 0, fmt.Errorf("close gzip stream: %w", err)
	}
	if err := writer.Close(); err != nil {
		return 0, fmt.Errorf("close final log stream: %w", err)
	}
	return written, nil
}

type LogArchivingCandidate struct {
	TaskID          uuid.UUID
	OrganizationID  uuid.UUID
	FleetSlug       string
	FleetScopeType  string
	ActiveStore     string
	State           string
	ProcessingUntil time.Time
}

/*
 * claimLogArchiving holds a row lock only while it records a lease.
 */
func claimLogArchiving(tx *gorm.DB, now, processingUntil time.Time) (*LogArchivingCandidate, error) {
	var candidate LogArchivingCandidate
	err := tx.Transaction(func(tx *gorm.DB) error {
		err := tx.Table("runner_task_log_lifecycles AS lifecycles").
			Select(
				"lifecycles.task_id, tasks.organization_id, "+
					"fleets.slug AS fleet_slug, fleets.scope_type AS fleet_scope_type, "+
					"lifecycles.active_store, lifecycles.state",
			).
			Joins("JOIN runner_tasks AS tasks ON tasks.id = lifecycles.task_id").
			Joins("JOIN runner_fleets AS fleets ON fleets.id = tasks.fleet_id").
			Clauses(clause.Locking{
				Strength: "UPDATE",
				Table:    clause.Table{Name: "lifecycles"},
				Options:  "SKIP LOCKED",
			}).
			Where("lifecycles.state IN ?", []string{
				models.RunnerTaskLogStateArchivable,
				models.RunnerTaskLogStateArchiving,
				models.RunnerTaskLogStateArchived,
			}).
			Where("lifecycles.processing_until IS NULL OR lifecycles.processing_until <= ?", now).
			Where(
				"lifecycles.state <> ? OR lifecycles.cleanup_after <= ?",
				models.RunnerTaskLogStateArchived,
				now,
			).
			Order("lifecycles.updated_at ASC").
			Take(&candidate).
			Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}

		updates := map[string]any{
			"processing_until": processingUntil,
			"updated_at":       now,
		}
		if candidate.State == models.RunnerTaskLogStateArchivable {
			updates["state"] = models.RunnerTaskLogStateArchiving
		}
		result := tx.Model(&models.RunnerTaskLogLifecycle{}).
			Where("task_id = ?", candidate.TaskID).
			Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return models.ErrTaskLogLifecycleNotFound
		}
		candidate.ProcessingUntil = processingUntil
		if candidate.State == models.RunnerTaskLogStateArchivable {
			candidate.State = models.RunnerTaskLogStateArchiving
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if candidate.TaskID == uuid.Nil {
		return nil, nil
	}
	return &candidate, nil
}

func (f *LogArchivingCandidate) recordFinalCursor(tx *gorm.DB, finalCursor string, now time.Time) error {
	result := tx.Model(&models.RunnerTaskLogLifecycle{}).
		Where(
			"task_id = ? AND processing_until = ? AND state = ?",
			f.TaskID,
			f.ProcessingUntil,
			models.RunnerTaskLogStateArchiving,
		).
		Updates(map[string]any{
			"final_cursor": finalCursor,
			"updated_at":   now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return models.ErrTaskLogLifecycleNotFound
	}
	return nil
}

func (f *LogArchivingCandidate) markArchived(
	tx *gorm.DB,
	finalObjectKey string,
	truncated bool,
	cleanupAfter, now time.Time,
) error {
	result := tx.Model(&models.RunnerTaskLogLifecycle{}).
		Where(
			"task_id = ? AND processing_until = ? AND state = ?",
			f.TaskID,
			f.ProcessingUntil,
			models.RunnerTaskLogStateArchiving,
		).
		Updates(map[string]any{
			"state":            models.RunnerTaskLogStateArchived,
			"final_object_key": finalObjectKey,
			"truncated":        truncated,
			"cleanup_after":    cleanupAfter,
			"processing_until": nil,
			"updated_at":       now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return models.ErrTaskLogLifecycleNotFound
	}
	f.State = models.RunnerTaskLogStateArchived
	return nil
}

func (f *LogArchivingCandidate) finishCleanup(tx *gorm.DB, now time.Time) error {
	result := tx.Model(&models.RunnerTaskLogLifecycle{}).
		Where(
			"task_id = ? AND processing_until = ? AND state = ?",
			f.TaskID,
			f.ProcessingUntil,
			models.RunnerTaskLogStateArchived,
		).
		Updates(map[string]any{
			"processing_until": nil,
			"cleanup_after":    nil,
			"updated_at":       now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return models.ErrTaskLogLifecycleNotFound
	}
	f.ProcessingUntil = time.Time{}
	return nil
}

func (f *LogArchivingCandidate) release(tx *gorm.DB, now time.Time) error {
	return tx.Model(&models.RunnerTaskLogLifecycle{}).
		Where("task_id = ? AND processing_until = ?", f.TaskID, f.ProcessingUntil).
		Updates(map[string]any{
			"processing_until": nil,
			"updated_at":       now,
		}).
		Error
}

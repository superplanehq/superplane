package workers

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	runnerlogs "github.com/superplanehq/superplane/pkg/runners/logs"
)

const (
	runnerTaskLogFinalizationBatchSize = 25
	runnerTaskLogProcessingLease       = 15 * time.Minute
)

type RunnerTaskLogCompactor struct {
	provider        blob.Provider
	interval        time.Duration
	liveReaderGrace time.Duration
	logger          *log.Entry
}

func NewRunnerTaskLogCompactor(
	provider blob.Provider,
	interval time.Duration,
	liveReaderGrace time.Duration,
) *RunnerTaskLogCompactor {
	return &RunnerTaskLogCompactor{
		provider:        provider,
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
	for range runnerTaskLogFinalizationBatchSize {
		now := time.Now()
		candidate, err := models.ClaimTaskLogFinalization(
			database.Conn(),
			now,
			now.Add(-w.liveReaderGrace),
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
			releaseErr := models.ReleaseTaskLogFinalization(
				database.Conn(),
				candidate.TaskID,
				candidate.ProcessingUntil,
				time.Now(),
			)
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

func (w *RunnerTaskLogCompactor) processTask(ctx context.Context, candidate models.TaskLogFinalization) error {
	if candidate.FinalizingAt == nil {
		if err := w.writeFinalObject(ctx, candidate); err != nil {
			return err
		}
		return models.MarkTaskLogFinalizing(
			database.Conn(),
			candidate.TaskID,
			candidate.ProcessingUntil,
			time.Now(),
		)
	}

	for sequence := int64(0); sequence < candidate.NextChunkSequence; sequence++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := w.provider.Delete(
			ctx,
			runnerlogs.ChunkKey(candidate.OrganizationID, candidate.TaskID, sequence),
		); err != nil {
			return fmt.Errorf("delete chunk %d: %w", sequence, err)
		}
	}
	return models.DeleteClaimedTaskLogUpload(
		database.Conn(),
		candidate.TaskID,
		candidate.ProcessingUntil,
	)
}

func (w *RunnerTaskLogCompactor) writeFinalObject(ctx context.Context, candidate models.TaskLogFinalization) error {
	reader, writer := io.Pipe()
	writeDone := make(chan error, 1)
	go func() {
		writeDone <- w.writeGzipStream(ctx, writer, candidate)
	}()

	err := w.provider.Put(
		ctx,
		runnerlogs.FinalKey(candidate.OrganizationID, candidate.TaskID),
		reader,
		blob.PutOptions{
			ContentType:     "application/x-ndjson",
			ContentEncoding: "gzip",
		},
	)
	_ = reader.Close()
	writeErr := <-writeDone
	if writeErr != nil {
		return writeErr
	}
	if err != nil {
		return fmt.Errorf("store final log object: %w", err)
	}
	return nil
}

func (w *RunnerTaskLogCompactor) writeGzipStream(ctx context.Context, writer *io.PipeWriter, candidate models.TaskLogFinalization) error {
	gzipWriter := gzip.NewWriter(writer)
	for sequence := int64(0); sequence < candidate.NextChunkSequence; sequence++ {
		chunk, err := w.provider.Get(
			ctx,
			runnerlogs.ChunkKey(candidate.OrganizationID, candidate.TaskID, sequence),
		)
		if err != nil {
			_ = writer.CloseWithError(err)
			return fmt.Errorf("read chunk %d: %w", sequence, err)
		}
		_, copyErr := io.Copy(gzipWriter, chunk)
		closeErr := chunk.Close()
		if copyErr != nil {
			_ = writer.CloseWithError(copyErr)
			return fmt.Errorf("copy chunk %d: %w", sequence, copyErr)
		}
		if closeErr != nil {
			_ = writer.CloseWithError(closeErr)
			return fmt.Errorf("close chunk %d: %w", sequence, closeErr)
		}
	}
	if err := gzipWriter.Close(); err != nil {
		_ = writer.CloseWithError(err)
		return fmt.Errorf("close gzip stream: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close final log stream: %w", err)
	}
	return nil
}

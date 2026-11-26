package workers

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"golang.org/x/sync/semaphore"
)

type NodeRequestWorker struct {
	executor  *NodeRequestExecutor
	semaphore *semaphore.Weighted
	logger    *log.Entry
}

func NewNodeRequestWorker(executor *NodeRequestExecutor) *NodeRequestWorker {
	return &NodeRequestWorker{
		executor:  executor,
		semaphore: semaphore.NewWeighted(25),
		logger:    log.WithField("worker", "NodeRequestWorker"),
	}
}

func (w *NodeRequestWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tickStart := time.Now()

			requests, err := models.ListNodeRequests()
			if err != nil {
				w.logger.Errorf("Error finding workflow nodes ready to be processed: %v", err)
			}

			checksRequests, checksErr := models.ListDuePullRequestChecksStartRequests(database.DB(ctx))
			if checksErr != nil {
				w.logger.WithError(checksErr).Error("Error finding checks startup requests")
			}
			requests = append(requests, checksRequests...)

			telemetry.RecordNodeRequestWorkerRequestsCount(context.Background(), len(requests))

			for _, request := range requests {
				if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
					w.logger.Errorf("Error acquiring semaphore: %v", err)
					continue
				}

				go func(request models.CanvasNodeRequest) {
					defer w.semaphore.Release(1)

					if err := w.executor.Execute(request); err != nil {
						w.logger.Errorf("Error processing request %s: %v", request.ID, err)
					}
				}(request)
			}

			telemetry.RecordNodeRequestWorkerTickDuration(context.Background(), time.Since(tickStart))
		}
	}
}

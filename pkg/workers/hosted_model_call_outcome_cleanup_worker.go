package workers

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	hostedModelCallOutcomeCleanupEvery = time.Minute
	hostedModelCallOutcomeRetention    = 25 * time.Hour
)

type HostedModelCallOutcomeCleanupWorker struct {
	logger *log.Entry
	now    func() time.Time
}

func NewHostedModelCallOutcomeCleanupWorker() *HostedModelCallOutcomeCleanupWorker {
	return &HostedModelCallOutcomeCleanupWorker{
		logger: log.WithFields(log.Fields{"worker": "HostedModelCallOutcomeCleanupWorker"}),
		now:    time.Now,
	}
}

func (w *HostedModelCallOutcomeCleanupWorker) Start(ctx context.Context) {
	w.tick(ctx)

	ticker := time.NewTicker(hostedModelCallOutcomeCleanupEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *HostedModelCallOutcomeCleanupWorker) tick(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}

	before := w.now().Add(-hostedModelCallOutcomeRetention)
	deleted, err := models.DeleteExpiredHostedModelCallOutcomes(database.DB(ctx), before)
	if err != nil {
		w.logger.WithError(err).Error("failed to delete expired hosted model call outcomes")
		return
	}
	if deleted == 0 {
		return
	}
	w.logger.WithField("deleted", deleted).Info("Deleted expired hosted model call outcomes")
}

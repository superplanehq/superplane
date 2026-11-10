package workers

import (
	"context"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/factories"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	humanReviewReconcileEvery = time.Minute
	humanReviewBatchSize      = 100
)

// FactoryHumanReviewWorker repairs missed webhooks and unfinished external
// publications. It runs with the existing node-request worker configuration.
type FactoryHumanReviewWorker struct {
	deps    factories.IntakeDependencies
	afterID uuid.UUID
}

func NewFactoryHumanReviewWorker(encryptor crypto.Encryptor, registry *registry.Registry) *FactoryHumanReviewWorker {
	return &FactoryHumanReviewWorker{deps: factories.IntakeDependencies{Encryptor: encryptor, Registry: registry}}
}

func (w *FactoryHumanReviewWorker) Start(ctx context.Context) {
	w.reconcile(ctx)
	ticker := time.NewTicker(humanReviewReconcileEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.reconcile(ctx)
		}
	}
}

func (w *FactoryHumanReviewWorker) reconcile(ctx context.Context) {
	pullRequests, err := models.ListFactoryHumanReviewPullRequestsToReconcile(database.DB(ctx), w.afterID, humanReviewBatchSize)
	if err != nil {
		log.WithError(err).Error("Human review: failed to list tracked pull requests")
		return
	}
	for _, pr := range pullRequests {
		if ctx.Err() != nil {
			return
		}
		if !factories.ScheduleFactoryHumanReview(ctx, w.deps, pr) {
			return
		}
		w.afterID = pr.ID
	}
	if len(pullRequests) < humanReviewBatchSize {
		w.afterID = uuid.Nil
	}
}

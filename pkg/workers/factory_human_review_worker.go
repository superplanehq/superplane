package workers

import (
	"context"
	"golang.org/x/sync/errgroup"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/factories"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

const humanReviewReconcileEvery = time.Minute

// FactoryHumanReviewWorker repairs missed webhooks and unfinished external
// publications. It runs with the existing node-request worker configuration.
type FactoryHumanReviewWorker struct{ deps factories.IntakeDependencies }

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
	pullRequests, err := models.ListOpenFactoryHumanReviewPullRequests(database.DB(ctx))
	if err != nil {
		log.WithError(err).Error("Human review: failed to list tracked pull requests")
		return
	}
	var group errgroup.Group
	group.SetLimit(4)
	for _, pr := range pullRequests {
		if ctx.Err() != nil {
			break
		}
		group.Go(func() error {
			if err := factories.ReconcileFactoryHumanReview(ctx, w.deps, pr); err != nil {
				log.WithError(err).WithField("pull_request", pr.ID).Warn("Human review reconciliation failed")
			}
			return nil
		})
	}
	_ = group.Wait()
}

package workers

import (
	"context"
	"errors"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/usage/pricebooksync"
)

const priceBookSyncEvery = 24 * time.Hour

// PriceBookSyncWorker keeps model rates current from provider catalogs.
type PriceBookSyncWorker struct {
	sync   *pricebooksync.Service
	logger *log.Entry
}

// NewPriceBookSyncWorker builds a worker that syncs OpenRouter catalog prices.
func NewPriceBookSyncWorker(encryptor crypto.Encryptor, reg *registry.Registry) *PriceBookSyncWorker {
	return &PriceBookSyncWorker{
		sync:   pricebooksync.New(encryptor, reg.HTTPContext()),
		logger: log.WithFields(log.Fields{"worker": "PriceBookSyncWorker"}),
	}
}

func (w *PriceBookSyncWorker) Start(ctx context.Context) {
	w.tick(ctx)

	ticker := time.NewTicker(priceBookSyncEvery)
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

func (w *PriceBookSyncWorker) tick(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}

	startedAt := time.Now()
	result, err := w.sync.Sync(ctx, database.Conn(), pricebooksync.Options{SkipWhenUnchanged: true})
	if errors.Is(err, pricebooksync.ErrNoPricedCatalogProvider) {
		w.logger.Info("Price book sync skipped because no enabled provider publishes catalog prices")
		return
	}
	if err != nil {
		w.logger.Errorf("Price book sync failed: %v", err)
		return
	}
	if !result.Published {
		w.logger.WithField("duration", time.Since(startedAt).String()).Info("Price book sync skipped because catalog rates are unchanged")
		return
	}

	w.logger.WithFields(log.Fields{
		"version":  result.Book.Version,
		"updated":  result.UpdatedCount,
		"added":    result.AddedCount,
		"duration": time.Since(startedAt).String(),
	}).Info("Published synced price book")
}

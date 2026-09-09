package workers

import (
	"context"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/usage/pricesync"
)

const defaultPriceBookSyncEvery = 24 * time.Hour

type PriceBookSyncWorker struct {
	logger   *log.Entry
	interval time.Duration
	scan     func(ctx context.Context, tx *gorm.DB) (*pricesync.Result, error)
}

func NewPriceBookSyncWorker(httpClient core.HTTPContext) *PriceBookSyncWorker {
	return &PriceBookSyncWorker{
		logger:   log.WithFields(log.Fields{"worker": "PriceBookSyncWorker"}),
		interval: priceBookSyncInterval(),
		scan: func(ctx context.Context, tx *gorm.DB) (*pricesync.Result, error) {
			return pricesync.ScanAndPublish(ctx, tx, pricesync.NewScanner(pricesync.Options{HTTP: httpClient}))
		},
	}
}

func (w *PriceBookSyncWorker) Start(ctx context.Context) {
	w.tick(ctx)

	ticker := time.NewTicker(w.interval)
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
	result, err := w.scan(ctx, database.DB(ctx))
	if err != nil {
		w.logger.Errorf("Price book scan failed: %v", err)
		return
	}

	w.logger.WithFields(log.Fields{
		"version":       result.Version,
		"previous":      result.PreviousVersion,
		"model_rates":   result.ModelRateCount,
		"compute_rates": result.ComputeRateCount,
		"sources":       result.Sources,
		"duration":      time.Since(startedAt).String(),
	}).Info("Price book scan published a new catalog")
}

func priceBookSyncInterval() time.Duration {
	raw := os.Getenv("PRICE_BOOK_SYNC_EVERY")
	if raw == "" {
		return defaultPriceBookSyncEvery
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed <= 0 {
		return defaultPriceBookSyncEvery
	}
	return parsed
}

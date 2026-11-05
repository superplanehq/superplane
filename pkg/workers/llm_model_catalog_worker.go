package workers

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/llmcatalog"
)

const (
	llmModelCatalogRefreshEvery   = 24 * time.Hour
	llmModelCatalogRefreshTimeout = 5 * time.Minute
)

// LLMModelCatalogWorker keeps provider model capabilities current.
type LLMModelCatalogWorker struct {
	catalog *llmcatalog.Service
	logger  *log.Entry
}

func NewLLMModelCatalogWorker(catalog *llmcatalog.Service) *LLMModelCatalogWorker {
	return &LLMModelCatalogWorker{
		catalog: catalog,
		logger:  log.WithField("worker", "LLMModelCatalogWorker"),
	}
}

func (w *LLMModelCatalogWorker) Start(ctx context.Context) {
	w.refresh(ctx)

	ticker := time.NewTicker(llmModelCatalogRefreshEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.refresh(ctx)
		}
	}
}

func (w *LLMModelCatalogWorker) refresh(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, llmModelCatalogRefreshTimeout)
	defer cancel()

	startedAt := time.Now()
	if err := w.catalog.RefreshConfigured(ctx, database.Conn().WithContext(ctx)); err != nil {
		w.logger.WithError(err).Error("Failed to refresh the LLM model catalog")
		return
	}
	w.logger.WithField("duration", time.Since(startedAt).String()).Info("Refreshed the LLM model catalog")
}

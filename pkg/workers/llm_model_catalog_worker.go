package workers

import (
	"context"
	"errors"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/llmcatalog"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

const (
	llmModelCatalogRefreshEvery   = 24 * time.Hour
	llmModelCatalogRefreshTimeout = 5 * time.Minute
)

// LLMModelCatalogWorker keeps provider model capabilities current.
type LLMModelCatalogWorker struct {
	catalog  *llmcatalog.Service
	registry *registry.Registry
	logger   *log.Entry
}

func NewLLMModelCatalogWorker(catalog *llmcatalog.Service, reg *registry.Registry) *LLMModelCatalogWorker {
	return &LLMModelCatalogWorker{
		catalog:  catalog,
		registry: reg,
		logger:   log.WithField("worker", "LLMModelCatalogWorker"),
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
	tx := database.DB(ctx)
	err := errors.Join(w.catalog.RefreshConfigured(ctx, tx), w.refreshClaudeIntegrations(ctx, tx))
	if err != nil {
		w.logger.WithError(err).Error("Failed to refresh the LLM model catalog")
		return
	}
	w.logger.WithField("duration", time.Since(startedAt).String()).Info("Refreshed the LLM model catalog")
}

func (w *LLMModelCatalogWorker) refreshClaudeIntegrations(ctx context.Context, tx *gorm.DB) error {
	integrations, err := models.ListClaudeCatalogIntegrations(tx)
	if err != nil {
		return err
	}
	var refreshErrors []error
	for i := range integrations {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(refreshErrors, err)...)
		}
		integration := &integrations[i]
		integrationCtx := contexts.NewIntegrationContext(tx, nil, integration, w.registry.Encryptor, w.registry, nil)
		if err := w.catalog.RefreshAnthropicIntegration(ctx, tx, integrationCtx); err != nil {
			refreshErrors = append(refreshErrors, fmt.Errorf("refresh Claude integration %s: %w", integration.ID, err))
		}
	}
	return errors.Join(refreshErrors...)
}

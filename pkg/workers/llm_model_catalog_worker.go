package workers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
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
	after    uuid.UUID
}

func NewLLMModelCatalogWorker(catalog *llmcatalog.Service, reg *registry.Registry) *LLMModelCatalogWorker {
	return &LLMModelCatalogWorker{
		catalog:  catalog,
		registry: reg,
		logger:   log.WithField("worker", "LLMModelCatalogWorker"),
	}
}

func (w *LLMModelCatalogWorker) Start(ctx context.Context) {
	for {
		w.refresh(ctx)
		delay := llmModelCatalogRefreshEvery
		// Continue bounded scans promptly instead of waiting a day after each window.
		if w.after != uuid.Nil {
			delay = time.Minute
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
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
	integrations, err := models.ListClaudeCatalogIntegrations(tx, w.after)
	if err != nil {
		return err
	}
	var refreshErrors []error
	for i := range integrations {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(refreshErrors, err)...)
		}
		integration := &integrations[i]
		// Advance before attempting credentials or network I/O, including failures.
		w.after = integration.ID
		integrationCtx := contexts.NewIntegrationContext(tx, nil, integration, w.registry.Encryptor, w.registry, nil)
		if err := w.catalog.RefreshAnthropicIntegration(ctx, tx, integrationCtx); err != nil {
			refreshErrors = append(refreshErrors, fmt.Errorf("refresh Claude integration %s: %w", integration.ID, err))
		}
	}
	if len(integrations) < models.LLMCatalogIntegrationBatchSize {
		w.after = uuid.Nil
	}
	return errors.Join(refreshErrors...)
}

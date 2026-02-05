package workers

import (
	"context"
	"fmt"
	"log"
	"time"

	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/oidc"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

type IntegrationRequestWorker struct {
	semaphore       *semaphore.Weighted
	registry        *registry.Registry
	encryptor       crypto.Encryptor
	oidcProvider    oidc.Provider
	baseURL         string
	webhooksBaseURL string
}

func NewIntegrationRequestWorker(encryptor crypto.Encryptor, registry *registry.Registry, oidcProvider oidc.Provider, baseURL string, webhooksBaseURL string) *IntegrationRequestWorker {
	return &IntegrationRequestWorker{
		encryptor:       encryptor,
		registry:        registry,
		oidcProvider:    oidcProvider,
		baseURL:         baseURL,
		webhooksBaseURL: webhooksBaseURL,
		semaphore:       semaphore.NewWeighted(25),
	}
}

func (w *IntegrationRequestWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			requests, err := models.ListIntegrationRequests()
			if err != nil {
				w.log("Error finding app installation requests: %v", err)
			}

			for _, request := range requests {
				if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
					w.log("Error acquiring semaphore: %v", err)
					continue
				}

				go func(request models.IntegrationRequest) {
					defer w.semaphore.Release(1)

					if err := w.LockAndProcessRequest(request); err != nil {
						w.log("Error processing request %s: %v", request.ID, err)
					}
				}(request)
			}
		}
	}
}

func (w *IntegrationRequestWorker) LockAndProcessRequest(request models.IntegrationRequest) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		r, err := models.LockIntegrationRequest(tx, request.ID)
		if err != nil {
			w.log("Request %s already being processed - skipping", request.ID)
			return nil
		}

		return w.processRequest(tx, r)
	})
}

func (w *IntegrationRequestWorker) processRequest(tx *gorm.DB, request *models.IntegrationRequest) error {
	switch request.Type {
	case models.IntegrationRequestTypeSync:
		return w.syncIntegration(tx, request)
	case models.IntegrationRequestTypeInvokeAction:
		return w.invokeIntegrationAction(tx, request)
	}

	return fmt.Errorf("unsupported integration request type %s", request.Type)
}

func (w *IntegrationRequestWorker) syncIntegration(tx *gorm.DB, request *models.IntegrationRequest) error {
	instance, err := models.FindUnscopedIntegrationInTransaction(tx, request.AppInstallationID)
	if err != nil {
		return fmt.Errorf("failed to find integration: %v", err)
	}

	integration, err := w.registry.GetIntegration(instance.AppName)
	if err != nil {
		return fmt.Errorf("integration %s not found", instance.AppName)
	}

	integrationCtx := contexts.NewIntegrationContext(tx, nil, instance, w.encryptor, w.registry)
	syncErr := integration.Sync(core.SyncContext{
		Logger:          logging.ForIntegration(*instance),
		HTTP:            w.registry.HTTPContext(),
		Integration:     integrationCtx,
		Configuration:   instance.Configuration.Data(),
		BaseURL:         w.baseURL,
		WebhooksBaseURL: w.webhooksBaseURL,
		OrganizationID:  instance.OrganizationID.String(),
		OIDC:            w.oidcProvider,
	})

	if syncErr != nil {
		instance.State = models.IntegrationStateError
		instance.StateDescription = fmt.Sprintf("Sync failed: %v", syncErr)
	} else {
		instance.StateDescription = ""
	}

	if err := tx.Save(instance).Error; err != nil {
		return fmt.Errorf("failed to save integration after sync: %v", err)
	}

	return request.Complete(tx)
}

func (w *IntegrationRequestWorker) invokeIntegrationAction(tx *gorm.DB, request *models.IntegrationRequest) error {
	integration, err := models.FindUnscopedIntegrationInTransaction(tx, request.AppInstallationID)
	if err != nil {
		return fmt.Errorf("failed to find app installation: %v", err)
	}

	integrationImpl, err := w.registry.GetIntegration(integration.AppName)
	if err != nil {
		return fmt.Errorf("integration %s not found", integration.AppName)
	}

	spec := request.Spec.Data()
	logger := logging.ForIntegration(*integration)
	integrationCtx := contexts.NewIntegrationContext(tx, nil, integration, w.encryptor, w.registry)
	actionCtx := core.IntegrationActionContext{
		WebhooksBaseURL: w.webhooksBaseURL,
		Name:            spec.InvokeAction.ActionName,
		Parameters:      spec.InvokeAction.Parameters,
		Configuration:   integration.Configuration.Data(),
		Logger:          logger,
		Integration:     integrationCtx,
		HTTP:            w.registry.HTTPContext(),
	}

	err = integrationImpl.HandleAction(actionCtx)
	if err != nil {
		logger.Errorf("error handling action: %v", err)
	}

	err = tx.Save(integration).Error
	if err != nil {
		logger.Errorf("failed to save integration %s: %v", integration.ID, err)
		return fmt.Errorf("failed to save integration: %w", err)
	}

	return request.Complete(tx)
}

func (w *IntegrationRequestWorker) log(format string, v ...any) {
	log.Printf("[IntegrationRequestWorker] "+format, v...)
}

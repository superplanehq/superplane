package workers

import (
	"context"
	"errors"
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

// requestLeaseDuration is how far into the future a claimed request's run_at is
// pushed while it is processed. It is simultaneously (a) the worst-case recovery
// latency for a request whose worker died mid-process (it becomes due again once
// the lease expires) and (b) a lower bound that must exceed the longest expected
// Sync/HandleHook duration, so a request still legitimately in flight is never
// re-leased by another worker.
const requestLeaseDuration = 5 * time.Minute

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

// LockAndProcessRequest processes a request in 3 phases so the external
// HTTP call (Sync/HandleHook) never holds a DB transaction open:
//
//   - Phase 1 (short tx): Lease the request by pushing run_at past the work window
//   - Phase 2 (no tx): Run integration.Sync / HandleHook with a non-transactional context
//   - Phase 3 (short tx): Persist instance state and mark the request completed
func (w *IntegrationRequestWorker) LockAndProcessRequest(request models.IntegrationRequest) error {
	claimed, err := w.claimRequest(request)
	if err != nil {
		return err
	}
	if claimed == nil {
		// Already claimed by another worker, or no longer due.
		return nil
	}

	return w.processRequest(claimed)
}

// claimRequest leases the request in a short transaction by pushing its run_at
// past the work window, so the poll loop (which only lists due pending requests)
// will not pick it up again while the external work runs outside this transaction.
// Returns nil if the request was already picked up by another worker.
func (w *IntegrationRequestWorker) claimRequest(request models.IntegrationRequest) (*models.IntegrationRequest, error) {
	var claimed *models.IntegrationRequest

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		r, err := models.LeaseIntegrationRequest(tx, request.ID, requestLeaseDuration)
		if err != nil {
			return err
		}

		claimed = r
		return nil
	})

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			w.log("Request %s already being processed - skipping", request.ID)
			return nil, nil
		}
		return nil, fmt.Errorf("failed to lease request %s: %w", request.ID, err)
	}

	return claimed, nil
}

func (w *IntegrationRequestWorker) processRequest(request *models.IntegrationRequest) error {
	switch request.Type {
	case models.IntegrationRequestTypeSync:
		return w.syncIntegration(request)
	case models.IntegrationRequestTypeInvokeAction:
		return w.invokeIntegrationAction(request)
	}

	return fmt.Errorf("unsupported integration request type %s", request.Type)
}

func (w *IntegrationRequestWorker) syncIntegration(request *models.IntegrationRequest) error {
	db := database.Conn()

	instance, err := models.FindUnscopedIntegrationInTransaction(db, request.AppInstallationID)
	if err != nil {
		return fmt.Errorf("failed to find integration: %v", err)
	}

	integration, err := w.registry.GetIntegration(instance.AppName)
	if err != nil {
		return fmt.Errorf("integration %s not found", instance.AppName)
	}

	//
	// Phase 2: run Sync outside any transaction so the external HTTP call does
	// not hold a DB connection/row lock. Secret and request writes during Sync
	// go through a non-transactional context (database.Conn()).
	//
	integrationCtx := contexts.NewIntegrationContext(db, nil, instance, w.encryptor, w.registry, nil)
	logging.ForIntegration(*instance).WithField("source", "sync").Info("Integration operation may write secrets")
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

	//
	// Phase 3: persist instance state and complete the claimed request.
	//
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(instance).Error; err != nil {
			return fmt.Errorf("failed to save integration after sync: %v", err)
		}

		return request.Complete(tx)
	})
}

func (w *IntegrationRequestWorker) invokeIntegrationAction(request *models.IntegrationRequest) error {
	db := database.Conn()

	integration, err := models.FindUnscopedIntegrationInTransaction(db, request.AppInstallationID)
	if err != nil {
		return fmt.Errorf("failed to find app installation: %v", err)
	}

	spec := request.Spec.Data()
	hookProvider, _, err := w.registry.FindIntegrationHook(integration.AppName, spec.InvokeAction.ActionName)
	if err != nil {
		return fmt.Errorf("failed to find hook: %v", err)
	}

	//
	// Phase 2: run the action hook outside any transaction.
	//
	logger := logging.ForIntegration(*integration)
	integrationCtx := contexts.NewIntegrationContext(db, nil, integration, w.encryptor, w.registry, nil)
	logger.WithField("source", "integration_action").Info("Integration operation may write secrets")
	hookCtx := core.IntegrationHookContext{
		WebhooksBaseURL: w.webhooksBaseURL,
		Name:            spec.InvokeAction.ActionName,
		Parameters:      spec.InvokeAction.Parameters,
		Configuration:   integration.Configuration.Data(),
		Logger:          logger,
		Integration:     integrationCtx,
		HTTP:            w.registry.HTTPContext(),
	}

	if err := hookProvider.HandleHook(hookCtx); err != nil {
		logger.Errorf("error handling action: %v", err)
	}

	//
	// Phase 3: persist instance state and complete the claimed request.
	//
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(integration).Error; err != nil {
			logger.Errorf("failed to save integration %s: %v", integration.ID, err)
			return fmt.Errorf("failed to save integration: %w", err)
		}

		return request.Complete(tx)
	})
}

func (w *IntegrationRequestWorker) log(format string, v ...any) {
	log.Printf("[IntegrationRequestWorker] "+format, v...)
}

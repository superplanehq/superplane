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
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/oidc"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

type AppInstallationRequestWorker struct {
	semaphore       *semaphore.Weighted
	registry        *registry.Registry
	encryptor       crypto.Encryptor
	oidcProvider    oidc.Provider
	baseURL         string
	webhooksBaseURL string
}

func NewAppInstallationRequestWorker(encryptor crypto.Encryptor, registry *registry.Registry, oidcProvider oidc.Provider, baseURL string, webhooksBaseURL string) *AppInstallationRequestWorker {
	return &AppInstallationRequestWorker{
		encryptor:       encryptor,
		registry:        registry,
		oidcProvider:    oidcProvider,
		baseURL:         baseURL,
		webhooksBaseURL: webhooksBaseURL,
		semaphore:       semaphore.NewWeighted(25),
	}
}

func (w *AppInstallationRequestWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			requests, err := models.ListAppInstallationRequests()
			if err != nil {
				w.log("Error finding app installation requests: %v", err)
			}

			for _, request := range requests {
				if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
					w.log("Error acquiring semaphore: %v", err)
					continue
				}

				go func(request models.AppInstallationRequest) {
					defer w.semaphore.Release(1)

					if err := w.LockAndProcessRequest(request); err != nil {
						w.log("Error processing request %s: %v", request.ID, err)
					}
				}(request)
			}
		}
	}
}

func (w *AppInstallationRequestWorker) LockAndProcessRequest(request models.AppInstallationRequest) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		r, err := models.LockAppInstallationRequest(tx, request.ID)
		if err != nil {
			w.log("Request %s already being processed - skipping", request.ID)
			return nil
		}

		return w.processRequest(tx, r)
	})
}

func (w *AppInstallationRequestWorker) processRequest(tx *gorm.DB, request *models.AppInstallationRequest) error {
	switch request.Type {
	case models.AppInstallationRequestTypeSync:
		return w.syncAppInstallation(tx, request)
	}

	return fmt.Errorf("unsupported app installation request type %s", request.Type)
}

func (w *AppInstallationRequestWorker) syncAppInstallation(tx *gorm.DB, request *models.AppInstallationRequest) error {
	installation, err := models.FindUnscopedAppInstallationInTransaction(tx, request.AppInstallationID)
	if err != nil {
		return fmt.Errorf("failed to find app installation: %v", err)
	}

	app, err := w.registry.GetApplication(installation.AppName)
	if err != nil {
		return fmt.Errorf("application %s not found", installation.AppName)
	}

	appCtx := contexts.NewAppInstallationContext(tx, nil, installation, w.encryptor, w.registry)
	syncErr := app.Sync(core.SyncContext{
		HTTP:            contexts.NewHTTPContext(w.registry.GetHTTPClient()),
		AppInstallation: appCtx,
		Configuration:   installation.Configuration.Data(),
		BaseURL:         w.baseURL,
		WebhooksBaseURL: w.webhooksBaseURL,
		OrganizationID:  installation.OrganizationID.String(),
		InstallationID:  installation.ID.String(),
		OIDC:            w.oidcProvider,
	})

	if syncErr != nil {
		installation.State = models.AppInstallationStateError
		installation.StateDescription = fmt.Sprintf("Sync failed: %v", syncErr)
	} else {
		installation.StateDescription = ""
	}

	if err := tx.Save(installation).Error; err != nil {
		return fmt.Errorf("failed to save app installation after sync: %v", err)
	}

	return request.Complete(tx)
}

func (w *AppInstallationRequestWorker) log(format string, v ...any) {
	log.Printf("[AppInstallationRequestWorker] "+format, v...)
}

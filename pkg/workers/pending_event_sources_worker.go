package workers

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

type PendingEventSourcesWorker struct {
	Encryptor crypto.Encryptor
	Registry  *registry.Registry
	BaseURL   string
}

func NewPendingEventSourcesWorker(encryptor crypto.Encryptor, registry *registry.Registry, baseURL string) (*PendingEventSourcesWorker, error) {
	return &PendingEventSourcesWorker{
		Encryptor: encryptor,
		Registry:  registry,
		BaseURL:   baseURL,
	}, nil
}

func (w *PendingEventSourcesWorker) Start() {
	for {
		err := w.Tick()
		if err != nil {
			log.Errorf("Error processing events: %v", err)
		}

		time.Sleep(15 * time.Second)
	}
}

func (w *PendingEventSourcesWorker) Tick() error {
	eventSources, err := models.ListPendingEventSources()
	if err != nil {
		return err
	}

	for _, eventSource := range eventSources {
		err := w.ProcessEventSource(eventSource)
		if err != nil {
			log.Errorf("Error processing event source %s: %v", eventSource.ID, err)
		}
	}

	return nil
}

func (w *PendingEventSourcesWorker) ProcessEventSource(eventSource models.EventSource) error {
	if eventSource.ResourceID == nil {
		log.Infof("Event source %s is not tied to any integration - skipping", eventSource.ID)
		return eventSource.UpdateState(models.EventSourceStateReady)
	}

	resource, err := models.FindResourceByID(*eventSource.ResourceID)
	if err != nil {
		return fmt.Errorf("error finding integration resource: %v", err)
	}

	integration, err := models.FindIntegrationByID(resource.IntegrationID)
	if err != nil {
		return fmt.Errorf("error finding integration: %v", err)
	}

	integrationImpl, err := w.Registry.NewResourceManager(context.Background(), integration)
	if err != nil {
		return fmt.Errorf("error creating integration: %v", err)
	}

	key, err := w.Encryptor.Decrypt(context.Background(), eventSource.Key, []byte(eventSource.ID.String()))
	if err != nil {
		return fmt.Errorf("error decrypting event source key: %v", err)
	}

	resources, err := integrationImpl.SetupWebhook(integrations.WebhookOptions{
		Resource: resource,
		URL:      fmt.Sprintf("%s/api/v1/sources/%s/semaphore", w.BaseURL, eventSource.ID.String()),
		ID:       eventSource.ID.String(),
		Key:      key,
	})

	if err != nil {
		return fmt.Errorf("error setting up event source for %s integration: %v", integration.Type, err)
	}

	//
	// Save resources and update integration state
	//
	resourceRecords := []models.Resource{}
	for _, resource := range resources {
		resourceRecords = append(resourceRecords, models.Resource{
			ExternalID:    resource.Id(),
			ResourceName:  resource.Name(),
			IntegrationID: integration.ID,
			ResourceType:  resource.Type(),
		})
	}

	return database.Conn().Transaction(func(tx *gorm.DB) error {
		err := tx.Create(&resourceRecords).Error
		if err != nil {
			return err
		}

		return eventSource.UpdateStateInTransaction(tx, models.EventSourceStateReady)
	})
}

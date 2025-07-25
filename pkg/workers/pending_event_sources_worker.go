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
	"gorm.io/gorm"
)

type PendingEventSourcesWorker struct {
	Encryptor crypto.Encryptor
	BaseURL   string
}

func NewPendingEventSourcesWorker(encryptor crypto.Encryptor, baseURL string) (*PendingEventSourcesWorker, error) {
	return &PendingEventSourcesWorker{Encryptor: encryptor, BaseURL: baseURL}, nil
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

	integrationImpl, err := integrations.NewIntegration(context.Background(), integration, w.Encryptor)
	if err != nil {
		return fmt.Errorf("error creating integration: %v", err)
	}

	key, err := w.Encryptor.Decrypt(context.Background(), eventSource.Key, []byte(eventSource.Name))
	if err != nil {
		return fmt.Errorf("error decrypting event source key: %v", err)
	}

	resources, err := integrationImpl.SetupEventSource(integrations.EventSourceOptions{
		Resource: resource,
		BaseURL:  w.BaseURL,
		ID:       eventSource.ID.String(),
		Name:     eventSource.Name,
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

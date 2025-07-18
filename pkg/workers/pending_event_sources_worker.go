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

	switch integration.Type {
	case models.IntegrationTypeSemaphore:
		return w.processSemaphoreSource(eventSource, integration, resource)
	default:
		return fmt.Errorf("integration type %s not supported", integration.Type)
	}
}

func (w *PendingEventSourcesWorker) processSemaphoreSource(eventSource models.EventSource, integration *models.Integration, resource *models.Resource) error {
	semaphore, err := integrations.NewIntegration(context.Background(), integration, w.Encryptor)
	if err != nil {
		return fmt.Errorf("error creating integration: %v", err)
	}

	now := time.Now()
	resources := []models.Resource{}

	//
	// Create Semaphore secret to store the event source key.
	//
	key, err := w.Encryptor.Decrypt(context.Background(), eventSource.Key, []byte(eventSource.Name))
	if err != nil {
		return fmt.Errorf("error decrypting event source key: %v", err)
	}

	resourceName := fmt.Sprintf("superplane-%s-%s", integration.Name, eventSource.Name)
	secret, err := w.createSemaphoreSecret(semaphore, resourceName, key)
	if err != nil {
		return fmt.Errorf("error creating semaphore secret: %v", err)
	}

	resources = append(resources, models.Resource{
		ExternalID:    secret.Id(),
		ResourceName:  secret.Name(),
		IntegrationID: integration.ID,
		ResourceType:  secret.Type(),
		CreatedAt:     &now,
	})

	//
	// Create a notification resource to receive events from Semaphore
	//
	notification, err := w.createSemaphoreNotification(semaphore, resourceName, *resource, eventSource)
	if err != nil {
		return fmt.Errorf("error creating notification: %v", err)
	}

	resources = append(resources, models.Resource{
		ExternalID:    notification.Id(),
		ResourceName:  notification.Name(),
		IntegrationID: integration.ID,
		ResourceType:  notification.Type(),
		CreatedAt:     &now,
	})

	//
	// Save resources and update integration state
	//
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		err := tx.Create(&resources).Error
		if err != nil {
			return err
		}

		return eventSource.UpdateStateInTransaction(tx, models.EventSourceStateReady)
	})
}

func (w *PendingEventSourcesWorker) createSemaphoreSecret(semaphore integrations.Integration, name string, key []byte) (integrations.Resource, error) {
	//
	// Check if secret already exists.
	//
	secret, err := semaphore.Get(integrations.ResourceTypeSecret, name)
	if err == nil {
		log.Infof("Semaphore secret %s already exists - %s", secret.Name(), secret.Id())
		return secret, nil
	}

	//
	// Secret does not exist, create it.
	//
	secret, err = semaphore.Create(integrations.ResourceTypeSecret, &integrations.Secret{
		Metadata: integrations.SecretMetadata{
			Name: name,
		},
		Data: integrations.SecretSpecData{
			EnvVars: []integrations.SecretSpecDataEnvVar{
				{
					Name:  "WEBHOOK_SECRET",
					Value: string(key),
				},
			},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("error creating secret: %v", err)
	}

	log.Infof("Created Semaphore secret %s - %s", secret.Name(), secret.Id())

	return secret, nil
}

func (w *PendingEventSourcesWorker) createSemaphoreNotification(semaphore integrations.Integration, name string, resource models.Resource, eventSource models.EventSource) (integrations.Resource, error) {
	notification, err := semaphore.Get(integrations.ResourceTypeNotification, name)
	if err == nil {
		log.Infof("Semaphore notification %s already exists - %s", notification.Name(), notification.Id())
		return notification, nil
	}

	notification, err = semaphore.Create(integrations.ResourceTypeNotification, &integrations.Notification{
		Metadata: integrations.NotificationMetadata{
			Name: name,
		},
		Spec: integrations.NotificationSpec{
			Rules: []integrations.NotificationRule{
				{
					Name: fmt.Sprintf("webhooks-for-%s", resource.Name()),
					Filter: integrations.NotificationRuleFilter{
						Branches:  []string{},
						Pipelines: []string{},
						Projects:  []string{resource.ResourceName},
						Results:   []string{},
					},
					Notify: integrations.NotificationRuleNotify{
						Webhook: integrations.NotificationNotifyWebhook{
							Endpoint: fmt.Sprintf("%s/api/v1/sources/%s/semaphore", w.BaseURL, eventSource.ID.String()),
							Secret:   name,
						},
					},
				},
			},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("error creating notification: %v", err)
	}

	log.Infof("Created Semaphore notification %s", notification.Id())

	return notification, nil
}

package workers

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type PendingIntegrationsWorker struct {
	Encryptor crypto.Encryptor
	BaseURL   string
}

func NewPendingIntegrationsWorker(encryptor crypto.Encryptor, baseURL string) (*PendingIntegrationsWorker, error) {
	return &PendingIntegrationsWorker{Encryptor: encryptor, BaseURL: baseURL}, nil
}

func (w *PendingIntegrationsWorker) Start() {
	for {
		err := w.Tick()
		if err != nil {
			log.Errorf("Error processing events: %v", err)
		}

		time.Sleep(time.Minute)
	}
}

func (w *PendingIntegrationsWorker) Tick() error {
	integrations, err := models.ListPendingIntegrations()
	if err != nil {
		return err
	}

	for _, integration := range integrations {
		err := w.ProcessIntegration(integration)
		if err != nil {
			log.Errorf("Error processing event %s: %v", integration.ID, err)
		}
	}

	return nil
}

// TODO: having this as part integrations.Integration.Init() would be the best
func (w *PendingIntegrationsWorker) ProcessIntegration(integration models.Integration) error {
	switch integration.Type {
	case models.IntegrationTypeSemaphore:
		return w.processSemaphoreIntegration(integration)
	default:
		return fmt.Errorf("integration type %s not supported", integration.Type)
	}
}

func (w *PendingIntegrationsWorker) processSemaphoreIntegration(integration models.Integration) error {
	semaphore, err := integrations.NewIntegration(context.Background(), &integration, w.Encryptor)
	if err != nil {
		return fmt.Errorf("error creating integration: %v", err)
	}

	now := time.Now()
	resources := []models.IntegrationResource{}

	//
	// Create Semaphore secret used to sign the webhooks we will receive from Semaphore.
	//
	plain, encrypted, err := w.genNewSecret(context.Background(), integration.ID.String())
	if err != nil {
		return fmt.Errorf("error generating secret key: %v", err)
	}

	secret, err := semaphore.CreateResource(integrations.ResourceTypeSecret, &integrations.Secret{
		Metadata: integrations.SecretMetadata{Name: fmt.Sprintf("superplane-integration-secret-%s", integration.ID)},
		Spec: integrations.SecretSpec{
			Data: integrations.SecretSpecData{
				EnvVars: []integrations.SecretSpecDataEnvVar{
					{
						Name:  "WEBHOOK_SECRET",
						Value: plain,
					},
				},
			},
		},
	})

	if err != nil {
		return fmt.Errorf("error creating secret: %v", err)
	}

	resources = append(resources, models.IntegrationResource{
		ID:            uuid.MustParse(secret.ID()),
		Name:          secret.Name(),
		IntegrationID: integration.ID,
		Type:          secret.Type(),
		CreatedAt:     &now,
		Data:          encrypted,
	})

	log.Infof("Created Semaphore secret %s", secret.ID())

	//
	// Create a notification resource to receive events from Semaphore
	//
	notification, err := semaphore.CreateResource(integrations.ResourceTypeNotification, &integrations.Notification{
		Metadata: integrations.NotificationMetadata{Name: fmt.Sprintf("superplane-integration-notification-%s", integration.ID)},
		Spec: integrations.NotificationSpec{
			Rules: []integrations.NotificationRule{
				{
					Name: "all",
					Filter: integrations.NotificationRuleFilter{
						Branches:  []string{},
						Pipelines: []string{},
						Projects:  []string{},
						Results:   []string{},
					},
					Notify: integrations.NotificationRuleNotify{
						Webhook: integrations.NotificationNotifyWebhook{
							URL:    fmt.Sprintf("%s/api/v1/integrations/%s/semaphore", w.BaseURL, integration.ID),
							Secret: secret.Name(),
						},
					},
				},
			},
		},
	})

	if err != nil {
		return fmt.Errorf("error creating notification: %v", err)
	}

	resources = append(resources, models.IntegrationResource{
		ID:            uuid.MustParse(notification.ID()),
		Name:          notification.Name(),
		IntegrationID: integration.ID,
		Type:          notification.Type(),
		CreatedAt:     &now,
	})

	log.Infof("Created Semaphore notification %s", notification.ID())

	//
	// Save resources and update integration state
	//
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		err := tx.Create(&resources).Error
		if err != nil {
			return err
		}

		return integration.UpdateStateInTransaction(tx, models.IntegrationStateActive)
	})
}

func (w *PendingIntegrationsWorker) genNewSecret(ctx context.Context, id string) (string, []byte, error) {
	plainKey, _ := crypto.Base64String(32)
	encrypted, err := w.Encryptor.Encrypt(ctx, []byte(plainKey), []byte(id))
	if err != nil {
		return "", nil, err
	}

	return plainKey, encrypted, nil
}

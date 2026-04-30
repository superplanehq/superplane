package semaphore

import (
	"crypto/sha256"
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type WebhookMetadata struct {
	Secret       WebhookSecretMetadata       `json:"secret"`
	Notification WebhookNotificationMetadata `json:"notification"`
}

type WebhookSecretMetadata struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type WebhookNotificationMetadata struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type WebhookConfiguration struct {
	Project string `json:"project"`
}

type SemaphoreWebhookHandler struct{}

func (h *SemaphoreWebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA := WebhookConfiguration{}
	if err := mapstructure.Decode(a, &configA); err != nil {
		return false, err
	}

	configB := WebhookConfiguration{}
	if err := mapstructure.Decode(b, &configB); err != nil {
		return false, err
	}

	return configA.Project == configB.Project, nil
}

func (h *SemaphoreWebhookHandler) Merge(current, requested any) (any, bool, error) {
	return current, false, nil
}

func (h *SemaphoreWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	configuration := WebhookConfiguration{}
	err = mapstructure.Decode(ctx.Webhook.GetConfiguration(), &configuration)
	if err != nil {
		return nil, fmt.Errorf("error decoding configuration: %v", err)
	}

	//
	// Semaphore doesn't let us use UUIDs in secret names,
	// so we sha256 the ID before creating the secret.
	//
	hash := sha256.New()
	hash.Write([]byte(ctx.Webhook.GetID()))
	suffix := fmt.Sprintf("%x", hash.Sum(nil))
	name := fmt.Sprintf("superplane-webhook-%x", suffix[:16])

	webhookSecret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return nil, fmt.Errorf("error getting webhook secret: %v", err)
	}

	//
	// Create Semaphore secret to store the event source key.
	//
	secret, err := upsertSecret(client, name, webhookSecret)
	if err != nil {
		return nil, fmt.Errorf("error creating Semaphore secret: %v", err)
	}

	//
	// Create a notification resource to receive events from Semaphore
	//
	notification, err := upsertNotification(client, name, ctx.Webhook.GetURL(), configuration.Project)
	if err != nil {
		return nil, fmt.Errorf("error creating Semaphore notification: %v", err)
	}

	return WebhookMetadata{
		Secret:       WebhookSecretMetadata{ID: secret.Metadata.ID, Name: secret.Metadata.Name},
		Notification: WebhookNotificationMetadata{ID: notification.Metadata.ID, Name: notification.Metadata.Name},
	}, nil
}

func (h *SemaphoreWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	metadata := WebhookMetadata{}
	err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata)
	if err != nil {
		return fmt.Errorf("error decoding webhook metadata: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	err = client.DeleteNotification(metadata.Notification.ID)
	if err != nil {
		return fmt.Errorf("error deleting notification: %v", err)
	}

	return client.DeleteSecret(metadata.Secret.Name)
}

func upsertSecret(client *Client, name string, key []byte) (*Secret, error) {
	//
	// Check if secret already exists.
	//
	secret, err := client.GetSecret(name)
	if err == nil {
		return secret, nil
	}

	//
	// Secret does not exist, create it.
	//
	secret, err = client.CreateWebhookSecret(name, string(key))
	if err != nil {
		return nil, fmt.Errorf("error creating secret: %v", err)
	}

	return secret, nil
}

func upsertNotification(client *Client, name, URL, project string) (*Notification, error) {
	//
	// Check if notification already exists.
	//
	notification, err := client.GetNotification(name)
	if err == nil {
		return notification, nil
	}

	//
	// Notification does not exist, create it.
	//
	notification, err = client.CreateNotification(&Notification{
		Metadata: NotificationMetadata{
			Name: name,
		},
		Spec: NotificationSpec{
			Rules: []NotificationRule{
				{
					Name: fmt.Sprintf("webhook-for-%s", project),
					Filter: NotificationRuleFilter{
						Branches:  []string{},
						Pipelines: []string{},
						Projects:  []string{project},
						Results:   []string{},
					},
					Notify: NotificationRuleNotify{
						Webhook: NotificationNotifyWebhook{
							Endpoint: URL,
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

	return notification, nil
}

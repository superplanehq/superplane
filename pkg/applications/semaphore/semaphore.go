package semaphore

import (
	"crypto/sha256"
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterApplication("semaphore", &Semaphore{})
}

type Semaphore struct{}

type Configuration struct {
	OrganizationURL string `json:"organizationUrl"`
	APIToken        string `json:"apiToken"`
}

type Metadata struct {
	Projects []string `json:"projects"`
}

func (s *Semaphore) Name() string {
	return "semaphore"
}

func (s *Semaphore) Label() string {
	return "Semaphore"
}

func (s *Semaphore) Icon() string {
	return "workflow"
}

func (s *Semaphore) Description() string {
	return "Run and react to your Semaphore workflows"
}

func (s *Semaphore) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "organizationUrl",
			Label:       "Organization URL",
			Type:        configuration.FieldTypeString,
			Description: "Semaphore organization URL",
			Placeholder: "e.g. https://superplane.semaphoreci.com",
			Required:    true,
		},
		{
			Name:        "apiToken",
			Label:       "API Token",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "Semaphore API token",
			Required:    true,
		},
	}
}

func (s *Semaphore) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("Failed to decode configuration: %v", err)
	}

	metadata := Metadata{}
	err = mapstructure.Decode(ctx.AppInstallation.GetMetadata(), &metadata)
	if err != nil {
		return fmt.Errorf("Failed to decode metadata: %v", err)
	}

	//
	// TODO: Decrypt the API token to validate it can be decrypted
	// TODO: list projects to check if credentials are correct.
	// TODO: save projects in metadata? What if they change?
	//

	ctx.AppInstallation.SetState("ready")
	return nil
}

func (s *Semaphore) HandleRequest(ctx core.HttpRequestContext) {
	// no-op
}

type WebhookConfiguration struct {
	Project string `json:"project"`
}

func (s *Semaphore) RequestWebhook(ctx core.AppInstallationContext, configuration any) error {
	config := WebhookConfiguration{}
	err := mapstructure.Decode(configuration, &config)
	if err != nil {
		return fmt.Errorf("Failed to decode configuration: %v", err)
	}

	hooks, err := ctx.ListWebhooks()
	if err != nil {
		return fmt.Errorf("Failed to list webhooks: %v", err)
	}

	for _, hook := range hooks {
		c := WebhookConfiguration{}
		err := mapstructure.Decode(hook.Configuration, &c)
		if err != nil {
			return err
		}

		if c.Project == config.Project {
			ctx.AssociateWebhook(hook.ID)
			return nil
		}
	}

	return ctx.CreateWebhook(configuration)
}

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

func (s *Semaphore) SetupWebhook(ctx core.AppInstallationContext, options core.WebhookOptions) (any, error) {
	client, err := NewClient(ctx)
	if err != nil {
		return nil, err
	}

	configuration := WebhookConfiguration{}
	err = mapstructure.Decode(options.Configuration, &configuration)
	if err != nil {
		return nil, fmt.Errorf("error decoding configuration: %v", err)
	}

	//
	// Semaphore doesn't let us use UUIDs in secret names,
	// so we sha256 the ID before creating the secret.
	//
	hash := sha256.New()
	hash.Write([]byte(options.ID))
	suffix := fmt.Sprintf("%x", hash.Sum(nil))
	name := fmt.Sprintf("superplane-webhook-%x", suffix[:16])

	//
	// Create Semaphore secret to store the event source key.
	//
	secret, err := upsertSecret(client, name, options.Secret)
	if err != nil {
		return nil, fmt.Errorf("error creating Semaphore secret: %v", err)
	}

	//
	// Create a notification resource to receive events from Semaphore
	//
	notification, err := upsertNotification(client, name, options.URL, configuration.Project)
	if err != nil {
		return nil, fmt.Errorf("error creating Semaphore notification: %v", err)
	}

	return WebhookMetadata{
		Secret:       WebhookSecretMetadata{ID: secret.Metadata.ID, Name: secret.Metadata.Name},
		Notification: WebhookNotificationMetadata{ID: notification.Metadata.ID, Name: notification.Metadata.Name},
	}, nil
}

func (s *Semaphore) CleanupWebhook(ctx core.AppInstallationContext, options core.WebhookOptions) error {
	metadata := WebhookMetadata{}
	err := mapstructure.Decode(options.Metadata, &metadata)
	if err != nil {
		return fmt.Errorf("error decoding webhook metadata: %v", err)
	}

	client, err := NewClient(ctx)
	if err != nil {
		return err
	}

	err = client.DeleteNotification(metadata.Notification.ID)
	if err != nil {
		return fmt.Errorf("error deleting notification: %v", err)
	}

	return client.DeleteSecret(metadata.Secret.Name)
}

func (s *Semaphore) Components() []core.Component {
	return []core.Component{
		&RunWorkflow{},
	}
}

func (s *Semaphore) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnPipelineDone{},
	}
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

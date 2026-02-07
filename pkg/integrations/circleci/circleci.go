package circleci

import (
	"crypto/sha256"
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("circleci", &CircleCI{})
}

type CircleCI struct{}

type Configuration struct {
	APIToken string `json:"apiToken"`
}

type Metadata struct {
	Projects []string `json:"projects"`
}

func (c *CircleCI) Name() string {
	return "circleci"
}

func (c *CircleCI) Label() string {
	return "CircleCI"
}

func (c *CircleCI) Icon() string {
	return "workflow"
}

func (c *CircleCI) Description() string {
	return "Trigger and monitor CircleCI pipelines"
}

func (c *CircleCI) Instructions() string {
	return "Create a Personal API Token in CircleCI → User Settings → Personal API Tokens"
}

func (c *CircleCI) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiToken",
			Label:       "API Token",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Description: "CircleCI Personal API Token",
			Placeholder: "Your CircleCI API token",
			Required:    true,
		},
	}
}

func (c *CircleCI) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (c *CircleCI) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	metadata := Metadata{}
	err = mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	// Verify the API token by getting current user info
	_, err = client.GetCurrentUser()
	if err != nil {
		return fmt.Errorf("error verifying API token: %v", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (c *CircleCI) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

type WebhookConfiguration struct {
	ProjectSlug string `json:"projectSlug"`
}

func (c *CircleCI) CompareWebhookConfig(a, b any) (bool, error) {
	configA := WebhookConfiguration{}
	if err := mapstructure.Decode(a, &configA); err != nil {
		return false, err
	}

	configB := WebhookConfiguration{}
	if err := mapstructure.Decode(b, &configB); err != nil {
		return false, err
	}

	return configA.ProjectSlug == configB.ProjectSlug, nil
}

func (c *CircleCI) Actions() []core.Action {
	return []core.Action{}
}

func (c *CircleCI) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

type WebhookMetadata struct {
	WebhookID string `json:"webhookId"`
	Name      string `json:"name"`
}

func (c *CircleCI) SetupWebhook(ctx core.SetupWebhookContext) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	configuration := WebhookConfiguration{}
	err = mapstructure.Decode(ctx.Webhook.GetConfiguration(), &configuration)
	if err != nil {
		return nil, fmt.Errorf("error decoding configuration: %v", err)
	}

	hash := sha256.New()
	hash.Write([]byte(ctx.Webhook.GetID()))
	suffix := fmt.Sprintf("%x", hash.Sum(nil))
	name := fmt.Sprintf("superplane-webhook-%s", suffix[:16])

	webhookSecret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return nil, fmt.Errorf("error getting webhook secret: %v", err)
	}

	// Create CircleCI webhook
	webhook, err := client.CreateWebhook(
		name,
		ctx.Webhook.GetURL(),
		string(webhookSecret),
		configuration.ProjectSlug,
		[]string{"workflow-completed"},
	)
	if err != nil {
		return nil, fmt.Errorf("error creating CircleCI webhook: %v", err)
	}

	return WebhookMetadata{
		WebhookID: webhook.ID,
		Name:      webhook.Name,
	}, nil
}

func (c *CircleCI) CleanupWebhook(ctx core.CleanupWebhookContext) error {
	metadata := WebhookMetadata{}
	err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata)
	if err != nil {
		return fmt.Errorf("error decoding webhook metadata: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	return client.DeleteWebhook(metadata.WebhookID)
}

func (c *CircleCI) Components() []core.Component {
	return []core.Component{
		&TriggerPipeline{},
	}
}

func (c *CircleCI) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnPipelineCompleted{},
	}
}

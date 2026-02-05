package sendgrid

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("sendgrid", &SendGrid{})
}

type SendGrid struct{}

const webhookVerificationKeySecret = "sendgridWebhookVerificationKey"

type Configuration struct {
	APIKey    string `json:"apiKey"`
	FromName  string `json:"fromName"`
	FromEmail string `json:"fromEmail"`
}

type Metadata struct {
	// No metadata needed for the base integration.
}

func (s *SendGrid) Name() string {
	return "sendgrid"
}

func (s *SendGrid) Label() string {
	return "SendGrid"
}

func (s *SendGrid) Icon() string {
	return "sendgrid"
}

func (s *SendGrid) Description() string {
	return "Send transactional and marketing email with SendGrid"
}

func (s *SendGrid) Instructions() string {
	return `### Connection

Configure the SendGrid integration in SuperPlane with:
- **API Key**: SendGrid API key with Mail Send and Mail Settings Read scopes
- **Default From Email**: Required sender email address for SendGrid actions
- **Default From Name**: Optional sender name for SendGrid actions

### Actions and Triggers

The SendGrid base integration establishes API access. Actions and triggers will be documented here once they are available.`
}

func (s *SendGrid) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiKey",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "SendGrid API key with Mail Send and Mail Settings Read scopes",
		},
		{
			Name:        "fromEmail",
			Label:       "Default From Email",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Default sender email address for SendGrid actions",
		},
		{
			Name:        "fromName",
			Label:       "Default From Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Default sender name for SendGrid actions",
		},
	}
}

func (s *SendGrid) Components() []core.Component {
	return []core.Component{
		&SendEmail{},
	}
}

func (s *SendGrid) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnEmailEvent{},
	}
}

func (s *SendGrid) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (s *SendGrid) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if config.APIKey == "" {
		return fmt.Errorf("apiKey is required")
	}
	if config.FromEmail == "" {
		return fmt.Errorf("fromEmail is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if err := client.Verify(); err != nil {
		return fmt.Errorf("failed to verify SendGrid credentials: %w", err)
	}

	ctx.Integration.SetMetadata(Metadata{})
	ctx.Integration.Ready()
	return nil
}

func (s *SendGrid) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (s *SendGrid) CompareWebhookConfig(a, b any) (bool, error) {
	configA := WebhookConfiguration{}
	configB := WebhookConfiguration{}

	if err := mapstructure.Decode(a, &configA); err != nil {
		return false, err
	}

	if err := mapstructure.Decode(b, &configB); err != nil {
		return false, err
	}

	return true, nil
}

func (s *SendGrid) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return []core.IntegrationResource{}, nil
}

func (s *SendGrid) SetupWebhook(ctx core.SetupWebhookContext) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	settings := EventWebhookSettings{
		Enabled:          true,
		URL:              ctx.Webhook.GetURL(),
		Processed:        true,
		Delivered:        true,
		Deferred:         true,
		Bounce:           true,
		Dropped:          true,
		Open:             true,
		Click:            true,
		SpamReport:       true,
		Unsubscribe:      true,
		GroupUnsubscribe: true,
		GroupResubscribe: true,
	}

	if err := client.UpdateEventWebhookSettings(settings); err != nil {
		return nil, fmt.Errorf("failed to update SendGrid webhook settings: %w", err)
	}

	publicKey, err := client.EnableEventWebhookSignature()
	if err != nil {
		return nil, fmt.Errorf("failed to enable SendGrid signed webhook: %w", err)
	}
	verificationKey := strings.TrimSpace(publicKey)

	if verificationKey != "" {
		if err := ctx.Integration.SetSecret(webhookVerificationKeySecret, []byte(verificationKey)); err != nil {
			return nil, fmt.Errorf("failed to store integration verification key: %w", err)
		}

		if err := ctx.Webhook.SetSecret([]byte(verificationKey)); err != nil {
			return nil, fmt.Errorf("failed to store webhook verification key: %w", err)
		}
	}

	return nil, nil
}

func (s *SendGrid) CleanupWebhook(ctx core.CleanupWebhookContext) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	settings, err := client.GetEventWebhookSettings()
	if err != nil {
		return fmt.Errorf("failed to fetch SendGrid webhook settings: %w", err)
	}

	webhookURL := strings.TrimSpace(settings.URL)
	if webhookURL == "" {
		webhookURL = ctx.Webhook.GetURL()
	}

	parsedURL, err := url.Parse(webhookURL)
	if err != nil || strings.ToLower(parsedURL.Scheme) != "https" {
		return nil
	}

	settings.Enabled = false
	settings.URL = webhookURL

	if err := client.UpdateEventWebhookSettings(*settings); err != nil {
		return fmt.Errorf("failed to disable SendGrid webhook: %w", err)
	}

	return nil
}

func (s *SendGrid) Actions() []core.Action {
	return []core.Action{}
}

func (s *SendGrid) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

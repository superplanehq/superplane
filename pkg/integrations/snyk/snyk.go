package snyk

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("snyk", &Snyk{})
}

type Snyk struct{}

type Configuration struct {
	APIToken       string `json:"apiToken"`
	OrganizationID string `json:"organizationId"`
}

type Metadata struct {
	Organizations []Organization `json:"organizations"`
	User          User           `json:"user"`
}

type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type User struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

func (s *Snyk) Name() string {
	return "snyk"
}

func (s *Snyk) Label() string {
	return "Snyk"
}

func (s *Snyk) Icon() string {
	return "shield"
}

func (s *Snyk) Description() string {
	return "Security workflow integration with Snyk"
}

func (s *Snyk) Instructions() string {
	return "To get a Snyk API token, go to your Snyk account settings."
}

func (s *Snyk) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiToken",
			Label:       "API Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Snyk API token for authentication",
		},
		{
			Name:        "organizationId",
			Label:       "Organization ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Snyk organization ID",
		},
	}
}

func (s *Snyk) Components() []core.Component {
	return []core.Component{
		&IgnoreIssue{},
	}
}

func (s *Snyk) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnNewIssueDetected{},
	}
}

func (s *Snyk) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (s *Snyk) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode config: %v", err)
	}

	if config.APIToken == "" {
		return fmt.Errorf("apiToken is required")
	}

	if config.OrganizationID == "" {
		return fmt.Errorf("organizationId is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	user, err := client.GetUser()
	if err != nil {
		return fmt.Errorf("error verifying credentials: %v", err)
	}

	ctx.Integration.SetMetadata(Metadata{
		User: User{
			ID:       user.Data.ID,
			Name:     user.Data.Attributes.Name,
			Email:    user.Data.Attributes.Email,
			Username: user.Data.Attributes.Username,
		},
	})
	ctx.Integration.Ready()
	return nil
}

func (s *Snyk) HandleRequest(ctx core.HTTPRequestContext) {
	// The trigger-specific HandleWebhook method handles webhook events
	// This integration-level handler is not needed
	ctx.Response.WriteHeader(200)
}

type SnykWebhook struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type WebhookConfiguration struct {
	EventType string `json:"eventType"`
	OrgID     string `json:"orgId"`
	ProjectID string `json:"projectId,omitempty"`
}

func (s *Snyk) CompareWebhookConfig(a, b any) (bool, error) {
	configA := WebhookConfiguration{}
	configB := WebhookConfiguration{}

	err := mapstructure.Decode(a, &configA)
	if err != nil {
		return false, fmt.Errorf("failed to decode config A: %w", err)
	}

	err = mapstructure.Decode(b, &configB)
	if err != nil {
		return false, fmt.Errorf("failed to decode config B: %w", err)
	}

	return configA.EventType == configB.EventType &&
		configA.OrgID == configB.OrgID &&
		configA.ProjectID == configB.ProjectID, nil
}

func (s *Snyk) SetupWebhook(ctx core.SetupWebhookContext) (any, error) {
	config := WebhookConfiguration{}
	err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config)
	if err != nil {
		return nil, fmt.Errorf("failed to decode webhook configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create Snyk client: %w", err)
	}

	webhookURL := ctx.Webhook.GetURL()

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return nil, fmt.Errorf("error getting webhook secret: %w", err)
	}

	webhookID, err := client.RegisterWebhook(config.OrgID, webhookURL, string(secret))
	if err != nil {
		return nil, fmt.Errorf("error registering Snyk webhook: %w", err)
	}

	return &SnykWebhook{
		ID:  webhookID,
		URL: webhookURL,
	}, nil
}

func (s *Snyk) CleanupWebhook(ctx core.CleanupWebhookContext) error {
	webhook := SnykWebhook{}
	err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &webhook)
	if err != nil {
		return fmt.Errorf("failed to decode webhook metadata: %w", err)
	}

	config := WebhookConfiguration{}
	err = mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config)
	if err != nil {
		return fmt.Errorf("failed to decode webhook configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Snyk client: %w", err)
	}

	err = client.DeleteWebhook(config.OrgID, webhook.ID)
	if err != nil {
		return fmt.Errorf("error deleting Snyk webhook: %w", err)
	}

	return nil
}

func (s *Snyk) Actions() []core.Action {
	return []core.Action{}
}

func (s *Snyk) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

func (s *Snyk) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	// For now, return empty resources - we can implement this later if needed
	return []core.IntegrationResource{}, nil
}

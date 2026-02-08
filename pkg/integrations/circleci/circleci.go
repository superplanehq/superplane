package circleci

import (
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
	APIToken string `json:"apiToken" mapstructure:"apiToken"`
}

type Metadata struct {
	UserLogin string `json:"userLogin" mapstructure:"userLogin"`
	UserID    string `json:"userId" mapstructure:"userId"`
}

func (c *CircleCI) Name() string        { return "circleci" }
func (c *CircleCI) Label() string       { return "CircleCI" }
func (c *CircleCI) Icon() string        { return "workflow" }
func (c *CircleCI) Description() string { return "Trigger and react to your CircleCI workflows" }

func (c *CircleCI) Instructions() string {
	return `To set up CircleCI integration:

1. Go to CircleCI → **User Settings** → **Personal API Tokens**
2. Create a token
3. Paste it below

Notes:
- The API token is treated as sensitive and will be encrypted.
- Webhook verification uses the signing secret configured when SuperPlane provisions the webhook.
`
}

func (c *CircleCI) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiToken",
			Label:       "API Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "CircleCI Personal API Token",
		},
	}
}

func (c *CircleCI) Components() []core.Component {
	return []core.Component{
		&TriggerPipeline{},
	}
}

func (c *CircleCI) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnWorkflowCompleted{},
	}
}

func (c *CircleCI) Sync(ctx core.SyncContext) error {
	cfg := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &cfg); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	if cfg.APIToken == "" {
		return fmt.Errorf("apiToken is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	me, err := client.GetCurrentUser()
	if err != nil {
		return fmt.Errorf("failed to verify CircleCI token: %w", err)
	}

	// Best-effort metadata (depends on what the API returns).
	ctx.Integration.SetMetadata(Metadata{
		UserLogin: me.Login,
		UserID:    me.ID,
	})

	ctx.Integration.Ready()
	return nil
}

func (c *CircleCI) Cleanup(ctx core.IntegrationCleanupContext) error { return nil }
func (c *CircleCI) Actions() []core.Action                           { return []core.Action{} }
func (c *CircleCI) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

func (c *CircleCI) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	// CircleCI API v2 does not provide a universal "list projects" endpoint for all org types.
	// For now, components/triggers accept a project slug directly, e.g. `gh/my-org/my-repo`.
	return []core.IntegrationResource{}, nil
}

func (c *CircleCI) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op: CircleCI integration receives events via provisioned webhooks on triggers.
}

type WebhookConfiguration struct {
	ProjectID   string   `json:"projectId" mapstructure:"projectId"`
	ProjectSlug string   `json:"projectSlug" mapstructure:"projectSlug"`
	Events      []string `json:"events" mapstructure:"events"`
}

func (c *CircleCI) CompareWebhookConfig(a, b any) (bool, error) {
	cfgA := WebhookConfiguration{}
	if err := mapstructure.Decode(a, &cfgA); err != nil {
		return false, err
	}
	cfgB := WebhookConfiguration{}
	if err := mapstructure.Decode(b, &cfgB); err != nil {
		return false, err
	}

	// Keep it simple: if project differs, webhook differs.
	return cfgA.ProjectID == cfgB.ProjectID, nil
}

type WebhookMetadata struct {
	WebhookID string `json:"webhookId" mapstructure:"webhookId"`
}

func (c *CircleCI) SetupWebhook(ctx core.SetupWebhookContext) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	cfg := WebhookConfiguration{}
	if err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &cfg); err != nil {
		return nil, fmt.Errorf("failed to decode webhook configuration: %w", err)
	}
	if cfg.ProjectID == "" {
		return nil, fmt.Errorf("projectId is required")
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to get webhook secret: %w", err)
	}

	events := cfg.Events
	if len(events) == 0 {
		events = []string{"workflow-completed"}
	}

	name := fmt.Sprintf("superplane-%s", ctx.Webhook.GetID())
	wh, err := client.CreateWebhook(CreateWebhookRequest{
		Name:         name,
		URL:          ctx.Webhook.GetURL(),
		VerifyTLS:    true,
		SigningSecret: string(secret),
		Scope: WebhookScope{
			ID:   cfg.ProjectID,
			Type: "project",
		},
		Events: events,
	})
	if err != nil {
		return nil, err
	}

	return WebhookMetadata{WebhookID: wh.ID}, nil
}

func (c *CircleCI) CleanupWebhook(ctx core.CleanupWebhookContext) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	meta := WebhookMetadata{}
	if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &meta); err != nil {
		return fmt.Errorf("failed to decode webhook metadata: %w", err)
	}
	if meta.WebhookID == "" {
		return nil
	}

	return client.DeleteWebhook(meta.WebhookID)
}

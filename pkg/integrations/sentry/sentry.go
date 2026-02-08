package sentry

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("sentry", &Sentry{})
}

type Sentry struct{}

type Configuration struct {
	AuthToken string `json:"authToken"`
	BaseURL   string `json:"baseURL"`
}

func (s *Sentry) Name() string {
	return "sentry"
}

func (s *Sentry) Label() string {
	return "Sentry"
}

func (s *Sentry) Icon() string {
	return "alert-triangle"
}

func (s *Sentry) Description() string {
	return "Trigger workflows from Sentry issue events and update issues from workflows"
}

func (s *Sentry) Instructions() string {
	return `Connect Sentry to SuperPlane using an Internal Integration token.

1. In Sentry: **Organization Settings** → **Developer Settings** → **New Internal Integration**
2. Name it (e.g. "SuperPlane"), then create an **Auth Token** with scopes: ` + "`org:read`" + `, ` + "`project:read`" + `, ` + "`event:write`" + `
3. Paste the token below. For the **On Issue Event** trigger, configure the webhook in your Sentry integration to point to the trigger's webhook URL and subscribe to Issue events. Use the same integration's Client Secret as the webhook secret in SuperPlane when prompted.`
}

func (s *Sentry) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "authToken",
			Label:       "Auth Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Sentry auth token (Bearer). Create via Organization Settings → Developer Settings → Internal Integration → Auth Token.",
			Placeholder: "sntrys_...",
		},
		{
			Name:        "baseURL",
			Label:       "Base URL",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Sentry API base URL. Leave empty for sentry.io (use https://eu.sentry.io for EU).",
			Placeholder: "https://sentry.io",
		},
	}
}

func (s *Sentry) Components() []core.Component {
	return []core.Component{
		&UpdateIssue{},
	}
}

func (s *Sentry) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnIssueEvent{},
	}
}

func (s *Sentry) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (s *Sentry) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("decode config: %w", err)
	}
	if config.AuthToken == "" {
		return fmt.Errorf("authToken is required")
	}
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}
	if err := client.ValidateToken(); err != nil {
		return fmt.Errorf("invalid Sentry token: %w", err)
	}
	ctx.Integration.Ready()
	return nil
}

func (s *Sentry) HandleRequest(ctx core.HTTPRequestContext) {}

func (s *Sentry) CompareWebhookConfig(a, b any) (bool, error) {
	var cfgA, cfgB WebhookConfiguration
	if err := mapstructure.Decode(a, &cfgA); err != nil {
		return false, err
	}
	if err := mapstructure.Decode(b, &cfgB); err != nil {
		return false, err
	}

	// Ignore WebhookSecret field - it's transient and stripped before storage (line 144)
	// Only compare Events to determine if webhooks can be shared
	if len(cfgA.Events) != len(cfgB.Events) {
		return false, nil
	}
	seen := make(map[string]bool)
	for _, e := range cfgA.Events {
		seen[e] = true
	}
	for _, e := range cfgB.Events {
		if !seen[e] {
			return false, nil
		}
	}
	return true, nil
}

func (s *Sentry) SetupWebhook(ctx core.SetupWebhookContext) (any, error) {
	var cfg WebhookConfiguration
	if err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &cfg); err != nil {
		return nil, fmt.Errorf("decode webhook config: %w", err)
	}
	if cfg.WebhookSecret != "" {
		if err := ctx.Webhook.SetSecret([]byte(cfg.WebhookSecret)); err != nil {
			return nil, fmt.Errorf("store webhook secret: %w", err)
		}
	}
	// Return config without the secret (only Events)
	return WebhookConfiguration{Events: cfg.Events}, nil
}

func (s *Sentry) CleanupWebhook(ctx core.CleanupWebhookContext) error {
	return nil
}

func (s *Sentry) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return nil, nil
}

func (s *Sentry) Actions() []core.Action {
	return nil
}

func (s *Sentry) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

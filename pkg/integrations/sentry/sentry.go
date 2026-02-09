package sentry

import (
	"fmt"
	"strings"

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

func looksLikeClientSecret(token string) bool {
	token = strings.TrimSpace(token)
	if len(token) != 64 {
		return false
	}
	if strings.HasPrefix(token, "sntry") {
		return false
	}
	for _, c := range token {
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
			continue
		}
		return false
	}
	return true
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
	return `**Connection (Auth Token):** Use a **Personal Auth Token** from Sentry. Create it in **User Settings** → **Auth Tokens** → **Create New Token** with scopes: ` + "`org:read`" + `, ` + "`project:read`" + `, ` + "`event:write`" + `. The token will start with ` + "`sntryu_`" + `—paste it below.

⚠️ **Do not use the Internal Integration hex value** (e.g. ` + "`a22b2023...`" + ` or ` + "`f3233f8f...`" + `). That is the **Client Secret**; Sentry's API returns 401 for it. Use it only in the **On Issue Event** trigger's "Webhook secret" field.

**On Issue Event trigger:** You need an Internal Integration in Sentry (Developer Settings) to get a webhook URL and the **Client Secret**. Paste the Client Secret into the trigger's "Webhook secret" field.

**For each Sentry trigger and action node:** Select your Sentry integration in the **Integration** dropdown, then click **Save** so the node is linked; otherwise you'll see "Component not configured - integration is required".`
}

func (s *Sentry) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "authToken",
			Label:       "Auth Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Personal Auth Token from Sentry (User Settings → Auth Tokens). Must start with sntryu_. Do not paste the Client Secret (hex string)—use that only in the trigger's Webhook secret. Scopes: org:read, project:read, event:write.",
			Placeholder: "sntryu_...",
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
		msg := err.Error()
		if strings.Contains(msg, "401") && looksLikeClientSecret(config.AuthToken) {
			msg = "connection returned 401. If you pasted a long hex string (no sntryu_/sntrys_ prefix), that is the Client Secret—use it only for the trigger's Webhook secret. For this field use the Auth Token from 'Create New Token' in the integration. " + msg
		}
		return fmt.Errorf("invalid Sentry token: %s", msg)
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

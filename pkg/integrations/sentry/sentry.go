package sentry

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("sentry", &Sentry{}, &SentryWebhookHandler{})
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
	return `Connect Sentry to SuperPlane using a Sentry **Personal Token**.

## API token

1. In Sentry: **Settings** → **Developer Settings** → **Personal Tokens**
2. Create a token with scopes:
   - ` + "`org:read`" + `, ` + "`org:write`" + `
   - ` + "`project:read`" + `, ` + "`project:write`" + `
   - ` + "`event:read`" + `, ` + "`event:write`" + `
3. Paste the token below (tokens typically start with ` + "`sntryu_`" + `).

## Webhook (required for triggers)

To use the **On Issue Event** trigger, you must configure a Sentry webhook to call SuperPlane:

1. Add the **On Issue Event** trigger to a canvas and save the canvas.
2. Copy the webhook URL shown in the trigger configuration sidebar.
3. In Sentry, create an **Internal Integration** and set its webhook URL to the SuperPlane webhook URL.
4. Enable **Issue** events (created/resolved/unresolved/assigned/archived).
5. Trigger an issue change in Sentry and check the trigger’s **Runs** tab.

Docs: https://docs.sentry.io/`
}

func (s *Sentry) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "authToken",
			Label:       "Auth Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Sentry personal token (Bearer). Create via Settings → Developer Settings → Personal Tokens. Recommended scopes: org:read/org:write, project:read/project:write, event:read/event:write.",
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
		return fmt.Errorf("invalid Sentry token: %w", err)
	}
	ctx.Integration.Ready()
	return nil
}

func (s *Sentry) HandleRequest(ctx core.HTTPRequestContext) {}

func (s *Sentry) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return nil, nil
}

func (s *Sentry) Actions() []core.Action {
	return nil
}

func (s *Sentry) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

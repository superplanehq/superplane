package honeycomb

import (
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("honeycomb", &Honeycomb{}, &HoneycombWebhookHandler{})
}

type Honeycomb struct{}

func (h *Honeycomb) Name() string {
	return "honeycomb"
}

func (h *Honeycomb) Label() string {
	return "Honeycomb"
}

func (h *Honeycomb) Icon() string {
	return "honeycomb"
}

func (h *Honeycomb) Description() string {
	return "Triggers and actions for Honeycomb"
}

func (h *Honeycomb) Instructions() string {
	return `
Connect Honeycomb to Superplane using a Honeycomb API key.

**Get your API key**:
Honeycomb → Account → Team Settings → API Keys → copy key → paste here.

**Trigger setup**:
After saving a Honeycomb trigger node, Superplane generates a Webhook URL and Shared Secret.
Add them in Honeycomb → Team Settings → Integrations → Webhooks.

Once configured, Honeycomb events will trigger your workflow automatically.
`
}

func (h *Honeycomb) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "site",
			Label:    "Honeycomb Site",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "api.honeycomb.io",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "US (api.honeycomb.io)", Value: "api.honeycomb.io"},
						{Label: "EU (api.eu1.honeycomb.io)", Value: "api.eu1.honeycomb.io"},
					},
				},
			},
			Description: "Select the Honeycomb API host for your account region.",
		},
		{
			Name:        "apiKey",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Description: "Honeycomb API key used for actions (e.g. Create Event).",
			Sensitive:   true,
			Required:    true,
		},
	}
}

func (h *Honeycomb) Components() []core.Component {
	return []core.Component{
		&CreateEvent{},
	}
}

func (h *Honeycomb) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnAlertFired{},
	}
}

func (h *Honeycomb) Sync(ctx core.SyncContext) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}
	if err := client.Validate(); err != nil {
		return err
	}
	ctx.Integration.Ready()
	return nil
}

func (h *Honeycomb) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (h *Honeycomb) Actions() []core.Action {
	return []core.Action{}
}

func (h *Honeycomb) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

func (h *Honeycomb) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return []core.IntegrationResource{}, nil
}

func (h *Honeycomb) HandleRequest(ctx core.HTTPRequestContext) {
	ctx.Response.WriteHeader(404)
	_, _ = ctx.Response.Write([]byte("not found"))
}

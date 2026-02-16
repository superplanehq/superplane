package incident

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("incident", &IncidentIO{}, &IncidentIOWebhookHandler{})
}

type IncidentIO struct{}

type Configuration struct {
	APIKey string `json:"apiKey"`
}

type Metadata struct{}

func (i *IncidentIO) Name() string {
	return "incident"
}

func (i *IncidentIO) Label() string {
	return "Incident"
}

func (i *IncidentIO) Icon() string {
	return "alert-triangle"
}

func (i *IncidentIO) Description() string {
	return "Manage and react to incidents in incident.io"
}

func (i *IncidentIO) Instructions() string {
	return `## Connect incident.io to SuperPlane

1. **Create an API key** in [incident.io Settings > API keys](https://app.incident.io/settings/api-keys). Grant the key permission to create incidents and read severities (and optionally view private incidents if you use private incidents).

2. **Paste the API key** below. The key is stored securely and used to validate the connection and to run Create Incident actions.

## On Incident trigger (webhooks)

incident.io sends webhooks via Svix. There is no API to register webhook endpoints; you configure them in the incident.io dashboard:

1. Add the **On Incident** trigger to your workflow and select the events you want (e.g. Incident created, Incident updated).
2. Copy the **webhook URL** shown for this trigger (after saving the canvas).
3. In incident.io go to **Settings > Webhooks** and create a new endpoint. Paste the SuperPlane webhook URL and subscribe to the same events (e.g. **Public incident created (v2)**, **Public incident updated (v2)**).
4. Copy the **Signing secret** from the new endpoint in incident.io and paste it into the trigger's **Signing secret** field in SuperPlane.

Without the signing secret, webhook requests cannot be verified and will be rejected.`
}

func (i *IncidentIO) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiKey",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "API key from incident.io. Create one in Settings > API keys.",
		},
	}
}

func (i *IncidentIO) Components() []core.Component {
	return []core.Component{
		&CreateIncident{},
	}
}

func (i *IncidentIO) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnIncident{},
	}
}

func (i *IncidentIO) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (i *IncidentIO) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode config: %w", err)
	}

	if config.APIKey == "" {
		return fmt.Errorf("API key is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	// Validate API key by listing severities
	_, err = client.ListSeverities()
	if err != nil {
		return fmt.Errorf("error validating API key (listing severities): %w", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (i *IncidentIO) HandleRequest(ctx core.HTTPRequestContext) {}

func (i *IncidentIO) Actions() []core.Action {
	return nil
}

func (i *IncidentIO) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

func (i *IncidentIO) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if resourceType != "severity" {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	severities, err := client.ListSeverities()
	if err != nil {
		return nil, fmt.Errorf("failed to list severities: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(severities))
	for _, s := range severities {
		resources = append(resources, core.IntegrationResource{
			Type: "severity",
			Name: s.Name,
			ID:   s.ID,
		})
	}
	return resources, nil
}

package datadog

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const installationInstructions = `
To configure Datadog webhooks to work with SuperPlane:

1. **Create a Webhook in Datadog**: Go to Integrations > Webhooks in Datadog
2. **Configure the Webhook**:
   - **Name**: Give it a descriptive name (e.g., "SuperPlane Integration")
   - **URL**: Use the webhook URL provided by SuperPlane for your trigger
   - **Custom Headers**: Add "X-Superplane-Signature-256" header with the HMAC-SHA256 signature of the request body using the webhook secret
3. **Configure Monitor Notifications**: In your Datadog monitors, add @webhook-<webhook-name> to the notification message
4. **Test**: Trigger a test alert to verify the integration is working

Note: The signature should be in the format "sha256=<hex-encoded-hmac>". See Datadog webhook documentation for details on custom headers.
`

func init() {
	registry.RegisterApplication("datadog", &Datadog{})
}

type Datadog struct{}

type Configuration struct {
	APIKey string `json:"apiKey"`
	AppKey string `json:"appKey"`
	Site   string `json:"site"`
}

func (d *Datadog) Name() string {
	return "datadog"
}

func (d *Datadog) Label() string {
	return "Datadog"
}

func (d *Datadog) Icon() string {
	return "chart-bar"
}

func (d *Datadog) Description() string {
	return "Monitor alerts and create events in Datadog"
}

func (d *Datadog) InstallationInstructions() string {
	return installationInstructions
}

func (d *Datadog) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "site",
			Label:    "Datadog Site",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "datadoghq.com",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "US1 (datadoghq.com)", Value: "datadoghq.com"},
						{Label: "US3 (us3.datadoghq.com)", Value: "us3.datadoghq.com"},
						{Label: "US5 (us5.datadoghq.com)", Value: "us5.datadoghq.com"},
						{Label: "EU (datadoghq.eu)", Value: "datadoghq.eu"},
						{Label: "AP1 (ap1.datadoghq.com)", Value: "ap1.datadoghq.com"},
					},
				},
			},
		},
		{
			Name:        "apiKey",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Datadog API Key for authentication",
		},
		{
			Name:        "appKey",
			Label:       "Application Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Datadog Application Key for authentication",
		},
	}
}

func (d *Datadog) Components() []core.Component {
	return []core.Component{
		&CreateEvent{},
	}
}

func (d *Datadog) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnMonitorAlert{},
	}
}

func (d *Datadog) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode config: %v", err)
	}

	if config.Site == "" {
		return fmt.Errorf("site is required")
	}

	if config.APIKey == "" {
		return fmt.Errorf("apiKey is required")
	}

	if config.AppKey == "" {
		return fmt.Errorf("appKey is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.AppInstallation)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	err = client.ValidateCredentials()
	if err != nil {
		return fmt.Errorf("invalid credentials: %v", err)
	}

	ctx.AppInstallation.SetState("ready", "")
	return nil
}

func (d *Datadog) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op - webhooks are handled by triggers
}

func (d *Datadog) CleanupWebhook(ctx core.CleanupWebhookContext) error {
	// no-op - Datadog webhooks are manually configured by users
	return nil
}

func (d *Datadog) CompareWebhookConfig(a, b any) (bool, error) {
	// Datadog webhooks are manually configured, so we don't compare configurations
	return true, nil
}

func (d *Datadog) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.ApplicationResource, error) {
	// Datadog doesn't expose selectable resources in this integration
	return []core.ApplicationResource{}, nil
}

func (d *Datadog) SetupWebhook(ctx core.SetupWebhookContext) (any, error) {
	// Datadog webhooks are manually configured by users in the Datadog UI
	// No automatic provisioning is supported
	return nil, nil
}

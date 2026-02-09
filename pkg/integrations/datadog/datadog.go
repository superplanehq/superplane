package datadog

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const installationInstructions = `
To configure Datadog to work with SuperPlane:

1. **Get API Keys**: In Datadog, go to Organization Settings > API Keys to get your API Key
2. **Get Application Key**: Go to Organization Settings > Application Keys to create an Application Key
3. **Select Site**: Choose the Datadog site that matches your account (US1, US3, US5, EU, or AP1)
4. **Enter Credentials**: Provide your API Key, Application Key, and Site in the integration configuration
`

func init() {
	registry.RegisterIntegration("datadog", &Datadog{})
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
	return "Create events in Datadog"
}

func (d *Datadog) Instructions() string {
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
	return []core.Trigger{}
}

func (d *Datadog) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
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

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	err = client.ValidateCredentials()
	if err != nil {
		return fmt.Errorf("invalid credentials: %v", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (d *Datadog) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op - webhooks are handled by triggers
}

func (d *Datadog) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return []core.IntegrationResource{}, nil
}

func (d *Datadog) Actions() []core.Action {
	return []core.Action{}
}

func (d *Datadog) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

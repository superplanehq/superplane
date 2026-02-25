package firehydrant

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("firehydrant", &FireHydrant{}, &FireHydrantWebhookHandler{})
}

type FireHydrant struct{}

type Configuration struct {
	APIKey string `json:"apiKey"`
}

type Metadata struct {
	Severities []Severity `json:"severities"`
}

func (f *FireHydrant) Name() string {
	return "firehydrant"
}

func (f *FireHydrant) Label() string {
	return "FireHydrant"
}

func (f *FireHydrant) Icon() string {
	return "flame"
}

func (f *FireHydrant) Description() string {
	return "Manage and react to incidents in FireHydrant"
}

func (f *FireHydrant) Instructions() string {
	return "To connect FireHydrant, create an API key in **Settings → API Keys**. This requires Owner permissions."
}

func (f *FireHydrant) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiKey",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "API key from FireHydrant. You can generate one in Settings > API Keys.",
		},
	}
}

func (f *FireHydrant) Components() []core.Component {
	return []core.Component{
		&CreateIncident{},
	}
}

func (f *FireHydrant) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnIncident{},
	}
}

func (f *FireHydrant) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (f *FireHydrant) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode config: %v", err)
	}

	if config.APIKey == "" {
		return fmt.Errorf("API key is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	// Validate connection by listing severities
	severities, err := client.ListSeverities()
	if err != nil {
		return fmt.Errorf("error listing severities: %v", err)
	}

	ctx.Integration.SetMetadata(Metadata{Severities: severities})
	ctx.Integration.Ready()
	return nil
}

func (f *FireHydrant) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (f *FireHydrant) Actions() []core.Action {
	return []core.Action{}
}

func (f *FireHydrant) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

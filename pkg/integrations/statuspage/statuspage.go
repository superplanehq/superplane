package statuspage

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("statuspage", &Statuspage{})
}

type Statuspage struct{}

type Configuration struct {
	APIKey string `json:"apiKey"`
}

func (s *Statuspage) Name() string {
	return "statuspage"
}

func (s *Statuspage) Label() string {
	return "Statuspage"
}

func (s *Statuspage) Icon() string {
	return "activity"
}

func (s *Statuspage) Description() string {
	return "Create and manage incidents on your Atlassian Statuspage"
}

func (s *Statuspage) Instructions() string {
	return ""
}

func (s *Statuspage) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiKey",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Sensitive:   true,
			Required:    true,
			Description: "Your Statuspage OAuth API key. Generate at your page settings in Statuspage.",
		},
	}
}

func (s *Statuspage) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (s *Statuspage) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.APIKey == "" {
		return fmt.Errorf("apiKey is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	_, err = client.ListPages()
	if err != nil {
		return fmt.Errorf("error verifying connection: %w", err)
	}

	ctx.Integration.Ready()
	return nil
}

func (s *Statuspage) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (s *Statuspage) Actions() []core.Action {
	return []core.Action{}
}

func (s *Statuspage) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

func (s *Statuspage) Components() []core.Component {
	return []core.Component{
		&CreateIncident{},
	}
}

func (s *Statuspage) Triggers() []core.Trigger {
	return []core.Trigger{}
}

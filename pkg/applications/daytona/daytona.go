package daytona

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterApplication("daytona", &Daytona{})
}

type Daytona struct{}

type Configuration struct {
	APIKey  string `json:"apiKey"`
	BaseURL string `json:"baseURL"`
}

type Metadata struct {
}

func (d *Daytona) Name() string {
	return "daytona"
}

func (d *Daytona) Label() string {
	return "Daytona"
}

func (d *Daytona) Icon() string {
	return "daytona"
}

func (d *Daytona) Description() string {
	return "Execute code in isolated sandbox environments"
}

func (d *Daytona) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiKey",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Daytona API key",
		},
		{
			Name:        "baseURL",
			Label:       "Base URL",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "API base URL (default: https://app.daytona.io/api)",
		},
	}
}

func (d *Daytona) Components() []core.Component {
	return []core.Component{
		&CreateSandbox{},
		&ExecuteCode{},
		&ExecuteCommand{},
		&DeleteSandbox{},
	}
}

func (d *Daytona) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (d *Daytona) InstallationInstructions() string {
	return ""
}

func (d *Daytona) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if config.APIKey == "" {
		return fmt.Errorf("apiKey is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.AppInstallation)
	if err != nil {
		return err
	}

	if err := client.Verify(); err != nil {
		return err
	}

	ctx.AppInstallation.SetMetadata(Metadata{})
	ctx.AppInstallation.SetState("ready", "")
	return nil
}

func (d *Daytona) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op - Daytona does not emit external events
}

func (d *Daytona) CompareWebhookConfig(a, b any) (bool, error) {
	return true, nil
}

func (d *Daytona) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.ApplicationResource, error) {
	return []core.ApplicationResource{}, nil
}

func (d *Daytona) SetupWebhook(ctx core.SetupWebhookContext) (any, error) {
	return nil, nil
}

func (d *Daytona) CleanupWebhook(ctx core.CleanupWebhookContext) error {
	return nil
}

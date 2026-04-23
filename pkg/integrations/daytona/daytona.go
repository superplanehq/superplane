package daytona

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("daytona", &Daytona{})
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
			Default:     "https://app.daytona.io/api",
			Description: "API base URL",
		},
	}
}

func (d *Daytona) Components() []core.Component {
	return []core.Component{
		&CreateSandbox{},
		&CreateRepositorySandbox{},
		&GetPreviewURLComponent{},
		&ExecuteCode{},
		&ExecuteCommand{},
		&DeleteSandbox{},
	}
}

func (d *Daytona) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (d *Daytona) Instructions() string {
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

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if err := client.Verify(); err != nil {
		return err
	}

	ctx.Integration.SetMetadata(Metadata{})
	ctx.Integration.Ready()
	return nil
}

func (d *Daytona) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (d *Daytona) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op - Daytona does not emit external events
}

func (d *Daytona) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case "snapshot":
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, err
		}

		snapshots, err := client.ListSnapshots()
		if err != nil {
			return nil, err
		}

		resources := make([]core.IntegrationResource, 0, len(snapshots))
		for _, s := range snapshots {
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: s.Name,
				ID:   s.ID,
			})
		}

		return resources, nil
	case "sandbox":
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, err
		}

		sandboxes, err := client.ListSandboxes()
		if err != nil {
			return nil, err
		}

		resources := make([]core.IntegrationResource, 0, len(sandboxes))
		for _, s := range sandboxes {
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: s.ID,
				ID:   s.ID,
			})
		}

		return resources, nil
	default:
		return []core.IntegrationResource{}, nil
	}
}

func (d *Daytona) Actions() []core.Action {
	return []core.Action{}
}

func (d *Daytona) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

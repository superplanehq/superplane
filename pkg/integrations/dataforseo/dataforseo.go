package dataforseo

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("dataforseo", &DataForSEO{})
}

type DataForSEO struct{}

type Configuration struct {
	APIKey string `json:"apiKey"`
}

func (d *DataForSEO) Name() string {
	return "dataforseo"
}

func (d *DataForSEO) Label() string {
	return "DataForSEO"
}

func (d *DataForSEO) Icon() string {
	return "dataforseo"
}

func (d *DataForSEO) Description() string {
	return "Run SEO site audits with DataForSEO"
}

func (d *DataForSEO) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiKey",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Base64 of your DataForSEO login:password, from https://app.dataforseo.com/api-access",
		},
	}
}

func (d *DataForSEO) Actions() []core.Action {
	return []core.Action{
		&RunSiteAudit{},
	}
}

func (d *DataForSEO) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (d *DataForSEO) Instructions() string {
	return ""
}

func (d *DataForSEO) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (d *DataForSEO) Sync(ctx core.SyncContext) error {
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

	ctx.Integration.Ready()
	return nil
}

func (d *DataForSEO) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (d *DataForSEO) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return []core.IntegrationResource{}, nil
}

func (d *DataForSEO) Hooks() []core.Hook {
	return []core.Hook{}
}

func (d *DataForSEO) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}

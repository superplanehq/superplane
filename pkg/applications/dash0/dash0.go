package dash0

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterApplication("dash0", &Dash0{})
}

type Dash0 struct{}

type Configuration struct {
	APIToken *string `json:"apiToken"`
	BaseURL  *string `json:"baseURL"`
}

type Metadata struct {
	// No metadata needed initially
}

func (d *Dash0) Name() string {
	return "dash0"
}

func (d *Dash0) Label() string {
	return "Dash0"
}

func (d *Dash0) Icon() string {
	return "database"
}

func (d *Dash0) Description() string {
	return "Connect to Dash0 to query data using Prometheus API"
}

func (d *Dash0) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiToken",
			Label:       "API Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Your Dash0 API token for authentication",
		},
		{
			Name:        "baseURL",
			Label:       "Prometheus API Base URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Your Dash0 Prometheus API base URL. Find this in Dash0 dashboard: Organization Settings > Endpoints > Prometheus API. You can use either the full endpoint URL (https://api.us-west-2.aws.dash0.com/api/prometheus) or just the base URL (https://api.us-west-2.aws.dash0.com)",
			Placeholder: "https://api.us-west-2.aws.dash0.com",
		},
	}
}

func (d *Dash0) Components() []core.Component {
	return []core.Component{
		&QueryPrometheus{},
	}
}

func (d *Dash0) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnIssueStatus{},
	}
}

func (d *Dash0) Sync(ctx core.SyncContext) error {
	configuration := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &configuration)
	if err != nil {
		return fmt.Errorf("failed to decode config: %v", err)
	}

	if configuration.APIToken == nil || *configuration.APIToken == "" {
		return fmt.Errorf("apiToken is required")
	}

	if configuration.BaseURL == nil || *configuration.BaseURL == "" {
		return fmt.Errorf("baseURL is required for Dash0 Cloud. Find your API URL in Dash0 dashboard under Organization Settings > Endpoints Reference")
	}

	// Validate connection by creating a client and making a test query
	client, err := NewClient(ctx.HTTP, ctx.AppInstallation)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	// Test with a simple PromQL query to validate the connection
	testQuery := "up"
	_, err = client.ExecutePrometheusInstantQuery(testQuery, "default")
	if err != nil {
		return fmt.Errorf("error validating connection: %v", err)
	}

	ctx.AppInstallation.SetMetadata(Metadata{})
	ctx.AppInstallation.SetState("ready", "")
	return nil
}

func (d *Dash0) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.ApplicationResource, error) {
	return []core.ApplicationResource{}, nil
}

func (d *Dash0) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (d *Dash0) CompareWebhookConfig(a, b any) (bool, error) {
	return false, nil
}

func (d *Dash0) SetupWebhook(ctx core.SetupWebhookContext) (any, error) {
	return nil, nil
}

func (d *Dash0) CleanupWebhook(ctx core.CleanupWebhookContext) error {
	return nil
}

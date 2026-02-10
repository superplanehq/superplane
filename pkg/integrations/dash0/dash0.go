package dash0

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

// init registers the Dash0 integration in the global integration registry.
func init() {
	registry.RegisterIntegrationWithWebhookHandler("dash0", &Dash0{}, &Dash0WebhookHandler{})
}

// Dash0 implements the SuperPlane Dash0 integration entrypoint.
type Dash0 struct{}

// Configuration stores persisted integration credentials and endpoint settings.
type Configuration struct {
	APIToken string `json:"apiToken"`
	BaseURL  string `json:"baseURL"`
	Dataset  string `json:"dataset"`
}

// Metadata stores integration-level metadata persisted by Sync.
type Metadata struct {
	// No metadata needed initially
}

// Name returns the stable integration identifier.
func (d *Dash0) Name() string {
	return "dash0"
}

// Label returns the display name used across the UI.
func (d *Dash0) Label() string {
	return "Dash0"
}

// Icon returns the Lucide icon name for this integration.
func (d *Dash0) Icon() string {
	return "database"
}

// Description returns a short integration summary.
func (d *Dash0) Description() string {
	return "Connect to Dash0 to query observability data, manage checks, and react to alert events"
}

// Instructions returns optional setup instructions shown in the integration modal.
func (d *Dash0) Instructions() string {
	return ""
}

// Configuration defines fields required to configure Dash0 connectivity.
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
			Label:       "API Base URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Your Dash0 API base URL. Find this in Dash0 dashboard: Organization Settings > Endpoints. You can use either the full Prometheus endpoint URL (https://api.us-west-2.aws.dash0.com/api/prometheus) or just the base URL (https://api.us-west-2.aws.dash0.com). SuperPlane derives Prometheus, alerting, config, and logs endpoints from this base URL.",
			Placeholder: "https://api.us-west-2.aws.dash0.com",
		},
		{
			Name:        "dataset",
			Label:       "Dataset",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     "default",
			Description: "Dash0 dataset used by config API operations (check rules and synthetic checks).",
			Placeholder: "default",
		},
	}
}

// Components returns all Dash0 actions exposed in the workflow builder.
func (d *Dash0) Components() []core.Component {
	return []core.Component{
		&QueryPrometheus{},
		&ListIssues{},
		&SendLogEvent{},
		&GetCheckDetails{},
		&CreateSyntheticCheck{},
		&UpdateSyntheticCheck{},
		&CreateCheckRule{},
		&UpdateCheckRule{},
	}
}

// Triggers returns all Dash0 triggers exposed in the workflow builder.
func (d *Dash0) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnAlertEvent{},
	}
}

// Sync validates connectivity and marks the integration as ready.
func (d *Dash0) Sync(ctx core.SyncContext) error {
	configuration := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &configuration)
	if err != nil {
		return fmt.Errorf("failed to decode config: %v", err)
	}

	if configuration.APIToken == "" {
		return fmt.Errorf("apiToken is required")
	}

	if configuration.BaseURL == "" {
		return fmt.Errorf("baseURL is required for Dash0 Cloud. Find your API URL in Dash0 dashboard under Organization Settings > Endpoints Reference")
	}

	// Validate connection by creating a client and making a test query
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	// Test with a simple PromQL query to validate the connection
	testQuery := "up"
	_, err = client.ExecutePrometheusInstantQuery(testQuery, "default")
	if err != nil {
		return fmt.Errorf("error validating connection: %v", err)
	}

	ctx.Integration.SetMetadata(Metadata{})
	ctx.Integration.Ready()
	return nil
}

// Cleanup is a no-op because integration-level resources are not provisioned.
func (d *Dash0) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

// ListResources resolves integration resources used by resource selector fields.
func (d *Dash0) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("dash0 list resources: create client: %w", err)
	}

	switch resourceType {
	case "check-rule":
		checkRules, listErr := client.ListCheckRules()
		if listErr != nil {
			ctx.Logger.Warnf("dash0 list resources: fetch check rules: %v", listErr)
			return []core.IntegrationResource{}, nil
		}

		resources := make([]core.IntegrationResource, 0, len(checkRules))
		for _, rule := range checkRules {
			resourceID := rule.ID
			if resourceID == "" {
				resourceID = rule.Origin
			}

			resourceName := rule.Name
			if resourceName == "" {
				resourceName = resourceID
			}

			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: resourceName,
				ID:   resourceID,
			})
		}

		return resources, nil
	case "synthetic-check":
		syntheticChecks, listErr := client.ListSyntheticChecks()
		if listErr != nil {
			ctx.Logger.Warnf("dash0 list resources: fetch synthetic checks: %v", listErr)
			return []core.IntegrationResource{}, nil
		}

		resources := make([]core.IntegrationResource, 0, len(syntheticChecks))
		for _, check := range syntheticChecks {
			resourceID := check.ID
			if resourceID == "" {
				resourceID = check.Origin
			}

			resourceName := check.Name
			if resourceName == "" {
				resourceName = resourceID
			}

			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: resourceName,
				ID:   resourceID,
			})
		}

		return resources, nil
	default:
		return []core.IntegrationResource{}, nil
	}
}

// HandleRequest is unused because Dash0 integration has no custom HTTP routes.
func (d *Dash0) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

// Actions returns no integration-level actions.
func (d *Dash0) Actions() []core.Action {
	return []core.Action{}
}

// HandleAction is unused because integration-level actions are not defined.
func (d *Dash0) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

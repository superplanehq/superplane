package dash0

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("dash0", &Dash0{}, &Dash0WebhookHandler{})
}

type Dash0 struct{}

type Configuration struct {
	APIToken string `json:"apiToken"`
	BaseURL  string `json:"baseURL"`
}

type Metadata struct {
	WebhookURL string `json:"webhookURL,omitempty"`
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

func (d *Dash0) Instructions() string {
	return `To set up the Dash0 integration:

1. Go to your Dash0 dashboard -> **Organization Settings** -> **API Tokens**
2. Click **Create API Token**, give it a name, and copy the token
3. Paste the token in the **API Token** field below
4. Copy your **Prometheus API Base URL** from **Organization Settings** -> **Endpoints** -> **Prometheus API**

### Webhook Setup (for triggers)

To receive Dash0 notifications:

1. After connecting, copy the **Webhook URL** shown on the integration details page
2. In your Dash0 dashboard, go to **Alerting** -> **Notification Channels**
3. Create a new channel with type **Webhook**
4. Paste the webhook URL and save`
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
		&ListIssues{},
		&CreateHTTPSyntheticCheck{},
		&UpdateHTTPSyntheticCheck{},
		&DeleteHTTPSyntheticCheck{},
	}
}

func (d *Dash0) Triggers() []core.Trigger {
	return []core.Trigger{&OnNotification{}}
}

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

	webhookURL := ""
	webhookID, err := ctx.Integration.EnsureIntegrationWebhook(struct{}{})
	if err != nil {
		return fmt.Errorf("error ensuring integration webhook: %v", err)
	}

	if webhookID != nil && ctx.WebhooksBaseURL != "" {
		webhookURL = fmt.Sprintf("%s/api/v1/webhooks/%s", ctx.WebhooksBaseURL, webhookID.String())
	}

	ctx.Integration.SetMetadata(Metadata{WebhookURL: webhookURL})
	ctx.Integration.Ready()
	return nil
}

func (d *Dash0) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (d *Dash0) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("error creating dash0 client: %w", err)
	}

	switch resourceType {
	case "check-rule":
		checkRules, err := client.ListCheckRules()
		if err != nil {
			ctx.Logger.Warnf("Error fetching check rules: %v", err)
			return []core.IntegrationResource{}, nil
		}

		resources := make([]core.IntegrationResource, 0, len(checkRules))
		for _, rule := range checkRules {
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: rule.Name,
				ID:   rule.ID,
			})
		}

		return resources, nil

	case "synthetic-check":
		checks, err := client.ListSyntheticChecks("default")
		if err != nil {
			ctx.Logger.Warnf("Error fetching synthetic checks: %v", err)
			return []core.IntegrationResource{}, nil
		}

		resources := make([]core.IntegrationResource, 0, len(checks))
		for _, check := range checks {
			id, name := extractSyntheticCheckIDAndName(check)
			if id == "" {
				id = name
			}
			if id == "" {
				continue
			}
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: name,
				ID:   id,
			})
		}

		return resources, nil

	default:
		return []core.IntegrationResource{}, nil
	}
}

// extractSyntheticCheckIDAndName extracts the check ID and display name from a raw
// API response map, handling multiple possible Dash0 API response shapes.
func extractSyntheticCheckIDAndName(check map[string]any) (id, name string) {
	// Try metadata.labels["dash0.com/id"] and metadata.name
	if metadata, ok := check["metadata"].(map[string]any); ok {
		if metaName, ok := metadata["name"].(string); ok {
			name = metaName
		}
		if labels, ok := metadata["labels"].(map[string]any); ok {
			if labelID, ok := labels["dash0.com/id"].(string); ok {
				id = labelID
			}
		}
		// Try display name from spec.plugin.display.name
		if spec, ok := check["spec"].(map[string]any); ok {
			if plugin, ok := spec["plugin"].(map[string]any); ok {
				if display, ok := plugin["display"].(map[string]any); ok {
					if displayName, ok := display["name"].(string); ok && displayName != "" {
						name = displayName
					}
				}
			}
		}
	}

	// Fallback: top-level "id" and "name" fields (flat API format)
	if id == "" {
		if topID, ok := check["id"].(string); ok {
			id = topID
		}
	}
	if name == "" {
		if topName, ok := check["name"].(string); ok {
			name = topName
		}
	}

	return id, name
}

func (d *Dash0) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (d *Dash0) Actions() []core.Action {
	return []core.Action{}
}

func (d *Dash0) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

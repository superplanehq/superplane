package newrelic

import (
	"context"
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const installationInstructions = `
To set up New Relic integration:

1. **Select Region**: Choose your New Relic region (US or EU)
2. **Get API Key**: In New Relic, go to Account Settings > API Keys
3. **Create API Key**: Click "Create API key" and give it a name
4. **Copy API Key**: Copy the generated API key (you won't be able to see it again)
5. **Paste API Key**: Paste the API key in the field below

## Usage

- **Report Metric**: Use this action to send custom telemetry (Gauge, Count, Summary) to New Relic.
- **Run NRQL Query**: Use this action to fetch data from New Relic for decision making or reporting.

## Note

The **Report Metric** action requires a **License Key** (Ingest - License) or an **API Key** (User) with ingest permissions. 
If you use a User API Key, it must have the necessary permissions. A License Key is generally recommended for metric ingestion.
The **Run NRQL Query** action requires a **User API Key** (NRAK-) with query permissions.
`

func init() {
	registry.RegisterIntegrationWithWebhookHandler("newrelic", &NewRelic{}, &NewRelicWebhookHandler{})
}

type NewRelic struct{}

type Configuration struct {
	APIKey string `json:"apiKey" mapstructure:"apiKey"`
	Site   string `json:"site" mapstructure:"site"`
}

type Metadata struct {
	Accounts []Account `json:"accounts" mapstructure:"accounts"`
}

func (n *NewRelic) Name() string {
	return "newrelic"
}

func (n *NewRelic) Label() string {
	return "New Relic"
}

func (n *NewRelic) Icon() string {
	return "newrelic"
}

func (n *NewRelic) Description() string {
	return "Monitor and manage your New Relic resources"
}

func (n *NewRelic) Instructions() string {
	return installationInstructions
}

func (n *NewRelic) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "site",
			Label:       "Region",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     "US",
			Description: "Your New Relic data region",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "United States (US)", Value: "US"},
						{Label: "Europe (EU)", Value: "EU"},
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
			Description: "New Relic API key from Account Settings > API Keys",
		},
	}
}

func (n *NewRelic) Components() []core.Component {
	return []core.Component{
		&ReportMetric{},
		&RunNRQLQuery{},
	}
}

func (n *NewRelic) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnIssue{},
	}
}

func (n *NewRelic) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.APIKey == "" {
		return fmt.Errorf("API key is required")
	}

	// Default to US region if not specified or not EU
	if config.Site != "EU" {
		config.Site = "US"
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Validate API key using NerdGraph (GraphQL) identity query
	err = client.ValidateAPIKey(context.Background())
	if err != nil {
		return fmt.Errorf("failed to validate API key: %w", err)
	}

	// Set empty metadata for now - we can fetch accounts later if needed
	ctx.Integration.SetMetadata(Metadata{
		Accounts: []Account{},
	})

	ctx.Integration.Ready()
	return nil
}

func (n *NewRelic) HandleRequest(ctx core.HTTPRequestContext) {
	// Webhooks will be handled by triggers
}

func (n *NewRelic) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (n *NewRelic) Actions() []core.Action {
	return []core.Action{}
}

func (n *NewRelic) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

func (n *NewRelic) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if resourceType != "account" {
		return []core.IntegrationResource{}, nil
	}

	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(metadata.Accounts))
	for _, account := range metadata.Accounts {
		resources = append(resources, core.IntegrationResource{
			Type: "account",
			Name: account.Name,
			ID:   fmt.Sprintf("%d", account.ID),
		})
	}

	return resources, nil
}

type NewRelicWebhookHandler struct{}

func (h *NewRelicWebhookHandler) CompareConfig(a, b any) (bool, error) {
	return false, nil
}

func (h *NewRelicWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	return nil, nil
}

func (h *NewRelicWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	return nil
}

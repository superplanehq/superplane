package newrelic

import (
	"context"
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const installationInstructions = `
To set up New Relic integration:

1. **Select Region**: Choose your New Relic region (US or EU)
2. **Provide API Keys**: You can provide one or both keys depending on which components you need.

## API Keys

New Relic uses two different types of API keys for different purposes:

- **User API Key** (starts with NRAK-): Required for **Run NRQL Query** and **On Issue** trigger. Get it from New Relic > Account Settings > API Keys > Create User Key.
- **License Key** (Ingest - License): Required for **Report Metric** action. Get it from New Relic > Account Settings > API Keys > Create Ingest License Key.

You may provide both keys to enable all components, or just the key(s) for the components you need.
`

func init() {
	registry.RegisterIntegrationWithWebhookHandler("newrelic", &Newrelic{}, &NewrelicWebhookHandler{})
}

type Newrelic struct{}

type Configuration struct {
	UserAPIKey string `json:"userApiKey" mapstructure:"userApiKey"`
	LicenseKey string `json:"licenseKey" mapstructure:"licenseKey"`
	Site       string `json:"site" mapstructure:"site"`
}

type Metadata struct {
	Accounts []Account `json:"accounts" mapstructure:"accounts"`
}

func (n *Newrelic) Name() string {
	return "newrelic"
}

func (n *Newrelic) Label() string {
	return "New Relic"
}

func (n *Newrelic) Icon() string {
	return "newrelic"
}

func (n *Newrelic) Description() string {
	return "Monitor and manage your New Relic resources"
}

func (n *Newrelic) Instructions() string {
	return installationInstructions
}

func (n *Newrelic) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "site",
			Label:       "Region",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     "US",
			Description: "Your newrelic data region",
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
			Name:        "userApiKey",
			Label:       "User API Key",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Sensitive:   true,
			Description: "User API Key (NRAK-...) for NRQL queries and triggers. Get from Account Settings > API Keys.",
		},
		{
			Name:        "licenseKey",
			Label:       "License Key (Ingest)",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Sensitive:   true,
			Description: "Ingest License Key for metric reporting. Get from Account Settings > API Keys.",
		},
	}
}

func (n *Newrelic) Components() []core.Component {
	return []core.Component{
		&ReportMetric{},
		&RunNRQLQuery{},
	}
}

func (n *Newrelic) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnIssue{},
	}
}

func (n *Newrelic) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.UserAPIKey == "" && config.LicenseKey == "" {
		return fmt.Errorf("at least one API key is required: provide a User API Key (for NRQL/triggers) and/or a License Key (for metrics)")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Validate and fetch accounts only when a User API Key is provided
	if config.UserAPIKey != "" {
		err = client.ValidateAPIKey(context.Background())
		if err != nil {
			return fmt.Errorf("failed to validate User API Key: %w", err)
		}

		accounts, err := client.ListAccounts(context.Background())
		if err != nil {
			if ctx.Logger != nil {
				ctx.Logger.Warnf("New Relic: failed to fetch accounts: %v", err)
			}
			accounts = []Account{}
		}

		ctx.Integration.SetMetadata(Metadata{
			Accounts: accounts,
		})
	} else {
		// Only License Key provided — skip NerdGraph validation
		if ctx.Logger != nil {
			ctx.Logger.Info("New Relic: No User API Key provided, skipping NerdGraph validation and account fetching")
		}
		ctx.Integration.SetMetadata(Metadata{
			Accounts: []Account{},
		})
	}

	ctx.Integration.Ready()
	return nil
}

func (n *Newrelic) HandleRequest(ctx core.HTTPRequestContext) {
	// Webhooks will be handled by triggers
}

func (n *Newrelic) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (n *Newrelic) Actions() []core.Action {
	return []core.Action{}
}

func (n *Newrelic) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

func (n *Newrelic) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
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

type NewrelicWebhookHandler struct{}

func (h *NewrelicWebhookHandler) CompareConfig(a, b any) (bool, error) {
	if a == nil && b == nil {
		return true, nil
	}
	return reflect.DeepEqual(a, b), nil
}

func (h *NewrelicWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	return map[string]any{"manual": true}, nil
}

func (h *NewrelicWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	return nil
}

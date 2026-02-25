package newrelic

import (
	"context"
	"fmt"
	"strconv"
	"strings"

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
	return "Newrelic"
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
	// Create the client first — it decrypts config via GetConfig(),
	// trims whitespace, and validates that at least one key is present.
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Validate and fetch accounts only when a User API Key is provided
	if client.UserAPIKey != "" {
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

type WebhookConfiguration struct {
	Account string `json:"account" mapstructure:"account"`
}

type WebhookMetadata struct {
	DestinationID string `json:"destinationID"`
	ChannelID     string `json:"channelID"`
}

type NewrelicWebhookHandler struct{}

func (h *NewrelicWebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA := WebhookConfiguration{}
	configB := WebhookConfiguration{}

	if err := mapstructure.Decode(a, &configA); err != nil {
		return false, err
	}
	if err := mapstructure.Decode(b, &configB); err != nil {
		return false, err
	}

	return configA.Account == configB.Account, nil
}

func (h *NewrelicWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	config := WebhookConfiguration{}
	if err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config); err != nil {
		return nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	accountID, err := strconv.ParseInt(strings.TrimSpace(config.Account), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid account ID %q: %w", config.Account, err)
	}

	webhookURL := ctx.Webhook.GetURL()
	// Mock localhost for local development if not using a tunnel
	if strings.Contains(webhookURL, "localhost") || strings.Contains(webhookURL, "127.0.0.1") {
		return WebhookMetadata{
			DestinationID: "mock-destination-id",
			ChannelID:     "mock-channel-id",
		}, nil
	}

	secretBytes, err := ctx.Webhook.GetSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to get webhook secret: %w", err)
	}
	secret := string(secretBytes)

	destName := "Superplane - Webhook Destination"
	channelName := "Superplane - AI Notification Channel"

	// 1. Destination
	destID, err := client.FindAIDestinationByName(context.Background(), accountID, destName)
	if err != nil {
		return nil, fmt.Errorf("failed to search for destination: %w", err)
	}
	if destID == "" {
		destID, err = client.CreateAIDestination(context.Background(), accountID, destName, webhookURL, secret)
		if err != nil {
			return nil, fmt.Errorf("failed to create AI destination: %w", err)
		}
	}

	// 2. Channel
	channelID, err := client.FindAIChannelByName(context.Background(), accountID, channelName)
	if err != nil {
		return nil, fmt.Errorf("failed to search for channel: %w", err)
	}
	if channelID == "" {
		channelID, err = client.CreateAIChannel(context.Background(), accountID, channelName, destID)
		if err != nil {
			return nil, fmt.Errorf("failed to create AI channel: %w", err)
		}
	}

	return WebhookMetadata{
		DestinationID: destID,
		ChannelID:     channelID,
	}, nil
}

func (h *NewrelicWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	// We might want to keep the destination/channel for other nodes,
	// but the core will only call Cleanup when the last node using this webhook is removed.
	metadata := WebhookMetadata{}
	if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata); err != nil {
		return nil // Non-critical
	}

	if metadata.DestinationID == "" || metadata.DestinationID == "mock-destination-id" {
		return nil
	}

	// For now, we'll leave the Destination/Channel to avoid accidental deletion
	// if they were shared or if we want to avoid recreating them frequently.
	return nil
}

func (h *NewrelicWebhookHandler) Merge(prev, curr any) (any, bool, error) {
	return curr, true, nil
}
